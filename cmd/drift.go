package cmd

import (
	"github.com/spf13/cobra"
)

// Output format constants shared across drift subcommands
const (
	formatJSON  = "json"
	formatCSV   = "csv"
	formatTable = "table"
)

// driftCmd represents the drift command
var driftCmd = &cobra.Command{
	Use:     "drift",
	GroupID: "main",
	Short:   "Analyze Terraform version drift and plan changes",
	Long: `The drift command helps you analyze terraform plan
and detect terraform & provider versions discrepancies in your
Terraform infrastructure.`,
}

func init() {
	rootCmd.AddCommand(driftCmd)
}
