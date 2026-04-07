package cmd

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ishuar/tfskel/internal/config"
	"github.com/ishuar/tfskel/internal/fs"
	"github.com/ishuar/tfskel/internal/generate"
	"github.com/ishuar/tfskel/internal/logger"
	"github.com/ishuar/tfskel/internal/strutil"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var scaffoldCmd = &cobra.Command{
	Use:     "scaffold [app-dir]",
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

Configuration:
  The scaffold command reads .tfskel.yaml from the current directory by default.

Arguments:
  <app-dir>: Name of the application directory to create (required unless --upgrade-all is set)

Subcommands:
  workflows   Generate per-environment GitHub Actions workflow files`,
	Example: `  # Scaffold structure for an app in dev environment (uses .tfskel.yaml)
  tfskel scaffold myapp --env dev --region us-east-1

  # Scaffold with custom configuration file
  tfskel scaffold myapp --config ./my-config.yaml --env dev --region us-east-1

  # Scaffold with custom templates
  tfskel scaffold myapp --env stg --region eu-central-1 --templates-dir ./templates

  # Upgrade all app directories in a region
  tfskel scaffold --upgrade-all --env prd --region eu-central-1

  # Upgrade all, skipping specific directories
  tfskel scaffold --upgrade-all --env prd --region eu-central-1 --skip base-infra,pre-prod

  # Generate GitHub workflow for a specific environment
  tfskel scaffold workflows --env dev

  # Using the short alias
  tfskel sc myapp --env dev --region us-east-1`,
	Args: func(cmd *cobra.Command, args []string) error {
		upgradeAll, err := cmd.Flags().GetBool("upgrade-all")
		if err != nil {
			return err
		}
		if upgradeAll {
			if len(args) > 0 {
				return ErrUpgradeAllWithAppDir
			}
			return nil
		}
		return cobra.ExactArgs(1)(cmd, args)
	},
	SilenceUsage: true,
	RunE:         runScaffold,
}

var scaffoldWorkflowsCmd = &cobra.Command{
	Use:   "workflows",
	Short: "Generate per-environment GitHub Actions workflow files",
	Long: `Generates per-environment GitHub Actions workflow files from templates.

This command creates:
  - .github/workflows/<env>-terraform-plan-apply.yaml

Use 'tfskel init' to generate shared reusable workflow files (lint.yaml,
reusable-detect-changes.yaml, reusable-terraform-plan-apply.yaml, reusable-lint.yaml).

Configuration:
  Reads .tfskel.yaml from the current directory. Workflow generation can be
  disabled entirely with 'workflows.create: false' in the config file.`,
	Example: `  # Generate workflow for dev environment
  tfskel scaffold workflows --env dev

  # Generate workflow with custom config
  tfskel scaffold workflows --env prd --config ./my-config.yaml`,
	SilenceUsage: true,
	RunE:         runScaffoldWorkflows,
}

var (
	env                string
	region             string
	templatesDir       string
	s3BucketName       string
	workflowsEnv       string
	scaffoldUpgrade    bool
	scaffoldForce      bool
	scaffoldUpgradeAll bool
	scaffoldSkip       string
	workflowUpgrade    bool
	workflowForce      bool
)

func init() {
	rootCmd.AddCommand(scaffoldCmd)
	scaffoldCmd.AddCommand(scaffoldWorkflowsCmd)

	// Required flags for scaffolding
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
	scaffoldCmd.Flags().BoolVar(&scaffoldUpgrade, "upgrade", false, "re-render files from updated templates (only files with source markers)")
	scaffoldCmd.Flags().BoolVar(&scaffoldForce, "force", false, "with --upgrade or --upgrade-all, overwrite files even without source markers")
	scaffoldCmd.Flags().BoolVar(&scaffoldUpgradeAll, "upgrade-all", false, "re-render templates for all app directories under envs/<env>/<region>/")
	scaffoldCmd.Flags().StringVar(&scaffoldSkip, "skip", "", "comma-separated directories to skip (requires --upgrade-all)")

	// Bind flags to viper - these should never fail unless there's a developer error
	// (flag name mismatch, missing flag, etc.) so we fail fast with panic
	mustBindPFlag := func(key string, flagName string) {
		if err := viper.BindPFlag(key, scaffoldCmd.Flags().Lookup(flagName)); err != nil {
			panic(fmt.Sprintf("failed to bind flag %s to config key %s: %v", flagName, key, err))
		}
	}

	mustBindPFlag("templates.dir", "templates-dir")
	mustBindPFlag("backend.s3.bucket_name", "s3-bucket-name")

	// scaffold workflows flags
	scaffoldWorkflowsCmd.Flags().StringVarP(&workflowsEnv, "env", "e", "", "target environment (e.g., dev, stg, prd) - required")
	if err := scaffoldWorkflowsCmd.MarkFlagRequired("env"); err != nil {
		panic(fmt.Sprintf("failed to mark env flag as required: %v", err))
	}
	scaffoldWorkflowsCmd.Flags().BoolVar(&workflowUpgrade, "upgrade", false, "re-render workflow files from updated templates")
	scaffoldWorkflowsCmd.Flags().BoolVar(&workflowForce, "force", false, "with --upgrade, overwrite files even without source markers")
}

func runScaffold(cmd *cobra.Command, args []string) error {
	// Validate mutually exclusive flags
	if scaffoldUpgradeAll && scaffoldUpgrade {
		return ErrUpgradeAllWithUpgrade
	}
	if scaffoldSkip != "" && !scaffoldUpgradeAll {
		return ErrSkipRequiresUpgradeAll
	}
	if scaffoldForce && !scaffoldUpgrade && !scaffoldUpgradeAll {
		return ErrForceRequiresUpgrade
	}

	if scaffoldUpgradeAll {
		return runScaffoldUpgradeAll(cmd)
	}

	// Initialize logger
	log := logger.NewWithOptions(viper.GetBool("verbose"), useColor)

	log.Debug("Starting scaffold command")
	log.Info("Starting Terraform directory scaffolding...")

	// Get app directory from positional argument
	appDir := args[0]

	// Validate and trim scaffolding parameters
	trimmedEnv, trimmedRegion, trimmedAppDir, err := validateScaffoldParams(env, region, appDir)
	if err != nil {
		return fmt.Errorf("invalid parameters: %w", err)
	}

	// Load configuration
	cfg, err := config.Load(cmd, viper.GetViper(), log)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}

	// Create filesystem abstraction
	var filesystem fs.FileSystem = fs.NewOSFileSystem()
	if dryRun {
		filesystem = fs.NewDryRunFileSystem(filesystem)
	}

	// Create and run the generator with trimmed scaffolding parameters
	generator := generate.NewGenerator(cfg, filesystem, log)
	generator.SetUpgrade(scaffoldUpgrade, scaffoldForce)
	generator.SetDryRun(dryRun)
	if err := generator.Run(trimmedEnv, trimmedRegion, trimmedAppDir); err != nil {
		return fmt.Errorf("failed to scaffold Terraform structure: %w", err)
	}

	if summary := generator.Summary(); summary != "" {
		log.Infof("Summary: %s", summary)
	}
	if dryRun {
		log.Info("Dry run complete — no files were written")
	} else {
		log.Success("Terraform directory scaffolding completed!")
	}
	return nil
}

func runScaffoldUpgradeAll(cmd *cobra.Command) error {
	log := logger.NewWithOptions(viper.GetBool("verbose"), useColor)

	log.Debug("Starting scaffold upgrade-all")
	log.Info("Starting batch template upgrade...")

	// Validate and trim env and region
	trimmedEnv, err := strutil.TrimAndValidateInput(env, "environment")
	if err != nil {
		return fmt.Errorf("invalid parameters: %w (use --env flag)", err)
	}
	trimmedRegion, err := strutil.TrimAndValidateInput(region, "region")
	if err != nil {
		return fmt.Errorf("invalid parameters: %w (use --region flag)", err)
	}

	// Load and validate config
	cfg, err := config.Load(cmd, viper.GetViper(), log)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}

	// Create filesystem abstraction
	var filesystem fs.FileSystem = fs.NewOSFileSystem()
	if dryRun {
		filesystem = fs.NewDryRunFileSystem(filesystem)
	}

	// Discover app directories
	basePath := filepath.Join("envs", trimmedEnv, trimmedRegion)
	if !filesystem.DirExists(basePath) {
		return fmt.Errorf("%w: %s", ErrBaseDirNotExist, basePath)
	}

	dirs, err := discoverAppDirs(filesystem, basePath, parseSkipList(scaffoldSkip))
	if err != nil {
		return fmt.Errorf("failed to read directory %s: %w", basePath, err)
	}
	if len(dirs) == 0 {
		return fmt.Errorf("%w in %s", ErrNoAppDirsFound, basePath)
	}

	log.Infof("Found %d app %s to upgrade in %s", len(dirs), dirWord(len(dirs)), basePath)

	// Single generator — tracker accumulates across all runs
	generator := generate.NewGenerator(cfg, filesystem, log)
	generator.SetUpgrade(true, scaffoldForce)
	generator.SetDryRun(dryRun)

	for _, dir := range dirs {
		log.Infof("==> Scaffolding: %s", dir)
		if err := generator.Run(trimmedEnv, trimmedRegion, dir); err != nil {
			return fmt.Errorf("failed to scaffold %s: %w", dir, err)
		}
	}

	if summary := generator.Summary(); summary != "" {
		log.Infof("Summary: %s", summary)
	}
	if dryRun {
		log.Info("Dry run complete — no files were written")
	} else {
		log.Success("Terraform directory scaffolding completed!")
	}
	return nil
}

// parseSkipList parses a comma-separated string into a set of directory names.
func parseSkipList(raw string) map[string]bool {
	if raw == "" {
		return nil
	}
	skips := make(map[string]bool)
	for name := range strings.SplitSeq(raw, ",") {
		trimmed := strings.TrimSpace(name)
		if trimmed != "" {
			skips[trimmed] = true
		}
	}
	return skips
}

// discoverAppDirs lists subdirectories in basePath, filtering out hidden dirs
// and entries in the skip set. Results are sorted alphabetically.
func discoverAppDirs(filesystem fs.FileSystem, basePath string, skip map[string]bool) ([]string, error) {
	entries, err := filesystem.ReadDir(basePath)
	if err != nil {
		return nil, err
	}

	var dirs []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		if skip[name] {
			continue
		}
		dirs = append(dirs, name)
	}
	sort.Strings(dirs)
	return dirs, nil
}

func dirWord(n int) string {
	if n == 1 {
		return "directory"
	}
	return "directories"
}

func runScaffoldWorkflows(cmd *cobra.Command, _ []string) error {
	log := logger.NewWithOptions(viper.GetBool("verbose"), useColor)
	log.Debug("Starting scaffold workflows command")

	trimmedEnv, err := strutil.TrimAndValidateInput(workflowsEnv, "environment")
	if err != nil {
		return fmt.Errorf("invalid parameters: %w (use --env flag)", err)
	}

	cfg, err := config.Load(cmd, viper.GetViper(), log)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}

	// Validate flag combination: --force requires --upgrade
	if workflowForce && !workflowUpgrade {
		return ErrForceRequiresUpgrade
	}

	var filesystem fs.FileSystem = fs.NewOSFileSystem()
	if dryRun {
		filesystem = fs.NewDryRunFileSystem(filesystem)
	}

	generator := generate.NewGenerator(cfg, filesystem, log)
	generator.SetUpgrade(workflowUpgrade, workflowForce)
	generator.SetDryRun(dryRun)
	if err := generator.RunWorkflows(trimmedEnv); err != nil {
		return fmt.Errorf("failed to generate workflow files: %w", err)
	}

	if summary := generator.Summary(); summary != "" {
		log.Infof("Summary: %s", summary)
	}
	if dryRun {
		log.Info("Dry run complete — no files were written")
	} else {
		log.Success("GitHub workflow files generated!")
	}
	return nil
}

// validateScaffoldParams validates and trims the scaffolding parameters
// Returns trimmed values if validation passes
// For appDir, any spaces are replaced with hyphens after trimming
func validateScaffoldParams(env, region, appDir string) (string, string, string, error) {
	trimmedEnv, err := strutil.TrimAndValidateInput(env, "environment")
	if err != nil {
		return "", "", "", fmt.Errorf("%w (use --env flag)", err)
	}

	trimmedRegion, err := strutil.TrimAndValidateInput(region, "region")
	if err != nil {
		return "", "", "", fmt.Errorf("%w (use --region flag)", err)
	}

	trimmedAppDir, err := strutil.TrimAndValidateInput(appDir, "app directory")
	if err != nil {
		return "", "", "", fmt.Errorf("%w (provide as argument)", err)
	}

	// Replace any spaces in appDir with hyphens
	trimmedAppDir = strutil.ReplaceSpacesWithHyphens(trimmedAppDir)

	return trimmedEnv, trimmedRegion, trimmedAppDir, nil
}
