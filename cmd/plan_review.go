package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/ishuar/tfskel/internal/logger"
	"github.com/ishuar/tfskel/internal/output"
	"github.com/ishuar/tfskel/internal/plan"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	planReviewFile    string
	planReviewFormat  string
	planReviewNoColor bool
)

var (
	// ErrFileRequired indicates the file flag was not provided
	ErrFileRequired = errors.New("json plan file is required")
	// ErrFileNotFound indicates the specified file does not exist
	ErrFileNotFound = errors.New("json plan file not found")
)

// planReviewCmd represents the plan review command
var planReviewCmd = &cobra.Command{
	Use:   "review",
	Short: "Analyze Terraform plan json file & output a human-readable terraform plan summary",
	Long: `Analyze a terraform plan JSON file to detect and
categorize infrastructure changes.This command helps you
understand the impact of planned changes before applying them.

The plan file must be in JSON format, generated with:
  terraform plan -out=tfplan.binary
  terraform show -json tfplan.binary > tfplan.json

Change Categories:
  - Additions      - New resources being created
  - Modifications  - Existing resources being updated
  - Deletions      - Resources being destroyed
  - Replacements   - Resources being recreated (delete + create)

Severity Levels:
  - Critical  - Deletions or replacements (data loss risk)
  - High      - Modifications to critical resources
  - Medium    - Standard resource modifications
  - Low       - Additions only`,

	Example: `  # Analyze a plan file
  tfskel plan review --json-file tfplan.json

  # Export analysis as JSON for CI/CD
  tfskel plan review --json-file tfplan.json --format json

  # Generate CSV report
  tfskel plan review --json-file tfplan.json --format csv > plan-analysis.csv

  # Analyze without colors (for logs)
  tfskel plan review --json-file tfplan.json --no-color`,
	RunE: runPlanReview,
}

func init() {
	planCmd.AddCommand(planReviewCmd)

	planReviewCmd.Flags().StringVar(&planReviewFile, "json-file", "",
		"Path to terraform plan JSON file (required)")
	if err := planReviewCmd.MarkFlagRequired("json-file"); err != nil {
		panic(fmt.Sprintf("failed to mark JSON plan file as required: %v", err))
	}

	planReviewCmd.Flags().StringVarP(&planReviewFormat, "format", "f", "table",
		"Output format: table, json, csv")
	planReviewCmd.Flags().BoolVar(&planReviewNoColor, "no-color", false,
		"Disable colored output")
}

func runPlanReview(cmd *cobra.Command, _ []string) error {
	log := logger.New(viper.GetBool("verbose"))

	// Validate plan file path
	if planReviewFile == "" {
		log.Error("JSON plan file is required. Use --json-file flag to specify the path.")
		cmd.SilenceUsage = true
		return ErrFileRequired
	}

	// Check if file exists
	if _, err := os.Stat(planReviewFile); err != nil {
		if os.IsNotExist(err) {
			log.Errorf("JSON plan file not found: %s", planReviewFile)
			log.Info("Generate plan with:")
			log.Info("  terraform plan -out=tfplan.binary")
			log.Info("  terraform show -json tfplan.binary > tfplan.json")
			cmd.SilenceUsage = true
			return fmt.Errorf("%w: %s", ErrFileNotFound, planReviewFile)
		}
		log.Errorf("Failed to access file: %v", err)
		cmd.SilenceUsage = true
		return fmt.Errorf("failed to access file: %w", err)
	}

	// Suppress logs for machine-readable formats
	if planReviewFormat == formatJSON || planReviewFormat == formatCSV {
		log.SetOutput(os.Stderr)
	}

	log.Info("Analyzing terraform plan...")
	log.Infof("JSON plan file: %s", planReviewFile)

	// Parse plan file using internal package
	planData, err := plan.ParsePlanFile(planReviewFile)
	if err != nil {
		log.Errorf("Failed to parse plan file: %v", err)
		cmd.SilenceUsage = true
		return fmt.Errorf("failed to parse plan file: %w", err)
	}

	// Analyze the plan using internal package
	analyzer := plan.NewPlanAnalyzerWithConfig(viper.GetViper())
	analysis := analyzer.Analyze(planData)

	if !analysis.HasChanges {
		log.Success("No changes detected in plan - infrastructure is up to date")
		return nil
	}

	log.Infof("Found %d resource changes", analysis.TotalChanges)

	// Load drift config for formatter settings
	driftConfig := output.LoadDriftConfig(viper.GetViper())

	// Format and output using internal package
	formatter := plan.NewPlanFormatterWithConfig(!planReviewNoColor, driftConfig.TopNCount)
	if err := formatter.Format(analysis, output.OutputFormat(planReviewFormat), os.Stdout); err != nil {
		log.Errorf("Failed to format output: %v", err)
		cmd.SilenceUsage = true
		return fmt.Errorf("failed to format output: %w", err)
	}

	// Return ExitError if changes detected for proper exit code handling
	exitCode := analysis.ExitCode()
	if exitCode != 0 {
		log.Warnf("Changes detected - exiting with code %d", exitCode)
		cmd.SilenceUsage = true
		return NewExitError(exitCode, "")
	}

	return nil
}
