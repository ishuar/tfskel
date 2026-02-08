package drift

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// PlanFormatter handles formatting of plan analysis results
type PlanFormatter struct {
	useColor bool
}

// NewPlanFormatter creates a new plan formatter
func NewPlanFormatter(useColor bool) *PlanFormatter {
	return &PlanFormatter{useColor: useColor}
}

// Format outputs the plan analysis in the specified format
func (f *PlanFormatter) Format(analysis *PlanAnalysis, format OutputFormat, w io.Writer) error {
	switch format {
	case FormatJSON:
		return f.formatJSON(analysis, w)
	case FormatCSV:
		return f.formatCSV(analysis, w)
	case FormatTable:
		return f.formatTable(analysis, w)
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}
}

// formatJSON outputs analysis as JSON
func (f *PlanFormatter) formatJSON(analysis *PlanAnalysis, w io.Writer) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(analysis)
}

// formatCSV outputs analysis as CSV
func (f *PlanFormatter) formatCSV(analysis *PlanAnalysis, w io.Writer) error {
	_, _ = fmt.Fprintln(w, "Address,Type,Name,Provider,Action,Severity")
	for _, rc := range analysis.ResourceChanges {
		_, _ = fmt.Fprintf(w, "%s,%s,%s,%s,%s,%s\n",
			rc.Address, rc.Type, rc.Name, rc.Provider, rc.ActionString, rc.Severity)
	}
	return nil
}

// formatTable outputs analysis as a formatted table
func (f *PlanFormatter) formatTable(analysis *PlanAnalysis, w io.Writer) error {
	// Header
	_, _ = fmt.Fprintln(w, "")
	_, _ = fmt.Fprintln(w, "=== Terraform Plan Analysis ===")
	_, _ = fmt.Fprintf(w, "Terraform Version: %s\n\n", analysis.TerraformVersion)

	// Summary stats
	_, _ = fmt.Fprintln(w, "Summary:")
	_, _ = fmt.Fprintf(w, "  Total Changes:   %d\n", analysis.TotalChanges)
	_, _ = fmt.Fprintf(w, "  Additions:       %d\n", analysis.Additions)
	_, _ = fmt.Fprintf(w, "  Modifications:   %d\n", analysis.Modifications)
	_, _ = fmt.Fprintf(w, "  Deletions:       %d\n", analysis.Deletions)
	_, _ = fmt.Fprintf(w, "  Replacements:    %d\n\n", analysis.Replacements)

	if len(analysis.ResourceChanges) == 0 {
		return nil
	}

	// Resource changes table
	_, _ = fmt.Fprintln(w, "Resource Changes:")
	_, _ = fmt.Fprintln(w, strings.Repeat("─", 72))
	_, _ = fmt.Fprintf(w, "%-50s %-15s %-10s\n", "ADDRESS", "ACTION", "SEVERITY")
	_, _ = fmt.Fprintln(w, strings.Repeat("─", 72))

	for _, rc := range analysis.ResourceChanges {
		action := rc.ActionString
		severity := string(rc.Severity)

		// Add color if enabled
		if f.useColor {
			action = f.colorizeAction(action)
			severity = f.colorizeSeverity(severity)
		}

		_, _ = fmt.Fprintf(w, "%-50s %-24s %-18s\n",
			f.truncate(rc.Address, 50), action, severity)
	}
	_, _ = fmt.Fprintln(w, strings.Repeat("─", 72))
	_, _ = fmt.Fprintln(w, "")

	return nil
}

// colorizeAction adds ANSI color codes to action strings
func (f *PlanFormatter) colorizeAction(action string) string {
	switch action {
	case "create":
		return "\033[32m" + action + "\033[0m" // green
	case "delete":
		return "\033[31m" + action + "\033[0m" // red
	case "replace":
		return "\033[33m" + action + "\033[0m" // yellow
	case "update":
		return "\033[36m" + action + "\033[0m" // cyan
	default:
		return action
	}
}

// colorizeSeverity adds ANSI color codes to severity strings
func (f *PlanFormatter) colorizeSeverity(severity string) string {
	switch severity {
	case "critical":
		return "\033[1;31m" + severity + "\033[0m" // bold red
	case "high":
		return "\033[31m" + severity + "\033[0m" // red
	case "medium":
		return "\033[33m" + severity + "\033[0m" // yellow
	case "low":
		return "\033[32m" + severity + "\033[0m" // green
	default:
		return severity
	}
}

// truncate shortens a string to maxLen, adding ellipsis if needed
func (f *PlanFormatter) truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
