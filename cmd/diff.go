package cmd

import (
	"github.com/spf13/cobra"
)

// Output format constants shared across diff subcommands
const (
	formatJSON  = "json"
	formatCSV   = "csv"
	formatTable = "table"
)

// diffCmd represents the diff command
var diffCmd = &cobra.Command{
	Use:     "diff",
	GroupID: "main",
	Short:   "Analyze Terraform configuration drift and plan changes",
	Long: `The diff command helps you analyze terraform plan
and detect terraform & provider versions discrepancies in your
Terraform infrastructure.`,
}

func init() {
	rootCmd.AddCommand(diffCmd)
}
