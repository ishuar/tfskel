package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/ishuar/tfskel/internal/config"
	"github.com/ishuar/tfskel/internal/format"
	"github.com/ishuar/tfskel/internal/logger"
	"github.com/ishuar/tfskel/internal/plan"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	reviewPlanFile         string
	reviewPlanFormat       string
	reviewPlanTopResources int
)

var (
	// ErrFileRequired indicates the file flag was not provided
	ErrFileRequired = errors.New("json plan file is required")
	// ErrFileNotFound indicates the specified file does not exist
	ErrFileNotFound = errors.New("json plan file not found")
	// ErrInvalidFormat indicates an unsupported output format was specified
	ErrInvalidFormat = errors.New("invalid format: must be one of table, json, csv")
)

// reviewPlanCmd represents the review plan command
var reviewPlanCmd = &cobra.Command{
	Use:   "plan",
	Short: "Analyze Terraform plan JSON file & output a human-readable Terraform plan summary",
	Long: `Analyze a Terraform plan JSON file to detect and
categorize infrastructure changes. This command helps you
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
  tfskel review plan --json-file tfplan.json

  # Export analysis as JSON for CI/CD
  tfskel review plan --json-file tfplan.json --format json

  # Generate CSV report
  tfskel review plan --json-file tfplan.json --format csv > plan-analysis.csv

  # Analyze without colors (for logs)
  tfskel review plan --json-file tfplan.json --no-color

  # Limit top resource summaries to 5 items
  tfskel review plan --json-file tfplan.json --top-resources-count 5`,
	RunE: runReviewPlan,
}

func init() {
	reviewCmd.AddCommand(reviewPlanCmd)

	reviewPlanCmd.Flags().StringVar(&reviewPlanFile, "json-file", "",
		"Path to Terraform plan JSON file (required)")
	if err := reviewPlanCmd.MarkFlagRequired("json-file"); err != nil {
		panic(fmt.Sprintf("failed to mark JSON plan file as required: %v", err))
	}

	reviewPlanCmd.Flags().StringVarP(&reviewPlanFormat, "format", "f", "table",
		"Output format: table, json, csv")
	reviewPlanCmd.Flags().IntVar(&reviewPlanTopResources, "top-resources-count", -1,
		"Number of resources to show in top-N summaries (default: 10, 0 = unlimited)")
}

func runReviewPlan(cmd *cobra.Command, _ []string) error {
	log := logger.New(viper.GetBool("verbose"))

	// Validate output format
	switch format.OutputFormat(reviewPlanFormat) {
	case format.FormatTable, format.FormatJSON, format.FormatCSV:
		// valid format
	default:
		log.Errorf("Invalid format: %s", reviewPlanFormat)
		cmd.SilenceUsage = true
		return fmt.Errorf("%w", ErrInvalidFormat)
	}

	// Check if file exists
	if _, err := os.Stat(reviewPlanFile); err != nil {
		if os.IsNotExist(err) {
			log.Errorf("JSON plan file not found: %s", reviewPlanFile)
			log.Info("Generate plan with:")
			log.Info("  terraform plan -out=tfplan.binary")
			log.Info("  terraform show -json tfplan.binary > tfplan.json")
			cmd.SilenceUsage = true
			return fmt.Errorf("%w: %s", ErrFileNotFound, reviewPlanFile)
		}
		log.Errorf("Failed to access file: %v", err)
		cmd.SilenceUsage = true
		return fmt.Errorf("failed to access file: %w", err)
	}

	// Suppress logs for machine-readable formats
	if reviewPlanFormat == string(format.FormatJSON) || reviewPlanFormat == string(format.FormatCSV) {
		log.SetOutput(os.Stderr)
	}

	log.Info("Reviewing terraform plan...")
	log.Infof("JSON plan file: %s", reviewPlanFile)

	// Parse plan file using internal package
	planData, err := plan.ParsePlanFile(reviewPlanFile)
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

	// Load plan analysis config for formatter settings
	planConfig := config.LoadPlanAnalysisConfig(viper.GetViper())

	// Override config with flag if explicitly set
	topResourcesCount := planConfig.TopResourcesCount
	if cmd.Flags().Changed("top-resources-count") {
		topResourcesCount = reviewPlanTopResources
	}

	// Color profile already initialized in root PersistentPreRunE
	// Use the global noColor flag value
	useColor := format.ShouldUseColor(noColor)

	// Format and output using internal package
	formatter := plan.NewPlanFormatterWithConfig(useColor, topResourcesCount)
	if err := formatter.Format(analysis, format.OutputFormat(reviewPlanFormat), os.Stdout); err != nil {
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
