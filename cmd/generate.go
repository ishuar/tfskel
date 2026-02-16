package cmd

import (
	"errors"
	"fmt"

	"github.com/ishuar/tfskel/internal/app"
	"github.com/ishuar/tfskel/internal/config"
	"github.com/ishuar/tfskel/internal/fs"
	"github.com/ishuar/tfskel/internal/logger"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	// ErrEnvironmentRequired indicates the --env flag was not provided
	ErrEnvironmentRequired = errors.New("environment is required (use --env flag)")
	// ErrRegionRequired indicates the --region flag was not provided
	ErrRegionRequired = errors.New("region is required (use --region flag)")
	// ErrAppDirRequired indicates no app directory argument was provided
	ErrAppDirRequired = errors.New("app directory name is required (provide as argument)")
	// ErrAccountMapping indicates account mapping is missing for the environment
	ErrAccountMapping = errors.New("account mapping is required for environment in your configuration")
)

var generateCmd = &cobra.Command{
	Use:   "generate <app-dir>",
	Short: "Generate Terraform project structure",
	Long: `Generate a complete Terraform project structure with environment-based organization.

This command creates:
  - Environment directories (dev, stg, prd)
  - Region-specific subdirectories (only for the specified region)
  - Application directories
  - Terraform configuration files from templates

Configuration:
  The generate command reads .tfskel.yaml from the current directory by default.
  Use --config flag to specify a different configuration file location.
  The --config flag takes precedence over the default location.

Arguments:
  app-dir: Name of the application directory to create (required)`,
	Example: `  # Generate structure for an app in dev environment (uses .tfskel.yaml)
  tfskel generate myapp --env dev --region us-east-1

  # Generate with custom configuration file
  tfskel generate myapp --config ./my-config.yaml --env dev --region us-east-1

  # Generate with custom templates
  tfskel generate myapp --env stg --region eu-central-1 --templates-dir ./templates`,
	Args: cobra.ExactArgs(1),
	RunE: runGenerate,
}

var (
	env                     string
	region                  string
	templatesDir            string
	s3BucketName            string
	extraTemplateExtensions []string
	createGithubWorkflows   bool
)

func init() {
	rootCmd.AddCommand(generateCmd)

	// Required flags for generation
	generateCmd.Flags().StringVarP(&env, "env", "e", "", "target environment (e.g., dev, stg, prd) - required")
	generateCmd.Flags().StringVarP(&region, "region", "r", "", "AWS region (e.g., us-east-1, eu-central-1) - required")
	// These are critical flags - errors should be handled during command setup, but cobra handles this internally
	if err := generateCmd.MarkFlagRequired("env"); err != nil {
		panic(fmt.Sprintf("failed to mark env flag as required: %v", err))
	}
	if err := generateCmd.MarkFlagRequired("region"); err != nil {
		panic(fmt.Sprintf("failed to mark region flag as required: %v", err))
	}

	// Optional flags
	generateCmd.Flags().StringVar(&templatesDir, "templates-dir", "", "directory containing custom template files (overrides defaults)")
	generateCmd.Flags().StringVar(&s3BucketName, "s3-bucket-name", "", "S3 bucket name for Terraform state")
	generateCmd.Flags().StringSliceVar(&extraTemplateExtensions, "extra-template-extensions", []string{"tf.tmpl"}, "template file extensions to process from templates-dir (tf.tmpl always included)")
	generateCmd.Flags().BoolVar(&createGithubWorkflows, "create-github-workflows", false, "create GitHub workflow files from default templates (disabled by default)")

	// Bind flags to viper for config file support (only for optional flags that can come from config)
	// Binding errors are extremely rare and non-critical - if binding fails, config just won't be set from flag
	//nolint:errcheck // BindPFlag errors are non-critical; flag values simply won't override config
	_ = viper.BindPFlag("generate.templates_dir", generateCmd.Flags().Lookup("templates-dir"))
	//nolint:errcheck // BindPFlag errors are non-critical; flag values simply won't override config
	_ = viper.BindPFlag("backend.s3.bucket_name", generateCmd.Flags().Lookup("s3-bucket-name"))
	//nolint:errcheck // BindPFlag errors are non-critical; flag values simply won't override config
	_ = viper.BindPFlag("generate.extra_template_extensions", generateCmd.Flags().Lookup("extra-template-extensions"))
	//nolint:errcheck // BindPFlag errors are non-critical; flag values simply won't override config
	_ = viper.BindPFlag("generate.github_workflows.create", generateCmd.Flags().Lookup("create-github-workflows"))
}

func runGenerate(cmd *cobra.Command, args []string) error {
	// Initialize logger
	log := logger.New(viper.GetBool("verbose"))

	log.Debug("Starting generate command")
	log.Info("Starting Terraform directory scaffolding...")

	// Get app directory from positional argument
	appDir := args[0]

	// Validate generation parameters
	if err := validateGenerateParams(env, region, appDir); err != nil {
		cmd.SilenceUsage = true
		return fmt.Errorf("invalid parameters: %w", err)
	}

	// Load configuration
	cfg, err := config.Load(cmd, viper.GetViper())
	if err != nil {
		cmd.SilenceUsage = true
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		cmd.SilenceUsage = true
		return fmt.Errorf("configuration validation failed: %w", err)
	}

	// Validate that account mapping exists for the environment
	if err := validateAccountMapping(cfg, env); err != nil {
		cmd.SilenceUsage = true
		return err
	}

	// Create filesystem abstraction
	filesystem := fs.NewOSFileSystem()

	// Create and run the generator with generation parameters
	generator := app.NewGenerator(cfg, filesystem, log)
	if err := generator.Run(env, region, appDir); err != nil {
		cmd.SilenceUsage = true
		return fmt.Errorf("generation failed: %w", err)
	}

	log.Success("Terraform directory scaffolding completed!")
	return nil
}

// validateGenerateParams validates the generation parameters
func validateGenerateParams(env, region, appDir string) error {
	if env == "" {
		return ErrEnvironmentRequired
	}
	if region == "" {
		return ErrRegionRequired
	}
	if appDir == "" {
		return ErrAppDirRequired
	}
	return nil
}

// validateAccountMapping checks if the account mapping exists for the environment
func validateAccountMapping(cfg *config.Config, env string) error {
	accountID := cfg.GetAccountID(env)
	if accountID == "000000000000" {
		return fmt.Errorf("%w '%s'", ErrAccountMapping, env)
	}
	return nil
}
