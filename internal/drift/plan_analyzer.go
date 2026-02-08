package drift

import (
	"fmt"
)

// PlanAnalyzer analyzes terraform plans and assesses change severity
type PlanAnalyzer struct {
	criticalResourceTypes []string
}

// NewPlanAnalyzer creates a new plan analyzer with default critical resource types
func NewPlanAnalyzer() *PlanAnalyzer {
	return &PlanAnalyzer{
		criticalResourceTypes: []string{
			// Databases - risk of data loss or service disruption
			"aws_db_instance",
			"aws_rds_cluster",
			"aws_rds_cluster_instance",
			"aws_dynamodb_table",
			"google_sql_database_instance",
			"google_sql_database",
			"azurerm_sql_database",
			"azurerm_postgresql_server",
			"azurerm_mysql_server",
			"azurerm_mssql_server",
			// Storage - risk of data loss
			"aws_s3_bucket",
			"aws_efs_file_system",
			"google_storage_bucket",
			"google_compute_disk",
			"azurerm_storage_account",
			"azurerm_storage_blob",
			// Networking - risk of service disruption
			"aws_vpc",
			"aws_subnet",
			"aws_route_table",
			"aws_security_group",
			"aws_network_acl",
			"google_compute_network",
			"google_compute_subnetwork",
			"google_compute_firewall",
			"azurerm_virtual_network",
			"azurerm_subnet",
			"azurerm_network_security_group",
		},
	}
}

// NewPlanAnalyzerWithTypes creates a plan analyzer with custom critical resource types
func NewPlanAnalyzerWithTypes(criticalTypes []string) *PlanAnalyzer {
	return &PlanAnalyzer{
		criticalResourceTypes: criticalTypes,
	}
}

// Analyze processes a terraform plan and produces detailed analysis
func (a *PlanAnalyzer) Analyze(plan *TerraformPlan) *PlanAnalysis {
	analysis := &PlanAnalysis{
		TerraformVersion: plan.TerraformVersion,
		ResourceChanges:  make([]AnalyzedResource, 0),
	}

	for _, rc := range plan.ResourceChanges {
		// Skip resources with no actions or no-op actions
		if len(rc.Change.Actions) == 0 || isNoOp(rc.Change.Actions) {
			continue
		}

		analyzed := AnalyzedResource{
			Address:      rc.Address,
			Type:         rc.Type,
			Name:         rc.Name,
			Provider:     rc.ProviderName,
			Actions:      rc.Change.Actions,
			ActionString: a.formatActions(rc.Change.Actions),
			Severity:     a.determineSeverity(rc.Change.Actions, rc.Type),
		}

		analysis.ResourceChanges = append(analysis.ResourceChanges, analyzed)
		analysis.TotalChanges++

		// Count by action type
		a.updateCounts(analysis, rc.Change.Actions)
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
	if containsAction(actions, "create") && !containsAction(actions, "delete") {
		analysis.Additions++
	} else if containsAction(actions, "delete") && containsAction(actions, "create") {
		analysis.Replacements++
	} else if containsAction(actions, "delete") {
		analysis.Deletions++
	} else if containsAction(actions, "update") {
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
