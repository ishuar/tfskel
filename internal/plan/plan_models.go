package plan

import (
	"encoding/json"
	"slices"
	"sort"
	"strings"
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
	// Output change tracking
	OutputChanges       []OutputChange `json:"output_changes,omitempty"`
	OutputAdditions     int            `json:"output_additions"`
	OutputDeletions     int            `json:"output_deletions"`
	OutputModifications int            `json:"output_modifications"`
	OutputReplacements  int            `json:"output_replacements"`
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
	// Change carries the raw before/after attribute maps and sensitivity marks
	// from the plan for consumers that need attribute-level detail (the AI
	// payload). Excluded from JSON output: machine formats expose only the
	// analyzed summary fields above.
	Change       ChangeDetail `json:"-"`
	ActionReason string       `json:"-"`
}

// OutputChange represents a change to a Terraform output value
type OutputChange struct {
	Name      string   `json:"name"`
	Actions   []string `json:"actions"`
	Sensitive bool     `json:"sensitive"`
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

// Severity order constants for sorting (lower = higher priority)
const (
	severityOrderCritical = 0
	severityOrderHigh     = 1
	severityOrderMedium   = 2
	severityOrderLow      = 3
	severityOrderUnknown  = 4
)

// Action string constants used across parsing, analysis, and formatting
const (
	ActionCreate  = "create"
	ActionDelete  = "delete"
	ActionUpdate  = "update"
	ActionRead    = "read"
	ActionNoOp    = "no-op"
	ActionReplace = "replace" // synthetic: delete + create
)

// Order returns the sort order for a severity level.
// Lower values are sorted first (higher priority).
func (s Severity) Order() int {
	switch s {
	case SeverityCritical:
		return severityOrderCritical
	case SeverityHigh:
		return severityOrderHigh
	case SeverityMedium:
		return severityOrderMedium
	case SeverityLow:
		return severityOrderLow
	default:
		return severityOrderUnknown
	}
}

// formatActions converts an action list to a human-readable string.
// Returns "replace" for delete+create combinations, joins multiple actions with comma.
func formatActions(actions []string) string {
	if len(actions) == 0 {
		return ActionNoOp
	}
	if len(actions) == 1 {
		return actions[0]
	}
	if slices.Contains(actions, ActionDelete) && slices.Contains(actions, ActionCreate) {
		return ActionReplace
	}
	return strings.Join(actions, ", ")
}

// sortResourcesBySeverity sorts resources by severity level (critical, high, medium, low).
// Returns a new sorted slice without modifying the original.
// Uses stable sort to maintain original order within same severity level.
func sortResourcesBySeverity(resources []AnalyzedResource) []AnalyzedResource {
	sorted := make([]AnalyzedResource, len(resources))
	copy(sorted, resources)
	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].Severity.Order() < sorted[j].Severity.Order()
	})
	return sorted
}
