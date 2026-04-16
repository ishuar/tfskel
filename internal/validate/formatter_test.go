package validate

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"strings"
	"testing"

	"github.com/ishuar/tfskel/internal/format"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// sampleReport builds a report with known findings for deterministic output tests.
func sampleReport() *Report {
	return &Report{
		Checks: []CheckResult{
			{Check: CheckConfig, Status: StatusFail, Total: 2, Passed: 1, Issues: 1},
			{Check: CheckTools, Status: StatusSkipped},
		},
		Findings: []Finding{
			{
				Check:     CheckConfig,
				Resource:  "versions.tf",
				Component: "terraform",
				Message:   "version constraint drift",
				Expected:  "~> 1.13",
				Actual:    "~> 0.12",
			},
		},
	}
}

// passingReport builds a report where all checks pass.
func passingReport() *Report {
	return &Report{
		Checks: []CheckResult{
			{Check: CheckConfig, Status: StatusPass, Total: 2, Passed: 2},
			{Check: CheckTools, Status: StatusPass, Total: 5, Passed: 5},
		},
	}
}

// --- JSON output tests ---

func TestFormatter_FormatJSON(t *testing.T) {
	t.Run("structure matches expected schema", func(t *testing.T) {
		f := &Formatter{useColor: false, terminalWidth: 120}
		report := sampleReport()

		var buf bytes.Buffer
		err := f.Format(report, format.FormatJSON, &buf)

		require.NoError(t, err)

		var result struct {
			Checks   []CheckResult `json:"checks"`
			Findings []Finding     `json:"findings"`
			ExitCode int           `json:"exitCode"`
		}
		require.NoError(t, json.Unmarshal(buf.Bytes(), &result))

		assert.Len(t, result.Checks, 2)
		assert.Len(t, result.Findings, 1)
		assert.Equal(t, 1, result.ExitCode)
	})

	t.Run("exit code 0 for passing report", func(t *testing.T) {
		f := &Formatter{useColor: false, terminalWidth: 120}
		report := passingReport()

		var buf bytes.Buffer
		require.NoError(t, f.Format(report, format.FormatJSON, &buf))

		var result struct {
			ExitCode int `json:"exitCode"`
		}
		require.NoError(t, json.Unmarshal(buf.Bytes(), &result))
		assert.Equal(t, 0, result.ExitCode)
	})

	t.Run("findings contain expected fields", func(t *testing.T) {
		f := &Formatter{useColor: false, terminalWidth: 120}

		report := &Report{
			Checks: []CheckResult{
				{Check: CheckConfig, Status: StatusFail, Total: 1, Passed: 0, Issues: 1},
			},
			Findings: []Finding{
				{
					Check:     CheckConfig,
					Resource:  "envs/dev/versions.tf",
					Component: "aws",
					Message:   "version constraint drift",
					Expected:  "~> 5.0",
					Actual:    "~> 4.0",
				},
			},
		}

		var buf bytes.Buffer
		require.NoError(t, f.Format(report, format.FormatJSON, &buf))

		var result struct {
			Findings []Finding `json:"findings"`
		}
		require.NoError(t, json.Unmarshal(buf.Bytes(), &result))
		require.Len(t, result.Findings, 1)

		finding := result.Findings[0]
		assert.Equal(t, CheckConfig, finding.Check)
		assert.Equal(t, "aws", finding.Component)
		assert.Equal(t, "~> 5.0", finding.Expected)
		assert.Equal(t, "~> 4.0", finding.Actual)
	})

	t.Run("empty report produces valid JSON", func(t *testing.T) {
		f := &Formatter{useColor: false, terminalWidth: 120}
		report := &Report{}

		var buf bytes.Buffer
		require.NoError(t, f.Format(report, format.FormatJSON, &buf))

		var result map[string]any
		require.NoError(t, json.Unmarshal(buf.Bytes(), &result))
		assert.Equal(t, float64(0), result["exitCode"])
	})

	t.Run("directory field is included in JSON output", func(t *testing.T) {
		f := &Formatter{useColor: false, terminalWidth: 120}
		report := &Report{
			Directory: "/home/user/project",
			Checks:    []CheckResult{{Check: CheckConfig, Status: StatusPass}},
		}

		var buf bytes.Buffer
		require.NoError(t, f.Format(report, format.FormatJSON, &buf))

		var result struct {
			Directory string `json:"directory"`
		}
		require.NoError(t, json.Unmarshal(buf.Bytes(), &result))
		assert.Equal(t, "/home/user/project", result.Directory)
	})
}

// --- CSV output tests ---

func TestFormatter_FormatCSV(t *testing.T) {
	t.Run("header row is correct", func(t *testing.T) {
		f := &Formatter{useColor: false, terminalWidth: 120}
		report := &Report{}

		var buf bytes.Buffer
		require.NoError(t, f.Format(report, format.FormatCSV, &buf))

		reader := csv.NewReader(strings.NewReader(buf.String()))
		records, err := reader.ReadAll()
		require.NoError(t, err)
		require.Len(t, records, 1, "empty report should have only the header row")

		assert.Equal(t, []string{"check", "resource", "component", "message", "expected", "actual", "detail"}, records[0])
	})

	t.Run("findings produce correct rows", func(t *testing.T) {
		f := &Formatter{useColor: false, terminalWidth: 120}
		report := sampleReport()

		var buf bytes.Buffer
		require.NoError(t, f.Format(report, format.FormatCSV, &buf))

		reader := csv.NewReader(strings.NewReader(buf.String()))
		records, err := reader.ReadAll()
		require.NoError(t, err)
		require.Len(t, records, 2, "header + 1 finding")

		row := records[1]
		assert.Equal(t, "config", row[0])
		assert.Equal(t, "versions.tf", row[1])
		assert.Equal(t, "version constraint drift", row[3])
	})

	t.Run("multiple findings produce multiple rows", func(t *testing.T) {
		f := &Formatter{useColor: false, terminalWidth: 120}
		report := &Report{
			Findings: []Finding{
				{Check: CheckConfig, Resource: "versions.tf", Component: "aws", Message: "drift", Expected: "~> 5.0", Actual: "~> 4.0"},
				{Check: CheckTools, Resource: "tflint", Message: "missing", Detail: "run 'mise install'"},
			},
		}

		var buf bytes.Buffer
		require.NoError(t, f.Format(report, format.FormatCSV, &buf))

		reader := csv.NewReader(strings.NewReader(buf.String()))
		records, err := reader.ReadAll()
		require.NoError(t, err)
		assert.Len(t, records, 3, "header + 2 findings")
	})
}

// --- Table output tests ---

func TestFormatter_FormatTable(t *testing.T) {
	t.Run("passing report contains success verdict", func(t *testing.T) {
		f := &Formatter{useColor: false, terminalWidth: 120}
		report := passingReport()

		var buf bytes.Buffer
		require.NoError(t, f.Format(report, format.FormatTable, &buf))

		output := buf.String()
		assert.Contains(t, output, "All checks passed")
		assert.Contains(t, output, "Validation Report")
	})

	t.Run("failing report contains failure verdict with count", func(t *testing.T) {
		f := &Formatter{useColor: false, terminalWidth: 120}
		report := sampleReport()

		var buf bytes.Buffer
		require.NoError(t, f.Format(report, format.FormatTable, &buf))

		output := buf.String()
		assert.Contains(t, output, "Validation failed")
		assert.Contains(t, output, "1 finding")
	})

	t.Run("skipped checks show skipped status", func(t *testing.T) {
		f := &Formatter{useColor: false, terminalWidth: 120}
		report := &Report{
			Checks: []CheckResult{
				{Check: CheckConfig, Status: StatusSkipped},
				{Check: CheckTools, Status: StatusSkipped},
			},
		}

		var buf bytes.Buffer
		require.NoError(t, f.Format(report, format.FormatTable, &buf))

		output := buf.String()
		assert.Contains(t, output, "skipped")
		assert.Contains(t, output, "All checks passed")
	})

	t.Run("version drift section shown for config findings", func(t *testing.T) {
		f := &Formatter{useColor: false, terminalWidth: 120}
		report := &Report{
			Checks: []CheckResult{
				{Check: CheckConfig, Status: StatusFail, Total: 1, Passed: 0, Issues: 1},
				{Check: CheckTools, Status: StatusPass, Total: 5, Passed: 5},
			},
			Findings: []Finding{
				{
					Check:     CheckConfig,
					Resource:  "envs/dev/versions.tf",
					Component: "terraform",
					Message:   "version constraint drift",
					Expected:  "~> 1.13",
					Actual:    "~> 0.12",
				},
			},
		}

		var buf bytes.Buffer
		require.NoError(t, f.Format(report, format.FormatTable, &buf))

		output := buf.String()
		assert.Contains(t, output, "Version Drift")
		assert.Contains(t, output, "envs/dev/versions.tf")
	})

	t.Run("plural issues in verdict", func(t *testing.T) {
		f := &Formatter{useColor: false, terminalWidth: 120}
		report := &Report{
			Checks: []CheckResult{
				{Check: CheckConfig, Status: StatusFail, Total: 1, Passed: 0, Issues: 1},
				{Check: CheckTools, Status: StatusFail, Total: 5, Passed: 3, Issues: 2},
			},
			Findings: []Finding{
				{Check: CheckConfig, Resource: "a"},
				{Check: CheckTools, Resource: "b"},
				{Check: CheckTools, Resource: "c"},
			},
		}

		var buf bytes.Buffer
		require.NoError(t, f.Format(report, format.FormatTable, &buf))

		output := buf.String()
		assert.Contains(t, output, "3 findings")
	})

	t.Run("tools summary shows unique tools and total findings", func(t *testing.T) {
		f := &Formatter{useColor: false, terminalWidth: 120}
		report := &Report{
			Checks: []CheckResult{
				{Check: CheckConfig, Status: StatusPass, Total: 1, Passed: 1},
				{Check: CheckTools, Status: StatusFail, Total: 5, Passed: 3, Issues: 3, AffectedResources: 2},
			},
			Findings: []Finding{
				{Check: CheckTools, Resource: "Terraform", Message: "installed globally"},
				{Check: CheckTools, Resource: "Terraform", Component: "version", Message: "version mismatch"},
				{Check: CheckTools, Resource: "TFLint", Message: "not installed"},
			},
		}

		var buf bytes.Buffer
		require.NoError(t, f.Format(report, format.FormatTable, &buf))

		output := buf.String()
		assert.Contains(t, output, "2 tools, 3 findings")
	})

	t.Run("project header renders when data is populated", func(t *testing.T) {
		f := &Formatter{useColor: false, terminalWidth: 120}
		report := &Report{
			Directory:    "/home/alice/infra",
			ProjectRoot:  "/home/alice/infra",
			ConfigPath:   "/home/alice/infra/.tfskel.yaml",
			Environments: []string{"dev", "prd", "stg"},
			Regions:      []string{"eu-central-1"},
			Checks: []CheckResult{
				{Check: CheckConfig, Status: StatusPass},
				{Check: CheckTools, Status: StatusPass},
			},
		}

		var buf bytes.Buffer
		require.NoError(t, f.Format(report, format.FormatTable, &buf))

		output := buf.String()
		assert.Contains(t, output, "Project root:")
		assert.Contains(t, output, "/home/alice/infra")
		assert.Contains(t, output, "Config:")
		assert.Contains(t, output, "/home/alice/infra/.tfskel.yaml")
		assert.Contains(t, output, "Environments:")
		assert.Contains(t, output, "dev, prd, stg")
		assert.Contains(t, output, "Regions:")
		assert.Contains(t, output, "eu-central-1")
		// Working dir matches project root → line should be omitted.
		assert.NotContains(t, output, "Working dir:")
	})

	t.Run("working dir line shown when scan dir differs from project root", func(t *testing.T) {
		f := &Formatter{useColor: false, terminalWidth: 120}
		report := &Report{
			Directory:   "/home/alice/infra/envs",
			ProjectRoot: "/home/alice/infra",
			ConfigPath:  "/home/alice/infra/.tfskel.yaml",
			Checks:      []CheckResult{{Check: CheckConfig, Status: StatusPass}},
		}

		var buf bytes.Buffer
		require.NoError(t, f.Format(report, format.FormatTable, &buf))

		output := buf.String()
		assert.Contains(t, output, "Project root:")
		assert.Contains(t, output, "/home/alice/infra")
		assert.Contains(t, output, "Working dir:")
		assert.Contains(t, output, "/home/alice/infra/envs")
	})

	t.Run("project header omits empty env and region lines", func(t *testing.T) {
		f := &Formatter{useColor: false, terminalWidth: 120}
		report := &Report{
			ProjectRoot: "/tmp/proj",
			ConfigPath:  "/tmp/proj/.tfskel.yaml",
			Checks:      []CheckResult{{Check: CheckConfig, Status: StatusPass}},
		}

		var buf bytes.Buffer
		require.NoError(t, f.Format(report, format.FormatTable, &buf))

		output := buf.String()
		assert.Contains(t, output, "Project root:")
		assert.NotContains(t, output, "Environments:")
		assert.NotContains(t, output, "Regions:")
	})
}

func TestFormatter_FormatJSON_HeaderFields(t *testing.T) {
	f := &Formatter{useColor: false, terminalWidth: 120}
	report := &Report{
		ProjectRoot:  "/home/alice/infra",
		ConfigPath:   "/home/alice/infra/.tfskel.yaml",
		Environments: []string{"dev", "prd"},
		Regions:      []string{"eu-central-1"},
		Checks:       []CheckResult{{Check: CheckConfig, Status: StatusPass}},
	}

	var buf bytes.Buffer
	require.NoError(t, f.Format(report, format.FormatJSON, &buf))

	var result struct {
		ProjectRoot  string   `json:"projectRoot"`
		ConfigPath   string   `json:"configPath"`
		Environments []string `json:"environments"`
		Regions      []string `json:"regions"`
	}
	require.NoError(t, json.Unmarshal(buf.Bytes(), &result))
	assert.Equal(t, "/home/alice/infra", result.ProjectRoot)
	assert.Equal(t, "/home/alice/infra/.tfskel.yaml", result.ConfigPath)
	assert.Equal(t, []string{"dev", "prd"}, result.Environments)
	assert.Equal(t, []string{"eu-central-1"}, result.Regions)
}

// --- Unsupported format ---

func TestFormatter_UnsupportedFormat(t *testing.T) {
	f := &Formatter{useColor: false, terminalWidth: 120}
	report := &Report{}

	var buf bytes.Buffer
	err := f.Format(report, "xml", &buf)

	require.ErrorIs(t, err, ErrUnsupportedFormat)
}

// --- Helper tests ---

func TestFindingsForCheck(t *testing.T) {
	findings := []Finding{
		{Check: CheckConfig, Resource: "a"},
		{Check: CheckTools, Resource: "b"},
		{Check: CheckConfig, Resource: "c"},
	}

	configFindings := findingsForCheck(findings, CheckConfig)
	assert.Len(t, configFindings, 2)
	assert.Equal(t, "a", configFindings[0].Resource)
	assert.Equal(t, "c", configFindings[1].Resource)

	toolFindings := findingsForCheck(findings, CheckTools)
	assert.Len(t, toolFindings, 1)
}

func TestCalculateOptimalWidth(t *testing.T) {
	tests := []struct {
		name          string
		terminalWidth int
		want          int
	}{
		{"narrow terminal clamps to min", 60, minTableWidth},
		{"wide terminal clamps to max", 200, maxTableWidth},
		{"normal terminal uses percentage", 120, (120 * format.PercentageWidthFactor) / format.PercentageDivisor},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &Formatter{terminalWidth: tt.terminalWidth}
			assert.Equal(t, tt.want, f.calculateOptimalWidth())
		})
	}
}
