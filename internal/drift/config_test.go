package drift

import (
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestLoadDriftConfig(t *testing.T) {
	tests := []struct {
		name             string
		viperSetup       func(*viper.Viper)
		wantResources    []string
		wantResourcesLen int
		wantTopNCount    int
	}{
		{
			name: "config with critical resources",
			viperSetup: func(v *viper.Viper) {
				v.Set("critical_resources", []string{"aws_iam_role", "aws_lambda_function"})
			},
			wantResources:    []string{"aws_iam_role", "aws_lambda_function"},
			wantResourcesLen: 2,
			wantTopNCount:    10, // default
		},
		{
			name: "config without critical resources",
			viperSetup: func(_ *viper.Viper) {
				// Don't set any critical_resources
			},
			wantResources:    nil,
			wantResourcesLen: 0,
			wantTopNCount:    10, // default
		},
		{
			name: "config with empty critical resources",
			viperSetup: func(v *viper.Viper) {
				v.Set("critical_resources", []string{})
			},
			wantResources:    []string{},
			wantResourcesLen: 0,
			wantTopNCount:    10, // default
		},
		{
			name: "config with custom top_n_count",
			viperSetup: func(v *viper.Viper) {
				v.Set("top_n_count", 20)
			},
			wantResources:    nil,
			wantResourcesLen: 0,
			wantTopNCount:    20,
		},
		{
			name: "config with zero top_n_count uses default",
			viperSetup: func(v *viper.Viper) {
				v.Set("top_n_count", 0)
			},
			wantResources:    nil,
			wantResourcesLen: 0,
			wantTopNCount:    10, // default, ignores 0
		},
		{
			name: "config with negative top_n_count uses default",
			viperSetup: func(v *viper.Viper) {
				v.Set("top_n_count", -5)
			},
			wantResources:    nil,
			wantResourcesLen: 0,
			wantTopNCount:    10, // default, ignores negative
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := viper.New()
			tt.viperSetup(v)

			cfg := LoadDriftConfig(v)

			assert.NotNil(t, cfg)
			assert.Len(t, cfg.CriticalResources, tt.wantResourcesLen)
			if tt.wantResources != nil {
				assert.Equal(t, tt.wantResources, cfg.CriticalResources)
			}
			assert.Equal(t, tt.wantTopNCount, cfg.TopNCount)
		})
	}
}

func TestNewPlanAnalyzerWithConfig(t *testing.T) {
	tests := []struct {
		name                  string
		viperSetup            func(*viper.Viper)
		expectedToContain     []string
		shouldContainDefaults bool
	}{
		{
			name: "with user-defined critical resources",
			viperSetup: func(v *viper.Viper) {
				v.Set("critical_resources", []string{"aws_iam_role", "aws_lambda_function"})
			},
			expectedToContain: []string{
				"aws_iam_role",
				"aws_lambda_function",
				"aws_rds_cluster", // Default
				"aws_vpc",         // Default
			},
			shouldContainDefaults: true,
		},
		{
			name: "without user-defined critical resources",
			viperSetup: func(_ *viper.Viper) {
				// Don't set any critical_resources
			},
			expectedToContain: []string{
				"aws_rds_cluster", // Default
				"aws_vpc",         // Default
				"aws_s3_bucket",   // Default
			},
			shouldContainDefaults: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := viper.New()
			tt.viperSetup(v)

			analyzer := NewPlanAnalyzerWithConfig(v)

			assert.NotNil(t, analyzer)
			assert.NotNil(t, analyzer.criticalResourceTypes)

			// Verify expected resources are present
			for _, resource := range tt.expectedToContain {
				assert.Contains(t, analyzer.criticalResourceTypes, resource,
					"should contain %s", resource)
			}

			// Verify defaults are included when expected
			if tt.shouldContainDefaults {
				assert.Greater(t, len(analyzer.criticalResourceTypes), 10,
					"should contain default critical resources")
			}
		})
	}
}

func TestNewPlanAnalyzerWithConfig_Integration(t *testing.T) {
	// This test verifies the full integration: config loading + merging + analyzer creation
	v := viper.New()
	v.Set("critical_resources", []string{"aws_iam_role", "aws_lambda_function", "aws_rds_cluster"})

	analyzer := NewPlanAnalyzerWithConfig(v)

	// Verify analyzer is functional
	assert.NotNil(t, analyzer)

	// Verify it has both defaults and user-defined resources
	assert.Contains(t, analyzer.criticalResourceTypes, "aws_iam_role")        // User-defined
	assert.Contains(t, analyzer.criticalResourceTypes, "aws_lambda_function") // User-defined
	assert.Contains(t, analyzer.criticalResourceTypes, "aws_rds_cluster")     // Both default and user-defined
	assert.Contains(t, analyzer.criticalResourceTypes, "aws_vpc")             // Default
	assert.Contains(t, analyzer.criticalResourceTypes, "aws_s3_bucket")       // Default

	// Verify no duplicates
	seen := make(map[string]bool)
	for _, resource := range analyzer.criticalResourceTypes {
		assert.False(t, seen[resource], "should not have duplicate: %s", resource)
		seen[resource] = true
	}
}

func TestNewPlanAnalyzer_BackwardCompatibility(t *testing.T) {
	// Verify the original NewPlanAnalyzer() still works without config
	analyzer := NewPlanAnalyzer()

	assert.NotNil(t, analyzer)
	assert.NotNil(t, analyzer.criticalResourceTypes)
	assert.Greater(t, len(analyzer.criticalResourceTypes), 0)

	// Verify it contains default resources
	assert.Contains(t, analyzer.criticalResourceTypes, "aws_rds_cluster")
	assert.Contains(t, analyzer.criticalResourceTypes, "aws_vpc")
	assert.Contains(t, analyzer.criticalResourceTypes, "aws_s3_bucket")
}
