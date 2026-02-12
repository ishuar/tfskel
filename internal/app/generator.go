package app

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"

	"github.com/ishuar/tfskel/internal/config"
	"github.com/ishuar/tfskel/internal/fs"
	"github.com/ishuar/tfskel/internal/logger"
	"github.com/ishuar/tfskel/internal/templates"
	"github.com/ishuar/tfskel/internal/util"
)

var (
	// ErrMetadataKeyNotFound indicates the requested metadata key was not found in template metadata
	ErrMetadataKeyNotFound = errors.New("metadata key not found")

	// Template category constants
	categoryGithub = "github"
)

// extractMetadata extracts JSON metadata from a comment line in format: ## tfskel-metadata: {...}
func extractMetadata(content, metadataKey string) (map[string]string, error) {
	// Look for pattern: ## tfskel-<metadataKey>: {JSON}
	pattern := fmt.Sprintf(`##\s*tfskel-%s:\s*({[^}]*})`, regexp.QuoteMeta(metadataKey))
	re := regexp.MustCompile(pattern)

	matches := re.FindStringSubmatch(content)
	if len(matches) < 2 {
		return nil, fmt.Errorf("%w: %s", ErrMetadataKeyNotFound, metadataKey)
	}

	var metadata map[string]string
	if err := json.Unmarshal([]byte(matches[1]), &metadata); err != nil {
		return nil, fmt.Errorf("failed to parse metadata JSON: %w", err)
	}

	return metadata, nil
}

// buildBackendMetadata creates metadata map for backend.tf
func buildBackendMetadata(bucketName string) map[string]string {
	return map[string]string{
		"bucket": bucketName,
	}
}

// buildVersionsMetadata creates metadata map for versions.tf (terraform version and provider version)
func buildVersionsMetadata(tfVersion, awsProviderVersion string) map[string]string {
	return map[string]string{
		"tf_ver":           tfVersion,
		"aws_provider_ver": awsProviderVersion,
	}
}

// compareMetadata returns true if metadata maps differ, along with list of changes
func compareMetadata(fileMetadata, configMetadata map[string]string) (bool, []string) {
	var changes []string

	// Check for added or changed keys
	for key, configValue := range configMetadata {
		if fileValue, exists := fileMetadata[key]; !exists {
			changes = append(changes, fmt.Sprintf("%s added: %s", key, configValue))
		} else if fileValue != configValue {
			changes = append(changes, fmt.Sprintf("%s changed: %s -> %s", key, fileValue, configValue))
		}
	}

	// Check for removed keys
	for key, fileValue := range fileMetadata {
		if _, exists := configMetadata[key]; !exists {
			changes = append(changes, fmt.Sprintf("%s removed (was: %s)", key, fileValue))
		}
	}

	return len(changes) > 0, changes
}

// compareTags returns true if tag maps differ, along with list of changes
func compareTags(fileTags, configTags map[string]string) (bool, []string) {
	return compareMetadata(fileTags, configTags)
}

// Generator orchestrates the Terraform project generation
type Generator struct {
	config   *config.Config
	fs       fs.FileSystem
	log      *logger.Logger
	renderer *templates.Renderer
}

// NewGenerator creates a new Generator instance
func NewGenerator(cfg *config.Config, filesystem fs.FileSystem, log *logger.Logger) *Generator {
	return &Generator{
		config: cfg,
		fs:     filesystem,
		log:    log,
	}
}

// Run executes the generation process with the provided generation parameters
func (g *Generator) Run(env, region, appDir string) error {
	// Initialize template renderer with custom templates if provided
	var renderer *templates.Renderer
	var err error
	if g.config.TemplatesDir != "" {
		g.log.Infof("Using custom templates from: %s", g.config.TemplatesDir)
		renderer, err = templates.NewRendererWithCustomTemplates(
			g.config.TemplatesDir,
			g.config.ExtraTemplateExtensions,
		)
	} else {
		g.log.Debug("Using default embedded templates")
		renderer, err = templates.NewRenderer()
	}
	if err != nil {
		return fmt.Errorf("failed to initialize template renderer: %w", err)
	}
	g.renderer = renderer

	// Create directory structure: envs/<env>/<region>/<app>
	appPath := filepath.Join("envs", env, region, appDir)

	// Check if directory already exists
	dirExists := g.fs.DirExists(appPath)

	if err := g.fs.MkdirAll(appPath, 0755); err != nil {
		return fmt.Errorf("failed to create directory structure: %w", err)
	}

	// Show appropriate message based on whether directory was created or already existed
	if dirExists {
		g.log.Infof("Directory %s already exists", appPath)
	} else {
		g.log.Successf("Created directory structure: %s", appPath)
	}

	// Generate files from templates
	if err := g.generateFiles(appPath, env, region, appDir); err != nil {
		return err
	}

	// Display success message
	absPath, err := filepath.Abs(appPath)
	if err != nil {
		absPath = appPath
	}
	g.log.Successf("Created directory: %s", absPath)

	return nil
}

// findProjectRoot returns the project root directory (containing envs folder) from an app path
// appPath is in format: envs/<env>/<region>/<app>
func findProjectRoot(appPath string) string { //nolint:unparam // keeping for clarity and future use
	// Navigate up from appPath to find project root
	// Example: envs/dev/us-east-1/myapp -> current directory (project root)
	parts := strings.Split(filepath.ToSlash(appPath), "/")
	if len(parts) > 0 && parts[0] == "envs" {
		// Return current directory (".") which is the project root
		return "."
	}
	// Fallback to current directory
	return "."
}

// determineOutputPath converts template path to output location based on category
// Template paths are like: root/.gitignore.tmpl, tf/backend.tf.tmpl, github/workflow.yaml.tmpl
func (g *Generator) determineOutputPath(tmplPath, appPath string, data *templates.Data) (string, bool) {
	// Normalize path separators
	tmplPath = filepath.ToSlash(tmplPath)
	parts := strings.Split(tmplPath, "/")

	if len(parts) < 2 {
		// Invalid template path format
		return "", false
	}

	category := parts[0]
	fileName := strings.TrimSuffix(parts[len(parts)-1], ".tmpl")

	switch category {
	case "root":
		// Place at project root
		projectRoot := findProjectRoot(appPath)
		return filepath.Join(projectRoot, fileName), true
	case "tf":
		// Place in app directory
		return filepath.Join(appPath, fileName), true
	case categoryGithub:
		// Place in .github/workflows/ directory at project root with dynamic naming
		projectRoot := findProjectRoot(appPath)

		// Check if this is a reusable workflow (no .tmpl extension in original, just .yaml)
		if strings.HasPrefix(fileName, "reusable-") {
			// Reusable workflows keep their original names
			return filepath.Join(projectRoot, ".github", "workflows", fileName), true
		}

		// Generate dynamic workflow name: {{.AppDir}}-{{.Env}}-{{.ShortRegion}}-{lint|terraform}.yaml
		dynamicFileName := g.generateWorkflowFileName(fileName, data)
		return filepath.Join(projectRoot, ".github", "workflows", dynamicFileName), true
	default:
		// Unknown category
		return "", false
	}
}

// generateWorkflowFileName creates dynamic workflow file names based on template data
// Pattern: {{.AppDir}}-{{.Env}}-{{.ShortRegion}}-{lint|terraform}.yaml
// Example: myapp-dev-euc1-lint.yaml, myapp-dev-euc1-terraform.yaml
// If name_template is provided in config, it uses that template instead
func (g *Generator) generateWorkflowFileName(originalFileName string, data *templates.Data) string {
	// Check if custom name template is provided
	if g.config.Generate != nil && g.config.Generate.GithubWorkflows != nil && g.config.Generate.GithubWorkflows.NameTemplate != "" {
		// Use custom template
		nameTemplate := g.config.Generate.GithubWorkflows.NameTemplate

		// Extract workflow type and add to data for template rendering
		workflowType := strings.TrimSuffix(originalFileName, ".yaml")

		// Create a temporary template to render the name
		tmpl, err := template.New("workflow_name").Parse(nameTemplate)
		if err != nil {
			g.log.Warnf("Failed to parse name_template, using default naming: %v", err)
			return g.generateDefaultWorkflowFileName(originalFileName, data)
		}

		// Create extended data map for template (no .Type - it's internal)
		dataMap := map[string]string{
			"AppDir":      data.AppDir,
			"Env":         data.Env,
			"Region":      data.Region,
			"ShortRegion": data.ShortRegion,
		}

		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, dataMap); err != nil {
			g.log.Warnf("Failed to execute name_template, using default naming: %v", err)
			return g.generateDefaultWorkflowFileName(originalFileName, data)
		}

		// Automatically append workflow type and .yaml extension
		return buf.String() + "-" + workflowType + ".yaml"
	}

	// Use default naming
	return g.generateDefaultWorkflowFileName(originalFileName, data)
}

// generateDefaultWorkflowFileName creates the default workflow file name
func (g *Generator) generateDefaultWorkflowFileName(originalFileName string, data *templates.Data) string {
	// Extract the workflow type from the original filename (e.g., "lint.yaml" -> "lint")
	workflowType := strings.TrimSuffix(originalFileName, ".yaml")

	// Build dynamic name: {{.AppDir}}-{{.Env}}-{{.ShortRegion}}-{{workflowType}}.yaml
	dynamicName := fmt.Sprintf("%s-%s-%s-%s.yaml", data.AppDir, data.Env, data.ShortRegion, workflowType)

	return dynamicName
}

// generateFiles iterates over all templates and generates files
// Custom templates override default templates with the same name
// Root-level templates are placed at project root, app-level templates go in the app directory
func (g *Generator) generateFiles(appPath, env, region, appDir string) error {
	g.log.Infof("Generating files in %s...", appPath)

	// Prepare template data
	data, err := g.prepareTemplateData(env, region, appDir)
	if err != nil {
		return err
	}

	// Check and update backend.tf if needed
	if err := g.updateBackendIfNeeded(appPath, data); err != nil {
		return err
	}

	// Check and update versions.tf if needed
	if err := g.updateVersionsIfNeeded(appPath, data); err != nil {
		return err
	}

	// Process all templates
	return g.processTemplates(appPath, data)
}

// prepareTemplateData extracts config values and builds template data
func (g *Generator) prepareTemplateData(env, region, appDir string) (*templates.Data, error) {
	shortRegion := util.TransformRegionName(region)

	// Extract nested config values with nil checks
	awsProviderVersion := "~> 6.0"
	defaultTags := make(map[string]string)
	s3BucketName := "CHANGE_ME_WITH_YOUR_GLOBALLY_UNIQUE_S3_BUCKET_NAME"

	if g.config.Provider != nil && g.config.Provider.AWS != nil {
		if g.config.Provider.AWS.Version != "" {
			awsProviderVersion = g.config.Provider.AWS.Version
		}
		if g.config.Provider.AWS.DefaultTags != nil {
			defaultTags = g.config.Provider.AWS.DefaultTags
		}
	}

	if g.config.Backend != nil && g.config.Backend.S3 != nil && g.config.Backend.S3.BucketName != "" {
		s3BucketName = g.config.Backend.S3.BucketName
	}

	// Build AWS role ARN for terraform workflows
	awsRoleArn := g.buildAWSRoleArn(env)

	// Create initial data for template rendering
	data := &templates.Data{
		Env:                env,
		Region:             region,
		AppDir:             appDir,
		AccountID:          g.config.GetAccountID(env),
		ShortRegion:        shortRegion,
		S3BucketName:       s3BucketName,
		TerraformVersion:   g.config.TerraformVersion,
		AWSProviderVersion: awsProviderVersion,
		DefaultTags:        defaultTags,
		AWSRoleArn:         awsRoleArn,
	}

	// Render bucket_name as a template if it contains Go template syntax
	if strings.Contains(s3BucketName, "{{") {
		renderedBucketName, err := g.renderBucketName(s3BucketName, data)
		if err != nil {
			return nil, fmt.Errorf("failed to render bucket_name template: %w", err)
		}
		data.S3BucketName = renderedBucketName
	}

	return data, nil
}

// buildAWSRoleArn constructs AWS role ARN from config or returns explicit ARN
// Priority: aws_role_arn > aws_role_name > default placeholder
func (g *Generator) buildAWSRoleArn(env string) string {
	if g.config.Generate == nil || g.config.Generate.GithubWorkflows == nil {
		// Return default placeholder
		return fmt.Sprintf("arn:aws:iam::%s:role/REPLACE_WITH_ROLE_TO_ASSUME", g.config.GetAccountID(env))
	}

	workflows := g.config.Generate.GithubWorkflows

	// If explicit ARN is provided, use it
	if workflows.AWSRoleArn != "" {
		return workflows.AWSRoleArn
	}

	// If role name is provided, construct ARN
	if workflows.AWSRoleName != "" {
		return fmt.Sprintf("arn:aws:iam::%s:role/%s", g.config.GetAccountID(env), workflows.AWSRoleName)
	}

	// Return default placeholder
	return fmt.Sprintf("arn:aws:iam::%s:role/REPLACE_WITH_ROLE_TO_ASSUME", g.config.GetAccountID(env))
}

// updateBackendIfNeeded checks and updates backend.tf if bucket_name changed
func (g *Generator) updateBackendIfNeeded(appPath string, data *templates.Data) error {
	backendPath := filepath.Join(appPath, "backend.tf")
	if !g.fs.FileExists(backendPath) {
		return nil
	}

	needsUpdate, err := g.shouldUpdateBackend(backendPath, data.S3BucketName)
	if err != nil {
		return fmt.Errorf("failed to check backend.tf for updates: %w", err)
	}

	if needsUpdate {
		if err := g.updateBackendFile(backendPath, data); err != nil {
			return fmt.Errorf("failed to update backend.tf: %w", err)
		}
		g.log.Successf("Updated backend.tf with new bucket_name: %s", data.S3BucketName)
	}

	return nil
}

// updateVersionsIfNeeded checks and updates versions.tf if configuration changed
func (g *Generator) updateVersionsIfNeeded(appPath string, data *templates.Data) error {
	versionsPath := filepath.Join(appPath, "versions.tf")
	if !g.fs.FileExists(versionsPath) {
		return nil
	}

	needsUpdate, changes, err := g.shouldUpdateVersions(versionsPath, data)
	if err != nil {
		return fmt.Errorf("failed to check versions.tf for updates: %w", err)
	}

	if needsUpdate {
		if err := g.updateVersionsFile(versionsPath, data); err != nil {
			return fmt.Errorf("failed to update versions.tf: %w", err)
		}
		for _, change := range changes {
			g.log.Successf("Updated versions.tf - %s", change)
		}
	}

	return nil
}

// processTemplates iterates through templates and generates files
func (g *Generator) processTemplates(appPath string, data *templates.Data) error {
	allTemplates := g.renderer.GetTemplateNames()

	for _, tmplPath := range allTemplates {
		if err := g.processTemplate(tmplPath, appPath, data); err != nil {
			return err
		}
	}

	return nil
}

// processTemplate handles generation of a single template file
func (g *Generator) processTemplate(tmplPath, appPath string, data *templates.Data) error {
	// Normalize path and check if it's a root template
	normalizedPath := filepath.ToSlash(tmplPath)
	parts := strings.Split(normalizedPath, "/")

	// Skip root templates - they are only handled by init command
	if len(parts) > 0 && parts[0] == "root" {
		g.log.Debugf("Skipping root template (init only): %s", tmplPath)
		return nil
	}

	// Skip github templates if create_github_workflows is not enabled
	if len(parts) > 0 && parts[0] == categoryGithub {
		if g.config.Generate == nil || g.config.Generate.GithubWorkflows == nil || !g.config.Generate.GithubWorkflows.Create {
			g.log.Debugf("Skipping github template (create-github-workflows not enabled): %s", tmplPath)
			return nil
		}
	}

	// Create a copy of data for this template to avoid modifying shared data
	templateData := *data

	// For github workflow templates (.tmpl files), compute and inject the workflow filename
	if len(parts) > 0 && parts[0] == categoryGithub && strings.HasSuffix(tmplPath, ".tmpl") {
		// Extract the original filename (e.g., "lint.yaml.tmpl" -> "lint.yaml")
		fileName := parts[len(parts)-1]
		fileName = strings.TrimSuffix(fileName, ".tmpl")

		// Check if this is NOT a reusable workflow
		if !strings.HasPrefix(fileName, "reusable-") {
			// Generate the workflow filename that will be created
			workflowFileName := g.generateWorkflowFileName(fileName, &templateData)
			// Inject it into template data for self-reference
			templateData.WorkflowFileName = workflowFileName
		}
	}

	// Determine output path
	outputPath, valid := g.determineOutputPath(tmplPath, appPath, &templateData)
	if !valid {
		g.log.Debugf("Skipping template with invalid path format: %s", tmplPath)
		return nil
	}

	// Skip if file already exists
	outputName := filepath.Base(outputPath)
	if g.fs.FileExists(outputPath) {
		g.log.Infof("%s already exists, skipping", outputName)
		return nil
	}

	// Ensure parent directory exists
	outputDir := filepath.Dir(outputPath)
	if err := g.fs.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory for %s: %w", outputName, err)
	}

	// Render and write template (use templateData which contains computed values)
	content, err := g.renderer.Render(tmplPath, &templateData)
	if err != nil {
		g.log.Infof("Skipping %s: failed to render: %v", outputName, err)
		return nil
	}

	if err := g.fs.WriteFile(outputPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", outputName, err)
	}

	// Log success
	templateSource := g.renderer.GetTemplateSource(tmplPath)
	if templateSource == "" {
		templateSource = tmplPath
	}
	g.log.Successf("Created %s from %s", outputName, templateSource)

	return nil
}

// renderBucketName renders a bucket name template with context variables
func (g *Generator) renderBucketName(bucketTemplate string, data *templates.Data) (string, error) {
	tmpl, err := template.New("bucket_name").Parse(bucketTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse bucket_name template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute bucket_name template: %w", err)
	}

	return buf.String(), nil
}

// shouldUpdateBackend checks if the backend.tf needs updating due to bucket_name changes
func (g *Generator) shouldUpdateBackend(backendPath string, expectedBucketName string) (bool, error) {
	content, err := g.fs.ReadFile(backendPath)
	if err != nil {
		return false, fmt.Errorf("failed to read backend.tf: %w", err)
	}

	// Extract metadata from file
	fileMetadata, err := extractMetadata(string(content), "metadata")
	if err != nil {
		// If no metadata found, file was not generated by tfskel with metadata support
		// This is expected for old files, so return true (needs update) but no error
		g.log.Debug("No metadata found in backend.tf, will regenerate to add metadata")
		return true, nil //nolint:nilerr // missing metadata is expected for old files, not an error
	}

	// Compare metadata with expected values
	configMetadata := buildBackendMetadata(expectedBucketName)
	needsUpdate, _ := compareMetadata(fileMetadata, configMetadata)

	return needsUpdate, nil
}

// shouldUpdateVersions checks if the versions.tf needs updating due to terraform_version,
// provider version, or default_tags changes
func (g *Generator) shouldUpdateVersions(versionsPath string, data *templates.Data) (bool, []string, error) {
	content, err := g.fs.ReadFile(versionsPath)
	if err != nil {
		return false, nil, fmt.Errorf("failed to read versions.tf: %w", err)
	}

	contentStr := string(content)
	var allChanges []string

	// Extract and compare metadata (terraform version and provider version)
	fileMetadata, err := extractMetadata(contentStr, "metadata")
	if err != nil {
		// If no metadata found, file was not generated by tfskel with metadata support
		// This is expected for old files, so return true (needs update) but no error
		g.log.Debug("No metadata found in versions.tf, will regenerate to add metadata")
		return true, []string{"versions.tf needs metadata initialization"}, nil //nolint:nilerr // missing metadata is expected for old files, not an error
	}

	configMetadata := buildVersionsMetadata(data.TerraformVersion, data.AWSProviderVersion)
	needsUpdate, changes := compareMetadata(fileMetadata, configMetadata)
	if needsUpdate {
		allChanges = append(allChanges, changes...)
	}

	// Extract and compare tags
	fileTags, err := extractMetadata(contentStr, "tags")
	if err != nil {
		// No tags metadata found, check if config has tags
		if len(data.DefaultTags) > 0 {
			allChanges = append(allChanges, "default_tags added")
		}
	} else {
		// Compare tags
		tagsChanged, tagChanges := compareTags(fileTags, data.DefaultTags)
		if tagsChanged {
			// Add all tag change messages
			allChanges = append(allChanges, tagChanges...)
		}
	}

	return len(allChanges) > 0, allChanges, nil
}

// updateBackendFile regenerates the backend.tf file with updated configuration
func (g *Generator) updateBackendFile(backendPath string, data *templates.Data) error {
	// Find backend.tf.tmpl template in tf/ category
	templateName := "tf/backend.tf.tmpl"

	// Render template with updated data
	content, err := g.renderer.Render(templateName, data)
	if err != nil {
		return fmt.Errorf("failed to render backend template: %w", err)
	}

	// Write updated file
	if err := g.fs.WriteFile(backendPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write backend.tf: %w", err)
	}

	return nil
}

// updateVersionsFile regenerates the versions.tf file with updated configuration
func (g *Generator) updateVersionsFile(versionsPath string, data *templates.Data) error {
	// Find versions.tf.tmpl template in tf/ category
	templateName := "tf/versions.tf.tmpl"

	// Render template with updated data
	content, err := g.renderer.Render(templateName, data)
	if err != nil {
		return fmt.Errorf("failed to render versions template: %w", err)
	}

	// Write updated file
	if err := g.fs.WriteFile(versionsPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write versions.tf: %w", err)
	}

	return nil
}
