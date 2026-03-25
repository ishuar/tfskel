package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/ishuar/tfskel/internal/config"
	"github.com/ishuar/tfskel/internal/fs"
	"github.com/ishuar/tfskel/internal/logger"
	"github.com/ishuar/tfskel/internal/templates"
	"github.com/ishuar/tfskel/internal/util"
)

// Template category and name constants
const (
	categoryGithub = "github"
	tmplBackendTF  = "tf/backend.tf.tmpl"
	tmplVersionsTF = "tf/versions.tf.tmpl"
)

var (
	// ErrMetadataKeyNotFound indicates the requested metadata key was not found in template metadata
	ErrMetadataKeyNotFound = errors.New("metadata key not found")

	// ErrInvalidWorkflowFileName indicates the name_template produced an invalid filename
	ErrInvalidWorkflowFileName = errors.New("name_template produced invalid filename")
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

	// Normalize keys to lowercase for tags metadata (Terraform convention)
	// This is needed when READING from files to ensure consistent comparison
	// with tags from config (which are also normalized in prepareTemplateData)
	if metadataKey == "tags" {
		normalized := make(map[string]string, len(metadata))
		for k, v := range metadata {
			normalized[strings.ToLower(k)] = v
		}
		return normalized, nil
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

// metadataCommentsForTemplate returns the tfskel-metadata and tfskel-tags-hash comment lines
// for known template types. Returns empty strings for templates that don't need metadata injection.
func metadataCommentsForTemplate(tmplName string, data *templates.Data, fileExt string) (metadataComment, tagsHashComment string) {
	switch tmplName {
	case tmplBackendTF:
		meta := buildBackendMetadata(data.S3BucketName)
		return BuildMetadataComment(meta, fileExt), ""
	case tmplVersionsTF:
		meta := buildVersionsMetadata(data.TerraformVersion, data.AWSProviderVersion)
		return BuildMetadataComment(meta, fileExt), BuildTagsHashComment(data.DefaultTags, fileExt)
	default:
		return "", ""
	}
}

// Generator orchestrates the Terraform project generation
//
// Thread-safety: Generator is designed for single-threaded CLI usage.
// The config field is read-only after initialization via NewGenerator.
// The renderer field is set once during Run() and then only read.
// Concurrent use of the same Generator instance is not supported.
type Generator struct {
	config   *config.Config
	fs       fs.FileSystem
	log      *logger.Logger
	renderer *templates.Renderer
	upgrade  bool // When true, re-render files whose template has changed
	force    bool // When true (with upgrade), overwrite files even without source markers
}

// NewGenerator creates a new Generator instance
func NewGenerator(cfg *config.Config, filesystem fs.FileSystem, log *logger.Logger) *Generator {
	return &Generator{
		config: cfg,
		fs:     filesystem,
		log:    log,
	}
}

// SetUpgrade enables the upgrade mode, optionally with force.
// When upgrade is true, files whose source template has changed are re-rendered.
// When force is also true, files without source markers are overwritten unconditionally.
func (g *Generator) SetUpgrade(upgrade, force bool) {
	g.upgrade = upgrade
	g.force = force
}

// initRenderer initializes the template renderer, using custom templates if configured.
func (g *Generator) initRenderer() error {
	var (
		renderer *templates.Renderer
		err      error
	)
	// Defensive: check Templates is initialized (always true from config.Load, but defensive for direct usage)
	if g.config.Templates != nil && g.config.Templates.Dir != "" {
		g.log.Infof("Using custom templates from: %s", g.config.Templates.Dir)
		renderer, err = templates.NewRendererWithCustomTemplates(g.config.Templates.Dir)
	} else {
		g.log.Debug("Using default embedded templates")
		renderer, err = templates.NewRenderer()
	}
	if err != nil {
		return fmt.Errorf("failed to initialize template renderer: %w", err)
	}
	g.renderer = renderer
	return nil
}

// Run executes the generation process with the provided generation parameters
func (g *Generator) Run(env, region, appDir string) error {
	if err := g.initRenderer(); err != nil {
		return err
	}

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

// RunWorkflows generates per-environment GitHub workflow files.
// It processes only github/*.tmpl files (dynamic, env-specific workflows).
// Static reusable workflows are generated by the init command instead.
func (g *Generator) RunWorkflows(env string) error {
	// Check if workflows are enabled
	if g.config.Workflows == nil || !g.config.Workflows.Create {
		g.log.Info("Workflow generation is disabled (workflows.create: false in config)")
		return nil
	}

	if err := g.initRenderer(); err != nil {
		return err
	}

	// Get account ID for the environment
	accountID, err := g.config.GetAccountID(env)
	if err != nil {
		return err
	}

	// Build minimal template data (env + accountID are sufficient for workflow templates)
	data := &templates.Data{
		Env:       env,
		AccountID: accountID,
	}

	// Build AWS role ARN
	awsRoleArn, err := g.buildAWSRoleArn(data)
	if err != nil {
		return fmt.Errorf("failed to build AWS role ARN: %w", err)
	}
	data.AWSRoleArn = awsRoleArn

	g.log.Infof("Generating workflow files for environment: %s", env)

	// Process only github .tmpl files
	for _, tmplPath := range g.renderer.GetTemplateNames() {
		normalizedPath := filepath.ToSlash(tmplPath)
		if !strings.HasPrefix(normalizedPath, categoryGithub+"/") || !strings.HasSuffix(tmplPath, ".tmpl") {
			continue // static files go via init
		}

		templateData := *data
		if err := g.enrichTemplateDataForWorkflow(tmplPath, &templateData); err != nil {
			return err
		}

		outputPath, valid := g.determineOutputPath(tmplPath, "", &templateData)
		if !valid {
			g.log.Debugf("Skipping template with invalid path format: %s", tmplPath)
			continue
		}

		if g.fs.FileExists(outputPath) {
			if g.upgrade {
				if err := g.upgradeFileIfEligible(tmplPath, outputPath, &templateData); err != nil {
					return err
				}
			} else {
				g.log.Infof("%s already exists, skipping", outputPath)
			}
			continue
		}

		if err := g.writeTemplateFile(tmplPath, outputPath, outputPath, &templateData); err != nil {
			return err
		}
	}

	return nil
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
		projectRoot := "."
		return filepath.Join(projectRoot, fileName), true
	case "tf":
		// Place in app directory
		return filepath.Join(appPath, fileName), true
	case categoryGithub:
		// Place in .github/workflows/ directory at project root
		projectRoot := "."

		// Only .tmpl files get dynamic naming (e.g. terraform-plan-apply.yaml.tmpl → dev-terraform-plan-apply.yaml).
		// Static files (reusable workflows, lint.yaml, etc.) keep their original names.
		isTemplate := strings.HasSuffix(parts[len(parts)-1], ".tmpl")
		if !isTemplate {
			return filepath.Join(projectRoot, ".github", "workflows", fileName), true
		}

		// Generate dynamic workflow name: e.g. dev-terraform-plan-apply.yaml
		dynamicFileName, err := g.generateWorkflowFileName(fileName, data)
		if err != nil {
			g.log.Errorf("Failed to generate workflow filename: %v", err)
			return "", false
		}
		return filepath.Join(projectRoot, ".github", "workflows", dynamicFileName), true
	default:
		// Unknown category
		return "", false
	}
}

// sanitizeWorkflowFileName validates and sanitizes a workflow filename to prevent path traversal
// Returns the sanitized filename and a boolean indicating if it's valid
func sanitizeWorkflowFileName(filename string) (string, bool) {
	// Reject paths containing dangerous characters or patterns BEFORE cleaning
	if strings.Contains(filename, "..") || strings.ContainsRune(filename, filepath.Separator) || strings.Contains(filename, "/") || strings.Contains(filename, "\\") {
		return "", false
	}

	// Reject empty or just extension
	if filename == "" || filename == ".yaml" || filename == "." {
		return "", false
	}

	// Additional validation: ensure the cleaned version matches the input
	cleaned := filepath.Clean(filename)
	if cleaned != filename {
		return "", false
	}

	return filename, true
}

// generateWorkflowFileName creates dynamic workflow file names based on template data.
// Default pattern: {env}-terraform-plan-apply.yaml (e.g., dev-terraform-plan-apply.yaml)
// If name is provided in config, it uses that as a plain string (no Go template rendering).
// Pattern with name: {env}-{name}.yaml (e.g., dev-my-terraform.yaml)
// Returns error if name contains Go template syntax or produces invalid filename.
func (g *Generator) generateWorkflowFileName(originalFileName string, data *templates.Data) (string, error) {
	// Default path: no custom name configured
	if g.config.Workflows == nil ||
		g.config.Workflows.Name == "" {
		return g.generateDefaultWorkflowFileName(originalFileName, data), nil
	}

	name := g.config.Workflows.Name

	// Reject Go template syntax — name must be a plain string
	if strings.Contains(name, "{{") || strings.Contains(name, "}}") {
		return "", fmt.Errorf("%w: name must be a plain string without Go template syntax (e.g. 'my-terraform'), got: %s", ErrInvalidWorkflowFileName, name)
	}

	// Normalize: strip trailing .yaml if user included it
	name, _ = strings.CutSuffix(name, ".yaml")

	// Build filename: {env}-{name}.yaml
	baseFileName := data.Env + "-" + name + ".yaml"

	// Validate and sanitize the filename to prevent path traversal
	sanitized, valid := sanitizeWorkflowFileName(baseFileName)
	if !valid {
		return "", fmt.Errorf("%w (potential path traversal): %s", ErrInvalidWorkflowFileName, baseFileName)
	}

	return sanitized, nil
}

// generateDefaultWorkflowFileName creates the default workflow file name
// Pattern: {env}-{workflowType}.yaml (e.g., dev-terraform-plan-apply.yaml, dev-terraform-destroy.yaml)
// The workflowType is derived from the template filename by stripping .yaml
// (e.g., "terraform-plan-apply.yaml" -> "terraform-plan-apply")
func (g *Generator) generateDefaultWorkflowFileName(originalFileName string, data *templates.Data) string {
	// Extract the workflow type from the original filename
	// e.g., "terraform-plan-apply.yaml" -> "terraform-plan-apply"
	workflowType := strings.TrimSuffix(originalFileName, ".yaml")

	return fmt.Sprintf("%s-%s.yaml", data.Env, workflowType)
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
			// Normalize tag keys to lowercase for Terraform compatibility
			// This is needed when WRITING to files to ensure consistent tag format
			// (matches the normalization in extractMetadata for comparison)
			for k, v := range g.config.Provider.AWS.DefaultTags {
				defaultTags[strings.ToLower(k)] = v
			}
		}
	}

	if g.config.Backend != nil && g.config.Backend.S3 != nil && g.config.Backend.S3.BucketName != "" {
		s3BucketName = g.config.Backend.S3.BucketName
	}

	// Get account ID for the environment
	accountID, err := g.config.GetAccountID(env)
	if err != nil {
		return nil, err
	}

	// Create initial data for template rendering
	data := &templates.Data{
		Env:                env,
		Region:             region,
		AppDir:             appDir,
		AccountID:          accountID,
		ShortRegion:        shortRegion,
		S3BucketName:       s3BucketName,
		TerraformVersion:   g.config.TerraformVersion,
		AWSProviderVersion: awsProviderVersion,
		DefaultTags:        defaultTags,
	}

	// Build AWS role ARN for terraform workflows (needs full context for templates)
	awsRoleArn, err := g.buildAWSRoleArn(data)
	if err != nil {
		return nil, fmt.Errorf("failed to build AWS role ARN: %w", err)
	}
	data.AWSRoleArn = awsRoleArn

	// Render bucket_name as a template if it contains Go template syntax
	renderedBucketName, err := g.renderer.RenderConfigValue(s3BucketName, "bucket_name", data)
	if err != nil {
		return nil, fmt.Errorf("failed to render bucket_name template: %w", err)
	}
	data.S3BucketName = renderedBucketName

	return data, nil
}

// buildAWSRoleArn constructs AWS role ARN from config or returns explicit ARN
// Priority: aws_role_arn > aws_role_name > default placeholder
// Accepts full template data to allow flexible template variables in role names/ARNs
func (g *Generator) buildAWSRoleArn(data *templates.Data) (string, error) {
	// Defensive: check Workflows is initialized
	if g.config.Workflows == nil {
		// Return default placeholder
		return fmt.Sprintf("arn:aws:iam::%s:role/REPLACE_WITH_ROLE_TO_ASSUME", data.AccountID), nil
	}

	workflows := g.config.Workflows

	// Create template data once for reuse in both ARN and role name rendering
	// This provides all available fields for maximum flexibility in templates
	templateData := &templates.Data{
		Env:         data.Env,
		Region:      data.Region,
		AppDir:      data.AppDir,
		AccountID:   data.AccountID,
		ShortRegion: data.ShortRegion,
	}

	// If explicit ARN is provided, render it as a template and use it
	if workflows.AWSRoleArn != "" {
		renderedArn, err := g.renderer.RenderConfigValue(workflows.AWSRoleArn, "aws_role_arn", templateData)
		if err != nil {
			return "", fmt.Errorf("invalid template syntax in aws_role_arn: %w", err)
		}
		return renderedArn, nil
	}

	// If role name is provided, render it as a template and construct ARN
	if workflows.AWSRoleName != "" {
		renderedRoleName, err := g.renderer.RenderConfigValue(workflows.AWSRoleName, "aws_role_name", templateData)
		if err != nil {
			return "", fmt.Errorf("invalid template syntax in aws_role_name: %w", err)
		}
		return fmt.Sprintf("arn:aws:iam::%s:role/%s", data.AccountID, renderedRoleName), nil
	}

	// Return default placeholder
	return fmt.Sprintf("arn:aws:iam::%s:role/REPLACE_WITH_ROLE_TO_ASSUME", data.AccountID), nil
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

// shouldSkipTemplate determines if a template should be skipped during generation
// Returns true if template should be skipped, along with the reason
func (g *Generator) shouldSkipTemplate(tmplPath string) (bool, string) {
	normalizedPath := filepath.ToSlash(tmplPath)
	parts := strings.Split(normalizedPath, "/")

	if len(parts) == 0 {
		return true, "invalid path format"
	}

	category := parts[0]

	// Skip root templates - they are only handled by init command
	if category == "root" {
		return true, "root template (init only)"
	}

	// Skip github templates - static ones are handled by 'tfskel init',
	// per-env templates are handled by 'tfskel scaffold workflows --env'
	if category == categoryGithub {
		return true, "github template (use 'tfskel init' or 'tfskel scaffold workflows --env')"
	}

	return false, ""
}

// enrichTemplateDataForWorkflow adds workflow-specific data to template context
// For github workflow templates, computes and injects the workflow filename
func (g *Generator) enrichTemplateDataForWorkflow(tmplPath string, data *templates.Data) error {
	normalizedPath := filepath.ToSlash(tmplPath)
	parts := strings.Split(normalizedPath, "/")

	if len(parts) == 0 {
		return nil
	}

	// Only process github workflow templates (.tmpl files)
	if parts[0] != categoryGithub || !strings.HasSuffix(tmplPath, ".tmpl") {
		return nil
	}

	// Extract the original filename
	fileName := parts[len(parts)-1]
	fileName = strings.TrimSuffix(fileName, ".tmpl")

	// Skip reusable workflows
	if strings.HasPrefix(fileName, "reusable-") {
		return nil
	}

	// Generate and inject workflow filename
	workflowFileName, err := g.generateWorkflowFileName(fileName, data)
	if err != nil {
		return fmt.Errorf("failed to generate workflow filename: %w", err)
	}

	data.WorkflowFileName = workflowFileName
	return nil
}

// sourceComment returns the source marker comment for the given template and output path,
// or an empty string if the file does not support markers.
func sourceComment(renderer *templates.Renderer, tmplName, outputPath string) string {
	hash := renderer.GetTemplateHash(tmplName)
	if hash == "" {
		return ""
	}
	return SourceCommentForFile(tmplName, hash, outputPath)
}

// writeTemplateFile creates the directory structure, renders the template with all markers, and writes the result.
// Render failures are logged and skipped (returns nil) since this is used during initial generation
// where a single template failure should not block the entire run.
func (g *Generator) writeTemplateFile(tmplPath, outputPath, outputName string, data *templates.Data) error {
	// Ensure parent directory exists
	outputDir := filepath.Dir(outputPath)
	if err := g.fs.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory for %s: %w", outputName, err)
	}

	if err := g.renderAndWriteFile(tmplPath, outputPath, data); err != nil {
		g.log.Infof("Skipping %s: %v", outputName, err)
		return nil
	}

	// Log success
	templateSource := g.renderer.GetTemplateSource(tmplPath)
	if templateSource == "" {
		templateSource = tmplPath
	}
	g.log.Successf("Created %s from %s", outputName, templateSource)

	return nil
}

// upgradeFileIfEligible checks whether an existing file should be upgraded from its template.
// It reads the file's source marker, re-renders the template, and compares the full rendered
// content against the existing file. This detects both template source changes and manual edits
// to the rendered file (content drift).
// Returns nil if the file was skipped or upgraded successfully.
func (g *Generator) upgradeFileIfEligible(tmplPath, outputPath string, data *templates.Data) error {
	// Check upgrade whitelist
	if !g.isUpgradeEligible(tmplPath) {
		g.log.Debugf("Template %s not in upgrade list, skipping", tmplPath)
		return nil
	}

	// Read existing file and extract source marker
	content, err := g.fs.ReadFile(outputPath)
	if err != nil {
		return fmt.Errorf("failed to read %s for upgrade check: %w", outputPath, err)
	}

	marker, err := ExtractSourceMarker(string(content))
	if err != nil {
		if errors.Is(err, ErrSourceMarkerNotFound) {
			if g.force {
				g.log.Infof("Upgrading %s (--force, no source marker)", outputPath)
				return g.renderAndWriteFile(tmplPath, outputPath, data)
			}
			g.log.Infof("%s has no source marker, skipping upgrade (use --force to override)", outputPath)
			return nil
		}
		// Malformed source marker (e.g. invalid JSON)
		if g.force {
			g.log.Infof("Upgrading %s (--force, invalid source marker: %v)", outputPath, err)
			return g.renderAndWriteFile(tmplPath, outputPath, data)
		}
		return fmt.Errorf("invalid source marker in %s: %w", outputPath, err)
	}

	// Verify the marker template matches this template
	if marker.Template != tmplPath {
		g.log.Debugf("%s was generated from %s, not %s, skipping", outputPath, marker.Template, tmplPath)
		return nil
	}

	// Re-render the template with markers and compare to detect any drift
	rendered, err := g.renderWithMarkers(tmplPath, outputPath, data)
	if err != nil {
		return err
	}

	if rendered == string(content) {
		g.log.Debugf("%s is up to date, skipping", outputPath)
		return nil
	}

	// Content differs, re-render
	g.log.Infof("Upgrading %s (content drift detected)", outputPath)
	if err := g.renderAndWriteFile(tmplPath, outputPath, data); err != nil {
		return err
	}
	g.log.Successf("Upgraded %s", outputPath)
	return nil
}

// isUpgradeEligible checks if a template is eligible for upgrade based on the config whitelist.
// If no whitelist is configured, all templates are eligible.
func (g *Generator) isUpgradeEligible(tmplPath string) bool {
	if g.config.Templates == nil || len(g.config.Templates.Upgrade) == 0 {
		return true // no whitelist = all eligible
	}
	baseName := filepath.Base(tmplPath)
	return slices.Contains(g.config.Templates.Upgrade, baseName)
}

// processTemplate handles generation of a single template file
func (g *Generator) processTemplate(tmplPath, appPath string, data *templates.Data) error {
	// Check if template should be skipped
	shouldSkip, reason := g.shouldSkipTemplate(tmplPath)
	if shouldSkip {
		g.log.Debugf("Skipping %s: %s", tmplPath, reason)
		return nil
	}

	// Create a copy of data for this template
	templateData := *data

	// Enrich data for workflow templates
	if err := g.enrichTemplateDataForWorkflow(tmplPath, &templateData); err != nil {
		g.log.Errorf("Failed to enrich template data for %s: %v", tmplPath, err)
		return err
	}

	// Determine output path
	outputPath, valid := g.determineOutputPath(tmplPath, appPath, &templateData)
	if !valid {
		g.log.Debugf("Skipping template with invalid path format: %s", tmplPath)
		return nil
	}

	// Handle existing files
	if g.fs.FileExists(outputPath) {
		if g.upgrade {
			return g.upgradeFileIfEligible(tmplPath, outputPath, &templateData)
		}
		g.log.Infof("%s already exists, skipping", outputPath)
		return nil
	}

	// Create output directory and write file
	return g.writeTemplateFile(tmplPath, outputPath, outputPath, &templateData)
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
		g.log.Debugf("No metadata found in versions.tf, will regenerate with: terraform_version=%s, aws_provider_version=%s",
			data.TerraformVersion, data.AWSProviderVersion)
		return true, []string{"initialized configuration tracking (see details with -v flag)"}, nil //nolint:nilerr // missing metadata is expected for old files, not an error
	}

	configMetadata := buildVersionsMetadata(data.TerraformVersion, data.AWSProviderVersion)
	needsUpdate, changes := compareMetadata(fileMetadata, configMetadata)
	if needsUpdate {
		allChanges = append(allChanges, changes...)
	}

	// Extract and compare tags hash
	fileTagsHash, err := ExtractTagsHash(contentStr)
	configTagsHash := ComputeTagsHash(data.DefaultTags)
	if err != nil {
		// No tags hash found — could be old file with JSON-based tfskel-tags or no tags at all
		if len(data.DefaultTags) > 0 {
			allChanges = append(allChanges, "initialized tags tracking")
		}
	} else if fileTagsHash != configTagsHash {
		// Hash mismatch — tags have changed
		allChanges = append(allChanges, "default_tags changed")
	}

	return len(allChanges) > 0, allChanges, nil
}

// renderWithMarkers renders a template and injects source and metadata marker comments.
func (g *Generator) renderWithMarkers(tmplName, outputPath string, data *templates.Data) (string, error) {
	content, err := g.renderer.Render(tmplName, data)
	if err != nil {
		return "", fmt.Errorf("failed to render %s: %w", tmplName, err)
	}

	if comment := sourceComment(g.renderer, tmplName, outputPath); comment != "" {
		content = InjectSourceMarker(content, comment)
	}

	ext := filepath.Ext(outputPath)
	metaComment, tagsHashComment := metadataCommentsForTemplate(tmplName, data, ext)
	if metaComment != "" || tagsHashComment != "" {
		content = InjectMetadataMarkers(content, metaComment, tagsHashComment)
	}

	return content, nil
}

// renderAndWriteFile renders a template with all markers and writes it to outputPath.
func (g *Generator) renderAndWriteFile(tmplName, outputPath string, data *templates.Data) error {
	content, err := g.renderWithMarkers(tmplName, outputPath, data)
	if err != nil {
		return err
	}

	if err := g.fs.WriteFile(outputPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", outputPath, err)
	}

	return nil
}

// updateBackendFile regenerates the backend.tf file with updated configuration
func (g *Generator) updateBackendFile(backendPath string, data *templates.Data) error {
	return g.renderAndWriteFile(tmplBackendTF, backendPath, data)
}

// updateVersionsFile regenerates the versions.tf file with updated configuration
func (g *Generator) updateVersionsFile(versionsPath string, data *templates.Data) error {
	return g.renderAndWriteFile(tmplVersionsTF, versionsPath, data)
}
