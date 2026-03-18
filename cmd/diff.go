package cmd

import (
	"github.com/spf13/cobra"
)

// diffCmd represents the diff command
var diffCmd = &cobra.Command{
	Use:     "diff",
	GroupID: "main",
	Short:   "Analyze .tfskel.yaml and Terraform configuration version drift",
	Long: `The diff command helps you detect .tfskel config, Terraform
& provider version discrepancies in your Terraform infrastructure.`,
}

func init() {
	rootCmd.AddCommand(diffCmd)
}
