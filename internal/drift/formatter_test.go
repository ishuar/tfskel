package drift

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewFormatter(t *testing.T) {
	formatter := NewFormatter(true)
	assert.NotNil(t, formatter)
	assert.True(t, formatter.useColor)
	assert.Equal(t, 0, formatter.tableWidth) // Initially 0, calculated during formatting
}

func TestNewFormatter_NoColor(t *testing.T) {
	formatter := NewFormatter(false)
	assert.NotNil(t, formatter)
	assert.False(t, formatter.useColor)
}

func TestFormatter_Format(t *testing.T) {
	report := &DriftReport{
		ScannedAt:  time.Now(),
		ScanRoot:   "/test",
		TotalFiles: 1,
		Summary:    DriftSummary{TerraformVersions: map[string]int{}, ProviderVersions: map[string]map[string]int{}},
		Records:    []DriftRecord{},
	}

	formatter := NewFormatter(false)
	buf := &bytes.Buffer{}
	err := formatter.Format(report, FormatTable, buf)
	assert.NoError(t, err)
}

func TestFormatter_Format_UnsupportedFormat(t *testing.T) {
	report := &DriftReport{
		ScannedAt:  time.Now(),
		ScanRoot:   "/test",
		TotalFiles: 1,
		Summary:    DriftSummary{},
		Records:    []DriftRecord{},
	}

	formatter := NewFormatter(false)
	buf := &bytes.Buffer{}
	err := formatter.Format(report, OutputFormat("invalid"), buf)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported format")
}

func TestFormatter_FormatJSON(t *testing.T) {
	report := &DriftReport{
		ScannedAt:      time.Date(2026, 2, 4, 12, 0, 0, 0, time.UTC),
		ScanRoot:       "/test/path",
		TotalFiles:     2,
		FilesWithDrift: 1,
		Summary: DriftSummary{
			TotalFiles:          2,
			FilesInSync:         1,
			FilesWithMinorDrift: 1,
			FilesWithMajorDrift: 0,
			TerraformVersions:   map[string]int{"~> 1.16": 2},
			ProviderVersions:    map[string]map[string]int{},
		},
		Records: []DriftRecord{
			{
				FilePath:             "envs/dev/versions.tf",
				TerraformExpected:    "~> 1.16",
				TerraformActual:      "~> 1.15",
				TerraformDriftStatus: StatusMinorDrift,
				HasDrift:             true,
				Providers:            []ProviderDrift{},
			},
		},
	}

	formatter := NewFormatter(false)
	buf := &bytes.Buffer{}
	err := formatter.Format(report, FormatJSON, buf)

	require.NoError(t, err)

	// Verify it's valid JSON
	var result map[string]interface{}
	err = json.Unmarshal(buf.Bytes(), &result)
	assert.NoError(t, err)

	// Verify key fields
	assert.Equal(t, "/test/path", result["scanRoot"])
	assert.Equal(t, float64(2), result["totalFiles"])
	assert.Equal(t, float64(1), result["filesWithDrift"])
}

func TestFormatter_FormatCSV(t *testing.T) {
	report := &DriftReport{
		ScannedAt:      time.Now(),
		ScanRoot:       "/test",
		TotalFiles:     2,
		FilesWithDrift: 2,
		Summary:        DriftSummary{},
		Records: []DriftRecord{
			{
				FilePath:             "envs/dev/versions.tf",
				TerraformExpected:    "~> 1.16",
				TerraformActual:      "~> 1.15",
				TerraformDriftStatus: StatusMinorDrift,
				HasDrift:             true,
				Providers: []ProviderDrift{
					{
						Name:        "aws",
						Source:      "hashicorp/aws",
						Expected:    "~> 6.0",
						Actual:      "~> 5.0",
						DriftStatus: StatusMajorDrift,
					},
				},
			},
		},
	}

	formatter := NewFormatter(false)
	buf := &bytes.Buffer{}
	err := formatter.Format(report, FormatCSV, buf)

	require.NoError(t, err)

	// Parse CSV
	reader := csv.NewReader(buf)
	records, err := reader.ReadAll()
	require.NoError(t, err)

	// Verify headers
	assert.Len(t, records, 3) // 1 header + 1 terraform + 1 provider
	assert.Equal(t, []string{"File Path", "Component Type", "Component Name", "Expected Version", "Actual Version", "Drift Status", "Severity"}, records[0])

	// Verify terraform record
	assert.Equal(t, "envs/dev/versions.tf", records[1][0])
	assert.Equal(t, "terraform", records[1][1])
	assert.Equal(t, "terraform", records[1][2])
	assert.Equal(t, "~> 1.16", records[1][3])
	assert.Equal(t, "~> 1.15", records[1][4])
	assert.Equal(t, "minor-drift", records[1][5])
	assert.Equal(t, "minor", records[1][6])

	// Verify provider record
	assert.Equal(t, "envs/dev/versions.tf", records[2][0])
	assert.Equal(t, "provider", records[2][1])
	assert.Equal(t, "aws", records[2][2])
	assert.Equal(t, "~> 6.0", records[2][3])
	assert.Equal(t, "~> 5.0", records[2][4])
	assert.Equal(t, "major-drift", records[2][5])
	assert.Equal(t, "major", records[2][6])
}

func TestFormatter_FormatStatus(t *testing.T) {
	tests := []struct {
		name   string
		status DriftStatus
		want   string
	}{
		{
			name:   "in sync",
			status: StatusInSync,
			want:   "OK",
		},
		{
			name:   "minor drift",
			status: StatusMinorDrift,
			want:   "minor drift",
		},
		{
			name:   "major drift",
			status: StatusMajorDrift,
			want:   "major drift",
		},
		{
			name:   "missing",
			status: StatusMissing,
			want:   "missing",
		},
		{
			name:   "not managed",
			status: StatusNotManaged,
			want:   "not managed",
		},
	}

	formatter := NewFormatter(false)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatter.formatStatus(tt.status)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestFormatter_FormatTable_WithDrift(t *testing.T) {
	report := &DriftReport{
		ScannedAt:      time.Now(),
		ScanRoot:       "/test",
		TotalFiles:     2,
		FilesWithDrift: 1,
		Summary: DriftSummary{
			TotalFiles:          2,
			FilesInSync:         1,
			FilesWithMinorDrift: 1,
			FilesWithMajorDrift: 0,
			TerraformVersions:   map[string]int{"~> 1.16": 1, "~> 1.15": 1},
			ProviderVersions:    map[string]map[string]int{},
		},
		Records: []DriftRecord{
			{
				FilePath:             "envs/dev/versions.tf",
				TerraformExpected:    "~> 1.16",
				TerraformActual:      "~> 1.15",
				TerraformDriftStatus: StatusMinorDrift,
				HasDrift:             true,
				Providers:            []ProviderDrift{},
			},
		},
	}

	formatter := NewFormatter(false)
	buf := &bytes.Buffer{}
	err := formatter.Format(report, FormatTable, buf)

	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "Terraform Version Drift Report")
	assert.Contains(t, output, "Quick Summary")
	assert.Contains(t, output, "Files with Drift")
	assert.Contains(t, output, "envs/dev/versions.tf")
	assert.Contains(t, output, "minor drift")
}

func TestFormatter_FormatTable_NoDrift(t *testing.T) {
	report := &DriftReport{
		ScannedAt:      time.Now(),
		ScanRoot:       "/test",
		TotalFiles:     1,
		FilesWithDrift: 0,
		Summary: DriftSummary{
			TotalFiles:          1,
			FilesInSync:         1,
			FilesWithMinorDrift: 0,
			FilesWithMajorDrift: 0,
			TerraformVersions:   map[string]int{"~> 1.16": 1},
			ProviderVersions:    map[string]map[string]int{},
		},
		Records: []DriftRecord{
			{
				FilePath:             "envs/dev/versions.tf",
				TerraformExpected:    "~> 1.16",
				TerraformActual:      "~> 1.16",
				TerraformDriftStatus: StatusInSync,
				HasDrift:             false,
				Providers:            []ProviderDrift{},
			},
		},
	}

	formatter := NewFormatter(false)
	buf := &bytes.Buffer{}
	err := formatter.Format(report, FormatTable, buf)

	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "Terraform Version Drift Report")
	assert.Contains(t, output, "All 1 files are in sync")
	// Should not have "Files with Drift" section
	assert.NotContains(t, output, "Files with Drift (")
}

func TestFormatter_FormatTable_WithErrors(t *testing.T) {
	report := &DriftReport{
		ScannedAt:      time.Now(),
		ScanRoot:       "/test",
		TotalFiles:     2,
		FilesWithDrift: 0,
		Summary: DriftSummary{
			TotalFiles:          2,
			FilesInSync:         1,
			FilesWithMinorDrift: 0,
			FilesWithMajorDrift: 0,
			FilesWithErrors:     1,
			TerraformVersions:   map[string]int{},
			ProviderVersions:    map[string]map[string]int{},
		},
		Records: []DriftRecord{},
	}

	formatter := NewFormatter(false)
	buf := &bytes.Buffer{}
	err := formatter.Format(report, FormatTable, buf)

	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "Files with Errors")
	assert.Contains(t, output, "1")
}

func TestFormatter_FormatTable_LongPaths(t *testing.T) {
	longPath := strings.Repeat("very/long/path/", 20) + "versions.tf"

	report := &DriftReport{
		ScannedAt:      time.Now(),
		ScanRoot:       "/test",
		TotalFiles:     1,
		FilesWithDrift: 1,
		Summary: DriftSummary{
			TotalFiles:          1,
			FilesWithMinorDrift: 1,
			TerraformVersions:   map[string]int{},
			ProviderVersions:    map[string]map[string]int{},
		},
		Records: []DriftRecord{
			{
				FilePath:             longPath,
				TerraformExpected:    "~> 1.16",
				TerraformActual:      "~> 1.15",
				TerraformDriftStatus: StatusMinorDrift,
				HasDrift:             true,
				Providers:            []ProviderDrift{},
			},
		},
	}

	formatter := NewFormatter(false)
	buf := &bytes.Buffer{}
	err := formatter.Format(report, FormatTable, buf)

	require.NoError(t, err)
	// Should not error even with long paths
	assert.NotEmpty(t, buf.String())
}

func TestFormatter_CalculateOptimalWidth(t *testing.T) {
	tests := []struct {
		name           string
		terminalWidth  int
		report         *DriftReport
		expectedMin    int
		expectedMax    int
	}{
		{
			name:          "default width with no drift",
			terminalWidth: 120,
			report: &DriftReport{
				Records: []DriftRecord{},
			},
			expectedMin: minDriftTableWidth,
			expectedMax: minDriftTableWidth,
		},
		{
			name:          "wide terminal with short paths",
			terminalWidth: 200,
			report: &DriftReport{
				Records: []DriftRecord{
					{FilePath: "short.tf", HasDrift: true},
				},
			},
			expectedMin: minDriftTableWidth,
			expectedMax: minDriftTableWidth,
		},
		{
			name:          "narrow terminal",
			terminalWidth: 80,
			report: &DriftReport{
				Records: []DriftRecord{},
			},
			expectedMin: 80,
			expectedMax: 80,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatter := NewFormatter(false)
			formatter.terminalWidth = tt.terminalWidth
			width := formatter.calculateOptimalWidth(tt.report)
			assert.GreaterOrEqual(t, width, tt.expectedMin)
			assert.LessOrEqual(t, width, tt.expectedMax)
		})
	}
}
