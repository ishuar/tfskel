package validate

import (
	"errors"
	"fmt"
	"strings"

	"github.com/ishuar/tfskel/internal/config"
)

// ErrUnknownCheck indicates an invalid check name was provided.
var ErrUnknownCheck = errors.New("unknown check")

// Runner orchestrates validation checks.
type Runner struct {
	cfg    *config.Config
	dir    string
	checks map[CheckName]bool // which checks to run (true = run)
}

// NewRunner creates a runner that will execute the specified checks.
// If checks is nil, all checks are run.
func NewRunner(cfg *config.Config, dir string, checks map[CheckName]bool) *Runner {
	if checks == nil {
		checks = map[CheckName]bool{
			CheckConfig: true,
			CheckTools:  true,
		}
	}

	return &Runner{
		cfg:    cfg,
		dir:    dir,
		checks: checks,
	}
}

// Run executes all selected checks and returns a unified report.
func (r *Runner) Run() *Report {
	report := &Report{Directory: r.dir}

	for _, check := range AllChecks() {
		if !r.checks[check] {
			report.Checks = append(report.Checks, CheckResult{
				Check:  check,
				Status: StatusSkipped,
			})
			continue
		}

		findings, result, err := r.runCheck(check, report)
		if err != nil {
			report.Checks = append(report.Checks, CheckResult{
				Check:  check,
				Status: StatusError,
			})
			report.Findings = append(report.Findings, Finding{
				Check:    check,
				Resource: string(check),
				Message:  fmt.Sprintf("check failed: %v", err),
			})
			continue
		}

		report.Checks = append(report.Checks, result)
		report.Findings = append(report.Findings, findings...)
	}

	return report
}

// runCheck dispatches to the appropriate check function.
func (r *Runner) runCheck(check CheckName, report *Report) ([]Finding, CheckResult, error) {
	switch check {
	case CheckConfig:
		return RunConfigCheck(r.cfg, r.dir)
	case CheckTools:
		findings, result, toolReport, err := RunToolCheck(r.dir)
		if err == nil {
			report.ToolReport = toolReport
		}
		return findings, result, err
	default:
		return nil, CheckResult{}, fmt.Errorf("%w: %s", ErrUnknownCheck, check)
	}
}

// ParseCheckSelection parses the --skip flag value into a check map.
// Returns nil (run all) when skip is empty.
func ParseCheckSelection(skip string) (map[CheckName]bool, error) {
	if skip == "" {
		return nil, nil //nolint:nilnil // nil means "run all checks"
	}

	names := strings.Split(skip, ",")
	checks := map[CheckName]bool{
		CheckConfig: true,
		CheckTools:  true,
	}
	for _, name := range names {
		name = strings.TrimSpace(name)
		if !ValidCheckName(name) {
			return nil, fmt.Errorf("%w %q; valid checks: config, tools", ErrUnknownCheck, name)
		}
		delete(checks, CheckName(name))
	}
	return checks, nil
}
