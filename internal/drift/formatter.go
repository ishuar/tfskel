package drift

import (
	"bytes"
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
	ErrUnsupportedFormat = errors.New("unsupported format")
)

// OutputFormat defines the output format type
type OutputFormat string

const (
	FormatTable OutputFormat = "table"
	FormatJSON  OutputFormat = "json"
	FormatCSV   OutputFormat = "csv"

	// Minimum width calculation constants
	minDriftTableWidth = 113 // File(40) + Type(16) + Expected(16) + Actual(16) + Status(15) + borders(10)

	// File path width thresholds for extra space calculation
	baseFilePathWidth     = 40 // Base file path width before adding extra space
	maxExtraSpaceForPaths = 30 // Maximum extra space to add for long paths

	// Table layout constants
	tableBorderPadding = 4 // Approximate characters needed for borders and padding
	pathDivisor        = 2 // Divisor for calculating extra space from path length
)

// Formatter handles different output formats for drift reports
type Formatter struct {
	useColor      bool
	terminalWidth int
	tableWidth    int // Consistent width for all tables
}

// NewFormatter creates a new formatter
func NewFormatter(useColor bool) *Formatter {
	width := 120 // default width
	if fd := int(os.Stdout.Fd()); term.IsTerminal(fd) {
		if w, _, err := term.GetSize(fd); err == nil && w > 0 {
			width = w
		}
	}
	return &Formatter{
		useColor:      useColor,
		terminalWidth: width,
		tableWidth:    0, // Will be calculated during formatting
	}
}

// Format formats the drift report in the specified format
func (f *Formatter) Format(report *DriftReport, format OutputFormat, writer io.Writer) error {
	switch format {
	case FormatTable:
		return f.formatTable(report, writer)
	case FormatJSON:
		return f.formatJSON(report, writer)
	case FormatCSV:
		return f.formatCSV(report, writer)
	default:
		return fmt.Errorf("%w: %s", ErrUnsupportedFormat, format)
	}
}

// calculateOptimalWidth determines the best width for all tables
func (f *Formatter) calculateOptimalWidth(report *DriftReport) int {
	// Calculate minimum width needed for drift table (the widest)
	minRequired := minDriftTableWidth

	// Check if we have long file paths that need more space
	for _, record := range report.Records {
		if record.HasDrift && len(record.FilePath) > baseFilePathWidth {
			// Add extra space for longer paths, up to a reasonable limit
			extraSpace := (len(record.FilePath) - baseFilePathWidth) / pathDivisor
			if extraSpace > maxExtraSpaceForPaths {
				extraSpace = maxExtraSpaceForPaths // Cap extra space
			}
			minRequired += extraSpace
			break
		}
	}

	// Use the smaller of terminal width or required width
	if minRequired < f.terminalWidth {
		return minRequired
	}
	return f.terminalWidth
}

// formatTable outputs a human-readable table
func (f *Formatter) formatTable(report *DriftReport, writer io.Writer) error {
	buf := &bytes.Buffer{}
	styles := f.setupStyles()

	// Calculate consistent width for all tables
	f.tableWidth = f.calculateOptimalWidth(report)

	// Write header
	if err := f.writeHeader(buf, report, styles); err != nil {
		return err
	}

	// Write summary section
	if err := f.writeSummary(buf, report, styles); err != nil {
		return err
	}

	// Write version distributions
	if err := f.writeTerraformVersions(buf, report, styles); err != nil {
		return err
	}
	if err := f.writeProviderVersions(buf, report, styles); err != nil {
		return err
	}

	// Write drift details
	if err := f.writeDriftDetails(buf, report, styles); err != nil {
		return err
	}

	// Final summary message
	fmt.Fprintf(buf, "\n%s\n\n", report.GetDriftSummaryText())

	// Single atomic write to output
	_, err := writer.Write(buf.Bytes())
	return err
}

// tableStyles holds all lipgloss styles for table formatting
type tableStyles struct {
	titleStyle  lipgloss.Style
	headerStyle lipgloss.Style
	mutedStyle  lipgloss.Style
	borderColor lipgloss.Color
	headerColor lipgloss.Color
	rowColor    lipgloss.Color
}

// setupStyles initializes all formatting styles
func (f *Formatter) setupStyles() tableStyles {
	borderColor := lipgloss.Color("14")
	headerColor := lipgloss.Color("14")
	rowColor := lipgloss.Color("252")

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(headerColor).MarginBottom(1)
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(headerColor).MarginTop(1)
	mutedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))

	if !f.useColor {
		titleStyle = titleStyle.Foreground(lipgloss.NoColor{})
		headerStyle = headerStyle.Foreground(lipgloss.NoColor{})
		mutedStyle = mutedStyle.Foreground(lipgloss.NoColor{})
		borderColor = lipgloss.Color("")
		headerColor = lipgloss.Color("")
		rowColor = lipgloss.Color("")
	}

	return tableStyles{
		titleStyle:  titleStyle,
		headerStyle: headerStyle,
		mutedStyle:  mutedStyle,
		borderColor: borderColor,
		headerColor: headerColor,
		rowColor:    rowColor,
	}
}

// writeHeader writes the report header
func (f *Formatter) writeHeader(writer io.Writer, report *DriftReport, styles tableStyles) error {
	if _, err := fmt.Fprintln(writer, styles.titleStyle.Render("━━━ Terraform Version Drift Report ━━━")); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(writer, "%s %s\n", styles.mutedStyle.Render("Scanned:"), report.ScanRoot); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(writer, "%s %s\n", styles.mutedStyle.Render("Time:"), report.ScannedAt.Format("2006-01-02 15:04:05")); err != nil {
		return err
	}
	return nil
}

// writeSummary writes the quick summary section
func (f *Formatter) writeSummary(writer io.Writer, report *DriftReport, styles tableStyles) error {
	if _, err := fmt.Fprintln(writer, styles.headerStyle.Render("Quick Summary")); err != nil {
		return err
	}

	summaryData := f.buildSummaryData(report)

	// Calculate column widths to fill the table width evenly
	// Subtract borders and padding
	availableWidth := f.tableWidth - tableBorderPadding
	labelWidth := availableWidth / 2
	valueWidth := availableWidth - labelWidth

	summaryTable := table.New().
		Border(lipgloss.RoundedBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(styles.borderColor)).
		Width(f.tableWidth).
		StyleFunc(func(row, col int) lipgloss.Style {
			if col == 0 {
				// First column: left-aligned labels
				return lipgloss.NewStyle().Bold(true).Foreground(styles.rowColor).Width(labelWidth).Align(lipgloss.Right)
			}
			// Second column: center-aligned values
			return lipgloss.NewStyle().Foreground(styles.rowColor).Width(valueWidth).Align(lipgloss.Center)
		}).
		Rows(summaryData...)

	if _, err := fmt.Fprintln(writer, summaryTable.Render()); err != nil {
		return err
	}
	return nil
}

// buildSummaryData constructs summary table rows
func (f *Formatter) buildSummaryData(report *DriftReport) [][]string {
	summaryData := [][]string{
		{"Total Files Scanned", fmt.Sprintf("%d", report.TotalFiles)},
		{"Files in Sync", fmt.Sprintf("%d", report.Summary.FilesInSync)},
	}

	if report.FilesWithDrift > 0 {
		summaryData = append(summaryData, []string{"Files with Drift", fmt.Sprintf("%d", report.FilesWithDrift)})
		if report.Summary.FilesWithMajorDrift > 0 {
			summaryData = append(summaryData, []string{"  ↳ Major Drift", fmt.Sprintf("%d", report.Summary.FilesWithMajorDrift)})
		}
		if report.Summary.FilesWithMinorDrift > 0 {
			summaryData = append(summaryData, []string{"  ↳ Minor Drift", fmt.Sprintf("%d", report.Summary.FilesWithMinorDrift)})
		}
	} else {
		summaryData = append(summaryData, []string{"Files with Drift", "0"})
	}

	if report.Summary.FilesWithErrors > 0 {
		summaryData = append(summaryData, []string{"Files with Errors", fmt.Sprintf("%d", report.Summary.FilesWithErrors)})
	}

	return summaryData
}

// writeTerraformVersions writes the terraform versions section
func (f *Formatter) writeTerraformVersions(writer io.Writer, report *DriftReport, styles tableStyles) error {
	if len(report.Summary.TerraformVersions) == 0 {
		return nil
	}

	if _, err := fmt.Fprintln(writer, styles.headerStyle.Render("Terraform Versions")); err != nil {
		return err
	}

	expectedVersion := ""
	if len(report.Records) > 0 {
		expectedVersion = report.Records[0].TerraformExpected
	}

	versionData := f.buildVersionData(report.Summary.TerraformVersions, expectedVersion)

	versionTable := table.New().
		Border(lipgloss.RoundedBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(styles.borderColor)).
		Width(f.tableWidth).
		StyleFunc(func(row, col int) lipgloss.Style {
			if row == -1 {
				// Headers: center-aligned
				return lipgloss.NewStyle().Bold(true).Foreground(styles.headerColor).Align(lipgloss.Center)
			}
			// All data rows: center-aligned
			return lipgloss.NewStyle().Foreground(styles.rowColor).Align(lipgloss.Center)
		}).
		Headers("Status", "Version", "Count").
		Rows(versionData...)

	if _, err := fmt.Fprintln(writer, versionTable.Render()); err != nil {
		return err
	}
	return nil
}

// buildVersionData constructs version table rows
func (f *Formatter) buildVersionData(versions map[string]int, expectedVersion string) [][]string {
	versionData := [][]string{}

	// Sort versions for consistent display
	sortedVersions := make([]string, 0, len(versions))
	for v := range versions {
		sortedVersions = append(sortedVersions, v)
	}
	sort.Strings(sortedVersions)

	for _, version := range sortedVersions {
		count := versions[version]
		status := "OK"
		if version != expectedVersion {
			status = "DRIFT"
		}
		versionData = append(versionData, []string{
			status,
			version,
			fmt.Sprintf("%d files", count),
		})
	}

	return versionData
}

// writeProviderVersions writes the provider versions section
func (f *Formatter) writeProviderVersions(writer io.Writer, report *DriftReport, styles tableStyles) error {
	if len(report.Summary.ProviderVersions) == 0 {
		return nil
	}

	if _, err := fmt.Fprintln(writer, styles.headerStyle.Render("Provider Versions")); err != nil {
		return err
	}

	// Sort providers
	providers := make([]string, 0, len(report.Summary.ProviderVersions))
	for p := range report.Summary.ProviderVersions {
		providers = append(providers, p)
	}
	sort.Strings(providers)

	// Build and render tables for each provider
	for _, provider := range providers {
		providerTable := f.buildProviderTable(provider, report.Summary.ProviderVersions[provider], styles)
		if _, err := fmt.Fprintln(writer, providerTable); err != nil {
			return err
		}
	}
	return nil
}

// buildProviderTable constructs a table for a single provider's versions
func (f *Formatter) buildProviderTable(provider string, versions map[string]int, styles tableStyles) string {
	providerData := [][]string{}

	// Sort versions
	versionList := make([]string, 0, len(versions))
	for v := range versions {
		versionList = append(versionList, v)
	}
	sort.Strings(versionList)

	for _, version := range versionList {
		count := versions[version]
		providerData = append(providerData, []string{version, fmt.Sprintf("%d files", count)})
	}

	return table.New().
		Border(lipgloss.RoundedBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(styles.borderColor)).
		Width(f.tableWidth).
		StyleFunc(func(row, col int) lipgloss.Style {
			if row == -1 {
				// Headers: center-aligned
				return lipgloss.NewStyle().Bold(true).Foreground(styles.headerColor).Align(lipgloss.Center)
			}
			// All data rows: center-aligned
			return lipgloss.NewStyle().Foreground(styles.rowColor).Align(lipgloss.Center)
		}).
		Headers(provider, "Count").
		Rows(providerData...).
		Render()
}

// writeDriftDetails writes detailed drift information
func (f *Formatter) writeDriftDetails(writer io.Writer, report *DriftReport, styles tableStyles) error {
	driftRecords := f.filterDriftRecords(report.Records)
	if len(driftRecords) == 0 {
		return nil
	}

	totalDriftItems := f.countDriftItems(driftRecords)
	if _, err := fmt.Fprintln(writer, styles.headerStyle.Render(fmt.Sprintf("Files with Drift (%d files, %d issues)", len(driftRecords), totalDriftItems))); err != nil {
		return err
	}

	// Sort by file path
	sort.Slice(driftRecords, func(i, j int) bool {
		return driftRecords[i].FilePath < driftRecords[j].FilePath
	})

	driftData := f.buildDriftData(driftRecords, styles)

	driftTable := table.New().
		Border(lipgloss.RoundedBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(styles.borderColor)).
		StyleFunc(func(row, col int) lipgloss.Style {
			if row == -1 {
				// Headers: center-aligned
				return lipgloss.NewStyle().Bold(true).Foreground(styles.headerColor).Align(lipgloss.Center)
			}
			// All other columns: center-aligned
			return lipgloss.NewStyle().Foreground(styles.rowColor).Align(lipgloss.Left)
		}).
		Width(f.tableWidth).
		Headers("File", "Type", "Expected", "Actual", "Status").
		Rows(driftData...)

	if _, err := fmt.Fprintln(writer, driftTable.Render()); err != nil {
		return err
	}
	return nil
}

// filterDriftRecords extracts only records with drift
func (f *Formatter) filterDriftRecords(records []DriftRecord) []DriftRecord {
	driftRecords := []DriftRecord{}
	for _, record := range records {
		if record.HasDrift {
			driftRecords = append(driftRecords, record)
		}
	}
	return driftRecords
}

// countDriftItems counts total drift issues across records
func (f *Formatter) countDriftItems(records []DriftRecord) int {
	totalDriftItems := 0
	for _, record := range records {
		if record.TerraformDriftStatus != StatusInSync {
			totalDriftItems++
		}
		for _, pd := range record.Providers {
			if pd.DriftStatus != StatusInSync && pd.DriftStatus != StatusNotManaged {
				totalDriftItems++
			}
		}
	}
	return totalDriftItems
}

// buildDriftData constructs drift detail table rows
func (f *Formatter) buildDriftData(records []DriftRecord, styles tableStyles) [][]string {
	driftData := [][]string{}
	maxPathLen := 100

	truncatePath := func(path string) string {
		if len(path) > maxPathLen {
			return "..." + path[len(path)-maxPathLen+3:]
		}
		return path
	}

	for _, record := range records {
		filePath := truncatePath(record.FilePath)

		// Terraform version drift
		if record.TerraformDriftStatus != StatusInSync {
			driftData = append(driftData, []string{
				filePath,
				"Terraform",
				record.TerraformExpected,
				record.TerraformActual,
				f.formatStatus(record.TerraformDriftStatus),
			})
		}

		// Provider drifts
		for _, pd := range record.Providers {
			if pd.DriftStatus != StatusInSync && pd.DriftStatus != StatusNotManaged {
				expected := pd.Expected
				if expected == "" {
					expected = styles.mutedStyle.Render("(not configured)")
				}

				displayPath := filePath
				if record.TerraformDriftStatus != StatusInSync {
					displayPath = styles.mutedStyle.Render("  ↳ " + filePath)
				}

				driftData = append(driftData, []string{
					displayPath,
					fmt.Sprintf("Provider: %s", pd.Name),
					expected,
					pd.Actual,
					f.formatStatus(pd.DriftStatus),
				})
			}
		}
	}

	return driftData
}

// formatJSON outputs JSON format
func (f *Formatter) formatJSON(report *DriftReport, writer io.Writer) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false) // Don't escape HTML entities like > to \u003e
	return encoder.Encode(report)
}

// formatCSV outputs CSV format
func (f *Formatter) formatCSV(report *DriftReport, writer io.Writer) error {
	csvWriter := csv.NewWriter(writer)
	defer csvWriter.Flush()

	// Header
	headers := []string{
		"File Path",
		"Component Type",
		"Component Name",
		"Expected Version",
		"Actual Version",
		"Drift Status",
		"Severity",
	}
	if err := csvWriter.Write(headers); err != nil {
		return err
	}

	// Write records
	for _, record := range report.Records {
		// Terraform version
		severity := "none"
		switch record.TerraformDriftStatus {
		case StatusMajorDrift:
			severity = "major"
		case StatusMinorDrift:
			severity = "minor"
		}

		row := []string{
			record.FilePath,
			"terraform",
			"terraform",
			record.TerraformExpected,
			record.TerraformActual,
			string(record.TerraformDriftStatus),
			severity,
		}
		if err := csvWriter.Write(row); err != nil {
			return err
		}

		// Providers
		for _, pd := range record.Providers {
			severity := "none"
			switch pd.DriftStatus {
			case StatusMajorDrift:
				severity = "major"
			case StatusMinorDrift:
				severity = "minor"
			}

			row := []string{
				record.FilePath,
				"provider",
				pd.Name,
				pd.Expected,
				pd.Actual,
				string(pd.DriftStatus),
				severity,
			}
			if err := csvWriter.Write(row); err != nil {
				return err
			}
		}
	}

	return nil
}

// formatStatus returns a plain status indicator
func (f *Formatter) formatStatus(status DriftStatus) string {
	switch status {
	case StatusInSync:
		return "OK"
	case StatusMinorDrift:
		return "minor drift"
	case StatusMajorDrift:
		return "major drift"
	case StatusMissing:
		return "missing"
	case StatusNotManaged:
		return "not managed"
	default:
		return string(status)
	}
}
