package drift

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDriftReport_GetDriftSummaryText(t *testing.T) {
	tests := []struct {
		name   string
		report *DriftReport
		want   string
	}{
		{
			name: "no drift",
			report: &DriftReport{
				TotalFiles:     5,
				FilesWithDrift: 0,
				Summary: DriftSummary{
					FilesInSync:         5,
					FilesWithMinorDrift: 0,
					FilesWithMajorDrift: 0,
					FilesWithErrors:     0,
				},
			},
			want: "All 5 files are in sync",
		},
		{
			name: "minor drift only",
			report: &DriftReport{
				TotalFiles:     10,
				FilesWithDrift: 3,
				Summary: DriftSummary{
					FilesInSync:         7,
					FilesWithMinorDrift: 3,
					FilesWithMajorDrift: 0,
					FilesWithErrors:     0,
				},
			},
			want: "3 of 10 files have drift (minor: 3, major: 0)",
		},
		{
			name: "major drift only",
			report: &DriftReport{
				TotalFiles:     8,
				FilesWithDrift: 2,
				Summary: DriftSummary{
					FilesInSync:         6,
					FilesWithMinorDrift: 0,
					FilesWithMajorDrift: 2,
					FilesWithErrors:     0,
				},
			},
			want: "2 of 8 files have drift (minor: 0, major: 2)",
		},
		{
			name: "mixed drift",
			report: &DriftReport{
				TotalFiles:     15,
				FilesWithDrift: 7,
				Summary: DriftSummary{
					FilesInSync:         8,
					FilesWithMinorDrift: 4,
					FilesWithMajorDrift: 3,
					FilesWithErrors:     0,
				},
			},
			want: "7 of 15 files have drift (minor: 4, major: 3)",
		},
		{
			name: "with errors",
			report: &DriftReport{
				TotalFiles:     12,
				FilesWithDrift: 3,
				Summary: DriftSummary{
					FilesInSync:         7,
					FilesWithMinorDrift: 2,
					FilesWithMajorDrift: 1,
					FilesWithErrors:     2,
				},
			},
			want: "3 of 12 files have drift (minor: 2, major: 1), 2 files with errors",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.report.GetDriftSummaryText()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestDriftReport_GetExitCode(t *testing.T) {
	tests := []struct {
		name   string
		report *DriftReport
		want   int
	}{
		{
			name: "no drift no errors",
			report: &DriftReport{
				FilesWithDrift: 0,
				Summary: DriftSummary{
					FilesWithErrors: 0,
				},
			},
			want: 0,
		},
		{
			name: "has drift no errors",
			report: &DriftReport{
				FilesWithDrift: 3,
				Summary: DriftSummary{
					FilesWithErrors: 0,
				},
			},
			want: 1,
		},
		{
			name: "no drift with errors",
			report: &DriftReport{
				FilesWithDrift: 0,
				Summary: DriftSummary{
					FilesWithErrors: 2,
				},
			},
			want: 2,
		},
		{
			name: "has drift and errors",
			report: &DriftReport{
				FilesWithDrift: 5,
				Summary: DriftSummary{
					FilesWithErrors: 1,
				},
			},
			want: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.report.ExitCode()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestVersionInfo(t *testing.T) {
	t.Run("creates version info with providers", func(t *testing.T) {
		vi := VersionInfo{
			FilePath:         "envs/dev/versions.tf",
			TerraformVersion: "~> 1.13",
			Providers: map[string]ProviderVer{
				"aws": {
					Source:  "hashicorp/aws",
					Version: "~> 6.0",
				},
			},
			ParseError: nil,
		}

		assert.Equal(t, "envs/dev/versions.tf", vi.FilePath)
		assert.Equal(t, "~> 1.13", vi.TerraformVersion)
		assert.Len(t, vi.Providers, 1)
		assert.Equal(t, "hashicorp/aws", vi.Providers["aws"].Source)
		assert.Equal(t, "~> 6.0", vi.Providers["aws"].Version)
		assert.Nil(t, vi.ParseError)
	})
}

func TestDriftRecord(t *testing.T) {
	t.Run("creates drift record with multiple provider drifts", func(t *testing.T) {
		record := DriftRecord{
			FilePath:             "envs/prod/versions.tf",
			TerraformExpected:    "~> 1.16",
			TerraformActual:      "~> 1.12",
			TerraformDriftStatus: StatusMajorDrift,
			Providers: []ProviderDrift{
				{
					Name:        "aws",
					Source:      "hashicorp/aws",
					Expected:    "~> 6.0",
					Actual:      "~> 4.0",
					DriftStatus: StatusMajorDrift,
				},
				{
					Name:        "random",
					Source:      "hashicorp/random",
					Expected:    "",
					Actual:      "~> 3.1",
					DriftStatus: StatusNotManaged,
				},
			},
			HasDrift: true,
		}

		assert.True(t, record.HasDrift)
		assert.Equal(t, "envs/prod/versions.tf", record.FilePath)
		assert.Equal(t, "~> 1.16", record.TerraformExpected)
		assert.Equal(t, "~> 1.12", record.TerraformActual)
		assert.Equal(t, StatusMajorDrift, record.TerraformDriftStatus)
		assert.Len(t, record.Providers, 2)
		assert.Equal(t, "aws", record.Providers[0].Name)
		assert.Equal(t, StatusMajorDrift, record.Providers[0].DriftStatus)
		assert.Equal(t, StatusNotManaged, record.Providers[1].DriftStatus)
	})
}

func TestDriftStatus(t *testing.T) {
	t.Run("drift status constants", func(t *testing.T) {
		assert.Equal(t, DriftStatus("in-sync"), StatusInSync)
		assert.Equal(t, DriftStatus("minor-drift"), StatusMinorDrift)
		assert.Equal(t, DriftStatus("major-drift"), StatusMajorDrift)
		assert.Equal(t, DriftStatus("missing"), StatusMissing)
		assert.Equal(t, DriftStatus("not-managed"), StatusNotManaged)
	})
}

func TestDriftSummary(t *testing.T) {
	t.Run("creates summary with version distributions", func(t *testing.T) {
		summary := DriftSummary{
			TotalFiles:          10,
			FilesInSync:         7,
			FilesWithMinorDrift: 2,
			FilesWithMajorDrift: 1,
			FilesWithErrors:     0,
			TerraformVersions: map[string]int{
				"~> 1.16": 8,
				"~> 1.15": 2,
			},
			ProviderVersions: map[string]map[string]int{
				"aws": {
					"~> 6.0": 9,
					"~> 5.0": 1,
				},
			},
		}

		assert.Equal(t, 10, summary.TotalFiles)
		assert.Equal(t, 7, summary.FilesInSync)
		assert.Equal(t, 2, summary.FilesWithMinorDrift)
		assert.Equal(t, 1, summary.FilesWithMajorDrift)
		assert.Equal(t, 0, summary.FilesWithErrors)
		assert.Equal(t, 8, summary.TerraformVersions["~> 1.16"])
		assert.Equal(t, 9, summary.ProviderVersions["aws"]["~> 6.0"])
	})
}
