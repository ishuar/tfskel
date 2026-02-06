package drift

import "time"

// VersionInfo represents version information extracted from a Terraform file
type VersionInfo struct {
	FilePath         string                 // Relative path from scan root
	TerraformVersion string                 // e.g., "~> 1.13"
	Providers        map[string]ProviderVer // Provider name -> version
	ParseError       error                  // Non-nil if file couldn't be parsed
}

// ProviderVer holds provider version details
type ProviderVer struct {
	Source  string // e.g., "hashicorp/aws"
	Version string // e.g., "~> 6.0"
}

// DriftStatus represents the drift status of a version
type DriftStatus string

const (
	StatusInSync     DriftStatus = "in-sync"
	StatusMinorDrift DriftStatus = "minor-drift"
	StatusMajorDrift DriftStatus = "major-drift"
	StatusMissing    DriftStatus = "missing"
	StatusNotManaged DriftStatus = "not-managed" // Not in .tfskel config
)

// DriftRecord represents a single drift finding
type DriftRecord struct {
	FilePath             string          `json:"filePath"`
	TerraformExpected    string          `json:"terraformExpected"`
	TerraformActual      string          `json:"terraformActual"`
	TerraformDriftStatus DriftStatus     `json:"terraformDriftStatus"`
	Providers            []ProviderDrift `json:"providers"`
	HasDrift             bool            `json:"hasDrift"`
}

// ProviderDrift represents drift for a specific provider
type ProviderDrift struct {
	Name        string      `json:"name"`   // e.g., "aws"
	Source      string      `json:"source"` // e.g., "hashicorp/aws"
	Expected    string      `json:"expected"`
	Actual      string      `json:"actual"`
	DriftStatus DriftStatus `json:"driftStatus"`
}

// DriftReport is the complete analysis result
type DriftReport struct {
	ScannedAt      time.Time     `json:"scannedAt"`
	ScanRoot       string        `json:"scanRoot"`
	TotalFiles     int           `json:"totalFiles"`
	FilesWithDrift int           `json:"filesWithDrift"`
	Records        []DriftRecord `json:"records"`
	Summary        DriftSummary  `json:"summary"`
}

// DriftSummary provides aggregated statistics
type DriftSummary struct {
	TotalFiles          int                       `json:"totalFiles"`
	FilesInSync         int                       `json:"filesInSync"`
	FilesWithMinorDrift int                       `json:"filesWithMinorDrift"`
	FilesWithMajorDrift int                       `json:"filesWithMajorDrift"`
	FilesWithErrors     int                       `json:"filesWithErrors"`
	TerraformVersions   map[string]int            `json:"terraformVersions"` // version -> count
	ProviderVersions    map[string]map[string]int `json:"providerVersions"`  // provider -> version -> count
}
