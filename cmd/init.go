package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ishuar/tfskel/internal/bootstrap"
	"github.com/ishuar/tfskel/internal/fs"
	"github.com/ishuar/tfskel/internal/logger"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var initCmd = &cobra.Command{
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
	RunE:         runInit,
}

var (
	initDir       string
	initWorkflows bool
	initUpgrade   bool
	initForce     bool
	initSkip      string
)

func init() {
	rootCmd.AddCommand(initCmd)
	initCmd.Flags().StringVarP(&initDir, "dir", "d", "", "directory to initialize (default: current directory)")
	initCmd.Flags().BoolVar(&initWorkflows, "workflows", false, "generate shared GitHub workflow files (reusable workflows and lint)")
	initCmd.Flags().BoolVar(&initUpgrade, "upgrade", false, "overwrite init-managed files with latest versions from this binary")
	initCmd.Flags().BoolVar(&initForce, "force", false, "overwrite files even without source markers (requires --upgrade)")
	initCmd.Flags().StringVar(&initSkip, "skip", "", "comma-separated files to skip during upgrade (requires --upgrade)")
}

func runInit(_ *cobra.Command, _ []string) error {
	log := logger.NewWithOptions(viper.GetBool("verbose"), useColor)
	log.Debug("Starting init command")

	targetDir := initDir
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

	if initForce && !initUpgrade {
		return ErrForceRequiresUpgrade
	}
	if initSkip != "" && !initUpgrade {
		return ErrInitSkipRequiresUpgrade
	}

	log.Infof("Initializing tfskel project structure in: %s", targetDir)

	params, err := bootstrap.DetermineParameters(targetDir, log)
	if err != nil {
		return err
	}

	createWorkflows := initWorkflows || params.CreateWorkflows
	if !createWorkflows {
		log.Debug("Skipping workflow creation (workflows.create: false in config and --workflows flag not set)")
	}

	var filesystem fs.FileSystem = fs.NewOSFileSystem()
	if dryRun {
		filesystem = fs.NewDryRunFileSystem(filesystem)
	}

	runner, err := bootstrap.NewRunner(filesystem, log, bootstrap.Options{
		Upgrade: initUpgrade,
		Force:   initForce,
		DryRun:  dryRun,
		Skip:    parseSkipList(initSkip),
	})
	if err != nil {
		return err
	}

	if err := runner.CreateProjectStructure(targetDir, params.TerraformVersion, params.Regions, params.Environments, createWorkflows); err != nil {
		return err
	}

	if dryRun {
		log.Info("Dry run complete — no files were written")
	} else {
		log.Successf("Successfully initialized tfskel project structure in: %s", targetDir)
	}

	log.Info("")
	log.Info("Next step: run 'tfskel validate' to check your setup and install tools")

	return nil
}
