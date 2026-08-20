// Package review owns the `tfskel review plan` flow: parse → analyze →
// filter → format → optional AI narrative → exit-code derivation.
//
// The module's central invariant: once the filter is applied, the analysis is
// the single source of truth for everything downstream — the formatted report
// and the AI payload both see exactly the filtered resource set. The AI step
// is additive: any AI failure warns via the logger but never surfaces as an
// error and never affects the exit code.
package review

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/ishuar/tfskel/internal/ai"
	"github.com/ishuar/tfskel/internal/format"
	"github.com/ishuar/tfskel/internal/logger"
	"github.com/ishuar/tfskel/internal/plan"
)

// Request carries everything a review run needs, resolved by the caller at
// the CLI seam: flag > config > default precedence is decided once out there,
// never re-derived in here.
type Request struct {
	// PlanFile is the path to the Terraform plan JSON file.
	PlanFile string
	// Format selects the report output format.
	Format format.OutputFormat
	// Filter restricts which analyzed resources are reported and sent to the
	// AI. Never nil; may be empty (no filtering).
	Filter *plan.ResourceFilter
	// TopResourcesCount bounds the top-N summary tables (0 = unlimited).
	TopResourcesCount int
	// UseColor toggles ANSI colors in the table format.
	UseColor bool
	// CriticalResources is the effective critical-resource list — defaults
	// merged with user config. It drives both severity classification and the
	// list sent to the AI, so the two can never disagree.
	CriticalResources []string
	// NewAIClient constructs the AI client when the narrative analysis was
	// requested; nil means no AI step. Injected as a factory so tests can
	// substitute a stub and so construction errors (e.g. a missing API key)
	// stay inside the additive-failure contract.
	NewAIClient func(ctx context.Context) (ai.Client, error)
}

// Result is the outcome of a review run.
type Result struct {
	// Analysis is the analyzed (and, when a filter was active, filtered) plan.
	Analysis *plan.PlanAnalysis
	// ExitCode is the process exit code the analysis calls for: 0 no changes,
	// 1 non-critical changes, 2 critical changes.
	ExitCode int
}

// Run executes the review flow and writes the report (and optional AI
// narrative) to out. Log lines go to log; report content never does.
func Run(ctx context.Context, req Request, out io.Writer, log *logger.Logger) (*Result, error) {
	planData, err := plan.ParsePlanFile(req.PlanFile)
	if err != nil {
		log.Errorf("Failed to parse plan file: %v", err)
		return nil, fmt.Errorf("failed to parse plan file: %w", err)
	}

	analyzer := plan.NewPlanAnalyzerWithTypes(req.CriticalResources)
	analysis := analyzer.Analyze(planData)

	if !analysis.HasChanges {
		log.Success("No changes detected in plan - infrastructure is up to date")
		return &Result{Analysis: analysis, ExitCode: plan.ExitCodeSuccess}, nil
	}
	log.Infof("Found %d resource changes", analysis.TotalChanges)

	totalResourceCount := len(analysis.ResourceChanges)
	if !req.Filter.IsEmpty() {
		analysis.ResourceChanges = plan.FilterResources(analysis.ResourceChanges, req.Filter)
		log.Infof("Filtered to %d of %d resources", len(analysis.ResourceChanges), totalResourceCount)
	}

	var formatter *plan.PlanFormatter
	if !req.Filter.IsEmpty() {
		formatter = plan.NewPlanFormatterFiltered(req.UseColor, req.TopResourcesCount, totalResourceCount, req.Filter.Descriptions())
	} else {
		formatter = plan.NewPlanFormatterWithConfig(req.UseColor, req.TopResourcesCount)
	}
	if err := formatter.Format(analysis, req.Format, out); err != nil {
		log.Errorf("Failed to format output: %v", err)
		return nil, fmt.Errorf("failed to format output: %w", err)
	}

	if req.NewAIClient != nil {
		appendAIAnalysis(ctx, req, analysis, out, log)
	}

	return &Result{Analysis: analysis, ExitCode: analysis.ExitCode()}, nil
}

// appendAIAnalysis streams the AI narrative below the report. All failure
// modes are non-fatal: a missing API key, a network blip, or an invalid model
// name produces a logger warning and an early return — the exit code stays
// whatever the structured analysis decided.
func appendAIAnalysis(ctx context.Context, req Request, analysis *plan.PlanAnalysis, out io.Writer, log *logger.Logger) {
	client, err := req.NewAIClient(ctx)
	if err != nil {
		log.Warnf("--ai requested but skipping AI analysis: %v", err)
		return
	}

	payload := ai.BuildPayload(analysis, req.CriticalResources)

	header := fmt.Sprintf("\n## AI Analysis\n\n- **Provider:** %s\n- **Model:** %s\n\n", client.Provider(), client.Model())
	if _, err := fmt.Fprint(out, header); err != nil {
		log.Warnf("AI analysis: failed to write section header: %v", err)
		return
	}
	if err := client.Explain(ctx, payload, out); err != nil {
		if errors.Is(err, ai.ErrResponseTruncated) {
			// Truncated output is still partially useful — finish the markdown
			// cleanly and warn the user how to get the full response next time.
			if _, werr := fmt.Fprint(out, "\n\n> **Output truncated** at the model's max-tokens ceiling. Re-run with `--ai-max-tokens <higher value>` (or set `ai.max_tokens` in your config) to see the full analysis.\n"); werr != nil {
				log.Warnf("AI analysis: failed to write truncation notice: %v", werr)
			}
			log.Warnf("AI analysis was truncated (%v) — raise --ai-max-tokens to get the full response", err)
			return
		}
		log.Warnf("AI analysis failed: %v", err)
		return
	}
	if _, err := fmt.Fprintln(out); err != nil {
		log.Warnf("AI analysis: failed to write trailing newline: %v", err)
	}
}
