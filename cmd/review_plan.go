package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/ishuar/tfskel/internal/ai"
	"github.com/ishuar/tfskel/internal/format"
	"github.com/ishuar/tfskel/internal/logger"
	"github.com/ishuar/tfskel/internal/plan"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	// ErrFileRequired indicates the file flag was not provided.
	ErrFileRequired = errors.New("json plan file is required")
	// ErrFileNotFound indicates the specified file does not exist.
	ErrFileNotFound = errors.New("json plan file not found")
	// ErrInvalidFormat indicates an unsupported output format was specified.
	ErrInvalidFormat = errors.New("invalid format")
	// ErrInvalidFilter indicates an invalid filter value was specified.
	ErrInvalidFilter = errors.New("invalid filter")
	// ErrAIIncompatibleFormat indicates --ai was combined with a machine-readable format.
	ErrAIIncompatibleFormat = errors.New("--ai is only supported with --format=table")
)

// reviewPlanOpts holds flag state for `tfskel review plan`.
type reviewPlanOpts struct {
	root              *rootOpts
	planFile          string
	outputFormat      string
	topResourcesCount int
	filterSeverity    []string
	minSeverity       string
	filterAction      []string
	ai                bool
	aiModel           string
	aiMaxTokens       int
}

// newReviewPlanCmd builds the `review plan` command.
//
//nolint:funlen // Flag registration is linear pflag boilerplate; extracting helpers would only forward arguments without hiding complexity.
func newReviewPlanCmd(root *rootOpts) *cobra.Command {
	opts := &reviewPlanOpts{root: root}

	cmd := &cobra.Command{
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
  tfskel review plan --json-file tfplan.json --top-resources-count 5

  # Show only critical and high severity changes
  tfskel review plan --json-file tfplan.json --filter-severity critical,high

  # Show changes at high severity or above (high + critical)
  tfskel review plan --json-file tfplan.json --min-severity high

  # Show only deletions and replacements
  tfskel review plan --json-file tfplan.json --filter-action delete,replace

  # Combine filters (AND semantics)
  tfskel review plan --json-file tfplan.json --min-severity high --filter-action delete

  # Append an AI-generated narrative analysis (requires ANTHROPIC_API_KEY)
  tfskel review plan --json-file tfplan.json --ai

  # Use a specific Claude model for the AI analysis
  tfskel review plan --json-file tfplan.json --ai --ai-model claude-opus-4-7`,
		SilenceUsage: true,
		RunE:         opts.run,
	}

	cmd.Flags().StringVar(&opts.planFile, "json-file", "", "Path to Terraform plan JSON file (required)")
	if err := cmd.MarkFlagRequired("json-file"); err != nil {
		panic(fmt.Sprintf("failed to mark JSON plan file as required: %v", err))
	}

	cmd.Flags().StringVarP(&opts.outputFormat, "format", "f", "table",
		"Output format: table, json, csv")
	cmd.Flags().IntVar(&opts.topResourcesCount, "top-resources-count", -1,
		"Number of resources to show in top-N summaries (default: 10, 0 = unlimited)")
	cmd.Flags().StringSliceVar(&opts.filterSeverity, "filter-severity", nil,
		"Filter by exact severity: critical, high, medium, low (comma-separated)")
	cmd.Flags().StringVar(&opts.minSeverity, "min-severity", "",
		"Show resources at or above this severity: low, medium, high, critical")
	cmd.Flags().StringSliceVar(&opts.filterAction, "filter-action", nil,
		"Filter by action: create, update, delete, replace (comma-separated)")
	cmd.MarkFlagsMutuallyExclusive("filter-severity", "min-severity")

	cmd.Flags().BoolVar(&opts.ai, "ai", false,
		"Append an AI-generated narrative analysis (blast radius, security, rollback) using Claude. Requires ANTHROPIC_API_KEY.")
	cmd.Flags().StringVar(&opts.aiModel, "ai-model", "",
		"Claude model override (default: from config or built-in default)")
	cmd.Flags().IntVar(&opts.aiMaxTokens, "ai-max-tokens", 0,
		"Maximum response tokens for the AI analysis (0 = use config or built-in default)")

	return cmd
}

func (o *reviewPlanOpts) run(cmd *cobra.Command, _ []string) error {
	log := logger.NewWithOptions(o.root.verbose, o.root.useColor)

	filter, err := o.validateInputs(log)
	if err != nil {
		return err
	}

	if err := o.ensurePlanFileReadable(log); err != nil {
		return err
	}

	if o.outputFormat == string(format.FormatJSON) || o.outputFormat == string(format.FormatCSV) {
		log.SetMachineOutput()
	}
	log.Info("Reviewing terraform plan...")
	log.Infof("JSON plan file: %s", o.planFile)

	planData, err := plan.ParsePlanFile(o.planFile)
	if err != nil {
		log.Errorf("Failed to parse plan file: %v", err)
		return fmt.Errorf("failed to parse plan file: %w", err)
	}

	analyzer := plan.NewPlanAnalyzerWithConfig(viper.GetViper())
	analysis := analyzer.Analyze(planData)

	if !analysis.HasChanges {
		log.Success("No changes detected in plan - infrastructure is up to date")
		return nil
	}
	log.Infof("Found %d resource changes", analysis.TotalChanges)

	totalResourceCount := len(analysis.ResourceChanges)
	if !filter.IsEmpty() {
		analysis.ResourceChanges = plan.FilterResources(analysis.ResourceChanges, filter)
		log.Infof("Filtered to %d of %d resources", len(analysis.ResourceChanges), totalResourceCount)
	}

	if err := o.formatAndWrite(cmd, log, analysis, filter, totalResourceCount); err != nil {
		return err
	}

	if o.ai {
		// AI is additive: any failure here warns to stderr but never affects
		// the exit code or propagates as a command error.
		o.runAIAnalysis(cmd.Context(), log, planData, analysis)
	}

	if exitCode := analysis.ExitCode(); exitCode != 0 {
		log.Warnf("Changes detected - exiting with code %d", exitCode)
		return NewExitError(exitCode, "")
	}
	return nil
}

// validateInputs validates the format and filter flags and returns the
// constructed ResourceFilter for later use.
func (o *reviewPlanOpts) validateInputs(log *logger.Logger) (*plan.ResourceFilter, error) {
	validFormats := []string{string(format.FormatTable), string(format.FormatJSON), string(format.FormatCSV)}
	switch format.OutputFormat(o.outputFormat) {
	case format.FormatTable, format.FormatJSON, format.FormatCSV:
	default:
		log.Errorf("Invalid format: %s", o.outputFormat)
		return nil, fmt.Errorf("%w: %s (must be one of %s)", ErrInvalidFormat, o.outputFormat, strings.Join(validFormats, ", "))
	}

	if o.ai && format.OutputFormat(o.outputFormat) != format.FormatTable {
		log.Errorf("--ai is only supported with --format=table (got %s)", o.outputFormat)
		return nil, fmt.Errorf("%w (got --format=%s)", ErrAIIncompatibleFormat, o.outputFormat)
	}

	filter := &plan.ResourceFilter{
		Severities:  o.filterSeverity,
		MinSeverity: o.minSeverity,
		Actions:     o.filterAction,
	}
	if err := filter.Validate(); err != nil {
		log.Errorf("Invalid filter: %v", err)
		return nil, fmt.Errorf("%w: %w", ErrInvalidFilter, err)
	}
	return filter, nil
}

// ensurePlanFileReadable verifies the plan file exists and is accessible,
// surfacing a helpful error message (with generation hints) when it is missing.
func (o *reviewPlanOpts) ensurePlanFileReadable(log *logger.Logger) error {
	_, err := os.Stat(o.planFile)
	if err == nil {
		return nil
	}
	if os.IsNotExist(err) {
		log.Errorf("JSON plan file not found: %s", o.planFile)
		log.Info("Generate plan with:")
		log.Info("  terraform plan -out=tfplan.binary")
		log.Info("  terraform show -json tfplan.binary > tfplan.json")
		return fmt.Errorf("%w: %s", ErrFileNotFound, o.planFile)
	}
	log.Errorf("Failed to access file: %v", err)
	return fmt.Errorf("failed to access file: %w", err)
}

// runAIAnalysis streams Claude's narrative analysis to stdout below the table.
// All failure modes are non-fatal: a missing API key, a network blip, or an
// invalid model name produces a stderr warning and an early return — the
// command's exit code stays whatever the structured analysis decided.
func (o *reviewPlanOpts) runAIAnalysis(ctx context.Context, log *logger.Logger, planData *plan.TerraformPlan, analysis *plan.PlanAnalysis) {
	cfg := ai.LoadConfig(viper.GetViper())
	if o.aiModel != "" {
		cfg.Model = o.aiModel
	}
	if o.aiMaxTokens > 0 {
		cfg.MaxTokens = o.aiMaxTokens
	}

	client, err := ai.NewClient(ctx, cfg)
	if err != nil {
		log.Warnf("--ai requested but skipping AI analysis: %v", err)
		return
	}

	planCfg := plan.LoadAnalysisConfig(viper.GetViper())
	payload := ai.BuildPayload(planData, analysis, planCfg.CriticalResources)

	header := fmt.Sprintf("\n## AI Analysis\n\n- **Provider:** %s\n- **Model:** %s\n\n", client.Provider(), client.Model())
	if _, err := fmt.Fprint(os.Stdout, header); err != nil {
		log.Warnf("AI analysis: failed to write section header: %v", err)
		return
	}
	if err := client.Explain(ctx, payload, os.Stdout); err != nil {
		if errors.Is(err, ai.ErrResponseTruncated) {
			// Truncated output is still partially useful — finish the markdown
			// cleanly and warn the user how to get the full response next time.
			fmt.Fprintf(os.Stdout, "\n\n> **Output truncated** at the model's max-tokens ceiling. Re-run with `--ai-max-tokens <higher value>` (or set `ai.max_tokens` in your config) to see the full analysis.\n")
			log.Warnf("AI analysis was truncated (%v) — raise --ai-max-tokens to get the full response", err)
			return
		}
		log.Warnf("AI analysis failed: %v", err)
		return
	}
	if _, err := fmt.Fprintln(os.Stdout); err != nil {
		log.Warnf("AI analysis: failed to write trailing newline: %v", err)
	}
}

// formatAndWrite builds a PlanFormatter (filtered or unfiltered) and writes the
// formatted analysis to stdout. Top-resource count falls back to config unless
// the CLI flag was set explicitly.
func (o *reviewPlanOpts) formatAndWrite(cmd *cobra.Command, log *logger.Logger, analysis *plan.PlanAnalysis, filter *plan.ResourceFilter, totalResourceCount int) error {
	planConfig := plan.LoadAnalysisConfig(viper.GetViper())
	topResourcesCount := planConfig.TopResourcesCount
	if cmd.Flags().Changed("top-resources-count") {
		topResourcesCount = o.topResourcesCount
	}

	var formatter *plan.PlanFormatter
	if !filter.IsEmpty() {
		formatter = plan.NewPlanFormatterFiltered(o.root.useColor, topResourcesCount, totalResourceCount, filter.Descriptions())
	} else {
		formatter = plan.NewPlanFormatterWithConfig(o.root.useColor, topResourcesCount)
	}
	if err := formatter.Format(analysis, format.OutputFormat(o.outputFormat), os.Stdout); err != nil {
		log.Errorf("Failed to format output: %v", err)
		return fmt.Errorf("failed to format output: %w", err)
	}
	return nil
}
