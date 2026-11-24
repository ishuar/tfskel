package plan

import (
	"image/color"

	"charm.land/lipgloss/v2"
)

// Named color constants for terminal output
const (
	colorRed    = lipgloss.Red
	colorGreen  = lipgloss.Green
	colorYellow = lipgloss.Yellow
	colorBlue   = lipgloss.Blue
	colorCyan   = lipgloss.Cyan
)

// colorizeAction adds color styling to action strings
func (f *PlanFormatter) colorizeAction(action string) string {
	if !f.useColor {
		return action
	}

	var actionColor color.Color
	switch action {
	case ActionCreate:
		actionColor = colorGreen
	case ActionDelete:
		actionColor = colorRed
	case ActionReplace:
		actionColor = colorYellow
	case ActionUpdate:
		actionColor = colorCyan
	case ActionRead:
		actionColor = colorBlue
	default:
		return action
	}

	return lipgloss.NewStyle().Foreground(actionColor).Render(action)
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
