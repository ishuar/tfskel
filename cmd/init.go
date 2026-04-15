package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ishuar/tfskel/internal/config"
	"github.com/ishuar/tfskel/internal/fs"
	"github.com/ishuar/tfskel/internal/generate"
	"github.com/ishuar/tfskel/internal/logger"
	"github.com/ishuar/tfskel/internal/templates"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.yaml.in/yaml/v4"
)

var (
	// ErrMissingAccountMapping indicates AWS account mapping configuration is missing
	ErrMissingAccountMapping = errors.New("provider.aws.account_mapping is missing or empty")
	// ErrForceRequiresUpgrade indicates --force was used without --upgrade
	ErrForceRequiresUpgrade = errors.New("--force can only be used together with --upgrade")
	// ErrScaffoldForceRequiresUpgrade indicates --force was used without --upgrade or --upgrade-all in scaffold
	ErrScaffoldForceRequiresUpgrade = errors.New("--force can only be used together with --upgrade or --upgrade-all")
	// ErrUpgradeAllWithAppDir indicates --upgrade-all was used with a positional <app-dir> argument
	ErrUpgradeAllWithAppDir = errors.New("cannot specify <app-dir> when --upgrade-all is set")
	// ErrUpgradeAllWithUpgrade indicates --upgrade-all and --upgrade were both set
	ErrUpgradeAllWithUpgrade = errors.New("--upgrade-all and --upgrade are mutually exclusive")
	// ErrSkipRequiresUpgradeAll indicates --skip was used without --upgrade-all
	ErrSkipRequiresUpgradeAll = errors.New("--skip can only be used with --upgrade-all")
	// ErrInitSkipRequiresUpgrade indicates --skip was used without --upgrade in init
	ErrInitSkipRequiresUpgrade = errors.New("--skip can only be used with --upgrade")
	// ErrBaseDirNotExist indicates the envs/<env>/<region>/ directory does not exist
	ErrBaseDirNotExist = errors.New("base directory does not exist; check --env and --region values")
	// ErrNoAppDirsFound indicates no app directories were found for --upgrade-all
	ErrNoAppDirsFound = errors.New("no app directories found")
)

const (
	defaultTerraformVersion = "1.13.1"
)

// initManagedFile pairs a target filename with its source template.
type initManagedFile struct {
	filename     string
	templateName string
}

// rootConfigFiles lists root config files managed by init (rendered with nil data).
var rootConfigFiles = []initManagedFile{
	{".pre-commit-config.yaml", "root/.pre-commit-config.yaml.tmpl"},
	{".tflint.hcl", "root/.tflint.hcl.tmpl"},
	{"trivy.yaml", "root/trivy.yaml.tmpl"},
}

// initRunner bundles dependencies for init command file operations.
type initRunner struct {
	fs       fs.FileSystem
	log      *logger.Logger
	renderer *templates.Renderer
	upgrade  bool
	force    bool
	dryRun   bool
	skip     map[string]bool
}

func newInitRunner(filesystem fs.FileSystem, log *logger.Logger, upgrade, force, dryRun bool, skip map[string]bool) (*initRunner, error) {
	renderer, err := templates.NewRenderer()
	if err != nil {
		return nil, fmt.Errorf("failed to create renderer: %w", err)
	}
	return &initRunner{
		fs:       filesystem,
		log:      log,
		renderer: renderer,
		upgrade:  upgrade,
		force:    force,
		dryRun:   dryRun,
		skip:     skip,
	}, nil
}

var initCmd = &cobra.Command{
	Use:     "init",
	GroupID: "main",
	Short:   "Initialize tfskel project structure",
	Long: `Initializes a new Terraform monorepo with an environment-and-region-based
directory layout with sensible defaults already in place.

Generates a .mise.toml file declaring all required tools (terraform, tflint,
trivy, pre-commit, awscli) so they can be installed with a single command:

  mise install

Tool versions default to "latest". After creation, .mise.toml is user-owned —
edit it directly to pin specific versions.`,
	Example: `  # Initialize in current directory (uses .tfskel.yaml if present)
  tfskel init

  # Initialize in specific directory
  tfskel init --dir /path/to/project

  # Initialize with explicit config file
  tfskel init --config /path/to/config.yaml`,
	SilenceUsage: true,
	RunE:         runInit,
}

var (
	initDir       string
	initWorkflows bool
	initUpgrade   bool
	initForce     bool
	initSkip      string
)

func init() {
	rootCmd.AddCommand(initCmd)
	initCmd.Flags().StringVarP(&initDir, "dir", "d", "", "directory to initialize (default: current directory)")
	initCmd.Flags().BoolVar(&initWorkflows, "workflows", false, "generate shared GitHub workflow files (reusable workflows and lint)")
	initCmd.Flags().BoolVar(&initUpgrade, "upgrade", false, "overwrite init-managed files with latest versions from this binary")
	initCmd.Flags().BoolVar(&initForce, "force", false, "overwrite files even without source markers (requires --upgrade)")
	initCmd.Flags().StringVar(&initSkip, "skip", "", "comma-separated files to skip during upgrade (requires --upgrade)")
}

func runInit(_ *cobra.Command, _ []string) error {
	// Initialize logger
	log := logger.NewWithOptions(viper.GetBool("verbose"), useColor)

	log.Debug("Starting init command")

	// Determine target directory
	targetDir := initDir
	if targetDir == "" {
		var err error
		targetDir, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
		log.Debugf("Using current working directory: %s", targetDir)
	}

	// Make absolute path
	targetDir, err := filepath.Abs(targetDir)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Validate flag combinations
	if initForce && !initUpgrade {
		return ErrForceRequiresUpgrade
	}
	if initSkip != "" && !initUpgrade {
		return ErrInitSkipRequiresUpgrade
	}

	log.Infof("Initializing tfskel project structure in: %s", targetDir)

	// Determine environments, regions, and terraform version
	// Priority: existing .tfskel.yaml in target dir > defaults
	environments, terraformVersion, regions, workflowsFromConfig, err := determineInitParameters(targetDir, log)
	if err != nil {
		return err
	}

	// Determine whether to create workflows: --workflows flag OR config workflows.create
	createWorkflows := initWorkflows || workflowsFromConfig

	// Create filesystem abstraction — DryRunFileSystem silently skips writes
	var filesystem fs.FileSystem = fs.NewOSFileSystem()
	if dryRun {
		filesystem = fs.NewDryRunFileSystem(filesystem)
	}

	runner, err := newInitRunner(filesystem, log, initUpgrade, initForce, dryRun, parseSkipList(initSkip))
	if err != nil {
		return err
	}

	if err := runner.createProjectStructure(targetDir, terraformVersion, regions, environments, createWorkflows); err != nil {
		return err
	}

	if dryRun {
		log.Info("Dry run complete — no files were written")
	} else {
		log.Successf("Successfully initialized tfskel project structure in: %s", targetDir)
	}

	log.Info("")
	log.Info("Next step: run 'tfskel validate' to check your setup and install tools")

	return nil
}

// extractVersionFromConstraint converts a version constraint to a simple version number
// Examples: "~> 1.13" -> "1.13.0", ">= 1.13.1" -> "1.13.1", "1.13.1" -> "1.13.1"
func extractVersionFromConstraint(constraint string) string {
	// Remove common constraint operators and trim spaces
	version := strings.TrimSpace(constraint)
	version = strings.TrimPrefix(version, "~>")
	version = strings.TrimPrefix(version, ">=")
	version = strings.TrimPrefix(version, "<=")
	version = strings.TrimPrefix(version, ">")
	version = strings.TrimPrefix(version, "<")
	version = strings.TrimPrefix(version, "=")
	version = strings.TrimSpace(version)

	// Add patch version if missing (e.g., "1.13" -> "1.13.0")
	if strings.Count(version, ".") == 1 {
		version += ".0"
	}

	return version
}

// determineInitParameters determines environments, terraform version, regions, and the workflows flag.
// Priority: existing .tfskel.yaml in target dir > defaults
func determineInitParameters(targetDir string, log *logger.Logger) ([]string, string, []string, bool, error) {
	// Default values for bootstrapping new projects
	defaultEnvironments := []string{"dev", "stg", "prd"}
	defaultRegions := []string{"eu-central-1"}

	// Check if .tfskel.yaml exists in target directory
	configPath := filepath.Join(targetDir, ".tfskel.yaml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// No config file exists, use defaults
		log.Debugf("No .tfskel.yaml found in target directory, using default environments: %v", defaultEnvironments)
		return defaultEnvironments, defaultTerraformVersion, defaultRegions, false, nil
	}

	// Config file exists, read it
	log.Debugf("Found existing .tfskel.yaml, reading configuration...")

	// Create a new viper instance for reading the target directory's config
	v := viper.New()
	v.SetConfigFile(configPath)
	v.SetConfigType("yaml")

	if err := v.ReadInConfig(); err != nil {
		// If we can't read the config, warn and use defaults
		log.Warnf("Failed to read existing .tfskel.yaml: %v, using defaults", err)
		return defaultEnvironments, defaultTerraformVersion, defaultRegions, false, nil
	}

	// Unmarshal into config struct
	cfg := &config.Config{}
	if err := v.Unmarshal(cfg); err != nil {
		log.Warnf("Failed to parse .tfskel.yaml: %v, using defaults", err)
		return defaultEnvironments, defaultTerraformVersion, defaultRegions, false, nil
	}

	// Ensure nested structures exist
	if cfg.Provider == nil {
		cfg.Provider = &config.Provider{}
	}
	if cfg.Provider.AWS == nil {
		cfg.Provider.AWS = &config.AWSProvider{}
	}

	// Extract environments from account_mapping
	var environments []string
	if len(cfg.Provider.AWS.AccountMapping) > 0 {
		// Use keys from account_mapping as environments
		for env := range cfg.Provider.AWS.AccountMapping {
			environments = append(environments, env)
		}
		// Sort for consistent output
		sort.Strings(environments)
		log.Infof("Using %d environment(s) from config account_mapping: %v", len(environments), environments)
	} else {
		// Config exists but no account_mapping - this is an error
		return nil, "", nil, false, fmt.Errorf("existing .tfskel.yaml found but %w; account mappings are required. Please add environment mappings to .tfskel.yaml", ErrMissingAccountMapping)
	}

	// Extract terraform version
	terraformVersion := defaultTerraformVersion
	if cfg.TerraformVersion != "" {
		terraformVersion = extractVersionFromConstraint(cfg.TerraformVersion)
		log.Debugf("Using Terraform version from config: %s", terraformVersion)
	}

	// Extract regions
	regions := defaultRegions
	if configRegions := cfg.GetRegions(); len(configRegions) > 0 {
		regions = configRegions
		log.Infof("Using %d region(s) from config: %v", len(regions), regions)
	} else {
		log.Warnf("No regions specified in config, using default: %v", defaultRegions)
	}

	createWorkflows := cfg.Workflows != nil && cfg.Workflows.Create
	return environments, terraformVersion, regions, createWorkflows, nil
}

func (r *initRunner) createProjectStructure(baseDir, terraformVersion string, regions, environments []string, createWorkflows bool) error {
	// Create base directory if it doesn't exist
	if err := r.fs.MkdirAll(baseDir, 0755); err != nil {
		return fmt.Errorf("failed to create base directory: %w", err)
	}

	// Create root configuration files from templates
	for _, file := range rootConfigFiles {
		if err := r.createFileFromTemplate(filepath.Join(baseDir, file.filename), file.templateName, nil); err != nil {
			return err
		}
	}

	// Create .gitignore with sensible Terraform defaults. This file is user-owned
	// after creation: tfskel seeds it once but does not manage it (no source marker,
	// skipped on --upgrade).
	if err := r.createUnmanagedFile(filepath.Join(baseDir, ".gitignore"), "root/.gitignore.tmpl", nil); err != nil {
		return err
	}

	// Create .mise.toml with sensible defaults. This file is user-owned after creation:
	// tfskel seeds it once but does not manage it (no source marker, skipped on --upgrade).
	if err := r.createUnmanagedFile(filepath.Join(baseDir, ".mise.toml"), "root/.mise.toml.tmpl", &templates.Data{
		TerraformVersion: terraformVersion,
		Environments:     environments,
	}); err != nil {
		return err
	}

	// Create .tfskel.yaml config file
	if err := r.createDefaultConfig(filepath.Join(baseDir, ".tfskel.yaml")); err != nil {
		return err
	}

	// Create environment directories using provided environments list
	r.log.Debugf("Creating directory structure for %d environment(s): %v", len(environments), environments)
	for _, env := range environments {
		envPath := filepath.Join(baseDir, "envs", env)

		// Create .terraform-version file
		tfVersionPath := filepath.Join(envPath, ".terraform-version")
		tfVersionData := &templates.Data{TerraformVersion: terraformVersion}
		if err := r.createFileFromTemplate(tfVersionPath, "root/.terraform-version.tmpl", tfVersionData); err != nil {
			return err
		}

		// Create region directories
		for _, region := range regions {
			regionPath := filepath.Join(envPath, region)

			// Log directory creation relative to baseDir
			relPath, relErr := filepath.Rel(baseDir, regionPath)
			if relErr != nil {
				relPath = regionPath
			}

			if r.fs.DirExists(regionPath) {
				r.log.Infof("Directory %s/ already exists", relPath)
				continue
			}

			if err := r.fs.MkdirAll(regionPath, 0755); err != nil {
				return fmt.Errorf("failed to create region directory %s: %w", regionPath, err)
			}

			if r.dryRun {
				r.log.Infof("[dry-run] Would create directory: %s/", relPath)
			} else {
				r.log.Successf("Created directory: %s/", relPath)
			}
		}
	}

	// Create static GitHub workflow files (reusable workflows and lint caller)
	if createWorkflows {
		staticWorkflowFiles := []struct {
			filename     string
			templateName string
		}{
			{"lint.yaml", "github/lint.yaml"},
			{"reusable-detect-changes.yaml", "github/reusable-detect-changes.yaml"},
			{"reusable-terraform-plan-apply.yaml", "github/reusable-terraform-plan-apply.yaml"},
			{"reusable-lint.yaml", "github/reusable-lint.yaml"},
		}
		for _, file := range staticWorkflowFiles {
			targetPath := filepath.Join(baseDir, ".github", "workflows", file.filename)
			if err := r.createFileFromTemplate(targetPath, file.templateName, nil); err != nil {
				return err
			}
		}
	}

	return nil
}

func (r *initRunner) createFileFromTemplate(targetPath, templateName string, data *templates.Data) error {
	// Compute a relative path for logging; fall back to base name on error
	logPath := targetPath
	if cwd, err := os.Getwd(); err == nil {
		if rel, err := filepath.Rel(cwd, targetPath); err == nil {
			logPath = rel
		}
	}

	// Check if file already exists
	if r.fs.FileExists(targetPath) {
		if r.upgrade {
			if r.skip[filepath.Base(targetPath)] {
				r.log.Infof("%s skipped (--skip)", logPath)
				return nil
			}
			return r.upgradeFile(targetPath, templateName, data, logPath)
		}
		r.log.Infof("%s already exists, skipping", logPath)
		return nil
	}

	// Render content
	content, err := r.renderTemplate(templateName, data)
	if err != nil {
		return err
	}

	// Inject source marker (skipped for files that don't support comments)
	if comment := generate.SourceCommentForFile(templateName, r.renderer.GetTemplateHash(templateName), targetPath); comment != "" {
		content = generate.InjectSourceMarker(content, comment)
	}

	// Ensure parent directory exists and write file
	if err := r.writeFile(targetPath, content); err != nil {
		return err
	}

	if r.dryRun {
		r.log.Infof("[dry-run] Would create %s", logPath)
	} else {
		r.log.Successf("Created %s", logPath)
	}
	return nil
}

// createUnmanagedFile renders a template and writes it only if the file does not
// already exist. Unlike createFileFromTemplate it does not inject a source marker
// and is always skipped during --upgrade, because the file is user-owned after
// initial creation.
func (r *initRunner) createUnmanagedFile(targetPath, templateName string, data *templates.Data) error {
	logPath := targetPath
	if cwd, err := os.Getwd(); err == nil {
		if rel, err := filepath.Rel(cwd, targetPath); err == nil {
			logPath = rel
		}
	}

	if r.fs.FileExists(targetPath) {
		r.log.Debugf("%s already exists, skipping (user-owned)", logPath)
		return nil
	}

	content, err := r.renderTemplate(templateName, data)
	if err != nil {
		return err
	}

	if err := r.writeFile(targetPath, content); err != nil {
		return err
	}

	if r.dryRun {
		r.log.Infof("[dry-run] Would create %s", logPath)
	} else {
		r.log.Successf("Created %s", logPath)
	}
	return nil
}

// upgradeFile handles the upgrade logic for an existing init-managed file.
func (r *initRunner) upgradeFile(targetPath, templateName string, data *templates.Data, logPath string) error {
	// Read existing file and check source marker
	existingContent, err := r.fs.ReadFile(targetPath)
	if err != nil {
		return fmt.Errorf("failed to read %s for upgrade check: %w", logPath, err)
	}

	upgradeVerb := "Upgrading"
	forceVerb := "Force upgrading"
	if r.dryRun {
		upgradeVerb = "[dry-run] Would upgrade"
		forceVerb = "[dry-run] Would force upgrade"
	}

	marker, markerErr := generate.ExtractSourceMarker(string(existingContent))

	switch {
	case errors.Is(markerErr, generate.ErrSourceMarkerNotFound) && !r.force:
		r.log.Infof("%s has no source marker, skipping upgrade (use --force to override)", logPath)
		return nil
	case errors.Is(markerErr, generate.ErrSourceMarkerNotFound):
		r.log.Infof("%s %s (--force, no source marker)", forceVerb, logPath)
	case markerErr != nil && !r.force:
		// Malformed source marker (e.g. invalid JSON)
		return fmt.Errorf("invalid source marker in %s: %w", logPath, markerErr)
	case markerErr != nil:
		r.log.Infof("%s %s (--force, invalid source marker: %v)", forceVerb, logPath, markerErr)
	}

	// Has marker: verify template match, then compare rendered content
	if markerErr == nil {
		return r.upgradeFromMarker(targetPath, templateName, data, logPath, marker, existingContent, upgradeVerb)
	}

	// No valid marker (--force path): re-render and overwrite
	content, err := r.renderTemplate(templateName, data)
	if err != nil {
		return err
	}
	if comment := generate.SourceCommentForFile(templateName, r.renderer.GetTemplateHash(templateName), targetPath); comment != "" {
		content = generate.InjectSourceMarker(content, comment)
	}

	return r.writeFile(targetPath, content)
}

// upgradeFromMarker handles upgrade when a valid source marker is present.
// It compares re-rendered content against the existing file to detect both
// template changes and data drift (e.g. version bumps in .tfskel.yaml).
func (r *initRunner) upgradeFromMarker(targetPath, templateName string, data *templates.Data, logPath string, marker *generate.SourceMarker, existingContent []byte, upgradeVerb string) error {
	if marker.Template != templateName {
		r.log.Debugf("%s source marker template mismatch (%s != %s), skipping", logPath, marker.Template, templateName)
		return nil
	}

	// Re-render and compare full content to detect both template and data drift
	rendered, err := r.renderTemplate(templateName, data)
	if err != nil {
		return err
	}
	if comment := generate.SourceCommentForFile(templateName, r.renderer.GetTemplateHash(templateName), targetPath); comment != "" {
		rendered = generate.InjectSourceMarker(rendered, comment)
	}

	if rendered == string(existingContent) {
		r.log.Debugf("%s is up to date, skipping", logPath)
		return nil
	}

	currentHash := r.renderer.GetTemplateHash(templateName)
	if marker.Hash != currentHash {
		r.log.Infof("%s %s (template: %s -> %s)", upgradeVerb, logPath, marker.Hash, currentHash)
	} else {
		r.log.Infof("%s %s (config drift detected)", upgradeVerb, logPath)
	}

	return r.writeFile(targetPath, rendered)
}

// renderTemplate renders a template with optional data for init command.
func (r *initRunner) renderTemplate(templateName string, data *templates.Data) (string, error) {
	if data == nil {
		data = &templates.Data{}
	}
	return r.renderer.Render(templateName, data)
}

// writeFile writes content to the file via the filesystem abstraction.
// In dry-run mode the DryRunFileSystem silently skips the actual write.
// Callers are responsible for logging the action with the appropriate verb.
func (r *initRunner) writeFile(targetPath, content string) error {
	if err := r.fs.WriteFile(targetPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write file %s: %w", targetPath, err)
	}
	return nil
}

func (r *initRunner) createDefaultConfig(configPath string) error {
	// Check if config file already exists
	if r.fs.FileExists(configPath) {
		r.log.Infof(".tfskel.yaml already exists, skipping")
		return nil
	}

	defaultConfig := map[string]any{
		"terraform_version": "~> 1.13",
		"provider": map[string]any{
			"aws": map[string]any{
				"version": "~> 6.0",
				"account_mapping": map[string]string{
					"dev": "REPLACE_WITH_YOUR_DEV_ACCOUNT_ID",
					"stg": "REPLACE_WITH_YOUR_STG_ACCOUNT_ID",
					"prd": "REPLACE_WITH_YOUR_PRD_ACCOUNT_ID",
				},
				"default_tags": map[string]string{
					"managed_by": "terraform",
				},
				"regions": []string{"eu-central-1"},
			},
		},
		"backend": map[string]any{
			"s3": map[string]any{
				"bucket_name": "CHANGE_ME_WITH_YOUR_GLOBALLY_UNIQUE_S3_BUCKET_NAME",
			},
		},
		"critical_resources": []string{},
	}

	// Marshal to YAML
	data, err := yaml.Marshal(defaultConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Add comments at the top
	header := `# tfskel configuration file
# This file contains default settings for your Terraform operations with tfskel
#
# For full configuration reference with all available options and examples:
# https://github.com/ishuar/tfskel/blob/main/.tfskel.example.yaml
#

`
	fullContent := []byte(header + string(data))

	// Write to file
	if err := r.fs.WriteFile(configPath, fullContent, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	if r.dryRun {
		r.log.Infof("[dry-run] Would create .tfskel.yaml")
	} else {
		r.log.Successf("Created .tfskel.yaml")
	}

	return nil
}
