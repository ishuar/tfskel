package drift

import "github.com/charmbracelet/lipgloss"

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
