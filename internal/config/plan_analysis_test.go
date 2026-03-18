package config

import (
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestLoadPlanAnalysisConfig(t *testing.T) {
	tests := []struct {
		name                  string
		viperSetup            func(*viper.Viper)
		wantResources         []string
		wantResourcesLen      int
		wantTopResourcesCount int
	}{
		{
			name: "config with critical resources",
			viperSetup: func(v *viper.Viper) {
				v.Set("critical_resources", []string{"aws_iam_role", "aws_lambda_function"})
			},
			wantResources:         []string{"aws_iam_role", "aws_lambda_function"},
			wantResourcesLen:      2,
			wantTopResourcesCount: 10, // default
		},
		{
			name: "config without critical resources",
			viperSetup: func(_ *viper.Viper) {
				// Don't set any critical_resources
			},
			wantResources:         nil,
			wantResourcesLen:      0,
			wantTopResourcesCount: 10, // default
		},
		{
			name: "config with empty critical resources",
			viperSetup: func(v *viper.Viper) {
				v.Set("critical_resources", []string{})
			},
			wantResources:         []string{},
			wantResourcesLen:      0,
			wantTopResourcesCount: 10, // default
		},
		{
			name: "config with custom top_resources_count",
			viperSetup: func(v *viper.Viper) {
				v.Set("top_resources_count", 20)
			},
			wantResources:         nil,
			wantResourcesLen:      0,
			wantTopResourcesCount: 20,
		},
		{
			name: "config with zero top_resources_count uses default",
			viperSetup: func(v *viper.Viper) {
				v.Set("top_resources_count", 0)
			},
			wantResources:         nil,
			wantResourcesLen:      0,
			wantTopResourcesCount: 10, // default, ignores 0
		},
		{
			name: "config with negative top_resources_count uses default",
			viperSetup: func(v *viper.Viper) {
				v.Set("top_resources_count", -5)
			},
			wantResources:         nil,
			wantResourcesLen:      0,
			wantTopResourcesCount: 10, // default, ignores negative
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := viper.New()
			tt.viperSetup(v)

			cfg := LoadPlanAnalysisConfig(v)

			assert.NotNil(t, cfg)
			assert.Len(t, cfg.CriticalResources, tt.wantResourcesLen)
			if tt.wantResources != nil {
				assert.Equal(t, tt.wantResources, cfg.CriticalResources)
			}
			assert.Equal(t, tt.wantTopResourcesCount, cfg.TopResourcesCount)
		})
	}
}
