package plan

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPlanAnalyzer(t *testing.T) {
	t.Run("creates analyzer with default critical resource types", func(t *testing.T) {
		analyzer := NewPlanAnalyzer()
		assert.NotNil(t, analyzer)
		assert.NotEmpty(t, analyzer.criticalResourceTypes)

		// Verify some expected critical types are present
		assert.Contains(t, analyzer.criticalResourceTypes, "aws_db_instance")
		assert.Contains(t, analyzer.criticalResourceTypes, "aws_s3_bucket")
		assert.Contains(t, analyzer.criticalResourceTypes, "aws_vpc")
	})
}

func TestNewPlanAnalyzerWithTypes(t *testing.T) {
	t.Run("creates analyzer with custom critical types", func(t *testing.T) {
		customTypes := []string{"custom_resource", "another_critical_resource"}
		analyzer := NewPlanAnalyzerWithTypes(customTypes)
		assert.NotNil(t, analyzer)
		assert.Equal(t, customTypes, analyzer.criticalResourceTypes)
	})
}

func TestPlanAnalyzer_Analyze(t *testing.T) {
	tests := []struct {
		name              string
		plan              *TerraformPlan
		wantTotalChanges  int
		wantAdditions     int
		wantModifications int
		wantDeletions     int
		wantReplacements  int
		wantHasChanges    bool
		wantExitCode      int
		validateResources func(t *testing.T, resources []AnalyzedResource)
	}{
		{
			name: "no changes - empty plan",
			plan: &TerraformPlan{
				FormatVersion:    "1.2",
				TerraformVersion: "1.14.3",
				ResourceChanges:  []ResourceChange{},
			},
			wantTotalChanges:  0,
			wantAdditions:     0,
			wantModifications: 0,
			wantDeletions:     0,
			wantReplacements:  0,
			wantHasChanges:    false,
			wantExitCode:      0,
		},
		{
			name: "no changes - all no-op",
			plan: &TerraformPlan{
				FormatVersion:    "1.2",
				TerraformVersion: "1.14.3",
				ResourceChanges: []ResourceChange{
					{
						Address:      "aws_instance.unchanged",
						Type:         "aws_instance",
						Name:         "unchanged",
						ProviderName: "registry.terraform.io/hashicorp/aws",
						Change: ChangeDetail{
							Actions: []string{"no-op"},
						},
					},
				},
			},
			wantTotalChanges:  0,
			wantAdditions:     0,
			wantModifications: 0,
			wantDeletions:     0,
			wantReplacements:  0,
			wantHasChanges:    false,
			wantExitCode:      0,
		},
		{
			name: "create action - low severity",
			plan: &TerraformPlan{
				FormatVersion:    "1.2",
				TerraformVersion: "1.14.3",
				ResourceChanges: []ResourceChange{
					{
						Address:      "aws_instance.new",
						Type:         "aws_instance",
						Name:         "new",
						ProviderName: "registry.terraform.io/hashicorp/aws",
						Change: ChangeDetail{
							Actions: []string{"create"},
						},
					},
				},
			},
			wantTotalChanges:  1,
			wantAdditions:     1,
			wantModifications: 0,
			wantDeletions:     0,
			wantReplacements:  0,
			wantHasChanges:    true,
			wantExitCode:      1,
			validateResources: func(t *testing.T, resources []AnalyzedResource) {
				t.Helper()
				require.Len(t, resources, 1)
				assert.Equal(t, "aws_instance.new", resources[0].Address)
				assert.Equal(t, "create", resources[0].ActionString)
				assert.Equal(t, SeverityLow, resources[0].Severity)
			},
		},
		{
			name: "update action - medium severity for non-critical resource",
			plan: &TerraformPlan{
				FormatVersion:    "1.2",
				TerraformVersion: "1.14.3",
				ResourceChanges: []ResourceChange{
					{
						Address:      "aws_instance.updated",
						Type:         "aws_instance",
						Name:         "updated",
						ProviderName: "registry.terraform.io/hashicorp/aws",
						Change: ChangeDetail{
							Actions: []string{"update"},
						},
					},
				},
			},
			wantTotalChanges:  1,
			wantAdditions:     0,
			wantModifications: 1,
			wantDeletions:     0,
			wantReplacements:  0,
			wantHasChanges:    true,
			wantExitCode:      1,
			validateResources: func(t *testing.T, resources []AnalyzedResource) {
				t.Helper()
				require.Len(t, resources, 1)
				assert.Equal(t, "update", resources[0].ActionString)
				assert.Equal(t, SeverityMedium, resources[0].Severity)
			},
		},
		{
			name: "update action - high severity for critical resource",
			plan: &TerraformPlan{
				FormatVersion:    "1.2",
				TerraformVersion: "1.14.3",
				ResourceChanges: []ResourceChange{
					{
						Address:      "aws_db_instance.database",
						Type:         "aws_db_instance",
						Name:         "database",
						ProviderName: "registry.terraform.io/hashicorp/aws",
						Change: ChangeDetail{
							Actions: []string{"update"},
						},
					},
				},
			},
			wantTotalChanges:  1,
			wantAdditions:     0,
			wantModifications: 1,
			wantDeletions:     0,
			wantReplacements:  0,
			wantHasChanges:    true,
			wantExitCode:      1,
			validateResources: func(t *testing.T, resources []AnalyzedResource) {
				t.Helper()
				require.Len(t, resources, 1)
				assert.Equal(t, "aws_db_instance.database", resources[0].Address)
				assert.Equal(t, "update", resources[0].ActionString)
				assert.Equal(t, SeverityHigh, resources[0].Severity)
			},
		},
		{
			name: "delete action - critical severity",
			plan: &TerraformPlan{
				FormatVersion:    "1.2",
				TerraformVersion: "1.14.3",
				ResourceChanges: []ResourceChange{
					{
						Address:      "aws_s3_bucket.data",
						Type:         "aws_s3_bucket",
						Name:         "data",
						ProviderName: "registry.terraform.io/hashicorp/aws",
						Change: ChangeDetail{
							Actions: []string{"delete"},
						},
					},
				},
			},
			wantTotalChanges:  1,
			wantAdditions:     0,
			wantModifications: 0,
			wantDeletions:     1,
			wantReplacements:  0,
			wantHasChanges:    true,
			wantExitCode:      2,
			validateResources: func(t *testing.T, resources []AnalyzedResource) {
				t.Helper()
				require.Len(t, resources, 1)
				assert.Equal(t, "delete", resources[0].ActionString)
				assert.Equal(t, SeverityCritical, resources[0].Severity)
			},
		},
		{
			name: "replace action - delete and create",
			plan: &TerraformPlan{
				FormatVersion:    "1.2",
				TerraformVersion: "1.14.3",
				ResourceChanges: []ResourceChange{
					{
						Address:      "aws_instance.replaced",
						Type:         "aws_instance",
						Name:         "replaced",
						ProviderName: "registry.terraform.io/hashicorp/aws",
						Change: ChangeDetail{
							Actions: []string{"delete", "create"},
						},
					},
				},
			},
			wantTotalChanges:  1,
			wantAdditions:     0,
			wantModifications: 0,
			wantDeletions:     0,
			wantReplacements:  1,
			wantHasChanges:    true,
			wantExitCode:      2,
			validateResources: func(t *testing.T, resources []AnalyzedResource) {
				t.Helper()
				require.Len(t, resources, 1)
				assert.Equal(t, "replace", resources[0].ActionString)
				assert.Equal(t, SeverityCritical, resources[0].Severity)
			},
		},
		{
			name: "mixed changes - multiple resources",
			plan: &TerraformPlan{
				FormatVersion:    "1.2",
				TerraformVersion: "1.14.3",
				ResourceChanges: []ResourceChange{
					{
						Address:      "aws_instance.new",
						Type:         "aws_instance",
						Name:         "new",
						ProviderName: "registry.terraform.io/hashicorp/aws",
						Change: ChangeDetail{
							Actions: []string{"create"},
						},
					},
					{
						Address:      "aws_security_group.updated",
						Type:         "aws_security_group",
						Name:         "updated",
						ProviderName: "registry.terraform.io/hashicorp/aws",
						Change: ChangeDetail{
							Actions: []string{"update"},
						},
					},
					{
						Address:      "aws_vpc.replaced",
						Type:         "aws_vpc",
						Name:         "replaced",
						ProviderName: "registry.terraform.io/hashicorp/aws",
						Change: ChangeDetail{
							Actions: []string{"delete", "create"},
						},
					},
					{
						Address:      "aws_subnet.deleted",
						Type:         "aws_subnet",
						Name:         "deleted",
						ProviderName: "registry.terraform.io/hashicorp/aws",
						Change: ChangeDetail{
							Actions: []string{"delete"},
						},
					},
				},
			},
			wantTotalChanges:  4,
			wantAdditions:     1,
			wantModifications: 1,
			wantDeletions:     1,
			wantReplacements:  1,
			wantHasChanges:    true,
			wantExitCode:      2,
			validateResources: func(t *testing.T, resources []AnalyzedResource) {
				t.Helper()
				require.Len(t, resources, 4)

				// Verify create
				assert.Equal(t, "create", resources[0].ActionString)
				assert.Equal(t, SeverityLow, resources[0].Severity)

				// Verify update (critical resource)
				assert.Equal(t, "update", resources[1].ActionString)
				assert.Equal(t, SeverityHigh, resources[1].Severity)

				// Verify replace
				assert.Equal(t, "replace", resources[2].ActionString)
				assert.Equal(t, SeverityCritical, resources[2].Severity)

				// Verify delete
				assert.Equal(t, "delete", resources[3].ActionString)
				assert.Equal(t, SeverityCritical, resources[3].Severity)
			},
		},
		{
			name: "data sources are skipped (not tracked)",
			plan: &TerraformPlan{
				FormatVersion:    "1.2",
				TerraformVersion: "1.14.3",
				ResourceChanges: []ResourceChange{
					{
						Address:      "data.aws_iam_policy_document.test",
						Mode:         "data",
						Type:         "aws_iam_policy_document",
						Name:         "test",
						ProviderName: "registry.terraform.io/hashicorp/aws",
						Change: ChangeDetail{
							Actions: []string{"read"},
						},
					},
				},
			},
			wantTotalChanges:  0,
			wantAdditions:     0,
			wantModifications: 0,
			wantDeletions:     0,
			wantReplacements:  0,
			wantHasChanges:    false,
			wantExitCode:      0,
			validateResources: func(t *testing.T, resources []AnalyzedResource) {
				t.Helper()
				require.Len(t, resources, 0, "data sources should not be tracked")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			analyzer := NewPlanAnalyzer()

			analysis := analyzer.Analyze(tt.plan)

			require.NotNil(t, analysis)
			assert.Equal(t, tt.plan.TerraformVersion, analysis.TerraformVersion)
			assert.Equal(t, tt.wantTotalChanges, analysis.TotalChanges)
			assert.Equal(t, tt.wantAdditions, analysis.Additions)
			assert.Equal(t, tt.wantModifications, analysis.Modifications)
			assert.Equal(t, tt.wantDeletions, analysis.Deletions)
			assert.Equal(t, tt.wantReplacements, analysis.Replacements)
			assert.Equal(t, tt.wantHasChanges, analysis.HasChanges)
			assert.Equal(t, tt.wantExitCode, analysis.ExitCode())

			if tt.validateResources != nil {
				tt.validateResources(t, analysis.ResourceChanges)
			}
		})
	}
}

func TestPlanAnalyzer_AnalyzeOutputChanges(t *testing.T) {
	tests := []struct {
		name                  string
		outputChanges         map[string]any
		wantCount             int
		wantAdditions         int
		wantDeletions         int
		wantModifications     int
		wantReplacements      int
		wantHasChanges        bool
		validateOutputChanges func(t *testing.T, changes []OutputChange)
	}{
		{
			name:          "nil output changes",
			outputChanges: nil,
			wantCount:     0,
		},
		{
			name:          "empty output changes",
			outputChanges: map[string]any{},
			wantCount:     0,
		},
		{
			name: "no-op outputs are skipped",
			outputChanges: map[string]any{
				"unchanged_output": map[string]any{
					"actions": []any{"no-op"},
				},
			},
			wantCount: 0,
		},
		{
			name: "create output",
			outputChanges: map[string]any{
				"api_endpoint": map[string]any{
					"actions":          []any{"create"},
					"before_sensitive": false,
					"after_sensitive":  false,
				},
			},
			wantCount:      1,
			wantAdditions:  1,
			wantHasChanges: true,
			validateOutputChanges: func(t *testing.T, changes []OutputChange) {
				t.Helper()
				require.Len(t, changes, 1)
				assert.Equal(t, "api_endpoint", changes[0].Name)
				assert.Equal(t, []string{"create"}, changes[0].Actions)
				assert.False(t, changes[0].Sensitive)
			},
		},
		{
			name: "delete output",
			outputChanges: map[string]any{
				"old_endpoint": map[string]any{
					"actions":          []any{"delete"},
					"before_sensitive": false,
					"after_sensitive":  false,
				},
			},
			wantCount:      1,
			wantDeletions:  1,
			wantHasChanges: true,
			validateOutputChanges: func(t *testing.T, changes []OutputChange) {
				t.Helper()
				require.Len(t, changes, 1)
				assert.Equal(t, "old_endpoint", changes[0].Name)
				assert.Equal(t, []string{"delete"}, changes[0].Actions)
			},
		},
		{
			name: "update output",
			outputChanges: map[string]any{
				"app_url": map[string]any{
					"actions":          []any{"update"},
					"before_sensitive": false,
					"after_sensitive":  false,
				},
			},
			wantCount:         1,
			wantModifications: 1,
			wantHasChanges:    true,
		},
		{
			name: "replace output (delete+create)",
			outputChanges: map[string]any{
				"replaced_output": map[string]any{
					"actions":          []any{"delete", "create"},
					"before_sensitive": false,
					"after_sensitive":  false,
				},
			},
			wantCount:        1,
			wantReplacements: 1,
			wantHasChanges:   true,
		},
		{
			name: "sensitive output",
			outputChanges: map[string]any{
				"db_password": map[string]any{
					"actions":          []any{"create"},
					"before_sensitive": false,
					"after_sensitive":  true,
				},
			},
			wantCount:      1,
			wantAdditions:  1,
			wantHasChanges: true,
			validateOutputChanges: func(t *testing.T, changes []OutputChange) {
				t.Helper()
				require.Len(t, changes, 1)
				assert.True(t, changes[0].Sensitive)
			},
		},
		{
			name: "mixed output changes",
			outputChanges: map[string]any{
				"new_output": map[string]any{
					"actions":          []any{"create"},
					"before_sensitive": false,
					"after_sensitive":  false,
				},
				"removed_output": map[string]any{
					"actions":          []any{"delete"},
					"before_sensitive": false,
					"after_sensitive":  false,
				},
				"changed_output": map[string]any{
					"actions":          []any{"update"},
					"before_sensitive": false,
					"after_sensitive":  false,
				},
				"stable_output": map[string]any{
					"actions": []any{"no-op"},
				},
			},
			wantCount:         3,
			wantAdditions:     1,
			wantDeletions:     1,
			wantModifications: 1,
			wantHasChanges:    true,
		},
		{
			name: "output changes only (no resource changes) sets HasChanges",
			outputChanges: map[string]any{
				"new_output": map[string]any{
					"actions":          []any{"create"},
					"before_sensitive": false,
					"after_sensitive":  false,
				},
			},
			wantCount:      1,
			wantAdditions:  1,
			wantHasChanges: true,
		},
		{
			name: "malformed output entry is skipped",
			outputChanges: map[string]any{
				"bad_entry": "not a map",
				"good_entry": map[string]any{
					"actions":          []any{"create"},
					"before_sensitive": false,
					"after_sensitive":  false,
				},
			},
			wantCount:      1,
			wantAdditions:  1,
			wantHasChanges: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			analyzer := NewPlanAnalyzer()
			plan := &TerraformPlan{
				TerraformVersion: "1.14.3",
				ResourceChanges:  []ResourceChange{},
				OutputChanges:    tt.outputChanges,
			}

			analysis := analyzer.Analyze(plan)

			require.NotNil(t, analysis)
			assert.Len(t, analysis.OutputChanges, tt.wantCount)
			assert.Equal(t, tt.wantAdditions, analysis.OutputAdditions)
			assert.Equal(t, tt.wantDeletions, analysis.OutputDeletions)
			assert.Equal(t, tt.wantModifications, analysis.OutputModifications)
			assert.Equal(t, tt.wantReplacements, analysis.OutputReplacements)
			assert.Equal(t, tt.wantHasChanges, analysis.HasChanges)
			// Output changes should NOT affect exit codes
			assert.Equal(t, 0, analysis.ExitCode())

			if tt.validateOutputChanges != nil {
				tt.validateOutputChanges(t, analysis.OutputChanges)
			}
		})
	}
}

func TestPlanAnalyzer_isCriticalResource(t *testing.T) {
	tests := []struct {
		name         string
		resourceType string
		want         bool
	}{
		{"aws database instance", "aws_db_instance", true},
		{"aws rds cluster", "aws_rds_cluster", true},
		{"aws s3 bucket", "aws_s3_bucket", true},
		{"aws vpc", "aws_vpc", true},
		{"aws security group", "aws_security_group", true},
		{"google sql database", "google_sql_database_instance", false}, // Not in AWS-only default list
		{"azure storage account", "azurerm_storage_account", false},    // Not in AWS-only default list
		{"non-critical resource", "aws_instance", false},
		{"non-critical resource 2", "aws_lambda_function", false},
	}

	analyzer := NewPlanAnalyzer()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := analyzer.isCriticalResource(tt.resourceType)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestPlanAnalyzer_determineSeverity(t *testing.T) {
	tests := []struct {
		name         string
		actions      []string
		resourceType string
		want         Severity
	}{
		{
			name:         "delete action - critical",
			actions:      []string{"delete"},
			resourceType: "aws_instance",
			want:         SeverityCritical,
		},
		{
			name:         "replace action - critical",
			actions:      []string{"delete", "create"},
			resourceType: "aws_instance",
			want:         SeverityCritical,
		},
		{
			name:         "update critical resource - high",
			actions:      []string{"update"},
			resourceType: "aws_db_instance",
			want:         SeverityHigh,
		},
		{
			name:         "update critical resource vpc - high",
			actions:      []string{"update"},
			resourceType: "aws_vpc",
			want:         SeverityHigh,
		},
		{
			name:         "update non-critical resource - medium",
			actions:      []string{"update"},
			resourceType: "aws_instance",
			want:         SeverityMedium,
		},
		{
			name:         "create action - low",
			actions:      []string{"create"},
			resourceType: "aws_instance",
			want:         SeverityLow,
		},
		{
			name:         "read action - low",
			actions:      []string{"read"},
			resourceType: "aws_iam_policy_document",
			want:         SeverityLow,
		},
		{
			name:         "no-op - low",
			actions:      []string{"no-op"},
			resourceType: "aws_instance",
			want:         SeverityLow,
		},
	}

	analyzer := NewPlanAnalyzer()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := analyzer.determineSeverity(tt.actions, tt.resourceType)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestPlanAnalysis_ExitCode(t *testing.T) {
	tests := []struct {
		name         string
		analysis     *PlanAnalysis
		wantExitCode int
	}{
		{
			name: "no changes",
			analysis: &PlanAnalysis{
				TotalChanges: 0,
			},
			wantExitCode: 0,
		},
		{
			name: "only additions",
			analysis: &PlanAnalysis{
				TotalChanges: 5,
				Additions:    5,
			},
			wantExitCode: 1,
		},
		{
			name: "only modifications",
			analysis: &PlanAnalysis{
				TotalChanges:  3,
				Modifications: 3,
			},
			wantExitCode: 1,
		},
		{
			name: "has deletions - critical",
			analysis: &PlanAnalysis{
				TotalChanges: 1,
				Deletions:    1,
			},
			wantExitCode: 2,
		},
		{
			name: "has replacements - critical",
			analysis: &PlanAnalysis{
				TotalChanges: 1,
				Replacements: 1,
			},
			wantExitCode: 2,
		},
		{
			name: "mixed with deletions - critical takes precedence",
			analysis: &PlanAnalysis{
				TotalChanges:  10,
				Additions:     5,
				Modifications: 3,
				Deletions:     2,
			},
			wantExitCode: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.analysis.ExitCode()
			assert.Equal(t, tt.wantExitCode, got)
		})
	}
}
