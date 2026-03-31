package cmd

import (
	"fmt"
	"os"

	"github.com/ishuar/tfskel/internal/config"
	"github.com/ishuar/tfskel/internal/format"
	"github.com/ishuar/tfskel/internal/logger"
	"github.com/ishuar/tfskel/internal/validate"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	validateFormat string
	validateOnly   string
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

By default all checks run. Use --only or --skip to control which checks execute.`,
	Example: `  # Run all checks
  tfskel validate

  # Run only version drift check
  tfskel validate --only config

  # Skip tool checks
  tfskel validate --skip tools

  # Machine-readable JSON output for CI
  tfskel validate --format json`,
	SilenceUsage: true,
	RunE:         runValidate,
}

func init() {
	rootCmd.AddCommand(validateCmd)

	validateCmd.Flags().StringVarP(&validateFormat, "format", "f", "table",
		"Output format: table, json, csv")
	validateCmd.Flags().StringVar(&validateOnly, "only", "",
		"Comma-separated checks to run (config, tools)")
	validateCmd.Flags().StringVar(&validateSkip, "skip", "",
		"Comma-separated checks to skip (config, tools)")
	validateCmd.MarkFlagsMutuallyExclusive("only", "skip")
}

func runValidate(cmd *cobra.Command, _ []string) error {
	log := logger.NewWithOptions(viper.GetBool("verbose"), useColor)

	// Parse check selection
	checks, err := validate.ParseCheckSelection(validateOnly, validateSkip)
	if err != nil {
		return err
	}

	// Suppress logs for machine-readable formats
	outputFormat := format.OutputFormat(validateFormat)
	if outputFormat == format.FormatJSON || outputFormat == format.FormatCSV {
		log.SetMachineOutput()
	}

	// Load config from .tfskel.yaml in the current directory (or --config path).
	// Validate must be run from the project root where .tfskel.yaml lives.
	cfg, err := config.Load(cmd, viper.GetViper(), log)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	log.Info("Running validation checks...")

	// Run checks against the current directory
	runner := validate.NewRunner(cfg, ".", checks)
	report := runner.Run()

	// Format output
	formatter := validate.NewFormatter(useColor)
	if err := formatter.Format(report, outputFormat, os.Stdout); err != nil {
		return fmt.Errorf("failed to format output: %w", err)
	}

	// Exit with appropriate code
	exitCode := report.ExitCode()
	if exitCode != 0 {
		return NewExitError(exitCode, "")
	}

	return nil
}
