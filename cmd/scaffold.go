package cmd

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/ishuar/tfskel/internal/fs"
	"github.com/ishuar/tfskel/internal/generate"
	"github.com/ishuar/tfskel/internal/logger"
	"github.com/ishuar/tfskel/internal/strutil"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// scaffoldOpts holds flag state for `tfskel scaffold`.
type scaffoldOpts struct {
	root         *rootOpts
	env          string
	region       string
	templatesDir string
	s3BucketName string
	upgrade      bool
	force        bool
	upgradeAll   bool
	skip         string
}

// scaffoldWorkflowsOpts holds flag state for `tfskel scaffold workflows`.
type scaffoldWorkflowsOpts struct {
	root    *rootOpts
	env     string
	upgrade bool
	force   bool
}

func newScaffoldCmd(root *rootOpts) *cobra.Command {
	opts := &scaffoldOpts{root: root}

	cmd := &cobra.Command{
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
		Args:         opts.args,
		SilenceUsage: true,
		RunE:         opts.run,
	}

	cmd.Flags().StringVarP(&opts.env, "env", "e", "", "target environment (e.g., dev, stg, prd) - required")
	cmd.Flags().StringVarP(&opts.region, "region", "r", "", "AWS region (e.g., us-east-1, eu-central-1) - required")
	if err := cmd.MarkFlagRequired("env"); err != nil {
		panic(fmt.Sprintf("failed to mark env flag as required: %v", err))
	}
	if err := cmd.MarkFlagRequired("region"); err != nil {
		panic(fmt.Sprintf("failed to mark region flag as required: %v", err))
	}

	cmd.Flags().StringVar(&opts.templatesDir, "templates-dir", "", "directory containing custom template files (all .tmpl files will be processed)")
	cmd.Flags().StringVar(&opts.s3BucketName, "s3-bucket-name", "", "S3 bucket name for Terraform state")
	cmd.Flags().BoolVar(&opts.upgrade, "upgrade", false, "re-render files from updated templates (only files with source markers)")
	cmd.Flags().BoolVar(&opts.force, "force", false, "overwrite files even without source markers (requires --upgrade or --upgrade-all)")
	cmd.Flags().BoolVar(&opts.upgradeAll, "upgrade-all", false, "re-render templates for all app directories under envs/<env>/<region>/")
	cmd.Flags().StringVar(&opts.skip, "skip", "", "comma-separated directories to skip (requires --upgrade-all)")

	// Viper binds here let config-file values back the CLI flags. Errors only
	// fire on programmer error (wrong flag name), so panic is correct.
	if err := viper.BindPFlag("templates.dir", cmd.Flags().Lookup("templates-dir")); err != nil {
		panic(fmt.Sprintf("failed to bind flag templates-dir: %v", err))
	}
	if err := viper.BindPFlag("backend.s3.bucket_name", cmd.Flags().Lookup("s3-bucket-name")); err != nil {
		panic(fmt.Sprintf("failed to bind flag s3-bucket-name: %v", err))
	}

	cmd.AddCommand(newScaffoldWorkflowsCmd(root))
	return cmd
}

func newScaffoldWorkflowsCmd(root *rootOpts) *cobra.Command {
	opts := &scaffoldWorkflowsOpts{root: root}

	cmd := &cobra.Command{
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
		RunE:         opts.run,
	}

	cmd.Flags().StringVarP(&opts.env, "env", "e", "", "target environment (e.g., dev, stg, prd) - required")
	if err := cmd.MarkFlagRequired("env"); err != nil {
		panic(fmt.Sprintf("failed to mark env flag as required: %v", err))
	}
	cmd.Flags().BoolVar(&opts.upgrade, "upgrade", false, "re-render workflow files from updated templates")
	cmd.Flags().BoolVar(&opts.force, "force", false, "with --upgrade, overwrite files even without source markers")

	return cmd
}

// args validates positional arguments for the scaffold command.
func (o *scaffoldOpts) args(cmd *cobra.Command, args []string) error {
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
}

func (o *scaffoldOpts) run(cmd *cobra.Command, args []string) error {
	if o.upgradeAll && o.upgrade {
		return ErrUpgradeAllWithUpgrade
	}
	if o.skip != "" && !o.upgradeAll {
		return ErrSkipRequiresUpgradeAll
	}
	if o.force && !o.upgrade && !o.upgradeAll {
		return ErrScaffoldForceRequiresUpgrade
	}

	if o.upgradeAll {
		return o.runUpgradeAll(cmd)
	}

	log := logger.NewWithOptions(o.root.verbose, o.root.useColor)
	log.Debug("Starting scaffold command")
	log.Info("Starting Terraform directory scaffolding...")

	appDir := args[0]

	trimmedEnv, trimmedRegion, trimmedAppDir, err := validateScaffoldParams(o.env, o.region, appDir)
	if err != nil {
		return fmt.Errorf("invalid parameters: %w", err)
	}

	cfg, err := loadAndValidateConfig(cmd, log)
	if err != nil {
		return err
	}

	var filesystem fs.FileSystem = fs.NewOSFileSystem()
	if o.root.dryRun {
		filesystem = fs.NewDryRunFileSystem(filesystem)
	}

	generator := generate.NewGenerator(cfg, filesystem, log)
	generator.SetUpgrade(o.upgrade, o.force)
	generator.SetDryRun(o.root.dryRun)
	if err := generator.Run(trimmedEnv, trimmedRegion, trimmedAppDir); err != nil {
		return fmt.Errorf("failed to scaffold Terraform structure: %w", err)
	}

	if summary := generator.Summary(); summary != "" {
		log.Infof("Summary: %s", summary)
	}
	if o.root.dryRun {
		log.Info("Dry run complete — no files were written")
	} else {
		log.Success("Terraform directory scaffolding completed!")
	}
	return nil
}

func (o *scaffoldOpts) runUpgradeAll(cmd *cobra.Command) error {
	log := logger.NewWithOptions(o.root.verbose, o.root.useColor)
	log.Debug("Starting scaffold upgrade-all")
	log.Info("Starting batch template upgrade...")

	trimmedEnv, err := strutil.TrimAndValidateInput(o.env, "environment")
	if err != nil {
		return fmt.Errorf("invalid parameters: %w (use --env flag)", err)
	}
	trimmedRegion, err := strutil.TrimAndValidateInput(o.region, "region")
	if err != nil {
		return fmt.Errorf("invalid parameters: %w (use --region flag)", err)
	}

	cfg, err := loadAndValidateConfig(cmd, log)
	if err != nil {
		return err
	}

	var filesystem fs.FileSystem = fs.NewOSFileSystem()
	if o.root.dryRun {
		filesystem = fs.NewDryRunFileSystem(filesystem)
	}

	basePath := filepath.Join("envs", trimmedEnv, trimmedRegion)
	if !filesystem.DirExists(basePath) {
		return fmt.Errorf("%w: %s", ErrBaseDirNotExist, basePath)
	}

	dirs, err := discoverAppDirs(filesystem, basePath, parseSkipList(o.skip))
	if err != nil {
		return fmt.Errorf("failed to read directory %s: %w", basePath, err)
	}
	if len(dirs) == 0 {
		return fmt.Errorf("%w in %s", ErrNoAppDirsFound, basePath)
	}

	log.Infof("Found %d app %s to upgrade in %s", len(dirs), dirWord(len(dirs)), basePath)

	// Single generator — tracker accumulates across all runs.
	generator := generate.NewGenerator(cfg, filesystem, log)
	generator.SetUpgrade(true, o.force)
	generator.SetDryRun(o.root.dryRun)

	for _, dir := range dirs {
		log.Infof("==> Scaffolding: %s", dir)
		if err := generator.Run(trimmedEnv, trimmedRegion, dir); err != nil {
			return fmt.Errorf("failed to scaffold %s: %w", dir, err)
		}
	}

	if summary := generator.Summary(); summary != "" {
		log.Infof("Summary: %s", summary)
	}
	if o.root.dryRun {
		log.Info("Dry run complete — no files were written")
	} else {
		log.Success("Terraform directory scaffolding completed!")
	}
	return nil
}

func (o *scaffoldWorkflowsOpts) run(cmd *cobra.Command, _ []string) error {
	log := logger.NewWithOptions(o.root.verbose, o.root.useColor)
	log.Debug("Starting scaffold workflows command")

	trimmedEnv, err := strutil.TrimAndValidateInput(o.env, "environment")
	if err != nil {
		return fmt.Errorf("invalid parameters: %w (use --env flag)", err)
	}

	cfg, err := loadAndValidateConfig(cmd, log)
	if err != nil {
		return err
	}

	if o.force && !o.upgrade {
		return ErrForceRequiresUpgrade
	}

	var filesystem fs.FileSystem = fs.NewOSFileSystem()
	if o.root.dryRun {
		filesystem = fs.NewDryRunFileSystem(filesystem)
	}

	generator := generate.NewGenerator(cfg, filesystem, log)
	generator.SetUpgrade(o.upgrade, o.force)
	generator.SetDryRun(o.root.dryRun)
	if err := generator.RunWorkflows(trimmedEnv); err != nil {
		return fmt.Errorf("failed to generate workflow files: %w", err)
	}

	if summary := generator.Summary(); summary != "" {
		log.Infof("Summary: %s", summary)
	}
	if o.root.dryRun {
		log.Info("Dry run complete — no files were written")
	} else {
		log.Success("GitHub workflow files generated!")
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
// and entries in the skip set. Results preserve the sorted order from ReadDir.
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
	return dirs, nil
}

func dirWord(n int) string {
	if n == 1 {
		return "directory"
	}
	return "directories"
}

// validateScaffoldParams validates and trims scaffolding parameters.
// For appDir, spaces are replaced with hyphens after trimming.
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

	trimmedAppDir = strutil.ReplaceSpacesWithHyphens(trimmedAppDir)

	return trimmedEnv, trimmedRegion, trimmedAppDir, nil
}
