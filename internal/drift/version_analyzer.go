package drift

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/ishuar/tfskel/internal/config"
)

// Analyzer compares detected versions against expected config
type Analyzer struct {
	config *config.Config
}

// NewAnalyzer creates a new drift analyzer
func NewAnalyzer(cfg *config.Config) *Analyzer {
	return &Analyzer{
		config: cfg,
	}
}

// Analyze compares detected version information with expected configuration
func (a *Analyzer) Analyze(scanRoot string, versionInfos []VersionInfo) *DriftReport {
	report := &DriftReport{
		ScannedAt:  time.Now(),
		ScanRoot:   scanRoot,
		TotalFiles: len(versionInfos),
		Summary: DriftSummary{
			TotalFiles:        len(versionInfos),
			TerraformVersions: make(map[string]int),
			ProviderVersions:  make(map[string]map[string]int),
		},
	}

	for _, info := range versionInfos {
		if info.ParseError != nil {
			report.Summary.FilesWithErrors++
			continue
		}

		record := a.analyzeVersionInfo(info)
		report.Records = append(report.Records, record)

		// Update summary statistics
		if record.HasDrift {
			report.FilesWithDrift++
			categorizeDriftSeverity(&report.Summary, record)
		} else {
			report.Summary.FilesInSync++
		}

		// Aggregate version counts
		if info.TerraformVersion != "" {
			report.Summary.TerraformVersions[info.TerraformVersion]++
		}

		for providerName, providerVer := range info.Providers {
			if report.Summary.ProviderVersions[providerName] == nil {
				report.Summary.ProviderVersions[providerName] = make(map[string]int)
			}
			report.Summary.ProviderVersions[providerName][providerVer.Version]++
		}
	}

	return report
}

// categorizeDriftSeverity determines if a drift is major or minor and updates summary counts
func categorizeDriftSeverity(summary *DriftSummary, record DriftRecord) {
	hasMajor := record.TerraformDriftStatus == StatusMajorDrift

	if !hasMajor {
		for _, pd := range record.Providers {
			if pd.DriftStatus == StatusMajorDrift {
				hasMajor = true
				break
			}
		}
	}

	if hasMajor {
		summary.FilesWithMajorDrift++
	} else {
		summary.FilesWithMinorDrift++
	}
}

// analyzeVersionInfo compares a single version info against config
func (a *Analyzer) analyzeVersionInfo(info VersionInfo) DriftRecord {
	record := DriftRecord{
		FilePath:             info.FilePath,
		TerraformExpected:    a.config.TerraformVersion,
		TerraformActual:      info.TerraformVersion,
		TerraformDriftStatus: a.compareTerraformVersion(a.config.TerraformVersion, info.TerraformVersion),
	}

	// Analyze providers
	for providerName, providerVer := range info.Providers {
		var expected string

		// Currently only support AWS provider from config
		// Can be extended to support more providers
		if providerName == "aws" && a.config.Provider != nil && a.config.Provider.AWS != nil {
			expected = a.config.Provider.AWS.Version
		}

		drift := ProviderDrift{
			Name:        providerName,
			Source:      providerVer.Source,
			Expected:    expected,
			Actual:      providerVer.Version,
			DriftStatus: a.compareProviderVersion(expected, providerVer.Version),
		}

		if drift.DriftStatus != StatusInSync && drift.DriftStatus != StatusNotManaged {
			record.HasDrift = true
		}

		record.Providers = append(record.Providers, drift)
	}

	// Check terraform version drift
	if record.TerraformDriftStatus != StatusInSync {
		record.HasDrift = true
	}

	return record
}

// compareTerraformVersion compares terraform versions and returns drift status
func (a *Analyzer) compareTerraformVersion(expected, actual string) DriftStatus {
	if actual == "" {
		return StatusMissing
	}

	if expected == actual {
		return StatusInSync
	}

	// Parse version constraints
	severity := a.compareSemverConstraints(expected, actual)
	return severity
}

// compareProviderVersion compares provider versions
func (a *Analyzer) compareProviderVersion(expected, actual string) DriftStatus {
	if expected == "" {
		return StatusNotManaged
	}

	if actual == "" {
		return StatusMissing
	}

	if expected == actual {
		return StatusInSync
	}

	severity := a.compareSemverConstraints(expected, actual)
	return severity
}

// compareSemverConstraints compares two version constraint strings
// and determines if it's major or minor drift
func (a *Analyzer) compareSemverConstraints(expected, actual string) DriftStatus {
	expectedMajor, expectedMinor := extractVersionNumbers(expected)
	actualMajor, actualMinor := extractVersionNumbers(actual)

	// If we can't parse, assume major drift for safety
	if expectedMajor == -1 || actualMajor == -1 {
		return StatusMajorDrift
	}

	// Major version difference
	if expectedMajor != actualMajor {
		return StatusMajorDrift
	}

	// Minor version difference (or patch)
	if expectedMinor != actualMinor {
		return StatusMinorDrift
	}

	// Different constraints but same version numbers (e.g., "= 1.13" vs "~> 1.13")
	if expected != actual {
		return StatusMinorDrift
	}

	return StatusInSync
}

// extractVersionNumbers parses version strings like "~> 1.13", ">= 5.0", "= 6.0"
// and returns major and minor version numbers
func extractVersionNumbers(version string) (major, minor int) {
	// Remove constraint operators
	version = strings.TrimSpace(version)
	version = regexp.MustCompile(`^[~>=<!\s]+`).ReplaceAllString(version, "")

	// Split by dot
	parts := strings.Split(version, ".")

	major = -1
	minor = -1

	if len(parts) > 0 {
		if m, err := strconv.Atoi(parts[0]); err == nil {
			major = m
		}
	}

	if len(parts) > 1 {
		if m, err := strconv.Atoi(parts[1]); err == nil {
			minor = m
		}
	}

	return major, minor
}

// HasCriticalDrift checks if there's any major version drift
func (r *DriftReport) HasCriticalDrift() bool {
	return r.Summary.FilesWithMajorDrift > 0
}

// ExitCode returns appropriate exit code for CI/CD
// 0 = no drift, 1 = drift detected, 2 = errors
func (r *DriftReport) ExitCode() int {
	if r.Summary.FilesWithErrors > 0 {
		return 2
	}
	if r.FilesWithDrift > 0 {
		return 1
	}
	return 0
}

// GetDriftSummaryText returns a human-readable summary
func (r *DriftReport) GetDriftSummaryText() string {
	major := r.Summary.FilesWithMajorDrift
	minor := r.Summary.FilesWithMinorDrift

	var msg string
	if r.FilesWithDrift == 0 {
		msg = fmt.Sprintf("All %d files are in sync", r.TotalFiles)
	} else {
		msg = fmt.Sprintf("%d of %d files have drift (minor: %d, major: %d)",
			r.FilesWithDrift, r.TotalFiles, minor, major)

		// Add error info inline if present
		if r.Summary.FilesWithErrors > 0 {
			msg += fmt.Sprintf(", %d files with errors", r.Summary.FilesWithErrors)
		}
	}

	return msg
}
