package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/ishuar/tfskel/internal/drift"
	"github.com/ishuar/tfskel/internal/logger"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	planFile    string
	planFormat  string
	planNoColor bool
)

var (
	ErrPlanFileRequired = errors.New("plan file is required")
	ErrPlanFileNotFound = errors.New("plan file not found")
)

// driftPlanCmd represents the drift plan command
var driftPlanCmd = &cobra.Command{
	Use:   "plan",
	Short: "Analyze terraform plan output for changes",
	Long: `Analyze a terraform plan JSON file to detect and categorize infrastructure changes.
This command helps you understand the impact of planned changes before applying them.

The plan file must be in JSON format, generated with:
  terraform plan -out=tfplan.binary
  terraform show -json tfplan.binary > tfplan.json

Change Categories:
  • Additions      - New resources being created
  • Modifications  - Existing resources being updated
  • Deletions      - Resources being destroyed
  • Replacements   - Resources being recreated (delete + create)

Severity Levels:
  • Critical  - Deletions or replacements (data loss risk)
  • High      - Modifications to critical resources
  • Medium    - Standard resource modifications
  • Low       - Additions only

Examples:
  # Analyze a plan file
  tfskel drift plan --plan-file tfplan.json

  # Export analysis as JSON for CI/CD
  tfskel drift plan --plan-file tfplan.json --format json

  # Generate CSV report
  tfskel drift plan --plan-file tfplan.json --format csv > plan-analysis.csv

  # Analyze without colors (for logs)
  tfskel drift plan --plan-file tfplan.json --no-color`,
	RunE: runDriftPlan,
}

func init() {
	driftCmd.AddCommand(driftPlanCmd)

	driftPlanCmd.Flags().StringVar(&planFile, "plan-file", "",
		"Path to terraform plan JSON file (required)")
	if err := driftPlanCmd.MarkFlagRequired("plan-file"); err != nil {
		// This should never happen with a valid flag name, but handle it for completeness
		panic(fmt.Sprintf("failed to mark plan-file as required: %v", err))
	}

	driftPlanCmd.Flags().StringVarP(&planFormat, "format", "f", "table",
		"Output format: table, json, csv")
	driftPlanCmd.Flags().BoolVar(&planNoColor, "no-color", false,
		"Disable colored output")
}

func runDriftPlan(cmd *cobra.Command, args []string) error {
	log := logger.New(viper.GetBool("verbose"))

	// Validate plan file path
	if planFile == "" {
		log.Error("Plan file is required. Use --plan-file flag to specify the path.")
		cmd.SilenceUsage = true
		return ErrPlanFileRequired
	}

	// Check if file exists
	if _, err := os.Stat(planFile); err != nil {
		if os.IsNotExist(err) {
			log.Errorf("Plan file not found: %s", planFile)
			log.Info("Generate plan with:")
			log.Info("  terraform plan -out=tfplan.binary")
			log.Info("  terraform show -json tfplan.binary > tfplan.json")
			cmd.SilenceUsage = true
			return fmt.Errorf("%w: %s", ErrPlanFileNotFound, planFile)
		}
		log.Errorf("Failed to access plan file: %v", err)
		cmd.SilenceUsage = true
		return fmt.Errorf("failed to access plan file: %w", err)
	}

	// Suppress logs for machine-readable formats
	if planFormat == "json" || planFormat == "csv" {
		log.SetOutput(os.Stderr)
	}

	log.Info("Analyzing terraform plan...")
	log.Infof("Plan file: %s", planFile)

	// Parse plan file using internal package
	plan, err := drift.ParsePlanFile(planFile)
	if err != nil {
		log.Errorf("Failed to parse plan file: %v", err)
		cmd.SilenceUsage = true
		return fmt.Errorf("failed to parse plan file: %w", err)
	}

	// Analyze the plan using internal package
	analyzer := drift.NewPlanAnalyzerWithConfig(viper.GetViper())
	analysis := analyzer.Analyze(plan)

	if !analysis.HasChanges {
		log.Success("No changes detected in plan - infrastructure is up to date")
		return nil
	}

	log.Infof("Found %d resource changes", analysis.TotalChanges)

	// Load drift config for formatter settings
	driftConfig := drift.LoadDriftConfig(viper.GetViper())

	// Format and output using internal package
	formatter := drift.NewPlanFormatterWithConfig(!planNoColor, driftConfig.TopNCount)
	if err := formatter.Format(analysis, drift.OutputFormat(planFormat), os.Stdout); err != nil {
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
