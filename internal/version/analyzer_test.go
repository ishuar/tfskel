package version

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ishuar/tfskel/internal/config"
)

func TestNewAnalyzer(t *testing.T) {
	t.Run("creates new analyzer", func(t *testing.T) {
		cfg := &config.Config{
			TerraformVersion: "~> 1.16",
		}
		analyzer := NewAnalyzer(cfg)
		assert.NotNil(t, analyzer)
		assert.Equal(t, cfg, analyzer.config)
	})
}

func TestAnalyzer_Analyze(t *testing.T) {
	cfg := &config.Config{
		TerraformVersion: "~> 1.16",
		Provider: &config.Provider{
			AWS: &config.AWSProvider{
				Version: "~> 6.0",
			},
		},
	}
	analyzer := NewAnalyzer(cfg)

	tests := []struct {
		name           string
		versionInfos   []VersionInfo
		wantTotalFiles int
		wantDriftFiles int
		wantInSync     int
		wantErrors     int
	}{
		{
			name: "all files in sync",
			versionInfos: []VersionInfo{
				{
					FilePath:         "env1/versions.tf",
					TerraformVersion: "~> 1.16",
					Providers: map[string]ProviderVer{
						"aws": {Version: "~> 6.0", Source: "hashicorp/aws"},
					},
				},
			},
			wantTotalFiles: 1,
			wantDriftFiles: 0,
			wantInSync:     1,
			wantErrors:     0,
		},
		{
			name: "terraform version drift",
			versionInfos: []VersionInfo{
				{
					FilePath:         "env1/versions.tf",
					TerraformVersion: "~> 1.15",
					Providers: map[string]ProviderVer{
						"aws": {Version: "~> 6.0", Source: "hashicorp/aws"},
					},
				},
			},
			wantTotalFiles: 1,
			wantDriftFiles: 1,
			wantInSync:     0,
			wantErrors:     0,
		},
		{
			name: "provider version drift",
			versionInfos: []VersionInfo{
				{
					FilePath:         "env2/versions.tf",
					TerraformVersion: "~> 1.16",
					Providers: map[string]ProviderVer{
						"aws": {Version: "~> 4.0", Source: "hashicorp/aws"},
					},
				},
			},
			wantTotalFiles: 1,
			wantDriftFiles: 1,
			wantInSync:     0,
			wantErrors:     0,
		},
		{
			name: "multiple files mixed drift",
			versionInfos: []VersionInfo{
				{
					FilePath:         "env1/versions.tf",
					TerraformVersion: "~> 1.15",
					Providers: map[string]ProviderVer{
						"aws": {Version: "~> 6.0", Source: "hashicorp/aws"},
					},
				},
				{
					FilePath:         "env2/versions.tf",
					TerraformVersion: "~> 1.16",
					Providers: map[string]ProviderVer{
						"aws": {Version: "~> 4.0", Source: "hashicorp/aws"},
					},
				},
				{
					FilePath:         "env3/versions.tf",
					TerraformVersion: "~> 1.16",
					Providers: map[string]ProviderVer{
						"aws": {Version: "~> 6.0", Source: "hashicorp/aws"},
					},
				},
			},
			wantTotalFiles: 3,
			wantDriftFiles: 2,
			wantInSync:     1,
			wantErrors:     0,
		},
		{
			name: "with parse errors",
			versionInfos: []VersionInfo{
				{
					FilePath:         "env1/versions.tf",
					TerraformVersion: "~> 1.16",
					Providers: map[string]ProviderVer{
						"aws": {Version: "~> 6.0", Source: "hashicorp/aws"},
					},
				},
				{
					FilePath:   "env2/bad.tf",
					ParseError: assert.AnError,
				},
			},
			wantTotalFiles: 2,
			wantDriftFiles: 0,
			wantInSync:     1,
			wantErrors:     1,
		},
		{
			name: "terraform and provider drift in same file",
			versionInfos: []VersionInfo{
				{
					FilePath:         "env1/versions.tf",
					TerraformVersion: "~> 1.15",
					Providers: map[string]ProviderVer{
						"aws": {Version: "~> 4.0", Source: "hashicorp/aws"},
					},
				},
			},
			wantTotalFiles: 1,
			wantDriftFiles: 1,
			wantInSync:     0,
			wantErrors:     0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			report := analyzer.Analyze("/test/path", tt.versionInfos)

			assert.Equal(t, tt.wantTotalFiles, report.TotalFiles, "TotalFiles mismatch")
			assert.Equal(t, tt.wantDriftFiles, report.FilesWithDrift, "FilesWithDrift mismatch")
			assert.Equal(t, tt.wantInSync, report.Summary.FilesInSync, "FilesInSync mismatch")
			assert.Equal(t, tt.wantErrors, report.Summary.FilesWithErrors, "FilesWithErrors mismatch")
			assert.Equal(t, tt.wantDriftFiles, report.Summary.FilesWithDrift, "Summary.FilesWithDrift mismatch")

			// Consistency: total = in-sync + drift + errors
			totalCategorized := report.Summary.FilesInSync + report.Summary.FilesWithDrift + report.Summary.FilesWithErrors
			assert.Equal(t, report.TotalFiles, totalCategorized,
				"Total files should equal sum of all categories")
		})
	}
}

func TestAnalyzer_CompareSemverConstraints(t *testing.T) {
	analyzer := NewAnalyzer(&config.Config{})

	tests := []struct {
		name     string
		expected string
		actual   string
		want     DriftStatus
	}{
		{
			name:     "exact match",
			expected: "~> 1.16",
			actual:   "~> 1.16",
			want:     StatusInSync,
		},
		{
			name:     "different minor version is drift",
			expected: "~> 1.16",
			actual:   "~> 1.15",
			want:     StatusDrift,
		},
		{
			name:     "different major version is drift",
			expected: "~> 2.0",
			actual:   "~> 1.16",
			want:     StatusDrift,
		},
		{
			name:     "semver vs constraint same major.minor is in-sync",
			expected: "1.13.1",
			actual:   "~> 1.13",
			want:     StatusInSync,
		},
		{
			name:     "constraint vs semver same major.minor is in-sync",
			expected: "~> 5.80",
			actual:   "5.80.0",
			want:     StatusInSync,
		},
		{
			name:     "different constraint operators same version is in-sync",
			expected: "= 1.13",
			actual:   "~> 1.13",
			want:     StatusInSync,
		},
		{
			name:     "semver vs constraint different minor is drift",
			expected: "1.14.0",
			actual:   "~> 1.13",
			want:     StatusDrift,
		},
		{
			name:     "semver vs constraint different major is drift",
			expected: "2.0.0",
			actual:   "~> 1.13",
			want:     StatusDrift,
		},
		{
			name:     "unparseable expected is drift",
			expected: "latest",
			actual:   "~> 1.13",
			want:     StatusDrift,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := analyzer.compareSemverConstraints(tt.expected, tt.actual)
			assert.Equal(t, tt.want, got)
		})
	}
}
