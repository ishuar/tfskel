package cmd

import (
	"fmt"

	"github.com/ishuar/tfskel/internal/app"
	"github.com/ishuar/tfskel/internal/config"
	"github.com/ishuar/tfskel/internal/fs"
	"github.com/ishuar/tfskel/internal/logger"
	"github.com/ishuar/tfskel/internal/util"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var addCmd = &cobra.Command{
	Use:     "add <app-dir>",
	GroupID: "main",
	Short:   "Add Terraform project structure for target application",
	Long: `Accepts any subcommand value as an <app-dir> input and
creates its root module directories.

This command creates:
  - Environment directory if not exists
  - Region subdirectory for the specified region
  - <app-dir> directory under the specified env & region
  - Terraform configuration files from go templates
  - Optional GitHub workflow files from templates

Configuration:
  The add command reads .tfskel.yaml from the current directory by default.

Arguments:
  <app-dir>: Name of the application directory as subcommand input to create (required)`,
	Example: `  # Add structure for an app in dev environment (uses .tfskel.yaml)
  tfskel add myapp --env dev --region us-east-1

  # Add with custom configuration file
  tfskel add myapp --config ./my-config.yaml --env dev --region us-east-1

  # Add with custom templates and GitHub workflows
  tfskel add myapp --env stg --region eu-central-1 --templates-dir ./templates --create-github-workflows`,
	Args: cobra.ExactArgs(1),
	RunE: runAdd,
}

var (
	env                   string
	region                string
	templatesDir          string
	s3BucketName          string
	createGithubWorkflows bool
)

func init() {
	rootCmd.AddCommand(addCmd)

	// Required flags for generation
	addCmd.Flags().StringVarP(&env, "env", "e", "", "target environment (e.g., dev, stg, prd) - required")
	addCmd.Flags().StringVarP(&region, "region", "r", "", "AWS region (e.g., us-east-1, eu-central-1) - required")
	// These are critical flags - errors should be handled during command setup, but cobra handles this internally
	if err := addCmd.MarkFlagRequired("env"); err != nil {
		panic(fmt.Sprintf("failed to mark env flag as required: %v", err))
	}
	if err := addCmd.MarkFlagRequired("region"); err != nil {
		panic(fmt.Sprintf("failed to mark region flag as required: %v", err))
	}

	// Optional flags
	addCmd.Flags().StringVar(&templatesDir, "templates-dir", "", "directory containing custom template files (all .tmpl files will be processed)")
	addCmd.Flags().StringVar(&s3BucketName, "s3-bucket-name", "", "S3 bucket name for Terraform state")
	addCmd.Flags().BoolVar(&createGithubWorkflows, "create-github-workflows", false, "create GitHub workflow files from default templates (disabled by default)")

	// Bind flags to viper - these should never fail unless there's a developer error
	// (flag name mismatch, missing flag, etc.) so we fail fast with panic
	mustBindPFlag := func(key string, flagName string) {
		if err := viper.BindPFlag(key, addCmd.Flags().Lookup(flagName)); err != nil {
			panic(fmt.Sprintf("failed to bind flag %s to config key %s: %v", flagName, key, err))
		}
	}

	mustBindPFlag("generate.templates_dir", "templates-dir")
	mustBindPFlag("backend.s3.bucket_name", "s3-bucket-name")
	mustBindPFlag("generate.github_workflows.create", "create-github-workflows")
}

func runAdd(cmd *cobra.Command, args []string) error {
	// Initialize logger
	log := logger.New(viper.GetBool("verbose"))

	log.Debug("Starting add command")
	log.Info("Starting Terraform directory scaffolding...")

	// Get app directory from positional argument
	appDir := args[0]

	// Validate and trim generation parameters
	trimmedEnv, trimmedRegion, trimmedAppDir, err := validateAddParams(env, region, appDir)
	if err != nil {
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

	// Create and run the generator with trimmed generation parameters
	generator := app.NewGenerator(cfg, filesystem, log)
	if err := generator.Run(trimmedEnv, trimmedRegion, trimmedAppDir); err != nil {
		cmd.SilenceUsage = true
		return fmt.Errorf("failed to add Terraform structure: %w", err)
	}

	log.Success("Terraform directory scaffolding completed!")
	return nil
}

// validateAddParams validates and trims the generation parameters
// Returns trimmed values if validation passes
func validateAddParams(env, region, appDir string) (string, string, string, error) {
	trimmedEnv, err := util.TrimAndValidateInput(env, "environment")
	if err != nil {
		return "", "", "", fmt.Errorf("%w (use --env flag)", err)
	}

	trimmedRegion, err := util.TrimAndValidateInput(region, "region")
	if err != nil {
		return "", "", "", fmt.Errorf("%w (use --region flag)", err)
	}

	trimmedAppDir, err := util.TrimAndValidateInput(appDir, "app directory")
	if err != nil {
		return "", "", "", fmt.Errorf("%w (provide as argument)", err)
	}

	return trimmedEnv, trimmedRegion, trimmedAppDir, nil
}
