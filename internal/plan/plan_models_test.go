package plan

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSeverityOrder(t *testing.T) {
	tests := []struct {
		name     string
		severity Severity
		want     int
	}{
		{"critical has lowest order (highest priority)", SeverityCritical, 0},
		{"high has second order", SeverityHigh, 1},
		{"medium has third order", SeverityMedium, 2},
		{"low has fourth order", SeverityLow, 3},
		{"unknown severity has highest order (lowest priority)", Severity("unknown"), 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.severity.Order()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestSortResourcesBySeverity(t *testing.T) {
	// Create resources in random order
	resources := []AnalyzedResource{
		{Name: "low1", Severity: SeverityLow},
		{Name: "high1", Severity: SeverityHigh},
		{Name: "critical1", Severity: SeverityCritical},
		{Name: "medium1", Severity: SeverityMedium},
		{Name: "low2", Severity: SeverityLow},
		{Name: "critical2", Severity: SeverityCritical},
	}

	sorted := sortResourcesBySeverity(resources)

	// Verify original is unchanged
	assert.Equal(t, "low1", resources[0].Name, "Original slice should not be modified")

	// Verify sorted order
	expected := []string{"critical1", "critical2", "high1", "medium1", "low1", "low2"}
	actual := make([]string, len(sorted))
	for i, r := range sorted {
		actual[i] = r.Name
	}
	assert.Equal(t, expected, actual, "Should be sorted by severity with stable order")
}

func TestFormatActions(t *testing.T) {
	tests := []struct {
		name    string
		actions []string
		want    string
	}{
		{
			name:    "empty actions",
			actions: []string{},
			want:    ActionNoOp,
		},
		{
			name:    "single action - create",
			actions: []string{ActionCreate},
			want:    ActionCreate,
		},
		{
			name:    "single action - update",
			actions: []string{ActionUpdate},
			want:    ActionUpdate,
		},
		{
			name:    "single action - delete",
			actions: []string{ActionDelete},
			want:    ActionDelete,
		},
		{
			name:    "replace actions",
			actions: []string{ActionDelete, ActionCreate},
			want:    ActionReplace,
		},
		{
			name:    "replace actions reversed",
			actions: []string{ActionCreate, ActionDelete},
			want:    ActionReplace,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatActions(tt.actions)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestContainsAction(t *testing.T) {
	tests := []struct {
		name    string
		actions []string
		action  string
		want    bool
	}{
		{
			name:    "action present",
			actions: []string{ActionCreate, ActionDelete},
			action:  ActionCreate,
			want:    true,
		},
		{
			name:    "action not present",
			actions: []string{ActionCreate, ActionUpdate},
			action:  ActionDelete,
			want:    false,
		},
		{
			name:    "empty actions",
			actions: []string{},
			action:  ActionCreate,
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := contains(tt.actions, tt.action)
			assert.Equal(t, tt.want, got)
		})
	}
}
