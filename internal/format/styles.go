package format

import (
	"os"

	"github.com/charmbracelet/lipgloss"
)

// CommonStyles holds shared lipgloss styles for table formatting
type CommonStyles struct {
	TitleStyle  lipgloss.Style
	HeaderStyle lipgloss.Style
	MutedStyle  lipgloss.Style
	BorderColor lipgloss.Color
	HeaderColor lipgloss.Color
	RowColor    lipgloss.Color
}

// NewCommonStyles creates consistent styles for both version and plan formatters
func NewCommonStyles(useColor bool) CommonStyles {
	borderColor := lipgloss.Color("14")
	headerColor := lipgloss.Color("14")
	rowColor := lipgloss.Color("252")

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(headerColor).MarginBottom(1)
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(headerColor).MarginTop(1)
	mutedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))

	if !useColor {
		titleStyle = titleStyle.Foreground(lipgloss.NoColor{})
		headerStyle = headerStyle.Foreground(lipgloss.NoColor{})
		mutedStyle = mutedStyle.Foreground(lipgloss.NoColor{})
		borderColor = lipgloss.Color("")
		headerColor = lipgloss.Color("")
		rowColor = lipgloss.Color("")
	}

	return CommonStyles{
		TitleStyle:  titleStyle,
		HeaderStyle: headerStyle,
		MutedStyle:  mutedStyle,
		BorderColor: borderColor,
		HeaderColor: headerColor,
		RowColor:    rowColor,
	}
}

// ShouldUseColor determines if colored output should be used based on flags and environment variables.
// It respects standard conventions: NO_COLOR (disables), FORCE_COLOR (enables), CI (disables),
// and falls back to the flag.
//
// Precedence order (highest to lowest):
//  1. NO_COLOR environment variable - always disables colors if set
//  2. FORCE_COLOR environment variable - enables colors if set to non-zero/non-false value
//  3. CI environment variable - disables colors when running in CI/CD (set by GitHub Actions, GitLab CI, etc.)
//  4. noColorFlag parameter - used as fallback if no environment variables are set
//
// References:
//   - NO_COLOR standard: https://no-color.org/
//   - FORCE_COLOR convention: common in CI/CD tools like GitHub Actions
func ShouldUseColor(noColorFlag bool) bool {
	// Check NO_COLOR environment variable (standard: https://no-color.org/)
	if _, ok := os.LookupEnv("NO_COLOR"); ok {
		return false
	}

	// Check FORCE_COLOR environment variable
	if forceColor := os.Getenv("FORCE_COLOR"); forceColor != "" && forceColor != "0" && forceColor != "false" {
		return true
	}

	// Check CI environment variable (set by GitHub Actions, GitLab CI, Jenkins, etc.)
	if ci := os.Getenv("CI"); ci == "true" || ci == "1" {
		return false
	}

	// Fall back to flag value
	return !noColorFlag
}
