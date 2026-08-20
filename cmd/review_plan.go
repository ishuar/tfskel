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
	"github.com/ishuar/tfskel/internal/review"
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
		"Append an AI-generated narrative analysis (blast radius, security, rollback). Provider via TFSKEL_AI_PROVIDER=anthropic|gemini (default: anthropic); requires ANTHROPIC_API_KEY or GEMINI_API_KEY.")
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

	// Resolve all config once at the CLI seam; flag > config > default.
	planCfg := plan.LoadAnalysisConfig(viper.GetViper())
	topResourcesCount := planCfg.TopResourcesCount
	if cmd.Flags().Changed("top-resources-count") {
		topResourcesCount = o.topResourcesCount
	}

	req := review.Request{
		PlanFile:          o.planFile,
		Format:            format.OutputFormat(o.outputFormat),
		Filter:            filter,
		TopResourcesCount: topResourcesCount,
		UseColor:          o.root.useColor,
		CriticalResources: plan.MergeCriticalResources(plan.DefaultCriticalResources(), planCfg.CriticalResources),
		NewAIClient:       o.aiClientFactory(),
	}

	result, err := review.Run(cmd.Context(), req, os.Stdout, log)
	if err != nil {
		return err
	}
	if result.ExitCode != 0 {
		log.Warnf("Changes detected - exiting with code %d", result.ExitCode)
		return NewExitError(result.ExitCode, "")
	}
	return nil
}

// aiClientFactory returns the AI-client constructor for the review module, or
// nil when --ai was not requested. Flag overrides are applied here so the
// factory closes over a fully resolved config.
func (o *reviewPlanOpts) aiClientFactory() func(ctx context.Context) (ai.Client, error) {
	if !o.ai {
		return nil
	}
	cfg := ai.LoadConfig(viper.GetViper())
	if o.aiModel != "" {
		cfg.Model = o.aiModel
	}
	if o.aiMaxTokens > 0 {
		cfg.MaxTokens = o.aiMaxTokens
	}
	return func(ctx context.Context) (ai.Client, error) {
		return ai.NewClient(ctx, cfg)
	}
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
