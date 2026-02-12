package config

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	// ErrAWSProviderRequired indicates AWS provider configuration is missing
	ErrAWSProviderRequired = errors.New("AWS provider configuration is required")
	// ErrAccountMappingRequired indicates AWS account mapping is missing from provider configuration
	ErrAccountMappingRequired = errors.New("AWS account mapping is required in provider configuration")
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

// GithubWorkflows holds GitHub workflows configuration
type GithubWorkflows struct {
	Create       bool   `mapstructure:"create"`
	NameTemplate string `mapstructure:"name_template"`
	AWSRoleName  string `mapstructure:"aws_role_name"`
	AWSRoleArn   string `mapstructure:"aws_role_arn"`
}

// Generate holds generate command specific configuration
type Generate struct {
	GithubWorkflows *GithubWorkflows `mapstructure:"github_workflows"`
}

// Config holds the application configuration
type Config struct {
	TerraformVersion        string    `mapstructure:"terraform_version"`
	Provider                *Provider `mapstructure:"provider"`
	Backend                 *Backend  `mapstructure:"backend"`
	Generate                *Generate `mapstructure:"generate"`
	TemplatesDir            string    `mapstructure:"templates_dir"`
	ExtraTemplateExtensions []string  `mapstructure:"extra_template_extensions"`
}

// Load reads configuration from viper and command line flags
func Load(cmd *cobra.Command, v *viper.Viper) (*Config, error) {
	cfg := &Config{}

	// Unmarshal viper config into struct
	if err := v.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Override with command line flags if provided
	applyFlagOverrides(cmd, cfg)

	// Set defaults
	setDefaults(cfg)

	// Normalize template extensions
	normalizeTemplateExtensions(cfg)

	return cfg, nil
}

// applyFlagOverrides applies command line flag values to the config
func applyFlagOverrides(cmd *cobra.Command, cfg *Config) {
	applyTemplatesDirOverride(cmd, cfg)
	applyS3BucketNameOverride(cmd, cfg)
	applyExtraTemplateExtensionsOverride(cmd, cfg)
	applyCreateGithubWorkflowsOverride(cmd, cfg)
}

func applyTemplatesDirOverride(cmd *cobra.Command, cfg *Config) {
	if !cmd.Flags().Changed("templates-dir") {
		return
	}
	templatesDir, err := cmd.Flags().GetString("templates-dir")
	if err == nil {
		cfg.TemplatesDir = templatesDir
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

func applyExtraTemplateExtensionsOverride(cmd *cobra.Command, cfg *Config) {
	if !cmd.Flags().Changed("extra-template-extensions") {
		return
	}
	extraExts, err := cmd.Flags().GetStringSlice("extra-template-extensions")
	if err == nil {
		cfg.ExtraTemplateExtensions = extraExts
	}
}

func applyCreateGithubWorkflowsOverride(cmd *cobra.Command, cfg *Config) {
	if !cmd.Flags().Changed("create-github-workflows") {
		return
	}
	createWorkflows, err := cmd.Flags().GetBool("create-github-workflows")
	if err != nil {
		return
	}
	if cfg.Generate == nil {
		cfg.Generate = &Generate{}
	}
	if cfg.Generate.GithubWorkflows == nil {
		cfg.Generate.GithubWorkflows = &GithubWorkflows{}
	}
	cfg.Generate.GithubWorkflows.Create = createWorkflows
}

// setDefaults initializes default values for unset configuration fields
func setDefaults(cfg *Config) {
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
	if cfg.Backend.S3.BucketName == "" {
		cfg.Backend.S3.BucketName = "CHANGE_ME_WITH_YOUR_GLOBALLY_UNIQUE_S3_BUCKET_NAME"
	}
}

// normalizeTemplateExtensions ensures tf.tmpl is always present and deduplicates extensions
func normalizeTemplateExtensions(cfg *Config) {
	if len(cfg.ExtraTemplateExtensions) == 0 {
		cfg.ExtraTemplateExtensions = []string{"tf.tmpl"}
		return
	}

	// Deduplicate and ensure tf.tmpl is always present
	extMap := make(map[string]bool)
	extMap["tf.tmpl"] = true // Always include tf.tmpl
	for _, ext := range cfg.ExtraTemplateExtensions {
		if ext != "" {
			extMap[ext] = true
		}
	}
	// Convert back to slice
	cfg.ExtraTemplateExtensions = make([]string, 0, len(extMap))
	for ext := range extMap {
		cfg.ExtraTemplateExtensions = append(cfg.ExtraTemplateExtensions, ext)
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
	return nil
}

// GetAccountID returns the AWS account ID for the specified environment
func (c *Config) GetAccountID(env string) string {
	if c.Provider != nil && c.Provider.AWS != nil &&
		c.Provider.AWS.AccountMapping != nil {
		if id, ok := c.Provider.AWS.AccountMapping[env]; ok {
			return id
		}
	}
	return "000000000000"
}

// GetRegions returns the list of configured AWS regions
func (c *Config) GetRegions() []string {
	if c.Provider != nil && c.Provider.AWS != nil &&
		c.Provider.AWS.Regions != nil {
		return c.Provider.AWS.Regions
	}
	return []string{}
}
