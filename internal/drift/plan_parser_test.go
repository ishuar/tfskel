package drift

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParsePlanFile(t *testing.T) {
	tests := []struct {
		name        string
		setupFile   func(t *testing.T) string
		wantErr     bool
		errContains string
		validate    func(t *testing.T, plan *TerraformPlan)
	}{
		{
			name: "valid plan file with resource changes",
			setupFile: func(t *testing.T) string {
				content := `{
  "format_version": "1.2",
  "terraform_version": "1.14.3",
  "resource_changes": [
    {
      "address": "aws_instance.example",
      "mode": "managed",
      "type": "aws_instance",
      "name": "example",
      "provider_name": "registry.terraform.io/hashicorp/aws",
      "change": {
        "actions": ["create"],
        "before": null,
        "after": {
          "ami": "ami-12345",
          "instance_type": "t2.micro"
        },
        "after_unknown": {
          "id": true
        },
        "before_sensitive": false,
        "after_sensitive": {
          "ami": false,
          "instance_type": false
        }
      }
    }
  ]
}`
				return createTempFile(t, content)
			},
			wantErr: false,
			validate: func(t *testing.T, plan *TerraformPlan) {
				assert.Equal(t, "1.2", plan.FormatVersion)
				assert.Equal(t, "1.14.3", plan.TerraformVersion)
				assert.Len(t, plan.ResourceChanges, 1)
				assert.Equal(t, "aws_instance.example", plan.ResourceChanges[0].Address)
				assert.Equal(t, "aws_instance", plan.ResourceChanges[0].Type)
				assert.Equal(t, []string{"create"}, plan.ResourceChanges[0].Change.Actions)
			},
		},
		{
			name: "plan with boolean sensitive fields",
			setupFile: func(t *testing.T) string {
				content := `{
  "format_version": "1.2",
  "terraform_version": "1.14.3",
  "resource_changes": [
    {
      "address": "data.aws_iam_policy_document.test",
      "mode": "data",
      "type": "aws_iam_policy_document",
      "name": "test",
      "provider_name": "registry.terraform.io/hashicorp/aws",
      "change": {
        "actions": ["read"],
        "before": null,
        "after": {
          "statement": []
        },
        "after_unknown": {},
        "before_sensitive": false,
        "after_sensitive": false
      }
    }
  ]
}`
				return createTempFile(t, content)
			},
			wantErr: false,
			validate: func(t *testing.T, plan *TerraformPlan) {
				assert.Equal(t, "1.2", plan.FormatVersion)
				assert.Len(t, plan.ResourceChanges, 1)
				// Verify sensitive fields are parsed (as RawMessage)
				assert.NotNil(t, plan.ResourceChanges[0].Change.BeforeSensitive)
				assert.NotNil(t, plan.ResourceChanges[0].Change.AfterSensitive)
			},
		},
		{
			name: "plan with nested sensitive fields",
			setupFile: func(t *testing.T) string {
				content := `{
  "format_version": "1.2",
  "terraform_version": "1.14.3",
  "resource_changes": [
    {
      "address": "aws_db_instance.example",
      "mode": "managed",
      "type": "aws_db_instance",
      "name": "example",
      "provider_name": "registry.terraform.io/hashicorp/aws",
      "change": {
        "actions": ["create"],
        "before": null,
        "after": {
          "password": "secret"
        },
        "after_unknown": {},
        "before_sensitive": {},
        "after_sensitive": {
          "password": true
        }
      }
    }
  ]
}`
				return createTempFile(t, content)
			},
			wantErr: false,
			validate: func(t *testing.T, plan *TerraformPlan) {
				assert.Len(t, plan.ResourceChanges, 1)
				assert.Equal(t, "aws_db_instance.example", plan.ResourceChanges[0].Address)
			},
		},
		{
			name: "plan with multiple action types",
			setupFile: func(t *testing.T) string {
				content := `{
  "format_version": "1.2",
  "terraform_version": "1.14.3",
  "resource_changes": [
    {
      "address": "aws_instance.replace",
      "mode": "managed",
      "type": "aws_instance",
      "name": "replace",
      "provider_name": "registry.terraform.io/hashicorp/aws",
      "change": {
        "actions": ["delete", "create"],
        "before": {"ami": "ami-old"},
        "after": {"ami": "ami-new"},
        "after_unknown": {},
        "before_sensitive": false,
        "after_sensitive": false
      }
    },
    {
      "address": "aws_security_group.update",
      "mode": "managed",
      "type": "aws_security_group",
      "name": "update",
      "provider_name": "registry.terraform.io/hashicorp/aws",
      "change": {
        "actions": ["update"],
        "before": {"ingress": []},
        "after": {"ingress": [{"port": 80}]},
        "after_unknown": {},
        "before_sensitive": false,
        "after_sensitive": false
      }
    },
    {
      "address": "aws_s3_bucket.delete",
      "mode": "managed",
      "type": "aws_s3_bucket",
      "name": "delete",
      "provider_name": "registry.terraform.io/hashicorp/aws",
      "change": {
        "actions": ["delete"],
        "before": {"bucket": "my-bucket"},
        "after": null,
        "after_unknown": {},
        "before_sensitive": false,
        "after_sensitive": false
      }
    }
  ]
}`
				return createTempFile(t, content)
			},
			wantErr: false,
			validate: func(t *testing.T, plan *TerraformPlan) {
				assert.Len(t, plan.ResourceChanges, 3)
				// Replace action
				assert.Equal(t, []string{"delete", "create"}, plan.ResourceChanges[0].Change.Actions)
				// Update action
				assert.Equal(t, []string{"update"}, plan.ResourceChanges[1].Change.Actions)
				// Delete action
				assert.Equal(t, []string{"delete"}, plan.ResourceChanges[2].Change.Actions)
			},
		},
		{
			name: "non-existent file",
			setupFile: func(t *testing.T) string {
				return "/path/to/nonexistent/file.json"
			},
			wantErr:     true,
			errContains: "failed to read plan file",
		},
		{
			name: "invalid JSON",
			setupFile: func(t *testing.T) string {
				content := `{invalid json`
				return createTempFile(t, content)
			},
			wantErr:     true,
			errContains: "invalid plan file format",
		},
		{
			name: "binary plan file",
			setupFile: func(t *testing.T) string {
				// Create a file that starts with binary data (not '{')
				content := "\x00\x01\x02\x03binary data"
				return createTempFile(t, content)
			},
			wantErr:     true,
			errContains: "plan file is in binary format",
		},
		{
			name: "JSON but not a terraform plan",
			setupFile: func(t *testing.T) string {
				content := `{"some": "json", "but": "not terraform plan"}`
				return createTempFile(t, content)
			},
			wantErr:     true,
			errContains: "invalid plan file: missing format_version",
		},
		{
			name: "empty plan file",
			setupFile: func(t *testing.T) string {
				content := `{
  "format_version": "1.2",
  "terraform_version": "1.14.3",
  "resource_changes": []
}`
				return createTempFile(t, content)
			},
			wantErr: false,
			validate: func(t *testing.T, plan *TerraformPlan) {
				assert.Equal(t, "1.2", plan.FormatVersion)
				assert.Equal(t, "1.14.3", plan.TerraformVersion)
				assert.Empty(t, plan.ResourceChanges)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filename := tt.setupFile(t)

			plan, err := ParsePlanFile(filename)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				assert.Nil(t, plan)
			} else {
				require.NoError(t, err)
				require.NotNil(t, plan)
				if tt.validate != nil {
					tt.validate(t, plan)
				}
			}
		})
	}
}

// createTempFile creates a temporary file with the given content for testing
func createTempFile(t *testing.T, content string) string {
	t.Helper()
	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, "plan.json")
	err := os.WriteFile(filename, []byte(content), 0644)
	require.NoError(t, err)
	return filename
}
