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
)

var generateCmd = &cobra.Command{
	Use:     "generate <app-dir>",
	GroupID: "main",
	Short:   "Generate Terraform project structure for target application",
	Long: `Accepts any subcommand value as an <app-dir> input and
creates its root module directories.

This command creates:
  - Environment directory if not exists
  - Region subdirectory for the specified region
  - <app-dir> directory under the specified env & region
  - Terraform configuration files from go templates
  - Optional GitHub workflow files from templates

Configuration:
  The generate command reads .tfskel.yaml from the current directory by default.

Arguments:
  <app-dir>: Name of the application directory as subcommand input to create (required)`,
	Example: `  # Generate structure for an app in dev environment (uses .tfskel.yaml)
  tfskel generate myapp --env dev --region us-east-1

  # Generate with custom configuration file
  tfskel generate myapp --config ./my-config.yaml --env dev --region us-east-1

  # Generate with custom templates and GitHub workflows
  tfskel generate myapp --env stg --region eu-central-1 --templates-dir ./templates --create-github-workflows`,
	Args: cobra.ExactArgs(1),
	RunE: runGenerate,
}

var (
	env                   string
	region                string
	templatesDir          string
	s3BucketName          string
	createGithubWorkflows bool
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
	generateCmd.Flags().StringVar(&templatesDir, "templates-dir", "", "directory containing custom template files (all .tmpl files will be processed)")
	generateCmd.Flags().StringVar(&s3BucketName, "s3-bucket-name", "", "S3 bucket name for Terraform state")
	generateCmd.Flags().BoolVar(&createGithubWorkflows, "create-github-workflows", false, "create GitHub workflow files from default templates (disabled by default)")

	// Bind flags to viper - these should never fail unless there's a developer error
	// (flag name mismatch, missing flag, etc.) so we fail fast with panic
	mustBindPFlag := func(key string, flagName string) {
		if err := viper.BindPFlag(key, generateCmd.Flags().Lookup(flagName)); err != nil {
			panic(fmt.Sprintf("failed to bind flag %s to config key %s: %v", flagName, key, err))
		}
	}

	mustBindPFlag("generate.templates_dir", "templates-dir")
	mustBindPFlag("backend.s3.bucket_name", "s3-bucket-name")
	mustBindPFlag("generate.github_workflows.create", "create-github-workflows")
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
