package plan

import (
	"github.com/charmbracelet/lipgloss"
)

// Named color constants for terminal output
const (
	colorRed    = lipgloss.Color("1")
	colorGreen  = lipgloss.Color("2")
	colorYellow = lipgloss.Color("3")
	colorBlue   = lipgloss.Color("4")
	colorCyan   = lipgloss.Color("6")
)

// colorizeAction adds color styling to action strings
func (f *PlanFormatter) colorizeAction(action string) string {
	if !f.useColor {
		return action
	}

	var color lipgloss.Color
	switch action {
	case ActionCreate:
		color = colorGreen
	case ActionDelete:
		color = colorRed
	case ActionReplace:
		color = colorYellow
	case ActionUpdate:
		color = colorCyan
	case ActionRead:
		color = colorBlue
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
	case string(SeverityCritical):
		style = lipgloss.NewStyle().Bold(true).Foreground(colorRed)
	case string(SeverityHigh):
		style = lipgloss.NewStyle().Foreground(colorRed)
	case string(SeverityMedium):
		style = lipgloss.NewStyle().Foreground(colorYellow)
	case string(SeverityLow):
		style = lipgloss.NewStyle().Foreground(colorGreen)
	default:
		return severity
	}

	return style.Render(severity)
}
