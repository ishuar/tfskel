package drift

import (
	"encoding/json"
)

// TerraformPlan represents the structure of terraform plan JSON output
type TerraformPlan struct {
	FormatVersion    string           `json:"format_version"`
	TerraformVersion string           `json:"terraform_version"`
	PlannedValues    map[string]any   `json:"planned_values,omitempty"`
	ResourceChanges  []ResourceChange `json:"resource_changes"`
	OutputChanges    map[string]any   `json:"output_changes,omitempty"`
	PriorState       map[string]any   `json:"prior_state,omitempty"`
	Configuration    map[string]any   `json:"configuration,omitempty"`
}

// ResourceChange represents a change to a resource in the plan
type ResourceChange struct {
	Address       string       `json:"address"`
	ModuleAddress string       `json:"module_address,omitempty"`
	Mode          string       `json:"mode"`
	Type          string       `json:"type"`
	Name          string       `json:"name"`
	ProviderName  string       `json:"provider_name"`
	Change        ChangeDetail `json:"change"`
	ActionReason  string       `json:"action_reason,omitempty"`
}

// ChangeDetail contains the details of what's changing
// Note: before_sensitive and after_sensitive can be either bool or map[string]any
type ChangeDetail struct {
	Actions         []string        `json:"actions"`
	Before          map[string]any  `json:"before"`
	After           map[string]any  `json:"after"`
	AfterUnknown    map[string]any  `json:"after_unknown,omitempty"`
	BeforeSensitive json.RawMessage `json:"before_sensitive,omitempty"`
	AfterSensitive  json.RawMessage `json:"after_sensitive,omitempty"`
}

// PlanAnalysis represents the analyzed plan results
type PlanAnalysis struct {
	TotalChanges     int                `json:"total_changes"`
	Additions        int                `json:"additions"`
	Modifications    int                `json:"modifications"`
	Deletions        int                `json:"deletions"`
	Replacements     int                `json:"replacements"`
	ResourceChanges  []AnalyzedResource `json:"resource_changes"`
	TerraformVersion string             `json:"terraform_version"`
	HasChanges       bool               `json:"has_changes"`
	// Groupings for better visualization.
	// Note: These maps are not thread-safe. For concurrent usage, synchronization is required.
	ByType     map[string]int `json:"by_type,omitempty"`
	ByModule   map[string]int `json:"by_module,omitempty"`
	BySeverity map[string]int `json:"by_severity,omitempty"`
	ByAction   map[string]int `json:"by_action,omitempty"`
}

// AnalyzedResource represents a resource with analyzed change information
type AnalyzedResource struct {
	Address       string   `json:"address"`
	Type          string   `json:"type"`
	Name          string   `json:"name"`
	Provider      string   `json:"provider"`
	Actions       []string `json:"actions"`
	ActionString  string   `json:"action_string"`
	Severity      Severity `json:"severity"`
	ModuleAddress string   `json:"module_address,omitempty"`
}

// Severity represents the risk level of a change
type Severity string

const (
	// SeverityLow indicates low-risk changes (additions)
	SeverityLow Severity = "low"
	// SeverityMedium indicates medium-risk changes (standard updates)
	SeverityMedium Severity = "medium"
	// SeverityHigh indicates high-risk changes (updates to critical resources)
	SeverityHigh Severity = "high"
	// SeverityCritical indicates critical changes (deletions or replacements)
	SeverityCritical Severity = "critical"
)

// Exit codes for plan analysis
const (
	// ExitCodeSuccess indicates no infrastructure changes detected
	ExitCodeSuccess = 0
	// ExitCodeChanges indicates non-critical changes detected (creates, updates)
	ExitCodeChanges = 1
	// ExitCodeCritical indicates critical changes detected (deletes, replacements)
	ExitCodeCritical = 2
)

// ExitCode returns the appropriate exit code based on analysis results.
// Returns ExitCodeSuccess (0) for no changes, ExitCodeChanges (1) for non-critical changes,
// and ExitCodeCritical (2) for critical changes (deletions or replacements).
func (a *PlanAnalysis) ExitCode() int {
	if a.Deletions > 0 || a.Replacements > 0 {
		return ExitCodeCritical
	}
	if a.TotalChanges > 0 {
		return ExitCodeChanges
	}
	return ExitCodeSuccess
}
