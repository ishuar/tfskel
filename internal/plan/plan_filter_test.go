package plan

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResourceFilter_IsEmpty(t *testing.T) {
	assert.True(t, (&ResourceFilter{}).IsEmpty())
	assert.False(t, (&ResourceFilter{Severities: []string{"high"}}).IsEmpty())
	assert.False(t, (&ResourceFilter{MinSeverity: "high"}).IsEmpty())
	assert.False(t, (&ResourceFilter{Actions: []string{"delete"}}).IsEmpty())
}

func TestResourceFilter_Validate(t *testing.T) {
	tests := []struct {
		name    string
		filter  ResourceFilter
		wantErr string
	}{
		{
			name:   "empty filter is valid",
			filter: ResourceFilter{},
		},
		{
			name:   "valid severities",
			filter: ResourceFilter{Severities: []string{"critical", "high"}},
		},
		{
			name:   "valid actions",
			filter: ResourceFilter{Actions: []string{"delete", "replace"}},
		},
		{
			name:   "case insensitive severity",
			filter: ResourceFilter{Severities: []string{"CRITICAL", "High"}},
		},
		{
			name:   "case insensitive action",
			filter: ResourceFilter{Actions: []string{"DELETE", "Create"}},
		},
		{
			name:    "invalid severity",
			filter:  ResourceFilter{Severities: []string{"critical", "bogus"}},
			wantErr: `unknown severity "bogus"`,
		},
		{
			name:    "invalid action",
			filter:  ResourceFilter{Actions: []string{"explode"}},
			wantErr: `unknown action "explode"`,
		},
		{
			name:   "valid min severity",
			filter: ResourceFilter{MinSeverity: "high"},
		},
		{
			name:   "case insensitive min severity",
			filter: ResourceFilter{MinSeverity: "HIGH"},
		},
		{
			name:    "invalid min severity",
			filter:  ResourceFilter{MinSeverity: "extreme"},
			wantErr: `unknown severity "extreme"`,
		},
		{
			name:    "mutually exclusive severity filters",
			filter:  ResourceFilter{Severities: []string{"critical"}, MinSeverity: "high"},
			wantErr: "mutually exclusive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.filter.Validate()
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestResourceFilter_Match(t *testing.T) {
	resources := []AnalyzedResource{
		{Name: "critical-delete", Severity: SeverityCritical, ActionString: ActionDelete},
		{Name: "critical-replace", Severity: SeverityCritical, ActionString: ActionReplace},
		{Name: "high-update", Severity: SeverityHigh, ActionString: ActionUpdate},
		{Name: "low-create", Severity: SeverityLow, ActionString: ActionCreate},
		{Name: "medium-update", Severity: SeverityMedium, ActionString: ActionUpdate},
	}

	tests := []struct {
		name      string
		filter    ResourceFilter
		wantNames []string
	}{
		{
			name:      "empty filter matches all",
			filter:    ResourceFilter{},
			wantNames: []string{"critical-delete", "critical-replace", "high-update", "low-create", "medium-update"},
		},
		{
			name:      "single severity",
			filter:    ResourceFilter{Severities: []string{"critical"}},
			wantNames: []string{"critical-delete", "critical-replace"},
		},
		{
			name:      "multiple severities (OR within)",
			filter:    ResourceFilter{Severities: []string{"critical", "high"}},
			wantNames: []string{"critical-delete", "critical-replace", "high-update"},
		},
		{
			name:      "single action",
			filter:    ResourceFilter{Actions: []string{"delete"}},
			wantNames: []string{"critical-delete"},
		},
		{
			name:      "multiple actions (OR within)",
			filter:    ResourceFilter{Actions: []string{"delete", "replace"}},
			wantNames: []string{"critical-delete", "critical-replace"},
		},
		{
			name:      "AND across severity and action",
			filter:    ResourceFilter{Severities: []string{"critical"}, Actions: []string{"delete"}},
			wantNames: []string{"critical-delete"},
		},
		{
			name:      "AND narrows: critical + update matches nothing",
			filter:    ResourceFilter{Severities: []string{"critical"}, Actions: []string{"update"}},
			wantNames: nil,
		},
		{
			name:      "case insensitive match",
			filter:    ResourceFilter{Severities: []string{"CRITICAL"}, Actions: []string{"DELETE"}},
			wantNames: []string{"critical-delete"},
		},
		{
			name:      "min severity critical: only critical",
			filter:    ResourceFilter{MinSeverity: "critical"},
			wantNames: []string{"critical-delete", "critical-replace"},
		},
		{
			name:      "min severity high: high and critical",
			filter:    ResourceFilter{MinSeverity: "high"},
			wantNames: []string{"critical-delete", "critical-replace", "high-update"},
		},
		{
			name:      "min severity medium: medium, high, critical",
			filter:    ResourceFilter{MinSeverity: "medium"},
			wantNames: []string{"critical-delete", "critical-replace", "high-update", "medium-update"},
		},
		{
			name:      "min severity low: everything",
			filter:    ResourceFilter{MinSeverity: "low"},
			wantNames: []string{"critical-delete", "critical-replace", "high-update", "low-create", "medium-update"},
		},
		{
			name:      "min severity AND action filter",
			filter:    ResourceFilter{MinSeverity: "high", Actions: []string{"delete"}},
			wantNames: []string{"critical-delete"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got []string
			for _, r := range resources {
				if tt.filter.Match(r) {
					got = append(got, r.Name)
				}
			}
			assert.Equal(t, tt.wantNames, got)
		})
	}
}

func TestFilterResources(t *testing.T) {
	resources := []AnalyzedResource{
		{Name: "a", Severity: SeverityCritical, ActionString: ActionDelete},
		{Name: "b", Severity: SeverityLow, ActionString: ActionCreate},
		{Name: "c", Severity: SeverityHigh, ActionString: ActionUpdate},
	}

	t.Run("nil filter returns original", func(t *testing.T) {
		got := FilterResources(resources, nil)
		assert.Equal(t, resources, got)
	})

	t.Run("empty filter returns original", func(t *testing.T) {
		got := FilterResources(resources, &ResourceFilter{})
		assert.Equal(t, resources, got)
	})

	t.Run("filters correctly", func(t *testing.T) {
		got := FilterResources(resources, &ResourceFilter{Severities: []string{"critical"}})
		require.Len(t, got, 1)
		assert.Equal(t, "a", got[0].Name)
	})

	t.Run("returns nil slice when nothing matches", func(t *testing.T) {
		got := FilterResources(resources, &ResourceFilter{Actions: []string{"replace"}})
		assert.Nil(t, got)
	})
}

func TestResourceFilter_Descriptions(t *testing.T) {
	tests := []struct {
		name   string
		filter ResourceFilter
		want   []string
	}{
		{
			name:   "empty",
			filter: ResourceFilter{},
			want:   nil,
		},
		{
			name:   "severity only",
			filter: ResourceFilter{Severities: []string{"critical", "high"}},
			want:   []string{"severity=critical,high"},
		},
		{
			name:   "action only",
			filter: ResourceFilter{Actions: []string{"delete"}},
			want:   []string{"action=delete"},
		},
		{
			name:   "both",
			filter: ResourceFilter{Severities: []string{"critical"}, Actions: []string{"delete", "replace"}},
			want:   []string{"severity=critical", "action=delete,replace"},
		},
		{
			name:   "min severity",
			filter: ResourceFilter{MinSeverity: "high"},
			want:   []string{"min-severity=high"},
		},
		{
			name:   "min severity and action",
			filter: ResourceFilter{MinSeverity: "high", Actions: []string{"delete"}},
			want:   []string{"min-severity=high", "action=delete"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.filter.Descriptions())
		})
	}
}
