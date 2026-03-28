package plan

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/ishuar/tfskel/internal/format"
	"golang.org/x/term"
)

// detectTerminalWidth returns the current terminal width, or the default if detection fails.
func detectTerminalWidth() int {
	if fd := int(os.Stdout.Fd()); term.IsTerminal(fd) {
		if w, _, err := term.GetSize(fd); err == nil && w > 0 {
			return w
		}
	}
	return format.DefaultTerminalWidth
}

var (
	// ErrUnsupportedPlanFormat indicates an unsupported output format
	ErrUnsupportedPlanFormat = errors.New("unsupported format")
)

// PlanFormatter handles formatting of plan analysis results
type PlanFormatter struct {
	useColor           bool
	terminalWidth      int
	tableWidth         int      // Consistent width for all tables
	topResourcesCount  int      // Number of resources to show in top-N summaries
	totalResourceCount int      // Original unfiltered count; 0 means not filtered
	activeFilters      []string // Human-readable filter descriptions
}

// NewPlanFormatter creates a new plan formatter with auto-detected terminal width
func NewPlanFormatter(useColor bool) *PlanFormatter {
	return &PlanFormatter{
		useColor:          useColor,
		terminalWidth:     detectTerminalWidth(),
		topResourcesCount: DefaultTopResourcesCount,
	}
}

// NewPlanFormatterWithConfig creates a new plan formatter with configuration.
// topResourcesCount: 0 = show all (unlimited), negative = use default (10), positive = use that limit.
func NewPlanFormatterWithConfig(useColor bool, topResourcesCount int) *PlanFormatter {
	if topResourcesCount < 0 {
		topResourcesCount = DefaultTopResourcesCount
	}
	return &PlanFormatter{
		useColor:          useColor,
		terminalWidth:     detectTerminalWidth(),
		topResourcesCount: topResourcesCount,
	}
}

// NewPlanFormatterFiltered creates a formatter with filter metadata.
// totalResourceCount is the original unfiltered count, activeFilters are human-readable descriptions.
func NewPlanFormatterFiltered(useColor bool, topResourcesCount int, totalResourceCount int, activeFilters []string) *PlanFormatter {
	if topResourcesCount < 0 {
		topResourcesCount = DefaultTopResourcesCount
	}
	return &PlanFormatter{
		useColor:           useColor,
		terminalWidth:      detectTerminalWidth(),
		topResourcesCount:  topResourcesCount,
		totalResourceCount: totalResourceCount,
		activeFilters:      activeFilters,
	}
}

// Format outputs the plan analysis in the specified format
func (f *PlanFormatter) Format(analysis *PlanAnalysis, outputFormat format.OutputFormat, w io.Writer) error {
	switch outputFormat {
	case format.FormatJSON:
		return f.formatJSON(analysis, w)
	case format.FormatCSV:
		return f.formatCSV(analysis, w)
	case format.FormatTable:
		return f.formatTable(analysis, w)
	default:
		return fmt.Errorf("%w: %s", ErrUnsupportedPlanFormat, outputFormat)
	}
}
