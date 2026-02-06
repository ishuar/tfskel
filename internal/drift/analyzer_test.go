package drift

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
		wantMinorDrift int
		wantMajorDrift int
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
			wantMinorDrift: 0,
			wantMajorDrift: 0,
			wantErrors:     0,
		},
		{
			name: "with minor drift in terraform version",
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
			wantMinorDrift: 1,
			wantMajorDrift: 0,
			wantErrors:     0,
		},
		{
			name: "with major drift in provider",
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
			wantMinorDrift: 0,
			wantMajorDrift: 1,
			wantErrors:     0,
		},
		{
			name: "mixed severity - multiple files",
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
			wantMinorDrift: 1,
			wantMajorDrift: 1,
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
			wantMinorDrift: 0,
			wantMajorDrift: 0,
			wantErrors:     1,
		},
		{
			name: "file with major drift classification (any major = major file)",
			versionInfos: []VersionInfo{
				{
					FilePath:         "env1/versions.tf",
					TerraformVersion: "~> 1.15", // minor drift
					Providers: map[string]ProviderVer{
						"aws": {Version: "~> 4.0", Source: "hashicorp/aws"}, // major drift
					},
				},
			},
			wantTotalFiles: 1,
			wantDriftFiles: 1,
			wantInSync:     0,
			wantMinorDrift: 0,
			wantMajorDrift: 1, // Should be major because ANY component has major drift
			wantErrors:     0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			report := analyzer.Analyze("/test/path", tt.versionInfos)

			// Validate all fields are correctly calculated
			assert.Equal(t, tt.wantTotalFiles, report.TotalFiles, "TotalFiles mismatch")
			assert.Equal(t, tt.wantDriftFiles, report.FilesWithDrift, "FilesWithDrift mismatch")
			assert.Equal(t, tt.wantInSync, report.Summary.FilesInSync, "FilesInSync mismatch")
			assert.Equal(t, tt.wantMinorDrift, report.Summary.FilesWithMinorDrift, "FilesWithMinorDrift mismatch")
			assert.Equal(t, tt.wantMajorDrift, report.Summary.FilesWithMajorDrift, "FilesWithMajorDrift mismatch")
			assert.Equal(t, tt.wantErrors, report.Summary.FilesWithErrors, "FilesWithErrors mismatch")

			// Validate consistency: total files should equal sum of categories
			totalCategorized := report.Summary.FilesInSync + report.Summary.FilesWithMinorDrift +
				report.Summary.FilesWithMajorDrift + report.Summary.FilesWithErrors
			assert.Equal(t, report.TotalFiles, totalCategorized,
				"Total files should equal sum of all categories")

			// Validate consistency: drift files should equal minor + major
			driftSum := report.Summary.FilesWithMinorDrift + report.Summary.FilesWithMajorDrift
			assert.Equal(t, report.FilesWithDrift, driftSum,
				"FilesWithDrift should equal FilesWithMinorDrift + FilesWithMajorDrift")
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
			name:     "minor drift",
			expected: "~> 1.16",
			actual:   "~> 1.15",
			want:     StatusMinorDrift,
		},
		{
			name:     "major drift",
			expected: "~> 2.0",
			actual:   "~> 1.16",
			want:     StatusMajorDrift,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := analyzer.compareSemverConstraints(tt.expected, tt.actual)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestDriftReport_HasCriticalDrift(t *testing.T) {
	tests := []struct {
		name   string
		report *DriftReport
		want   bool
	}{
		{
			name: "has major drift",
			report: &DriftReport{
				Summary: DriftSummary{
					FilesWithMajorDrift: 2,
					FilesWithMinorDrift: 1,
				},
			},
			want: true,
		},
		{
			name: "only minor drift",
			report: &DriftReport{
				Summary: DriftSummary{
					FilesWithMajorDrift: 0,
					FilesWithMinorDrift: 3,
				},
			},
			want: false,
		},
		{
			name: "no drift",
			report: &DriftReport{
				Summary: DriftSummary{
					FilesWithMajorDrift: 0,
					FilesWithMinorDrift: 0,
				},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.report.HasCriticalDrift()
			assert.Equal(t, tt.want, got)
		})
	}
}
