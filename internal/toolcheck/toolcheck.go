package toolcheck

import (
	"os"
	"path/filepath"
	"regexp"
)

// ToolStatus represents the detection result for a single tool.
type ToolStatus int

const (
	// StatusInstalled indicates the tool was found on PATH; version parsing is
	// attempted but may fail (Version can be empty).
	StatusInstalled ToolStatus = iota
	// StatusMissing indicates the tool was not found on PATH.
	StatusMissing
	// StatusMiseManaged indicates the tool is not installed but mise can install it.
	StatusMiseManaged
	// StatusMiseInstalled indicates the tool is installed by mise but not on PATH
	// (mise is not activated in the current shell).
	StatusMiseInstalled
	// StatusGlobalPath indicates the tool was found on the global PATH but is not
	// managed by mise. Running mise install will add a .mise.toml version.
	StatusGlobalPath
)

// ToolDef defines how to detect and version-check a tool.
type ToolDef struct {
	Name       string   // display name, e.g. "Terraform"
	Binary     string   // binary to look up on PATH, e.g. "terraform"
	VersionCmd []string // args to get version, e.g. ["--version"]
	MisePlugin string   // mise plugin name for .mise.toml
	Required   bool     // whether this tool is essential vs optional
}

// ToolResult holds the detection outcome for one tool.
type ToolResult struct {
	Name       string
	Binary     string
	Status     ToolStatus
	Version    string // parsed version string, empty if not found
	MisePlugin string
	Required   bool
}

// Report holds the full check results.
type Report struct {
	Tools         []ToolResult
	MiseFound     bool
	MiseActivated bool   // true if mise is activated in the current shell session
	Shell         string // detected shell name (e.g. "zsh", "bash", "fish")
}

// versionPattern extracts the first semver-like version from command output.
var versionPattern = regexp.MustCompile(`(\d+\.\d+\.\d+)`)

// DefaultTools returns the standard tool set for tfskel projects.
func DefaultTools() []ToolDef {
	return []ToolDef{
		{Name: "Terraform", Binary: "terraform", VersionCmd: []string{"--version"}, MisePlugin: "terraform", Required: true},
		{Name: "TFLint", Binary: "tflint", VersionCmd: []string{"--version"}, MisePlugin: "tflint", Required: true},
		{Name: "Trivy", Binary: "trivy", VersionCmd: []string{"--version"}, MisePlugin: "trivy", Required: true},
		{Name: "pre-commit", Binary: "pre-commit", VersionCmd: []string{"--version"}, MisePlugin: "pre-commit", Required: true},
		{Name: "AWS CLI", Binary: "aws", VersionCmd: []string{"--version"}, MisePlugin: "awscli", Required: false},
	}
}

// Checker performs tool detection.
type Checker struct {
	runner CommandRunner
	tools  []ToolDef
}

// NewChecker creates a Checker with the given runner and tool definitions.
func NewChecker(runner CommandRunner, tools []ToolDef) *Checker {
	return &Checker{runner: runner, tools: tools}
}

// CheckAll detects all configured tools and returns a Report.
// Individual tool detection failures do not cause CheckAll to return an error.
func (c *Checker) CheckAll() *Report {
	report := &Report{
		Tools: make([]ToolResult, 0, len(c.tools)),
		Shell: detectShell(),
	}

	// Check mise itself first
	_, err := c.runner.LookPath("mise")
	report.MiseFound = err == nil
	report.MiseActivated = os.Getenv("MISE_SHELL") != ""

	for _, tool := range c.tools {
		result := c.checkTool(tool, report.MiseFound)
		report.Tools = append(report.Tools, result)
	}

	return report
}

// checkTool detects a single tool and returns its result.
func (c *Checker) checkTool(def ToolDef, miseFound bool) ToolResult {
	result := ToolResult{
		Name:       def.Name,
		Binary:     def.Binary,
		MisePlugin: def.MisePlugin,
		Required:   def.Required,
	}

	_, err := c.runner.LookPath(def.Binary)
	if err != nil {
		if !miseFound {
			result.Status = StatusMissing
			return result
		}

		// mise is available — check if the tool is already installed via mise
		if _, miseErr := c.runner.RunCommand("mise", "which", def.Binary); miseErr == nil {
			result.Status = StatusMiseInstalled
		} else {
			result.Status = StatusMiseManaged
		}
		return result
	}

	result.Status = StatusInstalled

	// When mise is available, check if this tool comes from mise or global PATH
	if miseFound {
		if _, miseErr := c.runner.RunCommand("mise", "which", def.Binary); miseErr != nil {
			// mise which failed — tool is not from mise, it's on the global PATH
			result.Status = StatusGlobalPath
		}
	}

	// Tool found, try to get version
	output, err := c.runner.RunCommand(def.Binary, def.VersionCmd...)
	if err != nil {
		// If the binary is a mise shim but the underlying tool is not installed,
		// the version command will fail. Only reclassify StatusGlobalPath
		// (where mise which failed) — that points to an unresolved shim.
		// StatusInstalled (where mise which succeeded) means mise manages the
		// tool but something else is wrong; keep the original status so the
		// user sees an accurate diagnostic rather than a false "run mise install".
		if miseFound && result.Status == StatusGlobalPath {
			result.Status = StatusMiseManaged
		}
		return result
	}

	result.Version = extractVersion(output)
	return result
}

// detectShell returns the shell name from the $SHELL environment variable.
// Returns the basename (e.g. "zsh", "bash", "fish") or empty string if unset.
func detectShell() string {
	shell := os.Getenv("SHELL")
	if shell == "" {
		return ""
	}
	return filepath.Base(shell)
}

// extractVersion pulls the first semver match from command output.
func extractVersion(output string) string {
	match := versionPattern.FindStringSubmatch(output)
	if len(match) >= 2 {
		return match[1]
	}
	return ""
}
