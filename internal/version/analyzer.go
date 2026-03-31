package version

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
			report.Summary.FilesWithDrift++
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
// and determines if they target the same major.minor version.
func (a *Analyzer) compareSemverConstraints(expected, actual string) DriftStatus {
	expectedMajor, expectedMinor := extractVersionNumbers(expected)
	actualMajor, actualMinor := extractVersionNumbers(actual)

	// If we can't parse, assume drift for safety
	if expectedMajor == -1 || actualMajor == -1 {
		return StatusDrift
	}

	if expectedMajor != actualMajor || expectedMinor != actualMinor {
		return StatusDrift
	}

	// Same major.minor — treat as in-sync regardless of constraint operator
	// or patch version (e.g., "1.13.1" vs "~> 1.13" are both targeting 1.13).
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
	if r.FilesWithDrift == 0 {
		return fmt.Sprintf("All %d files are in sync", r.TotalFiles)
	}

	msg := fmt.Sprintf("%d of %d files have drift", r.FilesWithDrift, r.TotalFiles)
	if r.Summary.FilesWithErrors > 0 {
		msg += fmt.Sprintf(", %d files with errors", r.Summary.FilesWithErrors)
	}

	return msg
}
