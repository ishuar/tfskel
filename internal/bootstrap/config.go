package bootstrap

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ishuar/tfskel/internal/config"
	"github.com/ishuar/tfskel/internal/logger"
	"github.com/spf13/viper"
	"go.yaml.in/yaml/v4"
)

// ErrMissingAccountMapping indicates AWS account mapping configuration is missing.
var ErrMissingAccountMapping = errors.New("provider.aws.account_mapping is missing or empty")

// Parameters holds the inputs resolved from .tfskel.yaml (or defaults) that
// drive project creation.
type Parameters struct {
	Environments     []string
	TerraformVersion string
	Regions          []string
	CreateWorkflows  bool
}

// DetermineParameters resolves the environments, terraform version, regions, and workflows flag
// used for project creation. Priority: existing .tfskel.yaml in targetDir > defaults.
//
// Returns an error only when a .tfskel.yaml exists but has no provider.aws.account_mapping.
// All other read/parse failures fall back to defaults with a warning log.
func DetermineParameters(targetDir string, log *logger.Logger) (*Parameters, error) {
	defaults := &Parameters{
		Environments:     []string{"dev", "stg", "prd"},
		TerraformVersion: defaultTerraformVersion,
		Regions:          []string{"eu-central-1"},
	}

	configPath := filepath.Join(targetDir, ".tfskel.yaml")
	if _, statErr := os.Stat(configPath); os.IsNotExist(statErr) {
		log.Debugf("No .tfskel.yaml found in target directory, using default environments: %v", defaults.Environments)
		return defaults, nil
	}

	log.Debugf("Found existing .tfskel.yaml, reading configuration...")

	v := viper.New()
	v.SetConfigFile(configPath)
	v.SetConfigType("yaml")

	if readErr := v.ReadInConfig(); readErr != nil {
		log.Warnf("Failed to read existing .tfskel.yaml: %v, using defaults", readErr)
		return defaults, nil
	}

	cfg := &config.Config{}
	if unmarshalErr := v.Unmarshal(cfg); unmarshalErr != nil {
		log.Warnf("Failed to parse .tfskel.yaml: %v, using defaults", unmarshalErr)
		return defaults, nil
	}

	if cfg.Provider == nil || cfg.Provider.AWS == nil || len(cfg.Provider.AWS.AccountMapping) == 0 {
		return nil, fmt.Errorf("%w in .tfskel.yaml; add entries under that key to enable environment auto-discovery", ErrMissingAccountMapping)
	}

	params := &Parameters{
		TerraformVersion: defaultTerraformVersion,
		Regions:          defaults.Regions,
		CreateWorkflows:  cfg.Workflows != nil && cfg.Workflows.Create,
	}

	for envName := range cfg.Provider.AWS.AccountMapping {
		params.Environments = append(params.Environments, envName)
	}
	sort.Strings(params.Environments)
	log.Infof("Using %d environment(s) from config account_mapping: %v", len(params.Environments), params.Environments)

	if cfg.TerraformVersion != "" {
		params.TerraformVersion = extractVersionFromConstraint(cfg.TerraformVersion)
		log.Debugf("Using Terraform version from config: %s", params.TerraformVersion)
	}

	if configRegions := cfg.GetRegions(); len(configRegions) > 0 {
		params.Regions = configRegions
		log.Infof("Using %d region(s) from config: %v", len(params.Regions), params.Regions)
	} else {
		log.Warnf("No regions specified in config, using default: %v", defaults.Regions)
	}

	return params, nil
}

// extractVersionFromConstraint converts a version constraint to a simple version number.
// Examples: "~> 1.13" -> "1.13.0", ">= 1.13.1" -> "1.13.1", "1.13.1" -> "1.13.1".
func extractVersionFromConstraint(constraint string) string {
	version := strings.TrimSpace(constraint)
	for _, op := range []string{"~>", ">=", "<=", ">", "<", "="} {
		version = strings.TrimPrefix(version, op)
	}
	version = strings.TrimSpace(version)

	if strings.Count(version, ".") == 1 {
		version += ".0"
	}
	return version
}

// CreateDefaultConfig writes a seed .tfskel.yaml at configPath with placeholders
// the user is expected to replace. It is a no-op when the file already exists.
func (r *Runner) CreateDefaultConfig(configPath string) error {
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

	data, err := yaml.Marshal(defaultConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	header := `# tfskel configuration file
# This file contains default settings for your Terraform operations with tfskel
#
# For full configuration reference with all available options and examples:
# https://github.com/ishuar/tfskel/blob/main/.tfskel.example.yaml
#

`
	fullContent := []byte(header + string(data))

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
