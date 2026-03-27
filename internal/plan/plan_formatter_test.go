package plan

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/ishuar/tfskel/internal/format"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPlanFormatter(t *testing.T) {
	t.Run("creates formatter with color", func(t *testing.T) {
		formatter := NewPlanFormatter(true)
		assert.NotNil(t, formatter)
		assert.True(t, formatter.useColor)
	})

	t.Run("creates formatter without color", func(t *testing.T) {
		formatter := NewPlanFormatter(false)
		assert.NotNil(t, formatter)
		assert.False(t, formatter.useColor)
	})
}

func TestPlanFormatter_Format(t *testing.T) {
	analysis := &PlanAnalysis{
		TerraformVersion: "1.14.3",
		TotalChanges:     4,
		Additions:        1,
		Modifications:    1,
		Deletions:        1,
		Replacements:     1,
		HasChanges:       true,
		ResourceChanges: []AnalyzedResource{
			{
				Address:      "aws_instance.new",
				Type:         "aws_instance",
				Name:         "new",
				Provider:     "registry.terraform.io/hashicorp/aws",
				Actions:      []string{"create"},
				ActionString: "create",
				Severity:     SeverityLow,
			},
			{
				Address:      "aws_security_group.updated",
				Type:         "aws_security_group",
				Name:         "updated",
				Provider:     "registry.terraform.io/hashicorp/aws",
				Actions:      []string{"update"},
				ActionString: "update",
				Severity:     SeverityHigh,
			},
			{
				Address:      "aws_vpc.replaced",
				Type:         "aws_vpc",
				Name:         "replaced",
				Provider:     "registry.terraform.io/hashicorp/aws",
				Actions:      []string{"delete", "create"},
				ActionString: "replace",
				Severity:     SeverityCritical,
			},
			{
				Address:      "aws_s3_bucket.deleted",
				Type:         "aws_s3_bucket",
				Name:         "deleted",
				Provider:     "registry.terraform.io/hashicorp/aws",
				Actions:      []string{"delete"},
				ActionString: "delete",
				Severity:     SeverityCritical,
			},
		},
	}

	tests := []struct {
		name        string
		format      format.OutputFormat
		useColor    bool
		validate    func(t *testing.T, output string)
		wantErr     bool
		errContains string
	}{
		{
			name:     "JSON format",
			format:   format.FormatJSON,
			useColor: false,
			validate: func(t *testing.T, output string) {
				t.Helper()
				// Verify it's valid JSON
				var result map[string]any
				err := json.Unmarshal([]byte(output), &result)
				require.NoError(t, err)

				// Verify key fields
				assert.Equal(t, "1.14.3", result["terraform_version"])
				assert.Equal(t, float64(4), result["total_changes"])
				assert.Equal(t, float64(1), result["additions"])
				assert.Equal(t, float64(1), result["modifications"])
				assert.Equal(t, float64(1), result["deletions"])
				assert.Equal(t, float64(1), result["replacements"])
				if hasChanges, ok := result["has_changes"].(bool); ok {
					assert.True(t, hasChanges)
				}

				// Verify resource_changes array
				if resources, ok := result["resource_changes"].([]any); ok {
					assert.Len(t, resources, 4)
				}
			},
		},
		{
			name:     "CSV format",
			format:   format.FormatCSV,
			useColor: false,
			validate: func(t *testing.T, output string) {
				t.Helper()
				lines := strings.Split(strings.TrimSpace(output), "\n")

				// Check metadata comments
				assert.Contains(t, lines[0], "# Terraform Plan Analysis")
				assert.Contains(t, output, "# Terraform Version: 1.14.3")
				assert.Contains(t, output, "# Total Resource Changes: 4")

				// Find the CSV header line (after metadata comments)
				headerLine := -1
				for i, line := range lines {
					if strings.Contains(line, "Address,Type,Name,Provider,Action,Severity") {
						headerLine = i
						break
					}
				}
				assert.NotEqual(t, -1, headerLine, "CSV header should be present")

				// Check data rows (after header)
				// Note: Rows are now sorted by severity (critical, high, low)
				dataLines := lines[headerLine+1:]
				assert.Len(t, dataLines, 4) // 4 resources

				// First two rows should be critical (vpc.replaced and s3_bucket.deleted)
				assert.Contains(t, dataLines[0], "critical")
				assert.True(t,
					strings.Contains(dataLines[0], "aws_vpc.replaced") || strings.Contains(dataLines[0], "aws_s3_bucket.deleted"),
					"First row should be a critical resource")

				assert.Contains(t, dataLines[1], "critical")
				assert.True(t,
					strings.Contains(dataLines[1], "aws_vpc.replaced") || strings.Contains(dataLines[1], "aws_s3_bucket.deleted"),
					"Second row should be a critical resource")

				// Third row should be high (security_group.updated)
				assert.Contains(t, dataLines[2], "aws_security_group.updated")
				assert.Contains(t, dataLines[2], "update")
				assert.Contains(t, dataLines[2], "high")

				// Fourth row should be low (instance.new)
				assert.Contains(t, dataLines[3], "aws_instance.new")
				assert.Contains(t, dataLines[3], "create")
				assert.Contains(t, dataLines[3], "low")
			},
		},
		{
			name:     "table format without color",
			format:   format.FormatTable,
			useColor: false,
			validate: func(t *testing.T, output string) {
				t.Helper()
				// Check for key sections
				assert.Contains(t, output, "Terraform Plan Analysis")
				assert.Contains(t, output, "Terraform Version: 1.14.3")
				assert.Contains(t, output, "Summary")
				// New format uses lipgloss table with different structure
				assert.Contains(t, output, "Total Resource Changes")
				assert.Contains(t, output, "4") // total changes count
				assert.Contains(t, output, "Additions")
				assert.Contains(t, output, "Modifications")
				assert.Contains(t, output, "Deletions")
				assert.Contains(t, output, "Replacements")
				assert.Contains(t, output, "Resource Changes (detailed)")
				// New headers: Resource, Type, Action, Severity
				assert.Contains(t, output, "Resource")
				assert.Contains(t, output, "Type")
				assert.Contains(t, output, "Action")
				assert.Contains(t, output, "Severity")

				// Check for resources (just names, not full addresses)
				assert.Contains(t, output, "new")
				assert.Contains(t, output, "aws_instance")
				assert.Contains(t, output, "updated")
				assert.Contains(t, output, "aws_security_group")
				assert.Contains(t, output, "replaced")
				assert.Contains(t, output, "aws_vpc")
				assert.Contains(t, output, "deleted")
				assert.Contains(t, output, "aws_s3_bucket")

				// Check actions and severities
				assert.Contains(t, output, "create")
				assert.Contains(t, output, "update")
				assert.Contains(t, output, "replace")
				assert.Contains(t, output, "delete")
				assert.Contains(t, output, "low")
				assert.Contains(t, output, "high")
				assert.Contains(t, output, "critical")
			},
		},
		{
			name:     "table format with color",
			format:   format.FormatTable,
			useColor: true,
			validate: func(t *testing.T, output string) {
				t.Helper()
				// Check basic structure
				assert.Contains(t, output, "Terraform Plan Analysis")
				assert.Contains(t, output, "Resource Changes (detailed)")
				// Lipgloss renders colors but format may differ in tests
				// Just verify the content is present
				assert.Contains(t, output, "create")
				assert.Contains(t, output, "delete")
				assert.Contains(t, output, "critical")
			},
		},
		{
			name:        "unsupported format",
			format:      format.OutputFormat("invalid"),
			useColor:    false,
			wantErr:     true,
			errContains: "unsupported format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatter := NewPlanFormatter(tt.useColor)
			var buf bytes.Buffer

			err := formatter.Format(analysis, tt.format, &buf)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				require.NoError(t, err)
				output := buf.String()
				if tt.validate != nil {
					tt.validate(t, output)
				}
			}
		})
	}
}

func TestPlanFormatter_FormatEmptyAnalysis(t *testing.T) {
	emptyAnalysis := &PlanAnalysis{
		TerraformVersion: "1.14.3",
		TotalChanges:     0,
		ResourceChanges:  []AnalyzedResource{},
		HasChanges:       false,
	}

	tests := []struct {
		name     string
		format   format.OutputFormat
		validate func(t *testing.T, output string)
	}{
		{
			name:   "empty JSON",
			format: format.FormatJSON,
			validate: func(t *testing.T, output string) {
				t.Helper()
				var result map[string]any
				err := json.Unmarshal([]byte(output), &result)
				require.NoError(t, err)
				assert.Equal(t, float64(0), result["total_changes"])
				if hasChanges, ok := result["has_changes"].(bool); ok {
					assert.False(t, hasChanges)
				}
			},
		},
		{
			name:   "empty CSV",
			format: format.FormatCSV,
			validate: func(t *testing.T, output string) {
				t.Helper()
				lines := strings.Split(strings.TrimSpace(output), "\n")
				// Should have metadata comments + header, but no data rows
				assert.Contains(t, output, "# Terraform Plan Analysis")
				assert.Contains(t, output, "# Total Resource Changes: 0")
				assert.Contains(t, output, "Address,Type,Name,Provider,Action,Severity")
				// Verify no data rows after header
				headerFound := false
				for i, line := range lines {
					if strings.Contains(line, "Address,Type,Name,Provider,Action,Severity") {
						headerFound = true
						// Should be the last line (no data after it)
						assert.Equal(t, len(lines)-1, i, "CSV header should be the last line when empty")
						break
					}
				}
				assert.True(t, headerFound, "CSV header should be present")
			},
		},
		{
			name:   "empty table",
			format: format.FormatTable,
			validate: func(t *testing.T, output string) {
				t.Helper()
				assert.Contains(t, output, "Terraform Plan Analysis")
				assert.Contains(t, output, "Total Resource Changes")
				assert.Contains(t, output, "0")
				// Should not contain resource changes table when empty
				assert.NotContains(t, output, "Resource Changes (detailed)")
				assert.NotContains(t, output, "Showing")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatter := NewPlanFormatter(false)
			var buf bytes.Buffer

			err := formatter.Format(emptyAnalysis, tt.format, &buf)

			require.NoError(t, err)
			if tt.validate != nil {
				tt.validate(t, buf.String())
			}
		})
	}
}

func TestOutputFormat_Constants(t *testing.T) {
	// Verify output format constants are correctly defined
	assert.Equal(t, format.FormatJSON, format.OutputFormat("json"))
	assert.Equal(t, format.FormatCSV, format.OutputFormat("csv"))
	assert.Equal(t, format.FormatTable, format.OutputFormat("table"))
}
