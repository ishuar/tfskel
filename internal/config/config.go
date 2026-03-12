package config

import (
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/ishuar/tfskel/internal/logger"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	// ErrAWSProviderRequired indicates AWS provider configuration is missing
	ErrAWSProviderRequired = errors.New("AWS provider configuration is required")
	// ErrAccountMappingRequired indicates AWS account mapping is missing from provider configuration
	ErrAccountMappingRequired = errors.New("AWS account mapping is required in provider configuration")
	// ErrAccountMappingNotFound indicates the specified environment has no account mapping
	ErrAccountMappingNotFound = errors.New("no account mapping found for environment")
	// ErrInvalidAccountID indicates an AWS account ID is not properly formatted
	ErrInvalidAccountID = errors.New("AWS account ID must be a 12-digit number")
	// ErrInvalidBucketName indicates the S3 bucket name is not properly configured
	ErrInvalidBucketName = errors.New("backend.s3.bucket_name is invalid")
	// to avoid the regex on every call to validateAccountIDs, we can compile it once at package level
	accountIDPattern = regexp.MustCompile(`^\d{12}$`)
)

// AWSProvider holds AWS provider configuration
type AWSProvider struct {
	Version        string            `mapstructure:"version"`
	AccountMapping map[string]string `mapstructure:"account_mapping"`
	DefaultTags    map[string]string `mapstructure:"default_tags"`
	Regions        []string          `mapstructure:"regions"`
}

// Provider holds all provider configurations
type Provider struct {
	AWS *AWSProvider `mapstructure:"aws"`
}

// S3Backend holds S3 backend configuration
type S3Backend struct {
	BucketName string `mapstructure:"bucket_name"`
}

// Backend holds backend configuration
type Backend struct {
	S3 *S3Backend `mapstructure:"s3"`
}

// Templates holds template directory configuration
type Templates struct {
	Dir string `mapstructure:"dir"`
}

// Workflows holds GitHub workflows configuration
type Workflows struct {
	Create       bool   `mapstructure:"create"`
	NameTemplate string `mapstructure:"name_template"`
	AWSRoleName  string `mapstructure:"aws_role_name"`
	AWSRoleArn   string `mapstructure:"aws_role_arn"`
}

// Config holds the application configuration
type Config struct {
	TerraformVersion string     `mapstructure:"terraform_version"`
	Provider         *Provider  `mapstructure:"provider"`
	Backend          *Backend   `mapstructure:"backend"`
	Templates        *Templates `mapstructure:"templates"`
	Workflows        *Workflows `mapstructure:"workflows"`
}

// Load reads configuration from viper and command line flags
func Load(cmd *cobra.Command, v *viper.Viper) (*Config, error) {
	cfg := &Config{}

	// Unmarshal viper config into struct
	if err := v.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Check for deprecated root-level configuration
	checkDeprecatedConfig(v)

	// Override with command line flags if provided
	applyFlagOverrides(cmd, cfg)

	// Set defaults
	setDefaults(cfg)

	return cfg, nil
}

// applyFlagOverrides applies command line flag values to the config
func applyFlagOverrides(cmd *cobra.Command, cfg *Config) {
	applyTemplatesDirOverride(cmd, cfg)
	applyS3BucketNameOverride(cmd, cfg)
	applyWorkflowsOverride(cmd, cfg)
}

func applyTemplatesDirOverride(cmd *cobra.Command, cfg *Config) {
	if !cmd.Flags().Changed("templates-dir") {
		return
	}
	templatesDir, err := cmd.Flags().GetString("templates-dir")
	if err == nil {
		if cfg.Templates == nil {
			cfg.Templates = &Templates{}
		}
		cfg.Templates.Dir = templatesDir
	}
}

func applyS3BucketNameOverride(cmd *cobra.Command, cfg *Config) {
	if !cmd.Flags().Changed("s3-bucket-name") {
		return
	}
	bucketName, err := cmd.Flags().GetString("s3-bucket-name")
	if err != nil {
		return
	}
	if cfg.Backend == nil {
		cfg.Backend = &Backend{}
	}
	if cfg.Backend.S3 == nil {
		cfg.Backend.S3 = &S3Backend{}
	}
	cfg.Backend.S3.BucketName = bucketName
}

func applyWorkflowsOverride(cmd *cobra.Command, cfg *Config) {
	if !cmd.Flags().Changed("workflows") {
		return
	}
	createWorkflows, err := cmd.Flags().GetBool("workflows")
	if err != nil {
		return
	}
	// No nil check needed for Workflows - always initialized in setDefaults
	if cfg.Workflows == nil {
		cfg.Workflows = &Workflows{}
	}
	cfg.Workflows.Create = createWorkflows
}

// setDefaults initializes default values for unset configuration fields
func setDefaults(cfg *Config) {
	// Always initialize Templates and Workflows to avoid nil checks throughout codebase
	if cfg.Templates == nil {
		cfg.Templates = &Templates{}
	}
	if cfg.Workflows == nil {
		cfg.Workflows = &Workflows{}
	}

	if cfg.TerraformVersion == "" {
		cfg.TerraformVersion = "~> 1.13"
	}
	if cfg.Provider == nil {
		cfg.Provider = &Provider{}
	}
	if cfg.Provider.AWS == nil {
		cfg.Provider.AWS = &AWSProvider{}
	}
	if cfg.Provider.AWS.Version == "" {
		cfg.Provider.AWS.Version = "~> 6.0"
	}
	if cfg.Backend == nil {
		cfg.Backend = &Backend{}
	}
	if cfg.Backend.S3 == nil {
		cfg.Backend.S3 = &S3Backend{}
	}
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	// Validate that required configuration sections exist
	if c.Provider == nil || c.Provider.AWS == nil {
		return ErrAWSProviderRequired
	}
	if len(c.Provider.AWS.AccountMapping) == 0 {
		return ErrAccountMappingRequired
	}
	// Validate AWS account IDs
	if err := c.validateAccountIDs(); err != nil {
		return err
	}
	// Validate backend configuration
	// check that bucket name is not configured (nil)
	if c.Backend == nil || c.Backend.S3 == nil {
		return fmt.Errorf("%w: must not be empty", ErrInvalidBucketName)
	}
	// check that bucket name is not empty or just whitespace
	bucketName := strings.TrimSpace(c.Backend.S3.BucketName)
	if bucketName == "" {
		return fmt.Errorf("%w: must not be empty", ErrInvalidBucketName)
	}
	// Check if user left the example placeholder value
	if c.Backend.S3.BucketName == "CHANGE_ME_WITH_YOUR_GLOBALLY_UNIQUE_S3_BUCKET_NAME" {
		return fmt.Errorf("%w: placeholder value must be replaced with actual bucket name", ErrInvalidBucketName)
	}

	return nil
}

// validateAccountIDs checks that all AWS account IDs are valid 12-digit numbers
func (c *Config) validateAccountIDs() error {
	// AWS account IDs are exactly 12 digits

	for env, accountID := range c.Provider.AWS.AccountMapping {
		// Validate format: must be exactly 12 digits
		if !accountIDPattern.MatchString(accountID) {
			return fmt.Errorf("%w: Update the account mapping %q: %q",
				ErrInvalidAccountID, env, accountID)
		}
	}

	return nil
}

// GetAccountID returns the AWS account ID for the specified environment,
// or an error if no mapping exists for that environment.
func (c *Config) GetAccountID(env string) (string, error) {
	if c.Provider != nil && c.Provider.AWS != nil &&
		c.Provider.AWS.AccountMapping != nil {
		if id, ok := c.Provider.AWS.AccountMapping[env]; ok {
			return id, nil
		}
		// Show available keys to help the user fix it immediately
		available := make([]string, 0, len(c.Provider.AWS.AccountMapping))
		for k := range c.Provider.AWS.AccountMapping {
			available = append(available, k)
		}
		sort.Strings(available)
		return "", fmt.Errorf(
			"%w %q, available: [%s]",
			ErrAccountMappingNotFound, env, strings.Join(available, ", "),
		)
	}
	return "", ErrAWSProviderRequired
}

// GetRegions returns the list of configured AWS regions
func (c *Config) GetRegions() []string {
	if c.Provider != nil && c.Provider.AWS != nil &&
		c.Provider.AWS.Regions != nil {
		return c.Provider.AWS.Regions
	}
	return []string{}
}

// checkDeprecatedConfig checks for deprecated root-level configuration and logs warnings
func checkDeprecatedConfig(v *viper.Viper) {
	// Create a minimal logger for warnings (non-verbose mode)
	log := logger.New(false)

	// Check for old 'generate' section (replaced with 'templates' and 'workflows')
	if v.IsSet("generate") {
		log.Warn("DEPRECATION: 'generate' section is deprecated, use 'templates' and 'workflows' sections instead - this will be ignored")
	}

	// Check for old root-level templates_dir
	if v.IsSet("templates_dir") {
		log.Warnf("DEPRECATION: 'templates_dir' at root level is deprecated, use 'templates.dir' instead (current: %s) - this will be ignored",
			v.GetString("templates_dir"))
	}

	// Check for old root-level extra_template_extensions
	if v.IsSet("extra_template_extensions") {
		log.Warn("DEPRECATION: 'extra_template_extensions' is no longer supported - all .tmpl files are now processed as templates")
	}
}
