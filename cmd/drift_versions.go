package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ishuar/tfskel/internal/config"
	"github.com/ishuar/tfskel/internal/drift"
	"github.com/ishuar/tfskel/internal/logger"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	versionsFormat  string
	versionsNoColor bool
	versionsPath    string
)

var (
	ErrPathDoesNotExist = errors.New("path does not exist")
	ErrPathNotDirectory = errors.New("path is not a directory")
)

// driftVersionCmd represents the drift version command
var driftVersionCmd = &cobra.Command{
	Use:   "version",
	Short: "Detect version drift across Terraform configurations",
	Long: `Detect and report version inconsistencies for Terraform and providers
across your workspace. This command recursively scans .tf files, extracts
version information using HCL parsing, and compares against your .tfskel configuration.

The --path flag accepts:
  • Relative paths: ./envs, ../terraform, tfskel-function-test
  • Absolute paths: /full/path/to/terraform
  • Home directory: ~/terraform (~ is expanded automatically)
  • Current directory: . or ./ (scans recursively from current location)

Note: Hidden directories (starting with .) are automatically skipped.

Examples:
  # Check for drift in current directory and all subdirectories
  tfskel drift version
  tfskel drift version --path ./

  # Check specific subdirectory
  tfskel drift version --path ./envs

  # Check with absolute path
  tfskel drift version --path /home/user/terraform

  # Check home directory with JSON output
  tfskel drift version --path ~/terraform --format json

  # Generate CSV report for CI/CD
  tfskel drift version --format csv --no-color > drift-report.csv`,
	RunE: runDriftVersions,
}

func init() {
	driftCmd.AddCommand(driftVersionCmd)

	driftVersionCmd.Flags().StringVarP(&versionsFormat, "format", "f", "table",
		"Output format: table, json, csv")
	driftVersionCmd.Flags().BoolVar(&versionsNoColor, "no-color", false,
		"Disable colored output")
	driftVersionCmd.Flags().StringVarP(&versionsPath, "path", "p", ".",
		"Path to scan for Terraform files (default: current directory)")
}

func runDriftVersions(cmd *cobra.Command, args []string) error {
	log := logger.New(viper.GetBool("verbose"))

	// Validate and normalize path
	scanPath := versionsPath
	if scanPath == "" {
		scanPath = "."
	}

	// Check if path exists
	fileInfo, err := os.Stat(scanPath)
	if err != nil {
		if os.IsNotExist(err) {
			log.Errorf("Path does not exist: %s", scanPath)
			cmd.SilenceUsage = true // Don't show usage for validation errors
			return fmt.Errorf("%w: %s", ErrPathDoesNotExist, scanPath)
		}
		log.Errorf("Failed to access path %s: %v", scanPath, err)
		cmd.SilenceUsage = true
		return fmt.Errorf("failed to access path: %w", err)
	}

	if !fileInfo.IsDir() {
		log.Errorf("Path is not a directory: %s", scanPath)
		cmd.SilenceUsage = true
		return fmt.Errorf("%w: %s", ErrPathNotDirectory, scanPath)
	}

	// Get absolute path for clearer logging
	absPath, err := filepath.Abs(scanPath)
	if err != nil {
		absPath = scanPath // fallback to original path if absolute path fails
	}

	// Load configuration
	cfg, err := config.Load(cmd, viper.GetViper())
	if err != nil {
		log.Errorf("Failed to load configuration: %v", err)
		cmd.SilenceUsage = true
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Suppress logs for machine-readable formats (JSON/CSV)
	if versionsFormat == "json" || versionsFormat == "csv" {
		log.SetOutput(os.Stderr)
	}

	log.Info("Starting tfskel version drift detection...")
	log.Infof("Scanning path: %s", absPath)

	// Create detector and scan
	detector := drift.NewDetector(scanPath)
	versionInfos, err := detector.ScanDirectory()
	if err != nil {
		log.Errorf("Failed to scan directory: %v", err)
		cmd.SilenceUsage = true
		return fmt.Errorf("failed to scan directory: %w", err)
	}

	if len(versionInfos) == 0 {
		log.Warnf("No Terraform files with version information found in %s", absPath)
		return nil
	}

	log.Infof("Found %d files with version information", len(versionInfos))

	// Analyze drift
	analyzer := drift.NewAnalyzer(cfg)
	report := analyzer.Analyze(absPath, versionInfos)

	// Format and output
	format := drift.OutputFormat(versionsFormat)
	formatter := drift.NewFormatter(!versionsNoColor)

	if err := formatter.Format(report, format, os.Stdout); err != nil {
		log.Errorf("Failed to format output: %v", err)
		cmd.SilenceUsage = true
		return fmt.Errorf("failed to format output: %w", err)
	}

	// Exit with appropriate code for CI/CD
	exitCode := report.ExitCode()
	if exitCode != 0 {
		log.Warnf("Drift detected - exiting with code %d", exitCode)
		os.Exit(exitCode)
	}

	log.Success("No drift detected - all files are in sync")
	return nil
}
