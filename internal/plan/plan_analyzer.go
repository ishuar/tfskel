package plan

import (
	"sort"

	"github.com/spf13/viper"
)

// PlanAnalyzer analyzes terraform plans and assesses change severity
type PlanAnalyzer struct {
	criticalResourceTypes []string
}

// NewPlanAnalyzer creates a new plan analyzer with default critical resource types
func NewPlanAnalyzer() *PlanAnalyzer {
	return &PlanAnalyzer{
		criticalResourceTypes: DefaultCriticalResources(),
	}
}

// NewPlanAnalyzerWithConfig creates a plan analyzer with default critical resources
// merged with user-defined resources from viper config.
// This allows extending the default list via .tfskel.yaml configuration.
func NewPlanAnalyzerWithConfig(v *viper.Viper) *PlanAnalyzer {
	cfg := LoadAnalysisConfig(v)
	criticalResources := MergeCriticalResources(DefaultCriticalResources(), cfg.CriticalResources)
	return &PlanAnalyzer{
		criticalResourceTypes: criticalResources,
	}
}

// NewPlanAnalyzerWithTypes creates a plan analyzer with custom critical resource types
func NewPlanAnalyzerWithTypes(criticalTypes []string) *PlanAnalyzer {
	return &PlanAnalyzer{
		criticalResourceTypes: criticalTypes,
	}
}

// Analyze processes a terraform plan and produces detailed analysis.
// Returns an empty analysis if plan is nil.
func (a *PlanAnalyzer) Analyze(plan *TerraformPlan) *PlanAnalysis {
	if plan == nil {
		return &PlanAnalysis{
			TerraformVersion: "unknown",
			ResourceChanges:  []AnalyzedResource{},
			ByType:           make(map[string]int),
			ByModule:         make(map[string]int),
			BySeverity:       make(map[string]int),
			ByAction:         make(map[string]int),
		}
	}

	analysis := &PlanAnalysis{
		TerraformVersion: plan.TerraformVersion,
		ResourceChanges:  make([]AnalyzedResource, 0),
		ByType:           make(map[string]int),
		ByModule:         make(map[string]int),
		BySeverity:       make(map[string]int),
		ByAction:         make(map[string]int),
	}

	for _, rc := range plan.ResourceChanges {
		// Skip data sources - we only track managed resources
		if rc.Mode == "data" {
			continue
		}

		// Skip resources with no actions or no-op actions
		if len(rc.Change.Actions) == 0 || isNoOp(rc.Change.Actions) {
			continue
		}

		analyzed := AnalyzedResource{
			Address:       rc.Address,
			Type:          rc.Type,
			Name:          rc.Name,
			Provider:      rc.ProviderName,
			Actions:       rc.Change.Actions,
			ActionString:  formatActions(rc.Change.Actions),
			Severity:      a.determineSeverity(rc.Change.Actions, rc.Type),
			ModuleAddress: rc.ModuleAddress,
		}

		analysis.ResourceChanges = append(analysis.ResourceChanges, analyzed)
		analysis.TotalChanges++

		// Count by action type
		a.updateCounts(analysis, rc.Change.Actions)

		// Group by type
		analysis.ByType[rc.Type]++

		// Group by module
		module := "root"
		if rc.ModuleAddress != "" {
			module = rc.ModuleAddress
		}
		analysis.ByModule[module]++

		// Group by severity
		analysis.BySeverity[string(analyzed.Severity)]++

		// Group by action
		analysis.ByAction[analyzed.ActionString]++
	}

	// Analyze output changes
	a.analyzeOutputChanges(plan, analysis)

	analysis.HasChanges = analysis.TotalChanges > 0 || len(analysis.OutputChanges) > 0
	return analysis
}

// analyzeOutputChanges parses the plan's output_changes map and populates the analysis.
// The plan JSON stores output_changes as map[string]any where each value has:
//
//	{"actions": [...], "before_sensitive": bool, "after_sensitive": bool, ...}
func (a *PlanAnalyzer) analyzeOutputChanges(plan *TerraformPlan, analysis *PlanAnalysis) {
	if len(plan.OutputChanges) == 0 {
		return
	}

	// Collect and sort output names for deterministic ordering
	names := make([]string, 0, len(plan.OutputChanges))
	for name := range plan.OutputChanges {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		raw := plan.OutputChanges[name]
		entry, ok := raw.(map[string]any)
		if !ok {
			continue
		}

		actions := extractStringSlice(entry, "actions")
		if len(actions) == 0 || isNoOp(actions) {
			continue
		}

		sensitive := extractBool(entry, "before_sensitive") || extractBool(entry, "after_sensitive")

		analysis.OutputChanges = append(analysis.OutputChanges, OutputChange{
			Name:      name,
			Actions:   actions,
			Sensitive: sensitive,
		})

		switch {
		case contains(actions, ActionCreate) && !contains(actions, ActionDelete):
			analysis.OutputAdditions++
		case contains(actions, ActionDelete) && contains(actions, ActionCreate):
			analysis.OutputReplacements++
		case contains(actions, ActionDelete):
			analysis.OutputDeletions++
		case contains(actions, ActionUpdate):
			analysis.OutputModifications++
		}
	}
}

// extractStringSlice extracts a []string from a map entry's "key" field.
func extractStringSlice(m map[string]any, key string) []string {
	raw, ok := m[key]
	if !ok {
		return nil
	}
	arr, ok := raw.([]any)
	if !ok {
		return nil
	}
	result := make([]string, 0, len(arr))
	for _, v := range arr {
		if s, ok := v.(string); ok {
			result = append(result, s)
		}
	}
	return result
}

// extractBool extracts a bool from a map entry, returning false if missing or wrong type.
func extractBool(m map[string]any, key string) bool {
	raw, ok := m[key]
	if !ok {
		return false
	}
	b, ok := raw.(bool)
	return ok && b
}

// isNoOp checks if actions represent no operation
func isNoOp(actions []string) bool {
	return len(actions) == 1 && actions[0] == ActionNoOp
}

// determineSeverity assesses the risk level of a change
func (a *PlanAnalyzer) determineSeverity(actions []string, resourceType string) Severity {
	// Critical: Any deletion (data loss risk)
	if contains(actions, ActionDelete) {
		return SeverityCritical
	}

	// High: Updates to critical infrastructure resources
	if a.isCriticalResource(resourceType) && contains(actions, ActionUpdate) {
		return SeverityHigh
	}

	// Medium: Standard resource updates
	if contains(actions, ActionUpdate) {
		return SeverityMedium
	}

	// Low: Additions only (no risk)
	if contains(actions, ActionCreate) {
		return SeverityLow
	}

	return SeverityLow
}

// isCriticalResource checks if a resource type is considered critical
func (a *PlanAnalyzer) isCriticalResource(resourceType string) bool {
	return contains(a.criticalResourceTypes, resourceType)
}

// updateCounts updates the analysis counters based on actions
func (a *PlanAnalyzer) updateCounts(analysis *PlanAnalysis, actions []string) {
	switch {
	case contains(actions, ActionCreate) && !contains(actions, ActionDelete):
		analysis.Additions++
	case contains(actions, ActionDelete) && contains(actions, ActionCreate):
		analysis.Replacements++
	case contains(actions, ActionDelete):
		analysis.Deletions++
	case contains(actions, ActionUpdate):
		analysis.Modifications++
	}
}
