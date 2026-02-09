package cmd

import (
	"github.com/spf13/cobra"
)

// driftCmd represents the drift command
var driftCmd = &cobra.Command{
	Use:   "drift",
	Short: "Analyze Terraform drift and plan changes",
	Long: `The drift command helps you detect and analyze discrepancies in your
Terraform infrastructure. It provides multiple analysis modes:

  • version  - Detect version inconsistencies across configurations
  • plan     - Analyze terraform plan output for change review
  • all      - Combined version drift and plan analysis

Use drift subcommands for targeted analysis of your Terraform workspace.

Examples:
  # Check for version drift
  tfskel drift version

  # Analyze a terraform plan file
  tfskel drift plan --plan-file tfplan.json

  # Run both version and plan analysis
  tfskel drift all --plan-file tfplan.json`,
}

func init() {
	rootCmd.AddCommand(driftCmd)
}
