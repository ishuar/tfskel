package plan

import (
	"bytes"
	"slices"
	"testing"

	"github.com/ishuar/tfskel/internal/format"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlanFormatter_buildResourceData_SortsBySeverity(t *testing.T) {
	formatter := NewPlanFormatter(false)

	// Create resources in random order
	resources := []AnalyzedResource{
		{Name: "resource1", Type: "aws_instance", ActionString: "create", Severity: SeverityLow},
		{Name: "resource2", Type: "aws_s3_bucket", ActionString: "update", Severity: SeverityHigh},
		{Name: "resource3", Type: "aws_lambda", ActionString: "update", Severity: SeverityMedium},
		{Name: "resource4", Type: "aws_rds_cluster", ActionString: "delete", Severity: SeverityCritical},
		{Name: "resource5", Type: "aws_vpc", ActionString: "update", Severity: SeverityHigh},
		{Name: "resource6", Type: "aws_dynamodb", ActionString: "delete", Severity: SeverityCritical},
		{Name: "resource7", Type: "aws_ec2", ActionString: "create", Severity: SeverityLow},
	}

	data := formatter.buildResourceData(resources)

	// Verify sorting: critical, high, medium, low
	// Extract severity from the data (4th column)
	severities := make([]string, len(data))
	for i, row := range data {
		severities[i] = row[3] // Severity is the 4th column
	}

	// Expected order: critical (2), high (2), medium (1), low (2)
	expectedOrder := []string{"critical", "critical", "high", "high", "medium", "low", "low"}
	assert.Equal(t, expectedOrder, severities, "Resources should be sorted by severity: critical, high, medium, low")

	// Verify resource names are in expected order
	expectedNames := []string{"resource4", "resource6", "resource2", "resource5", "resource3", "resource1", "resource7"}
	actualNames := make([]string, len(data))
	for i, row := range data {
		actualNames[i] = row[0] // Name is the 1st column
	}
	assert.Equal(t, expectedNames, actualNames, "Resources should maintain their order within same severity")
}

func TestPlanFormatter_writeTableOutputChanges(t *testing.T) {
	tests := []struct {
		name          string
		outputChanges []OutputChange
		wantContains  []string
	}{
		{
			name: "renders output changes with actions",
			outputChanges: []OutputChange{
				{Name: "vpc_id", Actions: []string{"create"}, Sensitive: false},
				{Name: "db_endpoint", Actions: []string{"delete"}, Sensitive: false},
				{Name: "api_url", Actions: []string{"update"}, Sensitive: false},
			},
			wantContains: []string{
				"Output Changes",
				"3 total",
				"vpc_id", "create",
				"db_endpoint", "delete",
				"api_url", "update",
			},
		},
		{
			name: "marks sensitive outputs",
			outputChanges: []OutputChange{
				{Name: "db_password", Actions: []string{"create"}, Sensitive: true},
				{Name: "public_url", Actions: []string{"create"}, Sensitive: false},
			},
			wantContains: []string{
				"db_password (sensitive)",
				"public_url",
			},
		},
		{
			name: "formats replace action from delete+create",
			outputChanges: []OutputChange{
				{Name: "replaced_output", Actions: []string{"delete", "create"}, Sensitive: false},
			},
			wantContains: []string{
				"replaced_output",
				"replace",
			},
		},
		{
			name: "formats no-op action for empty actions",
			outputChanges: []OutputChange{
				{Name: "unchanged", Actions: []string{}, Sensitive: false},
			},
			wantContains: []string{
				"unchanged",
				"no-op",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatter := NewPlanFormatter(false)
			formatter.tableWidth = 100

			analysis := &PlanAnalysis{
				OutputChanges:       tt.outputChanges,
				OutputAdditions:     countOutputAction(tt.outputChanges, "create"),
				OutputDeletions:     countOutputAction(tt.outputChanges, "delete"),
				OutputModifications: countOutputAction(tt.outputChanges, "update"),
			}

			var buf bytes.Buffer
			err := formatter.writeTableOutputChanges(&buf, analysis, newTestStyles())
			require.NoError(t, err)

			output := buf.String()
			for _, want := range tt.wantContains {
				assert.Contains(t, output, want)
			}
		})
	}
}

// countOutputAction counts output changes containing the given action.
func countOutputAction(changes []OutputChange, action string) int {
	count := 0
	for _, oc := range changes {
		if slices.Contains(oc.Actions, action) {
			count++
		}
	}
	return count
}

// newTestStyles returns unstyled CommonStyles for deterministic test output.
func newTestStyles() format.CommonStyles {
	return format.NewCommonStyles(false)
}
