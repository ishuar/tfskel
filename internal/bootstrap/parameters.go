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
	defaultEnvironments := []string{"dev", "stg", "prd"}
	defaultRegions := []string{"eu-central-1"}
	newDefaults := func() *Parameters {
		return &Parameters{
			Environments:     append([]string(nil), defaultEnvironments...),
			TerraformVersion: defaultTerraformVersion,
			Regions:          append([]string(nil), defaultRegions...),
		}
	}

	configPath := filepath.Join(targetDir, ".tfskel.yaml")
	if _, statErr := os.Stat(configPath); os.IsNotExist(statErr) {
		log.Debugf("No .tfskel.yaml found in target directory, using default environments: %v", defaultEnvironments)
		return newDefaults(), nil
	}

	log.Debugf("Found existing .tfskel.yaml, reading configuration...")

	v := viper.New()
	v.SetConfigFile(configPath)
	v.SetConfigType("yaml")

	if readErr := v.ReadInConfig(); readErr != nil {
		log.Warnf("Failed to read existing .tfskel.yaml: %v, using defaults", readErr)
		return newDefaults(), nil
	}

	cfg := &config.Config{}
	if unmarshalErr := v.Unmarshal(cfg); unmarshalErr != nil {
		log.Warnf("Failed to parse .tfskel.yaml: %v, using defaults", unmarshalErr)
		return newDefaults(), nil
	}

	if cfg.Provider == nil || cfg.Provider.AWS == nil || len(cfg.Provider.AWS.AccountMapping) == 0 {
		return nil, fmt.Errorf(".tfskel.yaml: %w; add entries mapping environments to AWS account IDs", ErrMissingAccountMapping)
	}

	params := &Parameters{
		TerraformVersion: defaultTerraformVersion,
		Regions:          append([]string(nil), defaultRegions...),
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
		log.Warnf("No regions specified in config, using default: %v", defaultRegions)
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
