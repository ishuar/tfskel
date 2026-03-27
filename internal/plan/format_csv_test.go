package plan

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlanFormatter_formatCSV_SortsBySeverity(t *testing.T) {
	formatter := NewPlanFormatter(false)

	analysis := &PlanAnalysis{
		TerraformVersion: "1.14.3",
		TotalChanges:     4,
		ResourceChanges: []AnalyzedResource{
			{Address: "aws_instance.low", Name: "low", Type: "aws_instance", ActionString: "create", Severity: SeverityLow},
			{Address: "aws_s3_bucket.high", Name: "high", Type: "aws_s3_bucket", ActionString: "update", Severity: SeverityHigh},
			{Address: "aws_lambda.medium", Name: "medium", Type: "aws_lambda", ActionString: "update", Severity: SeverityMedium},
			{Address: "aws_rds.critical", Name: "critical", Type: "aws_rds_cluster", ActionString: "delete", Severity: SeverityCritical},
		},
	}

	var buf bytes.Buffer
	err := formatter.formatCSV(analysis, &buf)
	require.NoError(t, err)

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")

	// Find the header line and data lines
	var dataStartIdx int
	for i, line := range lines {
		if strings.HasPrefix(line, "Address,") {
			dataStartIdx = i + 1
			break
		}
	}

	// Verify data is sorted by severity
	dataLines := lines[dataStartIdx:]
	assert.Len(t, dataLines, 4)

	// CSV format: Address,Type,Name,Provider,Action,Severity
	// Check that severities are in order: critical, high, medium, low
	assert.Contains(t, dataLines[0], "critical", "First row should be critical")
	assert.Contains(t, dataLines[1], "high", "Second row should be high")
	assert.Contains(t, dataLines[2], "medium", "Third row should be medium")
	assert.Contains(t, dataLines[3], "low", "Fourth row should be low")
}

func TestPlanFormatter_formatCSV_OutputChanges(t *testing.T) {
	tests := []struct {
		name          string
		outputChanges []OutputChange
		wantContains  []string
	}{
		{
			name: "renders output changes section",
			outputChanges: []OutputChange{
				{Name: "vpc_id", Actions: []string{"create"}, Sensitive: false},
				{Name: "db_endpoint", Actions: []string{"delete"}, Sensitive: true},
				{Name: "api_url", Actions: []string{"update"}, Sensitive: false},
			},
			wantContains: []string{
				"Output,Action,Sensitive",
				"vpc_id,create,false",
				"db_endpoint,delete,true",
				"api_url,update,false",
				"# Output Changes: 3 total",
			},
		},
		{
			name: "formats replace action from delete+create",
			outputChanges: []OutputChange{
				{Name: "replaced_out", Actions: []string{"delete", "create"}, Sensitive: false},
			},
			wantContains: []string{
				"replaced_out,replace,false",
			},
		},
		{
			name:          "omits output section when empty",
			outputChanges: nil,
			wantContains: []string{
				"# Output Changes: 0 (added: 0, deleted: 0, modified: 0, replaced: 0)",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatter := NewPlanFormatter(false)
			analysis := &PlanAnalysis{
				TerraformVersion: "1.14.3",
				OutputChanges:    tt.outputChanges,
			}
			// Compute counts matching analyzer logic
			for _, oc := range tt.outputChanges {
				action := formatActions(oc.Actions)
				switch action {
				case "create":
					analysis.OutputAdditions++
				case "delete":
					analysis.OutputDeletions++
				case "update":
					analysis.OutputModifications++
				}
			}

			var buf bytes.Buffer
			err := formatter.formatCSV(analysis, &buf)
			require.NoError(t, err)

			output := buf.String()
			for _, want := range tt.wantContains {
				assert.Contains(t, output, want)
			}

			// Verify no output CSV rows when empty
			if len(tt.outputChanges) == 0 {
				assert.NotContains(t, output, "Output,Action,Sensitive")
			}
		})
	}
}
