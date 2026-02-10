package cmd

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
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

func runDriftAll(cmd *cobra.Command, args []string) error {
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
		OverallStatus: "clean",
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
		combined.OverallStatus = "critical"
		cmd.SilenceUsage = true
		return 1 // Return non-zero exit code for CI/CD
	}

	combined.VersionDrift = versionSummary
	if versionSummary.HasDrift {
		combined.HasIssues = true
		if versionSummary.MajorDrift > 0 {
			combined.OverallStatus = "critical"
		} else if combined.OverallStatus == "clean" {
			combined.OverallStatus = "warning"
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
		cmd.SilenceUsage = true
		return planExitCode
	}

	combined.PlanAnalysis = planAnalysis
	if planAnalysis.HasChanges {
		combined.HasIssues = true
		if planAnalysis.Deletions > 0 || planAnalysis.Replacements > 0 {
			combined.OverallStatus = "critical"
		} else if combined.OverallStatus != "critical" {
			combined.OverallStatus = "warning"
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
			{"version_drift", "total_files", fmt.Sprintf("%d", combined.VersionDrift.TotalFiles)},
			{"version_drift", "files_with_drift", fmt.Sprintf("%d", combined.VersionDrift.FilesWithDrift)},
			{"version_drift", "minor_drift", fmt.Sprintf("%d", combined.VersionDrift.MinorDrift)},
			{"version_drift", "major_drift", fmt.Sprintf("%d", combined.VersionDrift.MajorDrift)},
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
			{"plan_analysis", "total_changes", fmt.Sprintf("%d", combined.PlanAnalysis.TotalChanges)},
			{"plan_analysis", "additions", fmt.Sprintf("%d", combined.PlanAnalysis.Additions)},
			{"plan_analysis", "modifications", fmt.Sprintf("%d", combined.PlanAnalysis.Modifications)},
			{"plan_analysis", "deletions", fmt.Sprintf("%d", combined.PlanAnalysis.Deletions)},
			{"plan_analysis", "replacements", fmt.Sprintf("%d", combined.PlanAnalysis.Replacements)},
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
	fmt.Println("\n" + strings.Repeat("=", separatorWidth))
	fmt.Println("                 COMBINED DRIFT ANALYSIS RESULTS")
	fmt.Println(strings.Repeat("=", separatorWidth))

	// Version Drift Section
	if combined.VersionDrift != nil {
		fmt.Println("\n─── Version Drift Analysis ───")
		fmt.Printf("  Total Files Scanned:    %d\n", combined.VersionDrift.TotalFiles)
		fmt.Printf("  Files with Drift:       %d\n", combined.VersionDrift.FilesWithDrift)
		fmt.Printf("  Minor Drift:            %d\n", combined.VersionDrift.MinorDrift)
		fmt.Printf("  Major Drift:            %d\n", combined.VersionDrift.MajorDrift)
		status := "✔ Clean"
		if combined.VersionDrift.HasDrift {
			status = "✘ Drift Detected"
			if useColor {
				if combined.VersionDrift.MajorDrift > 0 {
					status = "\033[1;31m✘ Drift Detected\033[0m"
				} else {
					status = "\033[33m✘ Drift Detected\033[0m"
				}
			}
		} else if useColor {
			status = "\033[32m✔ Clean\033[0m"
		}
		fmt.Printf("  Status:                 %s\n", status)
	}

	// Plan Analysis Section
	if combined.PlanAnalysis != nil {
		fmt.Println("\n─── Plan Analysis ───")
		fmt.Printf("  Total Changes:          %d\n", combined.PlanAnalysis.TotalChanges)
		fmt.Printf("  Additions:              %d\n", combined.PlanAnalysis.Additions)
		fmt.Printf("  Modifications:          %d\n", combined.PlanAnalysis.Modifications)
		fmt.Printf("  Deletions:              %d\n", combined.PlanAnalysis.Deletions)
		fmt.Printf("  Replacements:           %d\n", combined.PlanAnalysis.Replacements)
		status := "✔ No Changes"
		if combined.PlanAnalysis.HasChanges {
			status = "✘ Changes Detected"
			if useColor {
				if combined.PlanAnalysis.Deletions > 0 || combined.PlanAnalysis.Replacements > 0 {
					status = "\033[1;31m✘ Changes Detected\033[0m"
				} else {
					status = "\033[33m✘ Changes Detected\033[0m"
				}
			}
		} else if useColor {
			status = "\033[32m✔ No Changes\033[0m"
		}
		fmt.Printf("  Status:                 %s\n", status)
	}

	// Overall Summary
	fmt.Println("\n" + strings.Repeat("─", separatorWidth))
	overallStatus := combined.OverallStatus
	if useColor {
		switch combined.OverallStatus {
		case "critical":
			overallStatus = "\033[1;31mCRITICAL\033[0m"
		case "warning":
			overallStatus = "\033[33mWARNING\033[0m"
		case "clean":
			overallStatus = "\033[32mCLEAN\033[0m"
		}
	}
	fmt.Printf("  Overall Status:         %s\n", overallStatus)
	fmt.Println(strings.Repeat("=", separatorWidth) + "\n")

	return nil
}
