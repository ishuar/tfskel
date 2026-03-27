package plan

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPlanFormatter_colorizeAction(t *testing.T) {
	tests := []struct {
		name   string
		action string
	}{
		{
			name:   "create - green",
			action: "create",
		},
		{
			name:   "delete - red",
			action: "delete",
		},
		{
			name:   "replace - yellow",
			action: "replace",
		},
		{
			name:   "update - cyan",
			action: "update",
		},
		{
			name:   "read - blue",
			action: "read",
		},
		{
			name:   "unknown action",
			action: "unknown",
		},
	}

	formatter := NewPlanFormatter(true)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatter.colorizeAction(tt.action)
			// Lipgloss Style.Render() returns the styled text
			// Just verify the action text is present
			assert.Contains(t, result, tt.action)
			// Result should not be empty
			assert.NotEmpty(t, result)
		})
	}

	// Test without color
	t.Run("no color", func(t *testing.T) {
		formatterNoColor := NewPlanFormatter(false)
		result := formatterNoColor.colorizeAction("create")
		assert.Equal(t, "create", result)
	})
}

func TestPlanFormatter_colorizeSeverity(t *testing.T) {
	tests := []struct {
		name     string
		severity string
	}{
		{
			name:     "critical - bold red",
			severity: "critical",
		},
		{
			name:     "high - red",
			severity: "high",
		},
		{
			name:     "medium - yellow",
			severity: "medium",
		},
		{
			name:     "low - green",
			severity: "low",
		},
		{
			name:     "unknown severity",
			severity: "unknown",
		},
	}

	formatter := NewPlanFormatter(true)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatter.colorizeSeverity(tt.severity)
			// Lipgloss Style.Render() returns the styled text
			// Just verify the severity text is present
			assert.Contains(t, result, tt.severity)
			// Result should not be empty
			assert.NotEmpty(t, result)
		})
	}

	// Test without color
	t.Run("no color", func(t *testing.T) {
		formatterNoColor := NewPlanFormatter(false)
		result := formatterNoColor.colorizeSeverity("critical")
		assert.Equal(t, "critical", result)
	})
}
