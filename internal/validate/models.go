package validate

import "github.com/ishuar/tfskel/internal/toolcheck"

// CheckName identifies which validation check produced a finding.
type CheckName string

// CheckName constants for the validation checks.
const (
	CheckConfig CheckName = "config"
	CheckTools  CheckName = "tools"
)

// AllChecks returns all valid check names in display order.
func AllChecks() []CheckName {
	return []CheckName{CheckConfig, CheckTools}
}

// ValidCheckName returns true if the given name is a valid check.
func ValidCheckName(name string) bool {
	switch CheckName(name) {
	case CheckConfig, CheckTools:
		return true
	}
	return false
}

// CheckStatus represents the outcome of a single check.
type CheckStatus string

// CheckStatus constants for check outcomes.
const (
	StatusPass    CheckStatus = "pass"
	StatusFail    CheckStatus = "fail"
	StatusError   CheckStatus = "error"
	StatusSkipped CheckStatus = "skipped"
)

// Finding represents a single diff between expected and actual state.
// All findings are equal — there is no severity hierarchy.
type Finding struct {
	Check     CheckName `json:"check"`
	Resource  string    `json:"resource"`            // file path, tool name, or provider name
	Component string    `json:"component,omitempty"` // e.g. "terraform", "aws" for config check
	Message   string    `json:"message"`
	Expected  string    `json:"expected,omitempty"` // what config/mise.toml says
	Actual    string    `json:"actual,omitempty"`   // what was found
	Detail    string    `json:"detail,omitempty"`   // remediation hint
}

// CheckResult holds the summary outcome for a single check.
type CheckResult struct {
	Check             CheckName   `json:"check"`
	Status            CheckStatus `json:"status"`
	Total             int         `json:"total"`
	Passed            int         `json:"passed"`
	Issues            int         `json:"issues"`
	AffectedResources int         `json:"affectedResources,omitempty"` // unique resources with findings (e.g. unique tools)
}

// Report is the top-level validation result.
type Report struct {
	Checks   []CheckResult `json:"checks"`
	Findings []Finding     `json:"findings"`

	// Directory is the scan root (the working directory at invocation time).
	// Distinct from ProjectRoot: when validate is invoked from a subdir of the
	// project, Directory is that subdir while ProjectRoot is filepath.Dir(configPath).
	// The table header renders both lines when they differ.
	Directory string `json:"directory,omitempty"`

	// ProjectRoot is the absolute path to the project root, defined as the
	// directory that contains the loaded .tfskel.yaml.
	ProjectRoot string `json:"projectRoot,omitempty"`

	// ConfigPath is the absolute path to the loaded tfskel config file.
	ConfigPath string `json:"configPath,omitempty"`

	// Environments are the environment names defined in config (sorted).
	Environments []string `json:"environments,omitempty"`

	// Regions are the AWS regions defined in config.
	Regions []string `json:"regions,omitempty"`

	// ToolReport holds the raw toolcheck report for detailed table rendering.
	// Excluded from JSON/CSV output — only used by the table formatter.
	ToolReport *toolcheck.Report `json:"-"`
}

// Exit codes for validate. They follow the same rule as `review plan
// --detailed-exitcode`: 1 always means "something broke", so a CI script
// checking `exit == 1` gets the same answer from every tfskel command.
const (
	// ExitCodeSuccess indicates every check passed (or was skipped).
	ExitCodeSuccess = 0
	// ExitCodeError indicates at least one check could not run.
	ExitCodeError = 1
	// ExitCodeFindings indicates every check ran and at least one reported findings.
	ExitCodeFindings = 2
)

// ExitCode returns the exit code for CI/CD.
// Returns ExitCodeError (1) if any check errored, ExitCodeFindings (2) if any
// check failed, and ExitCodeSuccess (0) otherwise.
func (r *Report) ExitCode() int {
	code := ExitCodeSuccess
	for _, c := range r.Checks {
		switch c.Status {
		case StatusError:
			return ExitCodeError
		case StatusFail:
			code = ExitCodeFindings
		case StatusPass, StatusSkipped:
			// no-op
		}
	}
	return code
}

// IssueCount returns the total number of findings across all checks.
func (r *Report) IssueCount() int {
	n := 0
	for _, c := range r.Checks {
		n += c.Issues
	}
	return n
}
