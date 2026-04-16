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

var (
	validateFormat string
	validateSkip   string
)

var validateCmd = &cobra.Command{
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
	RunE:         runValidate,
}

func init() {
	rootCmd.AddCommand(validateCmd)

	validateCmd.Flags().StringVarP(&validateFormat, "format", "f", "table",
		"Output format: table, json, csv")
	validateCmd.Flags().StringVar(&validateSkip, "skip", "",
		"Comma-separated checks to skip (config, tools)")
}

func runValidate(cmd *cobra.Command, _ []string) error {
	log := logger.NewWithOptions(viper.GetBool("verbose"), useColor)

	checks, err := validate.ParseCheckSelection(validateSkip)
	if err != nil {
		return err
	}

	// Suppress logs for machine-readable formats.
	outputFormat := format.OutputFormat(validateFormat)
	if outputFormat == format.FormatJSON || outputFormat == format.FormatCSV {
		log.SetMachineOutput()
	}

	// Load config from .tfskel.yaml in the current directory (or --config path).
	// Validate must be run from the project root where .tfskel.yaml lives.
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

	// Format output
	formatter := validate.NewFormatter(useColor)
	if err := formatter.Format(report, outputFormat, os.Stdout); err != nil {
		return fmt.Errorf("failed to format output: %w", err)
	}

	exitCode := report.ExitCode()
	if exitCode != 0 {
		return NewExitError(exitCode, "")
	}

	return nil
}
