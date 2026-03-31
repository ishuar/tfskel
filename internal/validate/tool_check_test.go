package validate

import (
	"testing"

	"github.com/ishuar/tfskel/internal/toolcheck"
	"github.com/stretchr/testify/assert"
)

func TestCheckToolVersions_Mismatch(t *testing.T) {
	dir := t.TempDir()
	writeMiseToml(t, dir, `
[tools]
terraform = "1.13.0"
tflint = "0.50.0"
`)

	report := &toolcheck.Report{
		Tools: []toolcheck.ToolResult{
			{Name: "Terraform", Binary: "terraform", MisePlugin: "terraform", Status: toolcheck.StatusInstalled, Version: "1.12.0"},
			{Name: "TFLint", Binary: "tflint", MisePlugin: "tflint", Status: toolcheck.StatusInstalled, Version: "0.49.0"},
		},
	}

	findings := checkToolVersions(dir, report)

	assert.Len(t, findings, 2)

	assert.Equal(t, "Terraform", findings[0].Resource)
	assert.Equal(t, "1.13.0", findings[0].Expected)
	assert.Equal(t, "1.12.0", findings[0].Actual)
	assert.Equal(t, "installed version does not match expected", findings[0].Message)

	assert.Equal(t, "TFLint", findings[1].Resource)
	assert.Equal(t, "0.50.0", findings[1].Expected)
	assert.Equal(t, "0.49.0", findings[1].Actual)
}

func TestCheckToolVersions_AllMatch(t *testing.T) {
	dir := t.TempDir()
	writeMiseToml(t, dir, `
[tools]
terraform = "1.13.0"
tflint = "0.50.0"
`)

	report := &toolcheck.Report{
		Tools: []toolcheck.ToolResult{
			{Name: "Terraform", Binary: "terraform", MisePlugin: "terraform", Status: toolcheck.StatusInstalled, Version: "1.13.0"},
			{Name: "TFLint", Binary: "tflint", MisePlugin: "tflint", Status: toolcheck.StatusInstalled, Version: "0.50.0"},
		},
	}

	findings := checkToolVersions(dir, report)
	assert.Empty(t, findings)
}

func TestCheckToolVersions_LatestSkipped(t *testing.T) {
	dir := t.TempDir()
	writeMiseToml(t, dir, `
[tools]
tflint = "latest"
`)

	report := &toolcheck.Report{
		Tools: []toolcheck.ToolResult{
			{Name: "TFLint", Binary: "tflint", MisePlugin: "tflint", Status: toolcheck.StatusInstalled, Version: "0.50.0"},
		},
	}

	findings := checkToolVersions(dir, report)
	assert.Empty(t, findings, "tools pinned to 'latest' should not produce version mismatch findings")
}

func TestCheckToolVersions_NoMiseToml(t *testing.T) {
	dir := t.TempDir() // No .mise.toml

	report := &toolcheck.Report{
		Tools: []toolcheck.ToolResult{
			{Name: "Terraform", Binary: "terraform", MisePlugin: "terraform", Status: toolcheck.StatusInstalled, Version: "1.12.0"},
		},
	}

	findings := checkToolVersions(dir, report)
	assert.Empty(t, findings, "without .mise.toml, version checks should be skipped gracefully")
}

func TestCheckToolVersions_NotInstalled(t *testing.T) {
	dir := t.TempDir()
	writeMiseToml(t, dir, `
[tools]
terraform = "1.13.0"
`)

	report := &toolcheck.Report{
		Tools: []toolcheck.ToolResult{
			{Name: "Terraform", Binary: "terraform", MisePlugin: "terraform", Status: toolcheck.StatusMissing},
		},
	}

	findings := checkToolVersions(dir, report)
	assert.Empty(t, findings, "missing tools should not have version comparison")
}

func TestCheckToolVersions_GlobalPathMismatch(t *testing.T) {
	dir := t.TempDir()
	writeMiseToml(t, dir, `
[tools]
terraform = "1.14.0"
`)

	report := &toolcheck.Report{
		Tools: []toolcheck.ToolResult{
			{Name: "Terraform", Binary: "terraform", MisePlugin: "terraform", Status: toolcheck.StatusGlobalPath, Version: "1.13.5"},
		},
	}

	findings := checkToolVersions(dir, report)
	assert.Len(t, findings, 1)
	assert.Equal(t, "1.14.0", findings[0].Expected)
	assert.Equal(t, "1.13.5", findings[0].Actual)
	assert.Equal(t, "installed version does not match expected", findings[0].Message)
}

func TestCheckToolVersions_ToolNotInMiseToml(t *testing.T) {
	dir := t.TempDir()
	writeMiseToml(t, dir, `
[tools]
terraform = "1.13.0"
`)

	report := &toolcheck.Report{
		Tools: []toolcheck.ToolResult{
			{Name: "TFLint", Binary: "tflint", MisePlugin: "tflint", Status: toolcheck.StatusInstalled, Version: "0.50.0"},
		},
	}

	findings := checkToolVersions(dir, report)
	assert.Empty(t, findings, "tools not pinned in .mise.toml should not produce findings")
}
