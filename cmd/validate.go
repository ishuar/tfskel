package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ishuar/tfskel/internal/format"
	"github.com/ishuar/tfskel/internal/logger"
	"github.com/ishuar/tfskel/internal/validate"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// validateOpts holds flag state for `tfskel validate`.
type validateOpts struct {
	root   *rootOpts
	format string
	skip   string
}

func newValidateCmd(root *rootOpts) *cobra.Command {
	opts := &validateOpts{root: root}

	cmd := &cobra.Command{
		Use:     "validate",
		GroupID: "main",
		Short:   "Validate project health: version drift and tool installation",
		Long: `Checks whether your project is in sync with .tfskel.yaml by running
two validation checks:

  config  — Terraform/provider version constraints and .terraform-version files match config
  tools   — required tools are installed and at expected versions (compared against .mise.toml)

By default all checks run. Use --skip to exclude specific checks.`,
		Example: `  # Run all checks
  tfskel validate

  # Skip tool checks
  tfskel validate --skip tools

  # Skip config checks
  tfskel validate --skip config

  # Machine-readable JSON output for CI
  tfskel validate --format json`,
		SilenceUsage: true,
		RunE:         opts.run,
	}

	cmd.Flags().StringVarP(&opts.format, "format", "f", "table",
		"Output format: table, json, csv")
	cmd.Flags().StringVar(&opts.skip, "skip", "",
		"Comma-separated checks to skip (config, tools)")

	return cmd
}

func (o *validateOpts) run(cmd *cobra.Command, _ []string) error {
	log := logger.NewWithOptions(o.root.verbose, o.root.useColor)

	checks, err := validate.ParseCheckSelection(o.skip)
	if err != nil {
		return err
	}

	outputFormat := format.OutputFormat(o.format)
	if outputFormat == format.FormatJSON || outputFormat == format.FormatCSV {
		log.SetMachineOutput()
	}

	cfg, err := loadAndValidateConfig(cmd, log)
	if err != nil {
		return err
	}

	log.Info("Running validation checks...")

	scanDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to resolve working directory: %w", err)
	}

	// viper.ConfigFileUsed() is guaranteed non-empty by loadAndValidateConfig above.
	// It may be relative (when --config is passed as a relative path); normalize to
	// absolute so the report header is unambiguous regardless of invocation.
	configPath, err := filepath.Abs(viper.ConfigFileUsed())
	if err != nil {
		return fmt.Errorf("failed to resolve config path: %w", err)
	}

	runner := validate.NewRunner(cfg, scanDir, checks, configPath)
	report := runner.Run()

	formatter := validate.NewFormatter(o.root.useColor)
	if err := formatter.Format(report, outputFormat, os.Stdout); err != nil {
		return fmt.Errorf("failed to format output: %w", err)
	}

	exitCode := report.ExitCode()
	if exitCode != 0 {
		return NewExitError(exitCode, "")
	}

	return nil
}
