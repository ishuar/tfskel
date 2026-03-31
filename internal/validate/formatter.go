package validate

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/ishuar/tfskel/internal/format"
	"github.com/ishuar/tfskel/internal/toolcheck"
	"golang.org/x/term"
)

const (
	symbolPass    = "✓"
	symbolFail    = "✗"
	symbolSkipped = "-"

	minTableWidth = 80
	maxTableWidth = 150
)

// ErrUnsupportedFormat indicates an unsupported output format was requested.
var ErrUnsupportedFormat = errors.New("unsupported format")

// Formatter handles output formatting for validation reports.
type Formatter struct {
	useColor      bool
	terminalWidth int
	tableWidth    int
}

// NewFormatter creates a new formatter.
func NewFormatter(useColor bool) *Formatter {
	width := format.DefaultTerminalWidth
	if fd := int(os.Stdout.Fd()); term.IsTerminal(fd) {
		if w, _, err := term.GetSize(fd); err == nil && w > 0 {
			width = w
		}
	}
	return &Formatter{
		useColor:      useColor,
		terminalWidth: width,
	}
}

// Format writes the report in the specified format.
func (f *Formatter) Format(report *Report, outputFormat format.OutputFormat, w io.Writer) error {
	switch outputFormat {
	case format.FormatTable:
		return f.formatTable(report, w)
	case format.FormatJSON:
		return f.formatJSON(report, w)
	case format.FormatCSV:
		return f.formatCSV(report, w)
	default:
		return fmt.Errorf("%w: %s", ErrUnsupportedFormat, outputFormat)
	}
}

// --- Table Output ---

func (f *Formatter) formatTable(report *Report, w io.Writer) error {
	buf := &bytes.Buffer{}
	styles := format.NewCommonStyles(f.useColor)
	f.tableWidth = f.calculateOptimalWidth()

	// Header
	if _, err := fmt.Fprintln(buf, styles.TitleStyle.Render("━━━ Validation Report ━━━")); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(buf, "%s  .\n", styles.MutedStyle.Render("Directory:")); err != nil {
		return err
	}

	// Detail sections for failed checks only.
	// Ordered by complexity: simple list first, detailed table last.
	if err := f.writeToolDetails(buf, report, styles); err != nil {
		return err
	}
	if err := f.writeConfigDetails(buf, report, styles); err != nil {
		return err
	}

	// Summary lines — after details so the recap is the last thing visible.
	if _, err := fmt.Fprintln(buf); err != nil {
		return err
	}
	for _, c := range report.Checks {
		if err := f.writeSummaryLine(buf, c, styles); err != nil {
			return err
		}
	}

	// Verdict
	if _, err := fmt.Fprintln(buf); err != nil {
		return err
	}
	issues := report.IssueCount()
	if issues == 0 {
		if _, err := fmt.Fprintf(buf, "%s All checks passed\n", f.colorize(symbolPass, lipgloss.Color("2"))); err != nil {
			return err
		}
	} else {
		noun := "findings"
		if issues == 1 {
			noun = "finding"
		}
		if _, err := fmt.Fprintf(buf, "%s Validation failed (%d %s)\n", f.colorize(symbolFail, lipgloss.Color("1")), issues, noun); err != nil {
			return err
		}
	}

	_, err := w.Write(buf.Bytes())
	return err
}

func (f *Formatter) writeSummaryLine(w io.Writer, c CheckResult, styles format.CommonStyles) error {
	var symbol, detail string

	switch c.Status {
	case StatusPass:
		symbol = f.colorize(symbolPass, lipgloss.Color("2"))
		detail = f.passSummary(c)
	case StatusFail:
		symbol = f.colorize(symbolFail, lipgloss.Color("1"))
		detail = f.failSummary(c)
	case StatusError:
		symbol = f.colorize(symbolFail, lipgloss.Color("1"))
		detail = "check failed"
	case StatusSkipped:
		symbol = symbolSkipped
		detail = "skipped"
	}

	// Pad check name to align details
	name := fmt.Sprintf("%-8s", c.Check)
	_, err := fmt.Fprintf(w, "  %s %s%s\n", symbol, styles.MutedStyle.Render(name), detail)
	return err
}

func (f *Formatter) passSummary(c CheckResult) string {
	switch c.Check {
	case CheckConfig:
		if c.Total == 0 {
			return "no .tf files found"
		}
		return fmt.Sprintf("%d files, all version constraints match", c.Total)
	case CheckTools:
		return fmt.Sprintf("%d tools installed", c.Passed)
	}
	return "pass"
}

func (f *Formatter) failSummary(c CheckResult) string {
	switch c.Check {
	case CheckConfig:
		return fmt.Sprintf("%d of %d files with version drift", c.Issues, c.Total)
	case CheckTools:
		noun := "findings"
		if c.Issues == 1 {
			noun = "finding"
		}
		return fmt.Sprintf("%d tool %s", c.Issues, noun)
	}
	return fmt.Sprintf("%d findings", c.Issues)
}

// writeConfigDetails writes the "Version Drift" table.
func (f *Formatter) writeConfigDetails(w io.Writer, report *Report, styles format.CommonStyles) error {
	findings := findingsForCheck(report.Findings, CheckConfig)
	if len(findings) == 0 {
		return nil
	}

	if _, err := fmt.Fprintln(w); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, styles.HeaderStyle.Render("Version Drift")); err != nil {
		return err
	}

	rows := f.buildConfigRows(findings, styles)

	t := table.New().
		Border(lipgloss.RoundedBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(styles.BorderColor)).
		Width(f.tableWidth).
		StyleFunc(func(row, _ int) lipgloss.Style {
			if row == -1 {
				return lipgloss.NewStyle().Bold(true).Foreground(styles.HeaderColor).Align(lipgloss.Center)
			}
			return lipgloss.NewStyle().Foreground(styles.RowColor).Align(lipgloss.Left)
		}).
		Headers("File", "Component", "Expected", "Actual").
		Rows(rows...)

	_, err := fmt.Fprintln(w, t.Render())
	return err
}

// buildConfigRows groups findings by file, using ↳ continuation for same-file entries.
func (f *Formatter) buildConfigRows(findings []Finding, styles format.CommonStyles) [][]string {
	rows := make([][]string, 0, len(findings))
	lastFile := ""

	for _, finding := range findings {
		file := finding.Resource
		if file == lastFile {
			file = styles.MutedStyle.Render("  ↳")
		} else {
			lastFile = finding.Resource
		}

		rows = append(rows, []string{
			file,
			finding.Component,
			finding.Expected,
			finding.Actual,
		})
	}

	return rows
}

// writeToolDetails writes the detailed tool status section.
// When the full toolcheck report is available, it renders the same per-tool
// breakdown as init (via toolcheck.FormatReport). Otherwise, falls back to
// listing findings.
func (f *Formatter) writeToolDetails(w io.Writer, report *Report, styles format.CommonStyles) error {
	// Use the full toolcheck report when available.
	if tr := report.ToolReport; tr != nil {
		if _, err := fmt.Fprintln(w); err != nil {
			return err
		}
		if _, err := fmt.Fprintln(w, styles.HeaderStyle.Render("Required Tools")); err != nil {
			return err
		}
		// FormatReport includes a "Running pre-flight checks...\n\n" header;
		// strip it since we render our own styled heading above.
		full := toolcheck.FormatReport(tr)
		full = strings.TrimPrefix(full, "Running pre-flight checks...\n\n")
		if _, err := fmt.Fprint(w, full); err != nil {
			return err
		}

		// Also show findings so users see the specific issues behind the count.
		return f.writeToolFindingsList(w, report, styles)
	}

	// Fallback: show findings (no toolcheck report stored).
	return f.writeToolFindings(w, report, styles)
}

// writeToolFindingsList renders tool findings as a bullet list.
// Used after the toolcheck overview to show the specific issues.
func (f *Formatter) writeToolFindingsList(w io.Writer, report *Report, styles format.CommonStyles) error {
	findings := findingsForCheck(report.Findings, CheckTools)
	if len(findings) == 0 {
		return nil
	}

	if _, err := fmt.Fprintln(w); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, styles.HeaderStyle.Render("Tool Findings")); err != nil {
		return err
	}

	for _, finding := range findings {
		sym := f.colorize(symbolFail, lipgloss.Color("1"))
		msg := finding.Resource
		if finding.Component != "" {
			msg += " (" + finding.Component + ")"
		}
		msg += ": " + finding.Message
		if finding.Detail != "" {
			msg += " — " + styles.MutedStyle.Render(finding.Detail)
		}
		if _, err := fmt.Fprintf(w, "  %s %s\n", sym, msg); err != nil {
			return err
		}
	}

	return nil
}

// writeToolFindings renders tool findings when the full toolcheck report
// is not available.
func (f *Formatter) writeToolFindings(w io.Writer, report *Report, styles format.CommonStyles) error {
	findings := findingsForCheck(report.Findings, CheckTools)
	if len(findings) == 0 {
		return nil
	}

	if _, err := fmt.Fprintln(w); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, styles.HeaderStyle.Render("Required Tools")); err != nil {
		return err
	}

	maxLen := 0
	for _, finding := range findings {
		if len(finding.Resource) > maxLen {
			maxLen = len(finding.Resource)
		}
	}

	for _, finding := range findings {
		sym := f.colorize(symbolFail, lipgloss.Color("1"))
		padding := strings.Repeat(" ", maxLen-len(finding.Resource)+2)
		detail := finding.Message
		if finding.Detail != "" {
			detail += " — " + finding.Detail
		}
		if _, err := fmt.Fprintf(w, "  %s %s%s%s\n", sym, finding.Resource, padding, styles.MutedStyle.Render(detail)); err != nil {
			return err
		}
	}

	for _, finding := range findings {
		if strings.Contains(finding.Detail, "mise install") {
			if _, err := fmt.Fprintln(w); err != nil {
				return err
			}
			if _, err := fmt.Fprintf(w, "  %s\n", styles.MutedStyle.Render("Run 'mise install' to install missing tools.")); err != nil {
				return err
			}
			break
		}
	}

	return nil
}

// --- JSON Output ---

func (f *Formatter) formatJSON(report *Report, w io.Writer) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false)

	output := struct {
		Checks   []CheckResult `json:"checks"`
		Findings []Finding     `json:"findings"`
		ExitCode int           `json:"exitCode"`
	}{
		Checks:   report.Checks,
		Findings: report.Findings,
		ExitCode: report.ExitCode(),
	}

	return encoder.Encode(output)
}

// --- CSV Output ---

func (f *Formatter) formatCSV(report *Report, w io.Writer) error {
	cw := csv.NewWriter(w)
	defer cw.Flush()

	if err := cw.Write([]string{"check", "resource", "component", "message", "expected", "actual", "detail"}); err != nil {
		return err
	}

	for _, finding := range report.Findings {
		row := []string{
			string(finding.Check),
			finding.Resource,
			finding.Component,
			finding.Message,
			finding.Expected,
			finding.Actual,
			finding.Detail,
		}
		if err := cw.Write(row); err != nil {
			return err
		}
	}

	return nil
}

// --- Helpers ---

func findingsForCheck(findings []Finding, check CheckName) []Finding {
	var result []Finding
	for _, finding := range findings {
		if finding.Check == check {
			result = append(result, finding)
		}
	}
	return result
}

func (f *Formatter) colorize(text string, color lipgloss.Color) string {
	if !f.useColor {
		return text
	}
	return lipgloss.NewStyle().Foreground(color).Render(text)
}

func (f *Formatter) calculateOptimalWidth() int {
	optimal := (f.terminalWidth * format.PercentageWidthFactor) / format.PercentageDivisor
	if optimal < minTableWidth {
		return minTableWidth
	}
	if optimal > maxTableWidth {
		return maxTableWidth
	}
	return optimal
}
