package plan

import (
	"errors"
	"fmt"
	"strings"
)

var (
	// ErrUnknownSeverity indicates an invalid severity filter value.
	ErrUnknownSeverity = errors.New("unknown severity")
	// ErrUnknownAction indicates an invalid action filter value.
	ErrUnknownAction = errors.New("unknown action")
	// ErrConflictingFilters indicates mutually exclusive filters were both set.
	ErrConflictingFilters = errors.New("--min-severity and --filter-severity are mutually exclusive")
)

// validSeverities contains all accepted severity filter values.
var validSeverities = []string{
	string(SeverityLow),
	string(SeverityMedium),
	string(SeverityHigh),
	string(SeverityCritical),
}

// validActions contains all accepted action filter values.
var validActions = []string{
	ActionCreate,
	ActionUpdate,
	ActionDelete,
	ActionReplace,
}

// ResourceFilter holds filter criteria for plan resources.
// All non-empty fields use AND semantics across dimensions,
// OR semantics within each dimension.
// MinSeverity and Severities are mutually exclusive.
type ResourceFilter struct {
	Severities  []string
	MinSeverity string
	Actions     []string
}

// IsEmpty reports whether no filters are active.
func (f *ResourceFilter) IsEmpty() bool {
	return len(f.Severities) == 0 && f.MinSeverity == "" && len(f.Actions) == 0
}

// Validate checks all filter values against known constants.
// Returns a descriptive error for unknown values or conflicting filters.
func (f *ResourceFilter) Validate() error {
	if len(f.Severities) > 0 && f.MinSeverity != "" {
		return fmt.Errorf("%w", ErrConflictingFilters)
	}
	for _, s := range f.Severities {
		if !containsFold(validSeverities, s) {
			return fmt.Errorf("%w %q: must be one of %s", ErrUnknownSeverity, s, strings.Join(validSeverities, ", "))
		}
	}
	if f.MinSeverity != "" && !containsFold(validSeverities, f.MinSeverity) {
		return fmt.Errorf("%w %q: must be one of %s", ErrUnknownSeverity, f.MinSeverity, strings.Join(validSeverities, ", "))
	}
	for _, a := range f.Actions {
		if !containsFold(validActions, a) {
			return fmt.Errorf("%w %q: must be one of %s", ErrUnknownAction, a, strings.Join(validActions, ", "))
		}
	}
	return nil
}

// Match reports whether the resource matches all active filters.
// Empty filter dimensions are skipped (match everything).
func (f *ResourceFilter) Match(r AnalyzedResource) bool {
	if len(f.Severities) > 0 && !containsFold(f.Severities, string(r.Severity)) {
		return false
	}
	if f.MinSeverity != "" && !f.meetsMinSeverity(r.Severity) {
		return false
	}
	if len(f.Actions) > 0 && !containsFold(f.Actions, r.ActionString) {
		return false
	}
	return true
}

// meetsMinSeverity reports whether s is at or above the minimum severity threshold.
// Lower Order() means higher severity, so s meets the threshold when its order <= threshold order.
func (f *ResourceFilter) meetsMinSeverity(s Severity) bool {
	threshold := Severity(strings.ToLower(f.MinSeverity))
	return s.Order() <= threshold.Order()
}

// Descriptions returns human-readable strings describing each active filter,
// e.g. ["severity=critical,high", "action=delete"].
func (f *ResourceFilter) Descriptions() []string {
	var descs []string
	if len(f.Severities) > 0 {
		descs = append(descs, "severity="+strings.Join(f.Severities, ","))
	}
	if f.MinSeverity != "" {
		descs = append(descs, "min-severity="+f.MinSeverity)
	}
	if len(f.Actions) > 0 {
		descs = append(descs, "action="+strings.Join(f.Actions, ","))
	}
	return descs
}

// FilterResources returns only the resources matching the filter.
// Returns the original slice unchanged if the filter is empty.
func FilterResources(resources []AnalyzedResource, filter *ResourceFilter) []AnalyzedResource {
	if filter == nil || filter.IsEmpty() {
		return resources
	}
	var filtered []AnalyzedResource
	for _, r := range resources {
		if filter.Match(r) {
			filtered = append(filtered, r)
		}
	}
	return filtered
}

// containsFold reports whether vals contains s, compared case-insensitively.
func containsFold(vals []string, s string) bool {
	for _, v := range vals {
		if strings.EqualFold(v, s) {
			return true
		}
	}
	return false
}
