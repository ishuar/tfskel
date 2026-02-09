package drift

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"golang.org/x/term"
)

var (
	// ErrUnsupportedPlanFormat indicates an unsupported output format
	ErrUnsupportedPlanFormat = errors.New("unsupported format")
)

// PlanFormatter handles formatting of plan analysis results
type PlanFormatter struct {
	useColor      bool
	terminalWidth int
	tableWidth    int // Consistent width for all tables
}

// NewPlanFormatter creates a new plan formatter with auto-detected terminal width
func NewPlanFormatter(useColor bool) *PlanFormatter {
	width := defaultTerminalWidth
	if fd := int(os.Stdout.Fd()); term.IsTerminal(fd) {
		if w, _, err := term.GetSize(fd); err == nil && w > 0 {
			width = w
		}
	}
	return &PlanFormatter{
		useColor:      useColor,
		terminalWidth: width,
		tableWidth:    0, // Will be calculated during formatting
	}
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
		return fmt.Errorf("%w: %s", ErrUnsupportedPlanFormat, format)
	}
}

// formatJSON outputs analysis as JSON
func (f *PlanFormatter) formatJSON(analysis *PlanAnalysis, w io.Writer) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(analysis)
}

// formatCSV outputs analysis as CSV with metadata and resource details using proper CSV escaping
func (f *PlanFormatter) formatCSV(analysis *PlanAnalysis, w io.Writer) error {
	csvWriter := csv.NewWriter(w)
	defer csvWriter.Flush()

	// Write metadata as comments (not CSV escaped)
	comments := []string{
		"# Terraform Plan Analysis",
		fmt.Sprintf("# Terraform Version: %s", analysis.TerraformVersion),
		fmt.Sprintf("# Total Changes: %d", analysis.TotalChanges),
		fmt.Sprintf("# Additions: %d, Modifications: %d, Deletions: %d, Replacements: %d",
			analysis.Additions, analysis.Modifications, analysis.Deletions, analysis.Replacements),
		"#",
	}
	for _, comment := range comments {
		if _, err := fmt.Fprintln(w, comment); err != nil {
			return fmt.Errorf("failed to write comment: %w", err)
		}
	}

	// Write header using CSV writer for consistency
	if err := csvWriter.Write([]string{"Address", "Type", "Name", "Provider", "Action", "Severity"}); err != nil {
		return fmt.Errorf("failed to write CSV header: %w", err)
	}

	// Write resource data with proper CSV escaping
	for _, rc := range analysis.ResourceChanges {
		if err := csvWriter.Write([]string{
			rc.Address,
			rc.Type,
			rc.Name,
			rc.Provider,
			rc.ActionString,
			string(rc.Severity), // Convert Severity type to string
		}); err != nil {
			return fmt.Errorf("failed to write CSV record: %w", err)
		}
	}

	return csvWriter.Error()
}

// formatTable outputs analysis as a formatted table with color styling
func (f *PlanFormatter) formatTable(analysis *PlanAnalysis, w io.Writer) error {
	styles := NewCommonStyles(f.useColor)

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

	if len(analysis.ResourceChanges) == 0 {
		return nil
	}

	// Write grouping sections
	if err := f.writeTableGroupings(w, analysis, styles); err != nil {
		return err
	}

	// Write detailed resource changes
	return f.writeTableResourceDetails(w, analysis, styles)
}

// writeTableHeader writes the table header section
func (f *PlanFormatter) writeTableHeader(w io.Writer, analysis *PlanAnalysis, styles CommonStyles) error {
	if _, err := fmt.Fprintln(w, styles.TitleStyle.Render("━━━ Terraform Plan Analysis ━━━")); err != nil {
		return fmt.Errorf("failed to write title: %w", err)
	}
	if _, err := fmt.Fprintf(w, "%s %s\n", styles.MutedStyle.Render("Terraform Version:"), analysis.TerraformVersion); err != nil {
		return fmt.Errorf("failed to write version: %w", err)
	}
	return nil
}

// writeTableSummary writes the summary statistics table
func (f *PlanFormatter) writeTableSummary(w io.Writer, analysis *PlanAnalysis, styles CommonStyles) error {
	if _, err := fmt.Fprintln(w, styles.HeaderStyle.Render("Summary")); err != nil {
		return fmt.Errorf("failed to write summary header: %w", err)
	}

	summaryData := [][]string{
		{"Total Changes", fmt.Sprintf("%d", analysis.TotalChanges)},
		{"Additions", fmt.Sprintf("%d", analysis.Additions)},
		{"Modifications", fmt.Sprintf("%d", analysis.Modifications)},
		{"Deletions", fmt.Sprintf("%d", analysis.Deletions)},
		{"Replacements", fmt.Sprintf("%d", analysis.Replacements)},
	}

	summaryTable := table.New().
		Border(lipgloss.RoundedBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(styles.BorderColor)).
		Width(f.tableWidth).
		StyleFunc(func(row, col int) lipgloss.Style {
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
func (f *PlanFormatter) writeTableGroupings(w io.Writer, analysis *PlanAnalysis, styles CommonStyles) error {
	// Changes by Resource Type
	if len(analysis.ByType) > 0 {
		if err := f.printGroupSummary(w, styles, "Changes by Resource Type", analysis.ByType, defaultTopNCount); err != nil {
			return err
		}
	}

	// Changes by Module
	if len(analysis.ByModule) > 1 { // Only show if more than root module
		if err := f.printGroupSummary(w, styles, "Changes by Module", analysis.ByModule, defaultTopNCount); err != nil {
			return err
		}
	}

	// Changes by Severity
	if len(analysis.BySeverity) > 0 {
		if err := f.printGroupSummary(w, styles, "Changes by Severity", analysis.BySeverity, severityTopNCount); err != nil {
			return err
		}
	}

	return nil
}

// writeTableResourceDetails writes the detailed resource changes table
func (f *PlanFormatter) writeTableResourceDetails(w io.Writer, analysis *PlanAnalysis, styles CommonStyles) error {
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

// colorizeAction adds color styling to action strings
func (f *PlanFormatter) colorizeAction(action string) string {
	if !f.useColor {
		return action
	}

	var color lipgloss.Color
	switch action {
	case "create":
		color = lipgloss.Color("2") // green
	case "delete":
		color = lipgloss.Color("1") // red
	case "replace":
		color = lipgloss.Color("3") // yellow
	case "update":
		color = lipgloss.Color("6") // cyan
	case "read":
		color = lipgloss.Color("4") // blue
	default:
		return action
	}

	return lipgloss.NewStyle().Foreground(color).Render(action)
}

// colorizeSeverity adds color styling to severity strings
func (f *PlanFormatter) colorizeSeverity(severity string) string {
	if !f.useColor {
		return severity
	}

	var style lipgloss.Style
	switch severity {
	case "critical":
		style = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("1")) // bold red
	case "high":
		style = lipgloss.NewStyle().Foreground(lipgloss.Color("1")) // red
	case "medium":
		style = lipgloss.NewStyle().Foreground(lipgloss.Color("3")) // yellow
	case "low":
		style = lipgloss.NewStyle().Foreground(lipgloss.Color("2")) // green
	default:
		return severity
	}

	return style.Render(severity)
}

// buildResourceData constructs resource table rows.
// Columns are auto-sized based on content without truncation.
func (f *PlanFormatter) buildResourceData(resources []AnalyzedResource) [][]string {
	data := make([][]string, 0, len(resources))

	for _, rc := range resources {
		// Show resource name with module path if present
		resourceName := rc.Name
		if rc.ModuleAddress != "" {
			// Show module path compactly
			modulePath := f.extractModuleName(rc.ModuleAddress)
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
	minWidth := minPlanTableWidth
	maxWidth := maxPlanTableWidth

	// Use 95% of terminal width to leave some margin
	optimalWidth := (f.terminalWidth * percentageWidthFactor) / percentageDivisor

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
func (f *PlanFormatter) printGroupSummary(w io.Writer, styles CommonStyles, title string, groups map[string]int, topN int) error {
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
		data[i] = []string{gc.name, fmt.Sprintf("%d", gc.count)}
	}

	groupTable := table.New().
		Border(lipgloss.RoundedBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(styles.BorderColor)).
		Width(f.tableWidth).
		StyleFunc(func(row, col int) lipgloss.Style {
			if row == -1 {
				// Headers: center-aligned
				return lipgloss.NewStyle().Bold(true).Foreground(styles.HeaderColor).Align(lipgloss.Center)
			}
			if col == 0 {
				// First column: left-aligned
				return lipgloss.NewStyle().Foreground(styles.RowColor).Align(lipgloss.Left)
			}
			// Second column: center-aligned
			return lipgloss.NewStyle().Foreground(styles.RowColor).Align(lipgloss.Center)
		}).
		Headers("Name", "Count").
		Rows(data...)

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
func (f *PlanFormatter) extractModuleName(moduleAddr string) string {
	// module.flowmotion -> flowmotion
	// module.vpc.module.subnets -> vpc.subnets
	parts := []string{}
	for _, part := range splitModuleAddress(moduleAddr) {
		if part != "module" {
			parts = append(parts, part)
		}
	}
	if len(parts) == 0 {
		return "root"
	}
	if len(parts) > 2 {
		// Show first and last for long module paths
		return parts[0] + "..." + parts[len(parts)-1]
	}
	result := ""
	for i, p := range parts {
		if i > 0 {
			result += "."
		}
		result += p
	}
	return result
}

// splitModuleAddress splits a module address string by dots into its component parts.
// This is a helper function for parsing module hierarchies.
// Example: "module.vpc.module.subnets" -> ["module", "vpc", "module", "subnets"]
func splitModuleAddress(addr string) []string {
	parts := []string{}
	current := ""
	for _, ch := range addr {
		if ch == '.' {
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		} else {
			current += string(ch)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}
	return parts
}
