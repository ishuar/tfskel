package ai

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/ishuar/tfskel/internal/plan"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseSensitive(t *testing.T) {
	tests := []struct {
		name string
		raw  json.RawMessage
		want any
	}{
		{name: "absent field", raw: nil, want: nil},
		{name: "empty field", raw: json.RawMessage(``), want: nil},
		{name: "false means nothing sensitive", raw: json.RawMessage(`false`), want: nil},
		{name: "true means whole value sensitive", raw: json.RawMessage(`true`), want: true},
		{name: "invalid json is treated as not sensitive", raw: json.RawMessage(`{broken`), want: nil},
		{
			name: "nested object mirrors value shape",
			raw:  json.RawMessage(`{"password":true}`),
			want: map[string]any{"password": true},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, parseSensitive(tt.raw))
		})
	}
}

func TestSanitizeAndDiff_EdgeCases(t *testing.T) {
	t.Run("empty self returns nil", func(t *testing.T) {
		assert.Nil(t, sanitizeAndDiff(nil, map[string]any{"a": 1}, nil))
	})

	t.Run("all keys unchanged returns nil", func(t *testing.T) {
		m := map[string]any{"a": "same", "b": float64(1)}
		assert.Nil(t, sanitizeAndDiff(m, map[string]any{"a": "same", "b": float64(1)}, nil))
	})

	t.Run("nil other keeps every key", func(t *testing.T) {
		got := sanitizeAndDiff(map[string]any{"a": "x", "b": "y"}, nil, nil)
		assert.Equal(t, map[string]any{"a": "x", "b": "y"}, got)
	})

	t.Run("whole-sensitive with all keys unchanged returns nil", func(t *testing.T) {
		m := map[string]any{"a": "same"}
		assert.Nil(t, sanitizeAndDiff(m, map[string]any{"a": "same"}, true))
	})

	t.Run("whole-sensitive redacts only changed keys", func(t *testing.T) {
		self := map[string]any{"changed": "new", "same": "v"}
		other := map[string]any{"changed": "old", "same": "v"}
		got := sanitizeAndDiff(self, other, true)
		assert.Equal(t, map[string]any{"changed": redactedMarker}, got)
	})

	t.Run("non-map non-bool sensitivity is ignored", func(t *testing.T) {
		got := sanitizeAndDiff(map[string]any{"a": "x"}, nil, "unexpected-shape")
		assert.Equal(t, map[string]any{"a": "x"}, got)
	})
}

func TestSanitizeValue_Nested(t *testing.T) {
	t.Run("nested map recurses with matching sensitivity subtree", func(t *testing.T) {
		v := map[string]any{
			"tags":     map[string]any{"team": "infra", "token": "s3cr3t"},
			"replicas": float64(3),
		}
		sensitive := map[string]any{
			"tags": map[string]any{"token": true},
		}
		got := sanitizeValue(v, sensitive)
		want := map[string]any{
			"tags":     map[string]any{"team": "infra", "token": redactedMarker},
			"replicas": float64(3),
		}
		assert.Equal(t, want, got)
	})

	t.Run("string at exactly the cap is not truncated", func(t *testing.T) {
		exact := strings.Repeat("x", maxAttrValueChars)
		assert.Equal(t, exact, sanitizeValue(exact, nil))
	})

	t.Run("non-string values pass through untouched", func(t *testing.T) {
		assert.Equal(t, float64(42), sanitizeValue(float64(42), nil))
		assert.Nil(t, sanitizeValue(nil, nil))
	})
}

// TestBuildPayload_ModuleAndActionReasonFlowThrough covers the pass-through
// fields that only appear on module resources and replacements.
func TestBuildPayload_ModuleAndActionReasonFlowThrough(t *testing.T) {
	analysis := &plan.PlanAnalysis{
		ResourceChanges: []plan.AnalyzedResource{
			{
				Address:       "module.db.aws_db_instance.main",
				Type:          "aws_db_instance",
				Name:          "main",
				ModuleAddress: "module.db",
				Actions:       []string{"delete", "create"},
				Severity:      plan.SeverityCritical,
				ActionReason:  "replace_because_cannot_update",
				Change: plan.ChangeDetail{
					Actions: []string{"delete", "create"},
					Before:  map[string]any{"instance_class": "db.t3.micro"},
					After:   map[string]any{"instance_class": "db.t3.small"},
				},
			},
		},
	}

	p := BuildPayload(analysis, nil)
	require.Len(t, p.Resources, 1)
	assert.Equal(t, "module.db", p.Resources[0].Module)
	assert.Equal(t, "replace_because_cannot_update", p.Resources[0].ActionReason)
}
