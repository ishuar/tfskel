package cmd

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/ishuar/tfskel/internal/config"
	"github.com/ishuar/tfskel/internal/drift"
	"github.com/ishuar/tfskel/internal/logger"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	// separatorWidth defines the width of separator lines in table output
	separatorWidth = 70

	// Status constants for overall analysis status
	statusCritical = "critical"
	statusWarning  = "warning"
	statusClean    = "clean"
)

var (
	allPath         string
	allPlanFile     string
	allFormat       string
	allNoColor      bool
	allSkipPlan     bool
	allSkipVersions bool
)

var (
	// ErrAllAnalysisFailed indicates that one or more analyses failed during execution
	ErrAllAnalysisFailed = errors.New("one or more analyses failed")
	// ErrInvalidSkipFlags indicates both analyses were skipped
	ErrInvalidSkipFlags = errors.New("invalid flags: cannot skip all analyses")
)

// CombinedAnalysis represents results from both version and plan analysis
type CombinedAnalysis struct {
	VersionDrift  *VersionDriftSummary `json:"version_drift,omitempty"`
	PlanAnalysis  *drift.PlanAnalysis  `json:"plan_analysis,omitempty"`
	OverallStatus string               `json:"overall_status"`
	HasIssues     bool                 `json:"has_issues"`
}

// VersionDriftSummary is a simplified version drift summary
type VersionDriftSummary struct {
	TotalFiles     int  `json:"total_files"`
	FilesWithDrift int  `json:"files_with_drift"`
	MinorDrift     int  `json:"minor_drift"`
	MajorDrift     int  `json:"major_drift"`
	HasDrift       bool `json:"has_drift"`
}

// driftAllCmd represents the drift all command
var driftAllCmd = &cobra.Command{
	Use:   "all",
	Short: "Run combined version drift and plan analysis",
	Long: `Run a comprehensive drift analysis combining both version drift detection
and terraform plan analysis. This provides a complete picture of your
infrastructure state and planned changes.

This command is ideal for:
  • Pre-commit hooks - Catch both version drift and plan issues
  • CI/CD pipelines - Comprehensive validation before deployment
  • Code reviews - Complete analysis for reviewers
  • Compliance checks - Ensure both versions and changes are validated

The command will:
  1. Scan for version drift across Terraform configurations
  2. Analyze terraform plan for resource changes
  3. Combine results and provide unified reporting
  4. Exit with code reflecting the most severe issue

Exit Codes:
  0 - No issues found
  1 - Version drift or plan changes detected
  2 - Critical changes (deletions/replacements) or major version drift

Examples:
  # Run full analysis
  tfskel drift all --plan-file tfplan.json

  # Analyze specific directory with plan
  tfskel drift all --path ./envs --plan-file tfplan.json

  # Skip plan analysis (versions only)
  tfskel drift all --skip-plan

  # Skip version analysis (plan only)
  tfskel drift all --plan-file tfplan.json --skip-versions

  # Export combined results as JSON
  tfskel drift all --plan-file tfplan.json --format json

  # CI/CD usage with no colors
  tfskel drift all --plan-file tfplan.json --no-color`,
	RunE: runDriftAll,
}

func init() {
	driftCmd.AddCommand(driftAllCmd)

	driftAllCmd.Flags().StringVarP(&allPath, "path", "p", ".",
		"Path to scan for Terraform files (default: current directory)")
	driftAllCmd.Flags().StringVar(&allPlanFile, "plan-file", "",
		"Path to terraform plan JSON file (optional)")
	driftAllCmd.Flags().StringVarP(&allFormat, "format", "f", "table",
		"Output format: table, json, csv")
	driftAllCmd.Flags().BoolVar(&allNoColor, "no-color", false,
		"Disable colored output")
	driftAllCmd.Flags().BoolVar(&allSkipPlan, "skip-plan", false,
		"Skip plan analysis (versions only)")
	driftAllCmd.Flags().BoolVar(&allSkipVersions, "skip-versions", false,
		"Skip version analysis (plan only)")
}

func runDriftAll(cmd *cobra.Command, _ []string) error {
	log := logger.New(viper.GetBool("verbose"))

	// Validate flags and get effective skip values
	skipVersion, skipPlan, err := validateAllFlags(log, cmd)
	if err != nil {
		return err
	}

	// Suppress logs for machine-readable formats
	if allFormat == "json" || allFormat == "csv" {
		log.SetOutput(os.Stderr)
	}

	log.Info("Starting comprehensive drift analysis...")

	combined := &CombinedAnalysis{
		OverallStatus: statusClean,
	}

	exitCode := executeAnalyses(log, cmd, combined, skipVersion, skipPlan)

	log.Info("\n=== Combined Analysis Complete ===")

	// Format and output combined results
	if err := formatCombinedAnalysis(combined, allFormat, !allNoColor); err != nil {
		log.Errorf("Failed to format output: %v", err)
		cmd.SilenceUsage = true
		return fmt.Errorf("failed to format output: %w", err)
	}

	// Return ExitError if issues detected for proper exit code handling
	if exitCode != 0 {
		log.Warnf("Issues detected - exiting with code %d", exitCode)
		cmd.SilenceUsage = true
		return NewExitError(exitCode, "")
	}

	log.Success("No issues detected - infrastructure is clean")
	return nil
}

// validateAllFlags validates the command flags for drift all and returns effective skip values
func validateAllFlags(log *logger.Logger, cmd *cobra.Command) (skipVersion bool, skipPlan bool, err error) {
	skipVersion = allSkipVersions
	skipPlan = allSkipPlan

	if skipVersion && skipPlan {
		log.Error("Cannot skip both version and plan analysis. Remove one of the skip flags.")
		cmd.SilenceUsage = true
		return false, false, ErrInvalidSkipFlags
	}

	if !skipPlan && allPlanFile == "" {
		log.Warn("No plan file provided. Use --plan-file or --skip-plan")
		skipPlan = true // Use local variable instead of mutating package-level flag
	}

	return skipVersion, skipPlan, nil
}

// executeAnalyses runs both version and plan analyses and returns the highest exit code
func executeAnalyses(log *logger.Logger, cmd *cobra.Command, combined *CombinedAnalysis, skipVersion bool, skipPlan bool) int {
	var exitCode int

	// Run version drift analysis
	if !skipVersion {
		versionExitCode := executeVersionAnalysis(log, cmd, combined)
		if versionExitCode > exitCode {
			exitCode = versionExitCode
		}
	}

	// Run plan analysis
	if !skipPlan {
		planExitCode := executePlanAnalysis(log, cmd, combined)
		if planExitCode > exitCode {
			exitCode = planExitCode
		}
	}

	return exitCode
}

// executeVersionAnalysis runs version drift analysis and updates combined results
func executeVersionAnalysis(log *logger.Logger, cmd *cobra.Command, combined *CombinedAnalysis) int {
	log.Info("\n[1/2] Running version drift analysis...")
	versionSummary, versionExitCode, err := runVersionAnalysis(allPath, log, cmd)
	if err != nil {
		log.Errorf("Version analysis failed: %v", err)
		// Mark combined analysis as having critical issues
		combined.HasIssues = true
		combined.OverallStatus = statusCritical
		cmd.SilenceUsage = true
		return 1 // Return non-zero exit code for CI/CD
	}

	combined.VersionDrift = versionSummary
	if versionSummary.HasDrift {
		combined.HasIssues = true
		if versionSummary.MajorDrift > 0 {
			combined.OverallStatus = statusCritical
		} else if combined.OverallStatus == statusClean {
			combined.OverallStatus = statusWarning
		}
	}

	return versionExitCode
}

// executePlanAnalysis runs plan analysis and updates combined results
func executePlanAnalysis(log *logger.Logger, cmd *cobra.Command, combined *CombinedAnalysis) int {
	log.Info("\n[2/2] Running plan analysis...")
	planAnalysis, planExitCode, err := runPlanAnalysisInternal(allPlanFile, log)
	if err != nil {
		log.Errorf("Plan analysis failed: %v", err)
		// Mark combined analysis as having critical issues on plan analysis failure
		combined.HasIssues = true
		combined.OverallStatus = statusCritical
		cmd.SilenceUsage = true
		return planExitCode
	}

	combined.PlanAnalysis = planAnalysis
	if planAnalysis.HasChanges {
		combined.HasIssues = true
		if planAnalysis.Deletions > 0 || planAnalysis.Replacements > 0 {
			combined.OverallStatus = statusCritical
		} else if combined.OverallStatus != statusCritical {
			combined.OverallStatus = statusWarning
		}
	}

	return planExitCode
}

func runVersionAnalysis(scanPath string, log *logger.Logger, cmd *cobra.Command) (*VersionDriftSummary, int, error) {
	// Validate path
	fileInfo, err := os.Stat(scanPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, 0, fmt.Errorf("%w: %s", ErrPathDoesNotExist, scanPath)
		}
		return nil, 0, fmt.Errorf("failed to access path: %w", err)
	}

	if !fileInfo.IsDir() {
		return nil, 0, fmt.Errorf("%w: %s", ErrPathNotDirectory, scanPath)
	}

	absPath, err := filepath.Abs(scanPath)
	if err != nil {
		absPath = scanPath
	}

	// Load configuration
	cfg, err := config.Load(cmd, viper.GetViper())
	if err != nil {
		return nil, 0, fmt.Errorf("failed to load configuration: %w", err)
	}

	log.Infof("Scanning path: %s", absPath)

	// Create detector and scan
	detector := drift.NewDetector(scanPath)
	versionInfos, err := detector.ScanDirectory()
	if err != nil {
		return nil, 0, fmt.Errorf("failed to scan directory: %w", err)
	}

	if len(versionInfos) == 0 {
		log.Warnf("No Terraform files with version information found")
		return &VersionDriftSummary{}, 0, nil
	}

	log.Infof("Found %d files with version information", len(versionInfos))

	// Analyze drift
	analyzer := drift.NewAnalyzer(cfg)
	report := analyzer.Analyze(absPath, versionInfos)

	// Create summary
	summary := &VersionDriftSummary{
		TotalFiles:     report.TotalFiles,
		FilesWithDrift: report.FilesWithDrift,
		MinorDrift:     report.Summary.FilesWithMinorDrift,
		MajorDrift:     report.Summary.FilesWithMajorDrift,
		HasDrift:       report.FilesWithDrift > 0,
	}

	return summary, report.ExitCode(), nil
}

func runPlanAnalysisInternal(planFile string, log *logger.Logger) (*drift.PlanAnalysis, int, error) {
	// Check if file exists
	if _, err := os.Stat(planFile); err != nil {
		if os.IsNotExist(err) {
			return nil, 1, fmt.Errorf("%w: %s", ErrPlanFileNotFound, planFile)
		}
		return nil, 1, fmt.Errorf("failed to access plan file: %w", err)
	}

	log.Infof("Analyzing plan file: %s", planFile)

	// Parse plan file using internal package
	plan, err := drift.ParsePlanFile(planFile)
	if err != nil {
		return nil, 1, fmt.Errorf("failed to parse plan file: %w", err)
	}

	// Analyze the plan using internal package
	analyzer := drift.NewPlanAnalyzerWithConfig(viper.GetViper())
	analysis := analyzer.Analyze(plan)

	if !analysis.HasChanges {
		log.Info("No changes detected in plan")
	}

	// Use the ExitCode method on analysis
	return analysis, analysis.ExitCode(), nil
}

// ErrUnsupportedCombinedFormat indicates an unsupported output format for combined analysis
var ErrUnsupportedCombinedFormat = errors.New("unsupported format")

func formatCombinedAnalysis(combined *CombinedAnalysis, format string, useColor bool) error {
	switch format {
	case "json":
		return formatCombinedJSON(combined)
	case "csv":
		return formatCombinedCSV(combined)
	case "table":
		return formatCombinedTable(combined, useColor)
	default:
		return fmt.Errorf("%w: %s", ErrUnsupportedCombinedFormat, format)
	}
}

func formatCombinedJSON(combined *CombinedAnalysis) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(combined)
}

func formatCombinedCSV(combined *CombinedAnalysis) error {
	csvWriter := csv.NewWriter(os.Stdout)
	defer csvWriter.Flush()

	// Write header
	if err := csvWriter.Write([]string{"Analysis Type", "Metric", "Value"}); err != nil {
		return fmt.Errorf("failed to write CSV header: %w", err)
	}

	// Write version drift data
	if combined.VersionDrift != nil {
		records := [][]string{
			{"version_drift", "total_files", strconv.Itoa(combined.VersionDrift.TotalFiles)},
			{"version_drift", "files_with_drift", strconv.Itoa(combined.VersionDrift.FilesWithDrift)},
			{"version_drift", "minor_drift", strconv.Itoa(combined.VersionDrift.MinorDrift)},
			{"version_drift", "major_drift", strconv.Itoa(combined.VersionDrift.MajorDrift)},
		}
		for _, record := range records {
			if err := csvWriter.Write(record); err != nil {
				return fmt.Errorf("failed to write version drift record: %w", err)
			}
		}
	}

	// Write plan analysis data
	if combined.PlanAnalysis != nil {
		records := [][]string{
			{"plan_analysis", "total_changes", strconv.Itoa(combined.PlanAnalysis.TotalChanges)},
			{"plan_analysis", "additions", strconv.Itoa(combined.PlanAnalysis.Additions)},
			{"plan_analysis", "modifications", strconv.Itoa(combined.PlanAnalysis.Modifications)},
			{"plan_analysis", "deletions", strconv.Itoa(combined.PlanAnalysis.Deletions)},
			{"plan_analysis", "replacements", strconv.Itoa(combined.PlanAnalysis.Replacements)},
		}
		for _, record := range records {
			if err := csvWriter.Write(record); err != nil {
				return fmt.Errorf("failed to write plan analysis record: %w", err)
			}
		}
	}

	// Write overall status
	if err := csvWriter.Write([]string{"overall", "status", combined.OverallStatus}); err != nil {
		return fmt.Errorf("failed to write overall status: %w", err)
	}

	return csvWriter.Error()
}

func formatCombinedTable(combined *CombinedAnalysis, useColor bool) error {
	printTableHeader()

	if combined.VersionDrift != nil {
		printVersionDriftSection(combined.VersionDrift, useColor)
	}

	if combined.PlanAnalysis != nil {
		printPlanAnalysisSection(combined.PlanAnalysis, useColor)
	}

	printOverallSummary(combined.OverallStatus, useColor)

	return nil
}

// printTableHeader prints the header section of the combined analysis table
func printTableHeader() {
	fmt.Println("\n" + strings.Repeat("=", separatorWidth))
	fmt.Println("                 COMBINED DRIFT ANALYSIS RESULTS")
	fmt.Println(strings.Repeat("=", separatorWidth))
}

// printVersionDriftSection prints the version drift analysis section
func printVersionDriftSection(vd *VersionDriftSummary, useColor bool) {
	fmt.Println("\n─── Version Drift Analysis ───")
	fmt.Printf("  Total Files Scanned:    %d\n", vd.TotalFiles)
	fmt.Printf("  Files with Drift:       %d\n", vd.FilesWithDrift)
	fmt.Printf("  Minor Drift:            %d\n", vd.MinorDrift)
	fmt.Printf("  Major Drift:            %d\n", vd.MajorDrift)

	status := formatVersionDriftStatus(vd, useColor)
	fmt.Printf("  Status:                 %s\n", status)
}

// formatVersionDriftStatus returns a formatted status string for version drift
func formatVersionDriftStatus(vd *VersionDriftSummary, useColor bool) string {
	if !vd.HasDrift {
		return formatStatus("✔ Clean", "green", useColor)
	}

	if vd.MajorDrift > 0 {
		return formatStatus("✘ Drift Detected", "red", useColor)
	}

	return formatStatus("✘ Drift Detected", "yellow", useColor)
}

// printPlanAnalysisSection prints the plan analysis section
func printPlanAnalysisSection(pa *drift.PlanAnalysis, useColor bool) {
	fmt.Println("\n─── Plan Analysis ───")
	fmt.Printf("  Total Changes:          %d\n", pa.TotalChanges)
	fmt.Printf("  Additions:              %d\n", pa.Additions)
	fmt.Printf("  Modifications:          %d\n", pa.Modifications)
	fmt.Printf("  Deletions:              %d\n", pa.Deletions)
	fmt.Printf("  Replacements:           %d\n", pa.Replacements)

	status := formatPlanAnalysisStatus(pa, useColor)
	fmt.Printf("  Status:                 %s\n", status)
}

// formatPlanAnalysisStatus returns a formatted status string for plan analysis
func formatPlanAnalysisStatus(pa *drift.PlanAnalysis, useColor bool) string {
	if !pa.HasChanges {
		return formatStatus("✔ No Changes", "green", useColor)
	}

	if pa.Deletions > 0 || pa.Replacements > 0 {
		return formatStatus("✘ Changes Detected", "red", useColor)
	}

	return formatStatus("✘ Changes Detected", "yellow", useColor)
}

// printOverallSummary prints the overall summary section
func printOverallSummary(status string, useColor bool) {
	fmt.Println("\n" + strings.Repeat("─", separatorWidth))

	colorMap := map[string]string{
		statusCritical: "red",
		statusWarning:  "yellow",
		statusClean:    "green",
	}

	colorName := colorMap[status]
	formattedStatus := formatStatus(strings.ToUpper(status), colorName, useColor)

	fmt.Printf("  Overall Status:         %s\n", formattedStatus)
	fmt.Println(strings.Repeat("=", separatorWidth) + "\n")
}

// formatStatus formats a status string with optional ANSI color codes
func formatStatus(text, colorName string, useColor bool) string {
	if !useColor {
		return text
	}

	colorCodes := map[string]string{
		"red":    "\033[1;31m",
		"yellow": "\033[33m",
		"green":  "\033[32m",
	}

	colorCode, exists := colorCodes[colorName]
	if !exists {
		return text
	}

	return fmt.Sprintf("%s%s\033[0m", colorCode, text)
}
