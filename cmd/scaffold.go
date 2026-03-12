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

var scaffoldCmd = &cobra.Command{
	Use:     "scaffold <app-dir>",
	Aliases: []string{"sc"},
	GroupID: "main",
	Short:   "Scaffold Terraform project structure for target application",
	Long: `Accepts any subcommand value as an <app-dir> input and
creates its root module directories.

This command creates:
  - Environment directory if not exists
  - Region subdirectory for the specified region
  - <app-dir> directory under the specified env & region
  - Terraform configuration files from go templates
  - Optional GitHub workflow files from templates

Configuration:
  The scaffold command reads .tfskel.yaml from the current directory by default.

Arguments:
  <app-dir>: Name of the application directory as subcommand input to create (required)`,
	Example: `  # Scaffold structure for an app in dev environment (uses .tfskel.yaml)
  tfskel scaffold myapp --env dev --region us-east-1

  # Scaffold with custom configuration file
  tfskel scaffold myapp --config ./my-config.yaml --env dev --region us-east-1

  # Scaffold with custom templates and GitHub workflows
  tfskel scaffold myapp --env stg --region eu-central-1 --templates-dir ./templates --workflows

  # Using the short alias
  tfskel sc myapp --env dev --region us-east-1`,
	Args: cobra.ExactArgs(1),
	RunE: runScaffold,
}

var (
	env          string
	region       string
	templatesDir string
	s3BucketName string
	workflows    bool
)

func init() {
	rootCmd.AddCommand(scaffoldCmd)

	// Required flags for generation
	scaffoldCmd.Flags().StringVarP(&env, "env", "e", "", "target environment (e.g., dev, stg, prd) - required")
	scaffoldCmd.Flags().StringVarP(&region, "region", "r", "", "AWS region (e.g., us-east-1, eu-central-1) - required")
	// These are critical flags - errors should be handled during command setup, but cobra handles this internally
	if err := scaffoldCmd.MarkFlagRequired("env"); err != nil {
		panic(fmt.Sprintf("failed to mark env flag as required: %v", err))
	}
	if err := scaffoldCmd.MarkFlagRequired("region"); err != nil {
		panic(fmt.Sprintf("failed to mark region flag as required: %v", err))
	}

	// Optional flags
	scaffoldCmd.Flags().StringVar(&templatesDir, "templates-dir", "", "directory containing custom template files (all .tmpl files will be processed)")
	scaffoldCmd.Flags().StringVar(&s3BucketName, "s3-bucket-name", "", "S3 bucket name for Terraform state")
	scaffoldCmd.Flags().BoolVar(&workflows, "workflows", false, "create GitHub workflow files from default templates (disabled by default)")

	// Bind flags to viper - these should never fail unless there's a developer error
	// (flag name mismatch, missing flag, etc.) so we fail fast with panic
	mustBindPFlag := func(key string, flagName string) {
		if err := viper.BindPFlag(key, scaffoldCmd.Flags().Lookup(flagName)); err != nil {
			panic(fmt.Sprintf("failed to bind flag %s to config key %s: %v", flagName, key, err))
		}
	}

	mustBindPFlag("templates.dir", "templates-dir")
	mustBindPFlag("backend.s3.bucket_name", "s3-bucket-name")
	mustBindPFlag("workflows.create", "workflows")
}

func runScaffold(cmd *cobra.Command, args []string) error {
	// Initialize logger
	log := logger.New(viper.GetBool("verbose"))

	log.Debug("Starting scaffold command")
	log.Info("Starting Terraform directory scaffolding...")

	// Get app directory from positional argument
	appDir := args[0]

	// Validate and trim generation parameters
	trimmedEnv, trimmedRegion, trimmedAppDir, err := validateScaffoldParams(env, region, appDir)
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
		return fmt.Errorf("failed to scaffold Terraform structure: %w", err)
	}

	log.Success("Terraform directory scaffolding completed!")
	return nil
}

// validateScaffoldParams validates and trims the generation parameters
// Returns trimmed values if validation passes
// For appDir, any spaces are replaced with hyphens after trimming
func validateScaffoldParams(env, region, appDir string) (string, string, string, error) {
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

	// Replace any spaces in appDir with hyphens
	trimmedAppDir = util.ReplaceSpacesWithHyphens(trimmedAppDir)

	return trimmedEnv, trimmedRegion, trimmedAppDir, nil
}
