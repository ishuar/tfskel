// Package plan parses Terraform plan JSON and turns it into the review
// report behind `tfskel review plan`.
//
// The pipeline is parse, analyze, filter, format:
//
//   - ParsePlanFile reads `terraform show -json` output and rejects binary
//     plan files with a hint to convert them.
//   - PlanAnalyzer classifies every resource change by action (create,
//     update, delete, replace) and severity. Severity is fixed by rule:
//     Critical for deletions and replacements, High for modifications to a
//     critical resource type, Medium for other modifications, Low for
//     additions. The critical type list is DefaultCriticalResources merged
//     with critical_resources from config.
//   - ResourceFilter narrows the analysis by severity, minimum severity or
//     action before anything downstream sees it.
//   - PlanFormatter renders table, JSON or CSV from the (filtered)
//     PlanAnalysis.
//
// PlanAnalysis.ExitCode derives the detailed exit code (0 no changes,
// 2 non-critical changes, 3 critical changes); the CLI decides whether to
// act on it. AnalysisConfig reads the review plan keys from viper.
//
// internal/ai builds its payload from the same PlanAnalysis, so the AI
// narrative and the table always describe the same resource set.
package plan
