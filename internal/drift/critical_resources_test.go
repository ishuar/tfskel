package drift

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultCriticalResources(t *testing.T) {
	resources := DefaultCriticalResources()

	// Verify we have a reasonable number of resources
	assert.Greater(t, len(resources), 0, "should have at least one default critical resource")

	// Verify key AWS resources are included
	expectedResources := []string{
		// Database
		"aws_rds_cluster",
		"aws_dynamodb_table",
		// Storage
		"aws_s3_bucket",
		"aws_efs_file_system",
		// Network
		"aws_vpc",
		"aws_subnet",
		"aws_security_group",
		// Security
		"aws_iam_role",
		"aws_kms_key",
		// WAF
		"aws_wafv2_web_acl",
	}

	for _, expected := range expectedResources {
		assert.Contains(t, resources, expected, "should contain %s", expected)
	}

	// Verify no duplicates in defaults
	seen := make(map[string]bool)
	for _, resource := range resources {
		assert.False(t, seen[resource], "should not have duplicate: %s", resource)
		seen[resource] = true
	}
}

func TestMergeCriticalResources(t *testing.T) {
	tests := []struct {
		name        string
		defaults    []string
		userDefined []string
		want        []string
	}{
		{
			name:        "empty user-defined list",
			defaults:    []string{"aws_vpc", "aws_subnet"},
			userDefined: []string{},
			want:        []string{"aws_vpc", "aws_subnet"},
		},
		{
			name:        "new user-defined resources",
			defaults:    []string{"aws_vpc", "aws_subnet"},
			userDefined: []string{"aws_iam_role", "aws_lambda_function"},
			want:        []string{"aws_vpc", "aws_subnet", "aws_iam_role", "aws_lambda_function"},
		},
		{
			name:        "overlapping resources",
			defaults:    []string{"aws_vpc", "aws_subnet"},
			userDefined: []string{"aws_subnet", "aws_iam_role"},
			want:        []string{"aws_vpc", "aws_subnet", "aws_iam_role"},
		},
		{
			name:        "empty default list",
			defaults:    []string{},
			userDefined: []string{"aws_iam_role", "aws_lambda_function"},
			want:        []string{"aws_iam_role", "aws_lambda_function"},
		},
		{
			name:        "both empty",
			defaults:    []string{},
			userDefined: []string{},
			want:        []string{},
		},
		{
			name:        "user-defined with empty strings",
			defaults:    []string{"aws_vpc"},
			userDefined: []string{"", "aws_iam_role", ""},
			want:        []string{"aws_vpc", "aws_iam_role"},
		},
		{
			name:        "duplicate user-defined resources",
			defaults:    []string{"aws_vpc"},
			userDefined: []string{"aws_iam_role", "aws_iam_role", "aws_lambda_function"},
			want:        []string{"aws_vpc", "aws_iam_role", "aws_lambda_function"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MergeCriticalResources(tt.defaults, tt.userDefined)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestMergeCriticalResources_OrderPreserved(t *testing.T) {
	defaults := []string{"aws_vpc", "aws_subnet", "aws_s3_bucket"}
	userDefined := []string{"aws_iam_role", "aws_lambda_function"}

	result := MergeCriticalResources(defaults, userDefined)

	// Verify defaults come first
	assert.Equal(t, "aws_vpc", result[0])
	assert.Equal(t, "aws_subnet", result[1])
	assert.Equal(t, "aws_s3_bucket", result[2])

	// Verify user-defined come after
	assert.Equal(t, "aws_iam_role", result[3])
	assert.Equal(t, "aws_lambda_function", result[4])
}

func TestMergeCriticalResources_NoDuplicates(t *testing.T) {
	defaults := []string{"aws_vpc", "aws_subnet", "aws_rds_cluster"}
	userDefined := []string{"aws_rds_cluster", "aws_iam_role", "aws_vpc"}

	result := MergeCriticalResources(defaults, userDefined)

	// Count occurrences
	counts := make(map[string]int)
	for _, resource := range result {
		counts[resource]++
	}

	// Verify no duplicates
	for resource, count := range counts {
		assert.Equal(t, 1, count, "resource %s should appear only once", resource)
	}

	// Verify all unique resources are present
	assert.Len(t, result, 4, "should have 4 unique resources")
	assert.Contains(t, result, "aws_vpc")
	assert.Contains(t, result, "aws_subnet")
	assert.Contains(t, result, "aws_rds_cluster")
	assert.Contains(t, result, "aws_iam_role")
}
