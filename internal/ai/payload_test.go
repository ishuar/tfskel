package ai

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/ishuar/tfskel/internal/plan"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildPayload_Sanitization(t *testing.T) {
	analysis := &plan.PlanAnalysis{
		TerraformVersion: "1.9.0",
		TotalChanges:     1,
		Modifications:    1,
		ResourceChanges: []plan.AnalyzedResource{
			{
				Address:  "aws_iam_role.admin",
				Type:     "aws_iam_role",
				Name:     "admin",
				Provider: "registry.terraform.io/hashicorp/aws",
				Actions:  []string{"update"},
				Severity: plan.SeverityHigh,
				Change: plan.ChangeDetail{
					Actions: []string{"update"},
					Before: map[string]any{
						"name":        "admin",
						"assume_role": "old-policy",
						"secret_arn":  "arn:aws:secretsmanager:...:secret-A",
					},
					After: map[string]any{
						"name":        "admin",
						"assume_role": "new-policy",
						"secret_arn":  "arn:aws:secretsmanager:...:secret-B",
					},
					// secret_arn is sensitive on both sides; nested-object form.
					BeforeSensitive: json.RawMessage(`{"secret_arn":true}`),
					AfterSensitive:  json.RawMessage(`{"secret_arn":true}`),
				},
			},
		},
	}

	p := BuildPayload(analysis, []string{"aws_iam_role"})

	require.Len(t, p.Resources, 1)
	r := p.Resources[0]

	assert.Equal(t, "high", r.Severity, "severity flows through")
	assert.Equal(t, []string{"aws_iam_role"}, p.CriticalResources)
	assert.Equal(t, "1.9.0", p.TerraformVersion)

	// Unchanged "name" must be dropped.
	_, hasNameBefore := r.Before["name"]
	_, hasNameAfter := r.After["name"]
	assert.False(t, hasNameBefore, "unchanged keys dropped from before")
	assert.False(t, hasNameAfter, "unchanged keys dropped from after")

	// Changed non-sensitive attr passes through.
	assert.Equal(t, "old-policy", r.Before["assume_role"])
	assert.Equal(t, "new-policy", r.After["assume_role"])

	// Sensitive attr redacted on both sides.
	assert.Equal(t, redactedMarker, r.Before["secret_arn"])
	assert.Equal(t, redactedMarker, r.After["secret_arn"])
}

func TestBuildPayload_WholeValueSensitive(t *testing.T) {
	// before_sensitive: true means the entire `before` map is sensitive.
	analysis := &plan.PlanAnalysis{
		ResourceChanges: []plan.AnalyzedResource{
			{
				Address: "aws_secretsmanager_secret_version.db",
				Type:    "aws_secretsmanager_secret_version",
				Name:    "db",
				Actions: []string{"update"},
				Change: plan.ChangeDetail{
					Actions:         []string{"update"},
					Before:          map[string]any{"secret_string": "old", "version_id": "v1"},
					After:           map[string]any{"secret_string": "new", "version_id": "v2"},
					BeforeSensitive: json.RawMessage(`true`),
					AfterSensitive:  json.RawMessage(`true`),
				},
			},
		},
	}

	p := BuildPayload(analysis, nil)
	require.Len(t, p.Resources, 1)
	r := p.Resources[0]

	for k, v := range r.Before {
		assert.Equalf(t, redactedMarker, v, "before key %q must be redacted", k)
	}
	for k, v := range r.After {
		assert.Equalf(t, redactedMarker, v, "after key %q must be redacted", k)
	}
}

func TestBuildPayload_TruncatesLongValues(t *testing.T) {
	big := strings.Repeat("x", maxAttrValueChars+200)
	analysis := &plan.PlanAnalysis{
		ResourceChanges: []plan.AnalyzedResource{
			{
				Address: "aws_instance.web",
				Type:    "aws_instance",
				Name:    "web",
				Actions: []string{"update"},
				Change: plan.ChangeDetail{
					Actions: []string{"update"},
					Before:  map[string]any{"user_data": "small"},
					After:   map[string]any{"user_data": big},
				},
			},
		},
	}

	p := BuildPayload(analysis, nil)
	got, ok := p.Resources[0].After["user_data"].(string)
	require.True(t, ok, "user_data should remain a string after truncation")
	assert.LessOrEqual(t, len(got), maxAttrValueChars+len(truncationSuffix))
	assert.True(t, strings.HasSuffix(got, truncationSuffix), "long value should be truncated with marker")
}

func TestBuildPayload_OutputChangesDropValues(t *testing.T) {
	analysis := &plan.PlanAnalysis{
		OutputChanges: []plan.OutputChange{
			{Name: "db_password", Actions: []string{"update"}, Sensitive: true},
			{Name: "endpoint", Actions: []string{"create"}, Sensitive: false},
		},
	}
	p := BuildPayload(analysis, nil)
	require.Len(t, p.OutputChanges, 2)

	// Confirm marshaled form does not contain any value fields — only name/actions/sensitive.
	raw, err := json.Marshal(p)
	require.NoError(t, err)
	assert.NotContains(t, string(raw), `"value"`)
}

func TestBuildPayload_WireKeys(t *testing.T) {
	analysis := &plan.PlanAnalysis{TerraformVersion: "1.9.0", TotalChanges: 1}
	p := BuildPayload(analysis, nil)
	raw, err := json.Marshal(p)
	require.NoError(t, err)
	assert.Contains(t, string(raw), `"terraform_version":"1.9.0"`, "wire key names are part of the prompt contract")
	assert.Contains(t, string(raw), `"counts":{"total":1`)
}

// TestBuildPayload_OnlyAnalyzedResourcesWithSeverity locks in the alignment
// invariant: the payload contains exactly the resources present in the
// analysis — a filtered analysis (or one that dropped data sources and no-ops)
// produces a payload of the same shape — and every resource carries a
// non-empty severity, as required by the prompt contract.
func TestBuildPayload_OnlyAnalyzedResourcesWithSeverity(t *testing.T) {
	raw := &plan.TerraformPlan{
		TerraformVersion: "1.9.0",
		ResourceChanges: []plan.ResourceChange{
			{
				Address: "aws_db_instance.main",
				Type:    "aws_db_instance",
				Name:    "main",
				Change:  plan.ChangeDetail{Actions: []string{"delete", "create"}},
			},
			{
				Address: "aws_s3_bucket.logs",
				Type:    "aws_s3_bucket",
				Name:    "logs",
				Change:  plan.ChangeDetail{Actions: []string{"create"}},
			},
			{
				// Data source: excluded by the analyzer, must never reach the model.
				Address: "data.aws_ami.latest",
				Mode:    "data",
				Type:    "aws_ami",
				Name:    "latest",
				Change:  plan.ChangeDetail{Actions: []string{"read"}},
			},
			{
				// No-op: excluded by the analyzer, must never reach the model.
				Address: "aws_vpc.main",
				Type:    "aws_vpc",
				Name:    "main",
				Change:  plan.ChangeDetail{Actions: []string{"no-op"}},
			},
		},
	}

	analyzer := plan.NewPlanAnalyzer()
	analysis := analyzer.Analyze(raw)

	// Simulate an active --min-severity=critical filter, exactly as the
	// command applies it before building the payload.
	filter := &plan.ResourceFilter{MinSeverity: "critical"}
	analysis.ResourceChanges = plan.FilterResources(analysis.ResourceChanges, filter)
	require.Len(t, analysis.ResourceChanges, 1, "only the replacement survives the filter")

	p := BuildPayload(analysis, nil)

	require.Len(t, p.Resources, 1, "payload must contain exactly the analyzed (filtered) resources")
	assert.Equal(t, "aws_db_instance.main", p.Resources[0].Address)
	for _, r := range p.Resources {
		assert.NotEmpty(t, r.Severity, "every resource on the wire must carry a computed severity")
	}
}
