package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ishuar/tfskel/internal/config"
	"github.com/ishuar/tfskel/internal/logger"
	"github.com/ishuar/tfskel/internal/templates"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.yaml.in/yaml/v4"
)

var (
	ErrUnsupportedDataType   = errors.New("unsupported data type for template rendering")
	ErrMissingAccountMapping = errors.New("provider.aws.account_mapping is missing or empty")
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize tfskel project structure",
	Long: `Initialize a tfskel project structure with environment directories and configuration files.

This creates:
- Root config files (.gitignore, .pre-commit-config.yaml, .tflint.hcl, trivy.yaml, .tfskel.yaml)
- Environment directories (dev, stg, prd) with region subdirectories
- .terraform-version files for each environment

Configuration:
  The init command reads .tfskel.yaml from the current directory (or --config path)
  to determine which regions and environments to create. If no config file exists, it uses defaults.

Recommendations:
  - Ensure required tools are installed: terraform, tflint, trivy, pre-commit
  - Install pre-commit hooks:
      pre-commit install --install-hooks`,
	Example: `  # Initialize in current directory (uses .tfskel.yaml if present)
  tfskel init

  # Initialize in specific directory
  tfskel init --dir /path/to/project

  # Initialize with explicit config file
  tfskel init --config /path/to/config.yaml`,
	RunE: runInit,
}

var (
	initDir string
)

func init() {
	rootCmd.AddCommand(initCmd)
	initCmd.Flags().StringVarP(&initDir, "dir", "d", "", "directory to initialize (default: current directory)")
}

func runInit(cmd *cobra.Command, args []string) error {
	// Initialize logger
	log := logger.New(viper.GetBool("verbose"))

	log.Debug("Starting init command")

	// Determine target directory
	targetDir := initDir
	if targetDir == "" {
		var err error
		targetDir, err = os.Getwd()
		if err != nil {
			cmd.SilenceUsage = true
			return fmt.Errorf("failed to get current directory: %w", err)
		}
		log.Debugf("Using current working directory: %s", targetDir)
	}

	// Make absolute path
	targetDir, err := filepath.Abs(targetDir)
	if err != nil {
		cmd.SilenceUsage = true
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	log.Infof("Initializing tfskel project structure in: %s", targetDir)

	// Determine environments, regions, and terraform version
	// Priority: existing .tfskel.yaml in target dir > defaults
	environments, terraformVersion, regions, err := determineInitParameters(targetDir, log)
	if err != nil {
		cmd.SilenceUsage = true
		return err
	}

	// Create the project structure
	if err := createProjectStructure(targetDir, terraformVersion, regions, environments, log); err != nil {
		cmd.SilenceUsage = true
		return err
	}

	log.Successf("Successfully initialized tfskel project structure in: %s", targetDir)

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

// determineInitParameters determines environments, terraform version, and regions
// Priority: existing .tfskel.yaml in target dir > defaults
func determineInitParameters(targetDir string, log *logger.Logger) ([]string, string, []string, error) {
	// Default values
	defaultEnvironments := []string{"dev", "stg", "prd"}
	defaultTerraformVersion := "1.13.1"
	defaultRegions := []string{"eu-central-1"}

	// Check if .tfskel.yaml exists in target directory
	configPath := filepath.Join(targetDir, ".tfskel.yaml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// No config file exists, use defaults
		log.Debugf("No .tfskel.yaml found in target directory, using default environments: %v", defaultEnvironments)
		return defaultEnvironments, defaultTerraformVersion, defaultRegions, nil
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
		return defaultEnvironments, defaultTerraformVersion, defaultRegions, nil
	}

	// Unmarshal into config struct
	cfg := &config.Config{}
	if err := v.Unmarshal(cfg); err != nil {
		log.Warnf("Failed to parse .tfskel.yaml: %v, using defaults", err)
		return defaultEnvironments, defaultTerraformVersion, defaultRegions, nil
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
		return nil, "", nil, fmt.Errorf("existing .tfskel.yaml found but %w; account mappings are required. Please add environment mappings to .tfskel.yaml", ErrMissingAccountMapping)
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

	return environments, terraformVersion, regions, nil
}

func createProjectStructure(baseDir string, terraformVersion string, regions []string, environments []string, log *logger.Logger) error {
	// Create base directory if it doesn't exist
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return fmt.Errorf("failed to create base directory: %w", err)
	}

	// Create root configuration files from templates
	rootConfigFiles := []struct {
		filename     string
		templateName string
	}{
		{".gitignore", "root/.gitignore.tmpl"},
		{".pre-commit-config.yaml", "root/.pre-commit-config.yaml.tmpl"},
		{".tflint.hcl", "root/.tflint.hcl.tmpl"},
		{"trivy.yaml", "root/trivy.yaml.tmpl"},
	}

	for _, file := range rootConfigFiles {
		if err := createFileFromTemplate(filepath.Join(baseDir, file.filename), file.templateName, nil, log); err != nil {
			return err
		}
	}

	// Create .tfskel.yaml config file
	if err := createDefaultConfig(filepath.Join(baseDir, ".tfskel.yaml"), log); err != nil {
		return err
	}

	// Create environment directories using provided environments list
	log.Debugf("Creating directory structure for %d environment(s): %v", len(environments), environments)
	for _, env := range environments {
		envPath := filepath.Join(baseDir, "envs", env)

		// Create .terraform-version file
		tfVersionPath := filepath.Join(envPath, ".terraform-version")
		data := map[string]string{
			"TerraformVersion": terraformVersion,
		}
		if err := createFileFromTemplate(tfVersionPath, "root/.terraform-version.tmpl", data, log); err != nil {
			return err
		}

		// Create region directories
		for _, region := range regions {
			regionPath := filepath.Join(envPath, region)

			// Check if directory already exists
			_, err := os.Stat(regionPath)
			dirExists := err == nil

			if err := os.MkdirAll(regionPath, 0755); err != nil {
				return fmt.Errorf("failed to create region directory %s: %w", regionPath, err)
			}

			// Log directory creation relative to baseDir
			relPath, err := filepath.Rel(baseDir, regionPath)
			if err != nil {
				// If relative path fails, use absolute path for logging
				relPath = regionPath
			}
			if dirExists {
				log.Infof("Directory %s/ already exists", relPath)
			} else {
				log.Successf("Created directory: %s/", relPath)
			}
		}
	}

	return nil
}

func createFileFromTemplate(targetPath string, templateName string, data interface{}, log *logger.Logger) error {
	// Check if file already exists
	if _, err := os.Stat(targetPath); err == nil {
		// File exists, skip creation
		baseName := filepath.Base(targetPath)
		log.Infof("%s already exists, skipping", baseName)
		return nil
	}

	// Ensure parent directory exists
	dir := filepath.Dir(targetPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// Create a new renderer
	renderer, err := templates.NewRenderer()
	if err != nil {
		return fmt.Errorf("failed to create renderer: %w", err)
	}

	// For simple templates without variables, render with empty data
	if data == nil {
		templateData := &templates.Data{}
		content, err := renderer.Render(templateName, templateData)
		if err != nil {
			return fmt.Errorf("failed to render template %s: %w", templateName, err)
		}

		if err := os.WriteFile(targetPath, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to write file %s: %w", targetPath, err)
		}
		baseName := filepath.Base(targetPath)
		log.Successf("Created %s", baseName)
		return nil
	}

	// For templates with variables (like terraform-version), use renderer with template functions
	if m, ok := data.(map[string]string); ok {
		// Use renderer to get access to template functions like stripConstraint
		content, err := renderer.Render(templateName, &templates.Data{
			TerraformVersion: m["TerraformVersion"],
		})
		if err != nil {
			return fmt.Errorf("failed to render template %s: %w", templateName, err)
		}

		file, err := os.Create(targetPath)
		if err != nil {
			return fmt.Errorf("failed to create file %s: %w", targetPath, err)
		}
		defer func() {
			if closeErr := file.Close(); closeErr != nil {
				// Log the error but don't override the main error
				fmt.Fprintf(os.Stderr, "Warning: failed to close file %s: %v\n", targetPath, closeErr)
			}
		}()

		if _, err := file.WriteString(content); err != nil {
			return fmt.Errorf("failed to write file %s: %w", targetPath, err)
		}

		// Get relative path or base name for logging
		baseName := filepath.Base(targetPath)
		log.Successf("Created %s", baseName)

		return nil
	}

	return ErrUnsupportedDataType
}

func createDefaultConfig(configPath string, log *logger.Logger) error {
	// Check if config file already exists
	if _, err := os.Stat(configPath); err == nil {
		// File exists, skip creation
		log.Infof(".tfskel.yaml already exists, skipping")
		return nil
	}

	defaultConfig := map[string]interface{}{
		"terraform_version": "~> 1.13",
		"provider": map[string]interface{}{
			"aws": map[string]interface{}{
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
		"backend": map[string]interface{}{
			"s3": map[string]interface{}{
				"bucket_name": "CHANGE_ME_WITH_YOUR_GLOBALLY_UNIQUE_S3_BUCKET_NAME",
			},
		},
	}

	// Marshal to YAML
	data, err := yaml.Marshal(defaultConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Add comments at the top
	header := `# tfskel configuration file
# This file contains default settings for your Terraform project scaffolding
#
# Required: Update provider.aws.account_mapping with your actual AWS account IDs
# Required: Update backend.s3.bucket_name with your actual S3 bucket name for Terraform state
# Optional: Customize terraform_version, provider versions, regions, and default_tags as needed
#
# For more information, visit: https://github.com/ishuar/tfskel

`
	fullContent := []byte(header + string(data))

	// Write to file
	if err := os.WriteFile(configPath, fullContent, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	log.Successf("Created .tfskel.yaml")

	return nil
}
