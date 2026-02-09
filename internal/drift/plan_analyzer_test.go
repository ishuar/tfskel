package drift

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
			name: "data sources with read action",
			plan: &TerraformPlan{
				FormatVersion:    "1.2",
				TerraformVersion: "1.14.3",
				ResourceChanges: []ResourceChange{
					{
						Address:      "data.aws_iam_policy_document.test",
						Type:         "aws_iam_policy_document",
						Name:         "test",
						ProviderName: "registry.terraform.io/hashicorp/aws",
						Change: ChangeDetail{
							Actions: []string{"read"},
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
				require.Len(t, resources, 1)
				assert.Equal(t, "read", resources[0].ActionString)
				// Read actions treated similar to create (low severity)
				assert.Equal(t, SeverityLow, resources[0].Severity)
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
		{"google sql database", "google_sql_database_instance", true},
		{"azure storage account", "azurerm_storage_account", true},
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

func TestPlanAnalyzer_formatActions(t *testing.T) {
	tests := []struct {
		name    string
		actions []string
		want    string
	}{
		{
			name:    "empty actions",
			actions: []string{},
			want:    "no-op",
		},
		{
			name:    "single action - create",
			actions: []string{"create"},
			want:    "create",
		},
		{
			name:    "single action - update",
			actions: []string{"update"},
			want:    "update",
		},
		{
			name:    "single action - delete",
			actions: []string{"delete"},
			want:    "delete",
		},
		{
			name:    "replace actions",
			actions: []string{"delete", "create"},
			want:    "replace",
		},
		{
			name:    "replace actions reversed",
			actions: []string{"create", "delete"},
			want:    "replace",
		},
	}

	analyzer := NewPlanAnalyzer()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := analyzer.formatActions(tt.actions)
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
			actions: []string{"create", "delete"},
			action:  "create",
			want:    true,
		},
		{
			name:    "action not present",
			actions: []string{"create", "update"},
			action:  "delete",
			want:    false,
		},
		{
			name:    "empty actions",
			actions: []string{},
			action:  "create",
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := containsAction(tt.actions, tt.action)
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
