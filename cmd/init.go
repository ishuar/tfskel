package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ishuar/tfskel/internal/bootstrap"
	"github.com/ishuar/tfskel/internal/fs"
	"github.com/ishuar/tfskel/internal/logger"
	"github.com/spf13/cobra"
)

// initOpts holds flag state for `tfskel init`.
type initOpts struct {
	root      *rootOpts
	dir       string
	workflows bool
	upgrade   bool
	force     bool
	skip      string
}

func newInitCmd(root *rootOpts) *cobra.Command {
	opts := &initOpts{root: root}

	cmd := &cobra.Command{
		Use:     "init",
		GroupID: "main",
		Short:   "Initialize tfskel project structure",
		Long: `Initializes a new Terraform monorepo with an environment-and-region-based
directory layout with sensible defaults already in place.

Generates a .mise.toml file declaring all required tools (terraform, tflint,
trivy, pre-commit, awscli) so they can be installed with a single command:

  mise install

Tool versions default to "latest". After creation, .mise.toml is user-owned —
edit it directly to pin specific versions.`,
		Example: `  # Initialize in current directory (uses .tfskel.yaml if present)
  tfskel init

  # Initialize in specific directory
  tfskel init --dir /path/to/project

  # Initialize with explicit config file
  tfskel init --config /path/to/config.yaml`,
		SilenceUsage: true,
		RunE:         opts.run,
	}

	cmd.Flags().StringVarP(&opts.dir, "dir", "d", "", "directory to initialize (default: current directory)")
	cmd.Flags().BoolVar(&opts.workflows, "workflows", false, "generate shared GitHub workflow files (reusable workflows and lint)")
	cmd.Flags().BoolVar(&opts.upgrade, "upgrade", false, "overwrite init-managed files with latest versions from this binary")
	cmd.Flags().BoolVar(&opts.force, "force", false, "overwrite files even without source markers (requires --upgrade)")
	cmd.Flags().StringVar(&opts.skip, "skip", "", "comma-separated files to skip during upgrade (requires --upgrade)")

	return cmd
}

func (o *initOpts) run(_ *cobra.Command, _ []string) error {
	log := logger.NewWithOptions(o.root.verbose, o.root.useColor)
	log.Debug("Starting init command")

	targetDir := o.dir
	if targetDir == "" {
		var err error
		targetDir, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
		log.Debugf("Using current working directory: %s", targetDir)
	}

	targetDir, err := filepath.Abs(targetDir)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	if o.force && !o.upgrade {
		return ErrForceRequiresUpgrade
	}
	if o.skip != "" && !o.upgrade {
		return ErrInitSkipRequiresUpgrade
	}

	log.Infof("Initializing tfskel project structure in: %s", targetDir)

	params, err := bootstrap.DetermineParameters(targetDir, log)
	if err != nil {
		return err
	}

	createWorkflows := o.workflows || params.CreateWorkflows
	if !createWorkflows {
		log.Debug("Skipping workflow creation (workflows.create: false in config and --workflows flag not set)")
	}

	var filesystem fs.FileSystem = fs.NewOSFileSystem()
	if o.root.dryRun {
		filesystem = fs.NewDryRunFileSystem(filesystem)
	}

	runner, err := bootstrap.NewRunner(filesystem, log, bootstrap.Options{
		Upgrade: o.upgrade,
		Force:   o.force,
		DryRun:  o.root.dryRun,
		Skip:    parseSkipList(o.skip),
	})
	if err != nil {
		return err
	}

	if err := runner.CreateProjectStructure(targetDir, params.TerraformVersion, params.Regions, params.Environments, createWorkflows); err != nil {
		return err
	}

	if o.root.dryRun {
		log.Info("Dry run complete — no files were written")
	} else {
		log.Successf("Successfully initialized tfskel project structure in: %s", targetDir)
	}

	log.Info("")
	log.Info("Next step: run 'tfskel validate' to check your setup and install tools")

	return nil
}
