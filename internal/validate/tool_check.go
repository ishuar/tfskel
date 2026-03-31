package validate

import (
	"errors"
	"strings"

	"github.com/ishuar/tfskel/internal/toolcheck"
)

// RunToolCheck detects missing or inactive required tools and compares installed
// versions against expected versions from .mise.toml.
func RunToolCheck(dir string) ([]Finding, CheckResult, *toolcheck.Report, error) {
	checker := toolcheck.NewChecker(&toolcheck.OSCommandRunner{}, toolcheck.DefaultTools())
	report := checker.CheckAll()

	var findings []Finding
	total := len(report.Tools)

	for _, tool := range report.Tools {
		switch tool.Status {
		case toolcheck.StatusInstalled:
			// OK — no finding needed
		case toolcheck.StatusMissing:
			findings = append(findings, Finding{
				Check:    CheckTools,
				Resource: tool.Name,
				Message:  "tool not found",
				Detail:   "run 'mise install'",
			})
		case toolcheck.StatusMiseManaged:
			findings = append(findings, Finding{
				Check:    CheckTools,
				Resource: tool.Name,
				Message:  "not installed, but mise can install it",
				Detail:   "run 'mise install'",
			})
		case toolcheck.StatusMiseInstalled:
			findings = append(findings, Finding{
				Check:    CheckTools,
				Resource: tool.Name,
				Message:  "installed by mise, but not activated",
				Detail:   "activate mise in your shell, then restart",
			})
		case toolcheck.StatusGlobalPath:
			findings = append(findings, Finding{
				Check:    CheckTools,
				Resource: tool.Name,
				Message:  "installed globally, not managed by mise",
				Detail:   "run 'mise install' to add .mise.toml version",
			})
		}
	}

	// Version mismatch detection: compare installed versions against .mise.toml pins.
	findings = append(findings, checkToolVersions(dir, report)...)

	result := CheckResult{
		Check:  CheckTools,
		Total:  total,
		Passed: total - len(findings),
		Issues: len(findings),
	}
	if len(findings) > 0 {
		result.Status = StatusFail
	} else {
		result.Status = StatusPass
	}

	return findings, result, report, nil
}

// checkToolVersions compares installed tool versions against versions pinned
// in .mise.toml. Returns findings for any mismatches.
func checkToolVersions(dir string, report *toolcheck.Report) []Finding {
	miseConfig, err := ParseMiseToml(dir)
	if err != nil {
		if errors.Is(err, ErrMiseTomlNotFound) {
			return nil
		}
		return nil
	}

	var findings []Finding

	for _, tool := range report.Tools {
		if (tool.Status != toolcheck.StatusInstalled && tool.Status != toolcheck.StatusGlobalPath) || tool.Version == "" {
			continue
		}

		expected, ok := miseConfig.Tools[tool.MisePlugin]
		if !ok || strings.EqualFold(expected, "latest") {
			continue
		}

		if tool.Version != expected {
			findings = append(findings, Finding{
				Check:     CheckTools,
				Resource:  tool.Name,
				Component: "version",
				Message:   "installed version does not match expected",
				Expected:  expected,
				Actual:    tool.Version,
				Detail:    "run 'mise install' to update",
			})
		}
	}

	return findings
}
