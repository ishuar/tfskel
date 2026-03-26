package plan

import (
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/ishuar/tfskel/internal/format"
)

// formatTable outputs analysis as a formatted table with color styling
func (f *PlanFormatter) formatTable(analysis *PlanAnalysis, w io.Writer) error {
	styles := format.NewCommonStyles(f.useColor)

	// Calculate optimal width for all tables
	f.tableWidth = f.calculateOptimalWidth()

	// Write header section
	if err := f.writeTableHeader(w, analysis, styles); err != nil {
		return err
	}

	// Write summary section
	if err := f.writeTableSummary(w, analysis, styles); err != nil {
		return err
	}

	if len(analysis.ResourceChanges) == 0 && len(analysis.OutputChanges) == 0 {
		return nil
	}

	if len(analysis.ResourceChanges) > 0 {
		// Write grouping sections
		if err := f.writeTableGroupings(w, analysis, styles); err != nil {
			return err
		}

		// Write detailed resource changes
		if err := f.writeTableResourceDetails(w, analysis, styles); err != nil {
			return err
		}
	}

	// Write output changes section at the bottom (if any)
	if len(analysis.OutputChanges) > 0 {
		if err := f.writeTableOutputChanges(w, analysis, styles); err != nil {
			return err
		}
	}

	return nil
}

// writeTableHeader writes the table header section
func (f *PlanFormatter) writeTableHeader(w io.Writer, analysis *PlanAnalysis, styles format.CommonStyles) error {
	if _, err := fmt.Fprintln(w, styles.TitleStyle.Render("━━━ Terraform Plan Analysis ━━━")); err != nil {
		return fmt.Errorf("failed to write title: %w", err)
	}
	if _, err := fmt.Fprintf(w, "%s %s\n", styles.MutedStyle.Render("Terraform Version:"), analysis.TerraformVersion); err != nil {
		return fmt.Errorf("failed to write version: %w", err)
	}
	return nil
}

// writeTableSummary writes the summary statistics table
func (f *PlanFormatter) writeTableSummary(w io.Writer, analysis *PlanAnalysis, styles format.CommonStyles) error {
	if _, err := fmt.Fprintln(w, styles.HeaderStyle.Render("Summary")); err != nil {
		return fmt.Errorf("failed to write summary header: %w", err)
	}

	summaryData := [][]string{
		{"Total Resource Changes", strconv.Itoa(analysis.TotalChanges)},
		{"Additions", strconv.Itoa(analysis.Additions)},
		{"Modifications", strconv.Itoa(analysis.Modifications)},
		{"Deletions", strconv.Itoa(analysis.Deletions)},
		{"Replacements", strconv.Itoa(analysis.Replacements)},
	}

	summaryTable := table.New().
		Border(lipgloss.RoundedBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(styles.BorderColor)).
		Width(f.tableWidth).
		StyleFunc(func(_, col int) lipgloss.Style {
			if col == 0 {
				// First column: right-aligned labels
				return lipgloss.NewStyle().Bold(true).Foreground(styles.RowColor).Align(lipgloss.Right)
			}
			// Second column: center-aligned values
			return lipgloss.NewStyle().Foreground(styles.RowColor).Align(lipgloss.Center)
		}).
		Rows(summaryData...)

	if _, err := fmt.Fprintln(w, summaryTable.Render()); err != nil {
		return fmt.Errorf("failed to write summary table: %w", err)
	}

	return nil
}

// writeTableGroupings writes all grouping sections
func (f *PlanFormatter) writeTableGroupings(w io.Writer, analysis *PlanAnalysis, styles format.CommonStyles) error {
	// Changes by Resource Type
	if len(analysis.ByType) > 0 {
		if err := f.printGroupSummary(w, styles, "Changes by Resource Type", analysis.ByType, f.topResourcesCount); err != nil {
			return err
		}
	}

	// Changes by Module
	if len(analysis.ByModule) > 1 { // Only show if more than root module
		if err := f.printGroupSummary(w, styles, "Changes by Module", analysis.ByModule, f.topResourcesCount); err != nil {
			return err
		}
	}

	// Changes by Severity
	if len(analysis.BySeverity) > 0 {
		if err := f.printGroupSummary(w, styles, "Changes by Severity", analysis.BySeverity, format.SeverityTopResourcesCount); err != nil {
			return err
		}
	}

	return nil
}

// writeTableResourceDetails writes the detailed resource changes table
func (f *PlanFormatter) writeTableResourceDetails(w io.Writer, analysis *PlanAnalysis, styles format.CommonStyles) error {
	if _, err := fmt.Fprintln(w, styles.HeaderStyle.Render("Resource Changes (detailed)")); err != nil {
		return fmt.Errorf("failed to write resource changes header: %w", err)
	}
	if _, err := fmt.Fprintln(w, styles.MutedStyle.Render(fmt.Sprintf("Showing %d resources", len(analysis.ResourceChanges)))); err != nil {
		return fmt.Errorf("failed to write resource count: %w", err)
	}

	resourceData := f.buildResourceData(analysis.ResourceChanges)

	resourceTable := table.New().
		Border(lipgloss.RoundedBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(styles.BorderColor)).
		Width(f.tableWidth).
		StyleFunc(func(row, col int) lipgloss.Style {
			if row == -1 {
				// Headers: center-aligned
				return lipgloss.NewStyle().Bold(true).Foreground(styles.HeaderColor).Align(lipgloss.Center)
			}
			// Data rows - left align for resource/type, center for action/severity
			if col == 0 || col == 1 {
				return lipgloss.NewStyle().Foreground(styles.RowColor).Align(lipgloss.Left)
			}
			return lipgloss.NewStyle().Foreground(styles.RowColor).Align(lipgloss.Center)
		}).
		Headers("Resource", "Type", "Action", "Severity").
		Rows(resourceData...)

	if _, err := fmt.Fprintln(w, resourceTable.Render()); err != nil {
		return fmt.Errorf("failed to write resource table: %w", err)
	}
	if _, err := fmt.Fprintln(w, ""); err != nil {
		return fmt.Errorf("failed to write trailing newline: %w", err)
	}

	return nil
}

// writeTableOutputChanges writes the output changes section to the table
func (f *PlanFormatter) writeTableOutputChanges(w io.Writer, analysis *PlanAnalysis, styles format.CommonStyles) error {
	header := fmt.Sprintf("Output Changes (%d total: %d added, %d deleted, %d modified, %d replaced)",
		len(analysis.OutputChanges), analysis.OutputAdditions, analysis.OutputDeletions, analysis.OutputModifications, analysis.OutputReplacements)
	if _, err := fmt.Fprintln(w, styles.HeaderStyle.Render(header)); err != nil {
		return fmt.Errorf("failed to write output changes header: %w", err)
	}

	data := make([][]string, 0, len(analysis.OutputChanges))
	for _, oc := range analysis.OutputChanges {
		action := formatActions(oc.Actions)
		name := oc.Name
		if oc.Sensitive {
			name += " (sensitive)"
		}

		if f.useColor {
			action = f.colorizeAction(action)
		}

		data = append(data, []string{name, action})
	}

	outputTable := f.newTwoColumnTable(styles, "Output", "Action", data)

	if _, err := fmt.Fprintln(w, outputTable.Render()); err != nil {
		return fmt.Errorf("failed to write output changes table: %w", err)
	}
	return nil
}

// newTwoColumnTable creates a styled two-column table with left-aligned first column and center-aligned second column.
func (f *PlanFormatter) newTwoColumnTable(styles format.CommonStyles, header1, header2 string, data [][]string) *table.Table {
	return table.New().
		Border(lipgloss.RoundedBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(styles.BorderColor)).
		Width(f.tableWidth).
		StyleFunc(func(row, col int) lipgloss.Style {
			if row == -1 {
				return lipgloss.NewStyle().Bold(true).Foreground(styles.HeaderColor).Align(lipgloss.Center)
			}
			if col == 0 {
				return lipgloss.NewStyle().Foreground(styles.RowColor).Align(lipgloss.Left)
			}
			return lipgloss.NewStyle().Foreground(styles.RowColor).Align(lipgloss.Center)
		}).
		Headers(header1, header2).
		Rows(data...)
}

// buildResourceData constructs resource table rows.
// Columns are auto-sized based on content without truncation.
// Resources are sorted by severity: critical, high, medium, low.
func (f *PlanFormatter) buildResourceData(resources []AnalyzedResource) [][]string {
	// Sort resources by severity using shared function
	sortedResources := sortResourcesBySeverity(resources)

	data := make([][]string, 0, len(sortedResources))

	for _, rc := range sortedResources {
		// Show resource name with module path if present
		resourceName := rc.Name
		if rc.ModuleAddress != "" {
			// Show module path compactly
			modulePath := extractModuleName(rc.ModuleAddress)
			resourceName = fmt.Sprintf("%s.%s", modulePath, rc.Name)
		}

		resourceType := rc.Type
		action := rc.ActionString
		severity := string(rc.Severity)

		// Apply color styling based on action and severity
		if f.useColor {
			action = f.colorizeAction(action)
			severity = f.colorizeSeverity(severity)
		}

		data = append(data, []string{resourceName, resourceType, action, severity})
	}

	return data
}

// calculateOptimalWidth determines the best width for all tables based on terminal size.
// Returns a width between 80-150 characters, or 95% of terminal width.
func (f *PlanFormatter) calculateOptimalWidth() int {
	// For plan analysis, we want tables to use most of the terminal width
	// but with some reasonable constraints
	minWidth := format.MinPlanTableWidth
	maxWidth := format.MaxPlanTableWidth

	// Use 95% of terminal width to leave some margin
	optimalWidth := (f.terminalWidth * format.PercentageWidthFactor) / format.PercentageDivisor

	if optimalWidth < minWidth {
		return minWidth
	}
	if optimalWidth > maxWidth {
		return maxWidth
	}
	return optimalWidth
}

// printGroupSummary prints a grouped summary table showing aggregated statistics.
// The groups map contains category names and their counts.
// If topN is > 0, only the top N items by count are displayed.
// Returns an error if writing to the output fails.
func (f *PlanFormatter) printGroupSummary(w io.Writer, styles format.CommonStyles, title string, groups map[string]int, topN int) error {
	if _, err := fmt.Fprintln(w, styles.HeaderStyle.Render(title)); err != nil {
		return fmt.Errorf("failed to write group summary title: %w", err)
	}

	// Sort groups by count (descending)
	type groupCount struct {
		name  string
		count int
	}

	sorted := make([]groupCount, 0, len(groups))
	for name, count := range groups {
		sorted = append(sorted, groupCount{name, count})
	}

	// Sort by count descending using standard library
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].count > sorted[j].count
	})

	// Limit to topN if specified
	if topN > 0 && len(sorted) > topN {
		sorted = sorted[:topN]
	}

	// Build table data
	data := make([][]string, len(sorted))
	for i, gc := range sorted {
		data[i] = []string{gc.name, strconv.Itoa(gc.count)}
	}

	groupTable := f.newTwoColumnTable(styles, "Name", "Count", data)

	if _, err := fmt.Fprintln(w, groupTable.Render()); err != nil {
		return fmt.Errorf("failed to write group table: %w", err)
	}
	return nil
}

// extractModuleName extracts a compact, human-readable module name from a full module address.
// It removes "module." prefixes and compacts long paths for display.
// Examples:
//   - "module.flowmotion" -> "flowmotion"
//   - "module.vpc.module.subnets" -> "vpc.subnets"
//   - "module.a.module.b.module.c" -> "a...c" (for long paths)
func extractModuleName(moduleAddr string) string {
	var parts []string
	for part := range strings.SplitSeq(moduleAddr, ".") {
		if part != "module" {
			parts = append(parts, part)
		}
	}
	if len(parts) == 0 {
		return "root"
	}
	if len(parts) > 2 {
		return parts[0] + "..." + parts[len(parts)-1]
	}
	return strings.Join(parts, ".")
}
