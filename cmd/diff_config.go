package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ishuar/tfskel/internal/config"
	"github.com/ishuar/tfskel/internal/diff"
	"github.com/ishuar/tfskel/internal/format"
	"github.com/ishuar/tfskel/internal/logger"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	configFormat string
	configDir    string
)

var (
	// ErrDirDoesNotExist indicates the specified directory does not exist
	ErrDirDoesNotExist = errors.New("directory does not exist")
	// ErrDirNotDirectory indicates the specified path is not a directory
	ErrDirNotDirectory = errors.New("target is not a directory")
)

// diffConfigCmd represents the diff config command
var diffConfigCmd = &cobra.Command{
	Use:   "config",
	Short: "Detect version drift across Terraform configurations",
	Long: `Detect and report version inconsistencies for Terraform
and providers across your workspace. This command recursively
scans .tf files, extracts version information using HCL parsing,
and compares against your .tfskel configuration.
Results can be output as JSON, table, or CSV

Note: Hidden directories (starting with .) are automatically skipped.`,

	Example: `  # Check for drift in current directory and all subdirectories
  tfskel diff config
  tfskel diff config --dir ./

  # Check specific subdirectory
  tfskel diff config --dir ./envs

  # Check home directory with JSON output
  tfskel diff config --dir ~/terraform --format json`,
	RunE: runDiffConfig,
}

func init() {
	diffCmd.AddCommand(diffConfigCmd)

	diffConfigCmd.Flags().StringVarP(&configFormat, "format", "f", "table",
		"Output format: table, json, csv")
	diffConfigCmd.Flags().StringVarP(&configDir, "dir", "d", ".",
		"Directory to scan for Terraform files (default: current directory)")
}

func runDiffConfig(cmd *cobra.Command, _ []string) error {
	log := logger.New(viper.GetBool("verbose"))

	// Validate and normalize directory
	scanDir := configDir
	if scanDir == "" {
		scanDir = "."
	}

	// Check if directory exists
	fileInfo, err := os.Stat(scanDir)
	if err != nil {
		if os.IsNotExist(err) {
			log.Errorf("Directory does not exist: %s", scanDir)
			cmd.SilenceUsage = true
			return fmt.Errorf("%w: %s", ErrDirDoesNotExist, scanDir)
		}
		log.Errorf("Failed to access directory %s: %v", scanDir, err)
		cmd.SilenceUsage = true
		return fmt.Errorf("failed to access directory: %w", err)
	}

	if !fileInfo.IsDir() {
		log.Errorf("Target is not a directory: %s", scanDir)
		cmd.SilenceUsage = true
		return fmt.Errorf("%w: %s", ErrDirNotDirectory, scanDir)
	}

	// Get absolute path for clearer logging
	absPath, err := filepath.Abs(scanDir)
	if err != nil {
		absPath = scanDir
	}

	// Load configuration
	cfg, err := config.Load(cmd, viper.GetViper())
	if err != nil {
		log.Errorf("Failed to load configuration: %v", err)
		cmd.SilenceUsage = true
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Suppress logs for machine-readable formats (JSON/CSV)
	if configFormat == string(format.FormatJSON) || configFormat == string(format.FormatCSV) {
		log.SetOutput(os.Stderr)
	}

	log.Info("Starting tfskel version drift detection...")
	log.Infof("Scanning directory: %s", absPath)

	// Create detector and scan
	detector := diff.NewDetector(scanDir)
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
	analyzer := diff.NewAnalyzer(cfg)
	report := analyzer.Analyze(absPath, versionInfos)

	// Format and output
	outputFormat := format.OutputFormat(configFormat)

	// Color profile already initialized in root PersistentPreRunE
	// Reuse the global useColor decision to avoid redundant env var checks
	formatter := diff.NewFormatter(useColor)

	if err := formatter.Format(report, outputFormat, os.Stdout); err != nil {
		log.Errorf("Failed to format output: %v", err)
		cmd.SilenceUsage = true
		return fmt.Errorf("failed to format output: %w", err)
	}

	// Exit with appropriate code for CI/CD
	exitCode := report.ExitCode()
	if exitCode != 0 {
		log.Warnf("Drift detected - exiting with code %d", exitCode)
		cmd.SilenceUsage = true
		return NewExitError(exitCode, "")
	}

	log.Success("No drift detected - all files are in sync")
	return nil
}
