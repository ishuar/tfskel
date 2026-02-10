package drift

import (
	"fmt"

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
	driftConfig := LoadDriftConfig(v)
	criticalResources := MergeCriticalResources(DefaultCriticalResources(), driftConfig.CriticalResources)
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
			ActionString:  a.formatActions(rc.Change.Actions),
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

	analysis.HasChanges = analysis.TotalChanges > 0
	return analysis
}

// isNoOp checks if actions represent no operation
func isNoOp(actions []string) bool {
	if len(actions) == 1 && actions[0] == "no-op" {
		return true
	}
	return false
}

// formatActions converts action list to human-readable string
func (a *PlanAnalyzer) formatActions(actions []string) string {
	if len(actions) == 0 {
		return "no-op"
	}
	if len(actions) == 1 {
		return actions[0]
	}
	// Handle replace (delete + create)
	if containsAction(actions, "delete") && containsAction(actions, "create") {
		return "replace"
	}
	return fmt.Sprintf("%v", actions)
}

// determineSeverity assesses the risk level of a change
func (a *PlanAnalyzer) determineSeverity(actions []string, resourceType string) Severity {
	// Critical: Any deletion (data loss risk)
	if containsAction(actions, "delete") {
		return SeverityCritical
	}

	// High: Updates to critical infrastructure resources
	if a.isCriticalResource(resourceType) && containsAction(actions, "update") {
		return SeverityHigh
	}

	// Medium: Standard resource updates
	if containsAction(actions, "update") {
		return SeverityMedium
	}

	// Low: Additions only (no risk)
	if containsAction(actions, "create") {
		return SeverityLow
	}

	return SeverityLow
}

// isCriticalResource checks if a resource type is considered critical
func (a *PlanAnalyzer) isCriticalResource(resourceType string) bool {
	for _, t := range a.criticalResourceTypes {
		if resourceType == t {
			return true
		}
	}
	return false
}

// updateCounts updates the analysis counters based on actions
func (a *PlanAnalyzer) updateCounts(analysis *PlanAnalysis, actions []string) {
	switch {
	case containsAction(actions, "create") && !containsAction(actions, "delete"):
		analysis.Additions++
	case containsAction(actions, "delete") && containsAction(actions, "create"):
		analysis.Replacements++
	case containsAction(actions, "delete"):
		analysis.Deletions++
	case containsAction(actions, "update"):
		analysis.Modifications++
	}
}

// containsAction checks if an action is in the actions list
func containsAction(actions []string, action string) bool {
	for _, a := range actions {
		if a == action {
			return true
		}
	}
	return false
}
