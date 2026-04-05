package toolcheck

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockRunner implements CommandRunner for testing.
// Keys in commands and errors are the full command string: "name arg1 arg2".
type mockRunner struct {
	paths    map[string]string // binary -> path (present means LookPath succeeds)
	commands map[string]string // "name arg1 arg2" -> output
	errors   map[string]error  // "name arg1 arg2" -> error
}

func (m *mockRunner) LookPath(file string) (string, error) {
	if path, ok := m.paths[file]; ok {
		return path, nil
	}
	return "", errors.New("executable file not found in $PATH")
}

func (m *mockRunner) RunCommand(name string, args ...string) (string, error) {
	key := name
	if len(args) > 0 {
		key = name + " " + strings.Join(args, " ")
	}
	if err, ok := m.errors[key]; ok && err != nil {
		return "", err
	}
	if output, ok := m.commands[key]; ok {
		return output, nil
	}
	return "", errors.New("command not found")
}

// --- Checker tests ---

func TestChecker_CheckAll(t *testing.T) {
	t.Run("all tools installed with mise", func(t *testing.T) {
		runner := &mockRunner{
			paths: map[string]string{
				"mise":       "/usr/local/bin/mise",
				"terraform":  "/usr/local/bin/terraform",
				"tflint":     "/usr/local/bin/tflint",
				"trivy":      "/usr/local/bin/trivy",
				"pre-commit": "/usr/local/bin/pre-commit",
				"aws":        "/usr/local/bin/aws",
			},
			commands: map[string]string{
				"terraform --version":  "Terraform v1.13.1\non linux_amd64",
				"tflint --version":     "TFLint version 0.50.0",
				"trivy --version":      "Version: 0.58.2",
				"pre-commit --version": "pre-commit 3.6.0",
				"aws --version":        "aws-cli/2.15.0 Python/3.11.6 Linux/6.1.0",
				// mise which succeeds for all — tools are from mise
				"mise which terraform":  "/home/user/.local/share/mise/installs/terraform/1.13.1/bin/terraform",
				"mise which tflint":     "/home/user/.local/share/mise/installs/tflint/0.50.0/bin/tflint",
				"mise which trivy":      "/home/user/.local/share/mise/installs/trivy/0.58.2/bin/trivy",
				"mise which pre-commit": "/home/user/.local/share/mise/installs/pre-commit/3.6.0/bin/pre-commit",
				"mise which aws":        "/home/user/.local/share/mise/installs/awscli/2.15.0/bin/aws",
			},
		}

		checker := NewChecker(runner, DefaultTools())
		report := checker.CheckAll()

		assert.True(t, report.MiseFound)
		require.Len(t, report.Tools, 5)
		for _, tool := range report.Tools {
			assert.Equal(t, StatusInstalled, tool.Status, "tool %s should be installed", tool.Name)
			assert.NotEmpty(t, tool.Version, "tool %s should have a version", tool.Name)
		}
	})

	t.Run("no tools installed and no mise", func(t *testing.T) {
		runner := &mockRunner{
			paths:    map[string]string{},
			commands: map[string]string{},
		}

		checker := NewChecker(runner, DefaultTools())
		report := checker.CheckAll()

		assert.False(t, report.MiseFound)
		for _, tool := range report.Tools {
			assert.Equal(t, StatusMissing, tool.Status, "tool %s should be missing", tool.Name)
			assert.Empty(t, tool.Version)
		}
	})

	t.Run("no tools installed but mise is available", func(t *testing.T) {
		runner := &mockRunner{
			paths: map[string]string{
				"mise": "/usr/local/bin/mise",
			},
			commands: map[string]string{},
		}

		checker := NewChecker(runner, DefaultTools())
		report := checker.CheckAll()

		assert.True(t, report.MiseFound)
		for _, tool := range report.Tools {
			assert.Equal(t, StatusMiseManaged, tool.Status,
				"tool %s should be mise-managed when mise is installed but tool is missing", tool.Name)
		}
	})

	t.Run("partial tool installation", func(t *testing.T) {
		runner := &mockRunner{
			paths: map[string]string{
				"mise":      "/usr/local/bin/mise",
				"terraform": "/usr/local/bin/terraform",
				"tflint":    "/usr/local/bin/tflint",
			},
			commands: map[string]string{
				"terraform --version": "Terraform v1.13.1\non darwin_arm64",
				"tflint --version":    "TFLint version 0.50.0",
				// mise which succeeds for tools on PATH
				"mise which terraform": "/home/user/.local/share/mise/installs/terraform/1.13.1/bin/terraform",
				"mise which tflint":    "/home/user/.local/share/mise/installs/tflint/0.50.0/bin/tflint",
			},
		}

		checker := NewChecker(runner, DefaultTools())
		report := checker.CheckAll()

		assert.True(t, report.MiseFound)

		// Build a map for easier assertions
		results := make(map[string]ToolResult)
		for _, r := range report.Tools {
			results[r.Binary] = r
		}

		assert.Equal(t, StatusInstalled, results["terraform"].Status)
		assert.Equal(t, "1.13.1", results["terraform"].Version)
		assert.Equal(t, StatusInstalled, results["tflint"].Status)
		assert.Equal(t, "0.50.0", results["tflint"].Version)
		assert.Equal(t, StatusMiseManaged, results["trivy"].Status)
		assert.Equal(t, StatusMiseManaged, results["pre-commit"].Status)
		assert.Equal(t, StatusMiseManaged, results["aws"].Status)
	})

	t.Run("tool installed by mise but not on PATH", func(t *testing.T) {
		runner := &mockRunner{
			paths: map[string]string{
				"mise":      "/usr/local/bin/mise",
				"terraform": "/usr/local/bin/terraform",
			},
			commands: map[string]string{
				"terraform --version":  "Terraform v1.13.1\non darwin_arm64",
				"mise which terraform": "/home/user/.local/share/mise/installs/terraform/1.13.1/bin/terraform",
				// mise which trivy succeeds — installed but not on PATH
				"mise which trivy": "/home/user/.local/share/mise/installs/trivy/0.58.2/trivy",
				// mise which tflint not present — not installed at all
			},
		}

		checker := NewChecker(runner, []ToolDef{
			{Name: "Terraform", Binary: "terraform", VersionCmd: []string{"--version"}, MisePlugin: "terraform", Required: true},
			{Name: "Trivy", Binary: "trivy", VersionCmd: []string{"--version"}, MisePlugin: "trivy", Required: true},
			{Name: "TFLint", Binary: "tflint", VersionCmd: []string{"--version"}, MisePlugin: "tflint", Required: true},
		})
		report := checker.CheckAll()

		results := make(map[string]ToolResult)
		for _, r := range report.Tools {
			results[r.Binary] = r
		}

		assert.Equal(t, StatusInstalled, results["terraform"].Status)
		assert.Equal(t, StatusMiseInstalled, results["trivy"].Status,
			"trivy should be mise-installed (installed but not on PATH)")
		assert.Equal(t, StatusMiseManaged, results["tflint"].Status,
			"tflint should be mise-managed (not installed at all)")
	})

	t.Run("tool on global PATH when mise available", func(t *testing.T) {
		runner := &mockRunner{
			paths: map[string]string{
				"mise":      "/usr/local/bin/mise",
				"terraform": "/usr/bin/terraform", // global PATH, not from mise
			},
			commands: map[string]string{
				"terraform --version": "Terraform v1.13.1\non darwin_arm64",
				// mise which terraform fails — not managed by mise
			},
		}

		checker := NewChecker(runner, []ToolDef{
			{Name: "Terraform", Binary: "terraform", VersionCmd: []string{"--version"}, MisePlugin: "terraform", Required: true},
		})
		report := checker.CheckAll()

		require.Len(t, report.Tools, 1)
		assert.Equal(t, StatusGlobalPath, report.Tools[0].Status)
		assert.Equal(t, "1.13.1", report.Tools[0].Version)
	})

	t.Run("tool on PATH via mise stays installed", func(t *testing.T) {
		runner := &mockRunner{
			paths: map[string]string{
				"mise":      "/usr/local/bin/mise",
				"terraform": "/home/user/.local/share/mise/installs/terraform/1.13.1/bin/terraform",
			},
			commands: map[string]string{
				"terraform --version":  "Terraform v1.13.1\non darwin_arm64",
				"mise which terraform": "/home/user/.local/share/mise/installs/terraform/1.13.1/bin/terraform",
			},
		}

		checker := NewChecker(runner, []ToolDef{
			{Name: "Terraform", Binary: "terraform", VersionCmd: []string{"--version"}, MisePlugin: "terraform", Required: true},
		})
		report := checker.CheckAll()

		require.Len(t, report.Tools, 1)
		assert.Equal(t, StatusInstalled, report.Tools[0].Status)
		assert.Equal(t, "1.13.1", report.Tools[0].Version)
	})

	t.Run("version command fails but tool exists without mise", func(t *testing.T) {
		runner := &mockRunner{
			paths: map[string]string{
				"terraform": "/usr/local/bin/terraform",
			},
			commands: map[string]string{},
			errors: map[string]error{
				"terraform --version": errors.New("exit status 1"),
			},
		}

		checker := NewChecker(runner, []ToolDef{
			{Name: "Terraform", Binary: "terraform", VersionCmd: []string{"--version"}, MisePlugin: "terraform", Required: true},
		})
		report := checker.CheckAll()

		require.Len(t, report.Tools, 1)
		assert.Equal(t, StatusInstalled, report.Tools[0].Status, "tool should still be installed even if version check fails (no mise)")
		assert.Empty(t, report.Tools[0].Version, "version should be empty when version check fails")
	})

	t.Run("mise which succeeds but version command fails", func(t *testing.T) {
		// mise which succeeds (tool is configured in mise), but the version
		// command fails. Keep StatusInstalled so the user sees an accurate
		// diagnostic — mise manages it but something is wrong with the install.
		runner := &mockRunner{
			paths: map[string]string{
				"mise":   "/usr/local/bin/mise",
				"tflint": "/home/user/.local/share/mise/shims/tflint",
			},
			commands: map[string]string{
				"mise which tflint": "/home/user/.local/share/mise/installs/tflint/0.61.0/bin/tflint",
			},
			errors: map[string]error{
				"tflint --version": errors.New("exit status 1"),
			},
		}

		checker := NewChecker(runner, []ToolDef{
			{Name: "TFLint", Binary: "tflint", VersionCmd: []string{"--version"}, MisePlugin: "tflint", Required: true},
		})
		report := checker.CheckAll()

		require.Len(t, report.Tools, 1)
		assert.Equal(t, StatusInstalled, report.Tools[0].Status,
			"mise which succeeded so tool is installed; version failure is a separate issue")
		assert.Empty(t, report.Tools[0].Version)
	})

	t.Run("mise shim on PATH but mise which fails and version fails", func(t *testing.T) {
		// Reproduces: mise shim exists on PATH (LookPath succeeds), but
		// "mise which" fails (tool not installed in mise) AND the version
		// command also fails (shim can't resolve to a real binary).
		// This should be StatusMiseManaged, not StatusGlobalPath.
		runner := &mockRunner{
			paths: map[string]string{
				"mise":   "/usr/local/bin/mise",
				"tflint": "/home/user/.local/share/mise/shims/tflint",
			},
			commands: map[string]string{
				// mise which fails — tool not installed via mise
			},
			errors: map[string]error{
				"mise which tflint": errors.New("tflint is not a mise bin"),
				"tflint --version":  errors.New("exit status 1"),
			},
		}

		checker := NewChecker(runner, []ToolDef{
			{Name: "TFLint", Binary: "tflint", VersionCmd: []string{"--version"}, MisePlugin: "tflint", Required: true},
		})
		report := checker.CheckAll()

		require.Len(t, report.Tools, 1)
		assert.Equal(t, StatusMiseManaged, report.Tools[0].Status,
			"shim on PATH with failing mise-which and failing version should be mise-managed")
		assert.Empty(t, report.Tools[0].Version)
	})

	t.Run("version output with no semver match", func(t *testing.T) {
		runner := &mockRunner{
			paths: map[string]string{
				"terraform": "/usr/local/bin/terraform",
			},
			commands: map[string]string{
				"terraform --version": "terraform development build",
			},
		}

		checker := NewChecker(runner, []ToolDef{
			{Name: "Terraform", Binary: "terraform", VersionCmd: []string{"--version"}, MisePlugin: "terraform", Required: true},
		})
		report := checker.CheckAll()

		require.Len(t, report.Tools, 1)
		assert.Equal(t, StatusInstalled, report.Tools[0].Status)
		assert.Empty(t, report.Tools[0].Version)
	})

	t.Run("empty tool list", func(t *testing.T) {
		runner := &mockRunner{
			paths:    map[string]string{},
			commands: map[string]string{},
		}

		checker := NewChecker(runner, nil)
		report := checker.CheckAll()

		assert.Empty(t, report.Tools)
		assert.False(t, report.MiseFound)
	})

	t.Run("preserves tool metadata in results", func(t *testing.T) {
		runner := &mockRunner{
			paths: map[string]string{
				"aws": "/usr/local/bin/aws",
			},
			commands: map[string]string{
				"aws --version": "aws-cli/2.15.0 Python/3.11.6",
			},
		}

		tools := []ToolDef{
			{Name: "AWS CLI", Binary: "aws", VersionCmd: []string{"--version"}, MisePlugin: "awscli", Required: false},
		}
		checker := NewChecker(runner, tools)
		report := checker.CheckAll()

		require.Len(t, report.Tools, 1)
		assert.Equal(t, "AWS CLI", report.Tools[0].Name)
		assert.Equal(t, "aws", report.Tools[0].Binary)
		assert.Equal(t, "awscli", report.Tools[0].MisePlugin)
		assert.False(t, report.Tools[0].Required)
	})
}

// --- extractVersion tests ---

func TestExtractVersion(t *testing.T) {
	tests := []struct {
		name     string
		output   string
		expected string
	}{
		{
			name:     "terraform version output",
			output:   "Terraform v1.13.1\non linux_amd64\n+ provider registry.terraform.io/hashicorp/aws v5.0.0",
			expected: "1.13.1",
		},
		{
			name:     "tflint version output",
			output:   "TFLint version 0.50.0\n+ ruleset.aws (0.45.0)",
			expected: "0.50.0",
		},
		{
			name:     "trivy version output",
			output:   "Version: 0.58.2\nVulnerability DB:\n  Version: 2",
			expected: "0.58.2",
		},
		{
			name:     "pre-commit version output",
			output:   "pre-commit 3.6.0",
			expected: "3.6.0",
		},
		{
			name:     "aws cli version output",
			output:   "aws-cli/2.15.0 Python/3.11.6 Linux/6.1.0 source/x86_64",
			expected: "2.15.0",
		},
		{
			name:     "version with v prefix",
			output:   "v1.2.3",
			expected: "1.2.3",
		},
		{
			name:     "no version in output",
			output:   "some tool - development build",
			expected: "",
		},
		{
			name:     "empty output",
			output:   "",
			expected: "",
		},
		{
			name:     "version buried in multiline output",
			output:   "Tool Name\nVersion: 12.34.56\nBuilt: 2024-01-01",
			expected: "12.34.56",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, extractVersion(tt.output))
		})
	}
}

// --- DefaultTools tests ---

func TestDefaultTools(t *testing.T) {
	tools := DefaultTools()

	assert.Len(t, tools, 5, "should have 5 default tools")

	// Verify each tool has required fields populated
	for _, tool := range tools {
		assert.NotEmpty(t, tool.Name, "tool name should not be empty")
		assert.NotEmpty(t, tool.Binary, "tool binary should not be empty")
		assert.NotEmpty(t, tool.VersionCmd, "tool version command should not be empty")
		assert.NotEmpty(t, tool.MisePlugin, "tool mise plugin should not be empty")
	}

	// Verify required tools
	requiredBinaries := map[string]bool{
		"terraform":  true,
		"tflint":     true,
		"trivy":      true,
		"pre-commit": true,
	}
	for _, tool := range tools {
		if expected, ok := requiredBinaries[tool.Binary]; ok {
			assert.Equal(t, expected, tool.Required, "tool %s required mismatch", tool.Name)
		}
	}

	// AWS CLI should be optional
	var awsTool *ToolDef
	for i := range tools {
		if tools[i].Binary == "aws" {
			awsTool = &tools[i]
			break
		}
	}
	require.NotNil(t, awsTool, "AWS CLI should be in default tools")
	assert.False(t, awsTool.Required, "AWS CLI should be optional")
}

// --- FormatReport tests ---

func TestFormatReport(t *testing.T) {
	t.Run("all installed with mise", func(t *testing.T) {
		report := &Report{
			MiseFound: true,
			Tools: []ToolResult{
				{Name: "Terraform", Status: StatusInstalled, Version: "1.13.1"},
				{Name: "TFLint", Status: StatusInstalled, Version: "0.50.0"},
			},
		}

		output := FormatReport(report)
		assert.Contains(t, output, "✓ Terraform")
		assert.Contains(t, output, "v1.13.1")
		assert.Contains(t, output, "✓ TFLint")
		assert.Contains(t, output, "v0.50.0")
		assert.Contains(t, output, "mise: ✓ installed")
		assert.Contains(t, output, "All tools are installed")
		assert.NotContains(t, output, "mise install")
	})

	t.Run("missing tools without mise", func(t *testing.T) {
		report := &Report{
			MiseFound: false,
			Tools: []ToolResult{
				{Name: "Terraform", Status: StatusMissing},
				{Name: "Trivy", Status: StatusMissing},
			},
		}

		output := FormatReport(report)
		assert.Contains(t, output, "✗ Terraform")
		assert.Contains(t, output, "✗ Trivy")
		assert.Contains(t, output, "mise: ✗ not found")
		assert.Contains(t, output, "brew install mise")
		assert.Contains(t, output, "curl https://mise.run | sh")
	})

	t.Run("mise-managed tools", func(t *testing.T) {
		report := &Report{
			MiseFound: true,
			Tools: []ToolResult{
				{Name: "Terraform", Status: StatusInstalled, Version: "1.13.1"},
				{Name: "Trivy", Status: StatusMiseManaged},
			},
		}

		output := FormatReport(report)
		assert.Contains(t, output, "✓ Terraform")
		assert.Contains(t, output, "↓ Trivy")
		assert.Contains(t, output, "will be installed by mise")
	})

	t.Run("mise-installed tools show activation hint for zsh", func(t *testing.T) {
		report := &Report{
			MiseFound: true,
			Shell:     "zsh",
			Tools: []ToolResult{
				{Name: "Terraform", Status: StatusInstalled, Version: "1.13.1"},
				{Name: "Trivy", Status: StatusMiseInstalled},
			},
		}

		output := FormatReport(report)
		assert.Contains(t, output, "! Trivy")
		assert.Contains(t, output, "installed by mise, but not activated")
		assert.Contains(t, output, "mise activate zsh")
		assert.Contains(t, output, "~/.zshrc")
		assert.NotContains(t, output, "mise install")
	})

	t.Run("mise-installed tools show activation hint for bash", func(t *testing.T) {
		report := &Report{
			MiseFound: true,
			Shell:     "bash",
			Tools: []ToolResult{
				{Name: "Trivy", Status: StatusMiseInstalled},
			},
		}

		output := FormatReport(report)
		assert.Contains(t, output, "mise activate bash")
		assert.Contains(t, output, "~/.bashrc")
	})

	t.Run("mise-installed tools show activation hint for fish", func(t *testing.T) {
		report := &Report{
			MiseFound: true,
			Shell:     "fish",
			Tools: []ToolResult{
				{Name: "Trivy", Status: StatusMiseInstalled},
			},
		}

		output := FormatReport(report)
		assert.Contains(t, output, "mise activate fish | source")
		assert.Contains(t, output, "config.fish")
	})

	t.Run("mise-installed tools show all shells when shell unknown", func(t *testing.T) {
		report := &Report{
			MiseFound: true,
			Shell:     "",
			Tools: []ToolResult{
				{Name: "Trivy", Status: StatusMiseInstalled},
			},
		}

		output := FormatReport(report)
		assert.Contains(t, output, "mise activate bash")
		assert.Contains(t, output, "mise activate zsh")
		assert.Contains(t, output, "mise activate fish")
	})

	t.Run("mise-installed with mise already activated shows reshim hint", func(t *testing.T) {
		report := &Report{
			MiseFound:     true,
			MiseActivated: true,
			Shell:         "zsh",
			Tools: []ToolResult{
				{Name: "Trivy", Status: StatusMiseInstalled},
			},
		}

		output := FormatReport(report)
		assert.Contains(t, output, "mise reshim && exec $SHELL")
		assert.NotContains(t, output, "mise activate")
	})

	t.Run("mise-installed without activation shows activate and restart hints", func(t *testing.T) {
		report := &Report{
			MiseFound:     true,
			MiseActivated: false,
			Shell:         "zsh",
			Tools: []ToolResult{
				{Name: "Trivy", Status: StatusMiseInstalled},
			},
		}

		output := FormatReport(report)
		assert.Contains(t, output, "mise activate zsh")
		assert.Contains(t, output, "exec $SHELL")
	})

	t.Run("mixed mise-installed and mise-managed shows both hints", func(t *testing.T) {
		report := &Report{
			MiseFound: true,
			Shell:     "zsh",
			Tools: []ToolResult{
				{Name: "Trivy", Status: StatusMiseInstalled},
				{Name: "TFLint", Status: StatusMiseManaged},
			},
		}

		output := FormatReport(report)
		assert.Contains(t, output, "! Trivy")
		assert.Contains(t, output, "↓ TFLint")
		assert.Contains(t, output, "mise install")
		assert.Contains(t, output, "mise activate zsh")
	})

	t.Run("global PATH tools show tilde and mise install hint", func(t *testing.T) {
		report := &Report{
			MiseFound: true,
			Tools: []ToolResult{
				{Name: "Terraform", Status: StatusGlobalPath, Version: "1.13.1"},
			},
		}

		output := FormatReport(report)
		assert.Contains(t, output, "~ Terraform")
		assert.Contains(t, output, "v1.13.1")
		assert.Contains(t, output, "Installed globally (mise install will add .mise.toml version):")
		assert.Contains(t, output, "Run: mise install")
		assert.NotContains(t, output, "All tools are installed")
	})

	t.Run("mix of installed and global PATH tools", func(t *testing.T) {
		report := &Report{
			MiseFound: true,
			Tools: []ToolResult{
				{Name: "Terraform", Status: StatusInstalled, Version: "1.13.1"},
				{Name: "TFLint", Status: StatusGlobalPath, Version: "0.50.0"},
			},
		}

		output := FormatReport(report)
		assert.Contains(t, output, "✓ Terraform")
		assert.Contains(t, output, "~ TFLint")
		assert.Contains(t, output, "Run: mise install")
		assert.NotContains(t, output, "All tools are installed")
	})

	t.Run("global PATH tool without version", func(t *testing.T) {
		report := &Report{
			MiseFound: true,
			Tools: []ToolResult{
				{Name: "Terraform", Status: StatusGlobalPath, Version: ""},
			},
		}

		output := FormatReport(report)
		assert.Contains(t, output, "~ Terraform")
		assert.Contains(t, output, "Installed globally (mise install will add .mise.toml version):")
		assert.NotContains(t, output, "(on PATH")
	})

	t.Run("installed tool without version", func(t *testing.T) {
		report := &Report{
			Tools: []ToolResult{
				{Name: "Terraform", Status: StatusInstalled, Version: ""},
			},
		}

		output := FormatReport(report)
		assert.Contains(t, output, "✓ Terraform")
		assert.Contains(t, output, "(installed)")
	})

	t.Run("empty report", func(t *testing.T) {
		report := &Report{
			Tools: []ToolResult{},
		}

		output := FormatReport(report)
		assert.Contains(t, output, "Running pre-flight checks...")
	})

	t.Run("alignment with varying name lengths", func(t *testing.T) {
		report := &Report{
			Tools: []ToolResult{
				{Name: "Terraform", Status: StatusInstalled, Version: "1.13.1"},
				{Name: "pre-commit", Status: StatusInstalled, Version: "3.6.0"},
				{Name: "AWS CLI", Status: StatusMissing},
			},
		}

		output := FormatReport(report)
		// All lines should be present, alignment is visual
		assert.Contains(t, output, "Terraform")
		assert.Contains(t, output, "pre-commit")
		assert.Contains(t, output, "AWS CLI")
	})
}

// --- detectShell tests ---

func TestDetectShell(t *testing.T) {
	t.Run("extracts basename from SHELL env var", func(t *testing.T) {
		t.Setenv("SHELL", "/bin/zsh")
		assert.Equal(t, "zsh", detectShell())
	})

	t.Run("handles bash", func(t *testing.T) {
		t.Setenv("SHELL", "/usr/bin/bash")
		assert.Equal(t, "bash", detectShell())
	})

	t.Run("handles fish", func(t *testing.T) {
		t.Setenv("SHELL", "/usr/local/bin/fish")
		assert.Equal(t, "fish", detectShell())
	})

	t.Run("returns empty when SHELL is unset", func(t *testing.T) {
		t.Setenv("SHELL", "")
		assert.Equal(t, "", detectShell())
	})
}

// --- shellRCFile tests ---

func TestShellRCFile(t *testing.T) {
	assert.Equal(t, "~/.zshrc", shellRCFile("zsh"))
	assert.Equal(t, "~/.bashrc", shellRCFile("bash"))
	assert.Equal(t, "~/.config/fish/config.fish", shellRCFile("fish"))
}

// --- miseActivationHint tests ---

func TestMiseActivationHint(t *testing.T) {
	t.Run("zsh", func(t *testing.T) {
		hint := miseActivationHint("zsh")
		assert.Contains(t, hint, `mise activate zsh`)
		assert.Contains(t, hint, "~/.zshrc")
	})

	t.Run("bash", func(t *testing.T) {
		hint := miseActivationHint("bash")
		assert.Contains(t, hint, `mise activate bash`)
		assert.Contains(t, hint, "~/.bashrc")
	})

	t.Run("fish", func(t *testing.T) {
		hint := miseActivationHint("fish")
		assert.Contains(t, hint, "mise activate fish | source")
		assert.Contains(t, hint, "config.fish")
	})

	t.Run("unknown shell shows all options", func(t *testing.T) {
		hint := miseActivationHint("")
		assert.Contains(t, hint, "mise activate bash")
		assert.Contains(t, hint, "mise activate zsh")
		assert.Contains(t, hint, "mise activate fish")
	})
}
