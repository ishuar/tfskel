package validate

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ishuar/tfskel/internal/config"
	"github.com/ishuar/tfskel/internal/templates"
	"github.com/ishuar/tfskel/internal/version"
)

// RunConfigCheck detects version constraint drift in .tf files against .tfskel.yaml
// and checks .terraform-version files for version mismatches.
func RunConfigCheck(cfg *config.Config, scanDir string) ([]Finding, CheckResult, error) {
	detector := version.NewDetector(scanDir)
	versionInfos, err := detector.ScanDirectory()
	if err != nil {
		return nil, CheckResult{Check: CheckConfig, Status: StatusError},
			fmt.Errorf("failed to scan directory: %w", err)
	}

	var findings []Finding
	totalFiles := 0

	if len(versionInfos) > 0 {
		analyzer := version.NewAnalyzer(cfg)
		report := analyzer.Analyze(scanDir, versionInfos)
		totalFiles = report.TotalFiles

		for _, record := range report.Records {
			if !record.HasDrift {
				continue
			}

			// Terraform version drift
			if record.TerraformDriftStatus != version.StatusInSync &&
				record.TerraformDriftStatus != version.StatusNotManaged {
				findings = append(findings, Finding{
					Check:     CheckConfig,
					Resource:  record.FilePath,
					Component: "terraform",
					Message:   "version constraint drift",
					Expected:  record.TerraformExpected,
					Actual:    record.TerraformActual,
				})
			}

			// Provider drift
			for _, pd := range record.Providers {
				if pd.DriftStatus == version.StatusInSync || pd.DriftStatus == version.StatusNotManaged {
					continue
				}
				findings = append(findings, Finding{
					Check:     CheckConfig,
					Resource:  record.FilePath,
					Component: pd.Name,
					Message:   "version constraint drift",
					Expected:  pd.Expected,
					Actual:    pd.Actual,
				})
			}
		}
	}

	// Check .terraform-version files for version mismatches.
	tvFindings, tvCount := checkTerraformVersionFiles(cfg, scanDir)
	findings = append(findings, tvFindings...)
	totalFiles += tvCount

	// Count unique files with findings so Passed reflects file-level metrics.
	uniqueFiles := make(map[string]bool, len(findings))
	for _, f := range findings {
		uniqueFiles[f.Resource] = true
	}

	result := CheckResult{
		Check:             CheckConfig,
		Total:             totalFiles,
		Passed:            totalFiles - len(uniqueFiles),
		Issues:            len(findings),
		AffectedResources: len(uniqueFiles),
	}
	if len(findings) > 0 {
		result.Status = StatusFail
	} else {
		result.Status = StatusPass
	}

	return findings, result, nil
}

// checkTerraformVersionFiles walks the scan directory for .terraform-version files
// and compares their content against the configured terraform version.
func checkTerraformVersionFiles(cfg *config.Config, scanDir string) ([]Finding, int) {
	if cfg == nil {
		return nil, 0
	}
	expectedVersion := templates.StripConstraint(cfg.TerraformVersion)
	if expectedVersion == "" {
		return nil, 0
	}

	var findings []Finding
	count := 0

	// Walk the directory tree looking for .terraform-version files.
	if err := filepath.WalkDir(scanDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip hidden directories (except the root).
		if d.IsDir() && d.Name() != "." && strings.HasPrefix(d.Name(), ".") {
			return filepath.SkipDir
		}

		if d.Name() != ".terraform-version" {
			return nil
		}

		count++

		data, readErr := os.ReadFile(path)
		if readErr != nil {
			relPath := path
			if rel, relErr := filepath.Rel(scanDir, path); relErr == nil {
				relPath = rel
			}
			findings = append(findings, Finding{
				Check:     CheckConfig,
				Resource:  relPath,
				Component: "terraform-version-file",
				Message:   "could not read .terraform-version file",
				Detail:    readErr.Error(),
			})
			return nil
		}

		actual := strings.TrimSpace(string(data))
		if actual == "" {
			return nil
		}

		if !strings.HasPrefix(actual, expectedVersion) {
			relPath := path
			if rel, relErr := filepath.Rel(scanDir, path); relErr == nil {
				relPath = rel
			}
			findings = append(findings, Finding{
				Check:     CheckConfig,
				Resource:  relPath,
				Component: "terraform-version-file",
				Message:   "version mismatch",
				Expected:  expectedVersion,
				Actual:    actual,
				Detail:    "run 'tfskel init --upgrade' to update",
			})
		}

		return nil
	}); err != nil {
		return nil, 0
	}

	return findings, count
}
