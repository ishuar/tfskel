package cmd

import (
	"github.com/spf13/cobra"
)

// planCmd represents the plan command
var planCmd = &cobra.Command{
	Use:     "plan",
	GroupID: "main",
	Short:   "Analyze and review Terraform plan changes",
	Long: `The plan command helps you analyze terraform plan output
to understand the impact of planned changes before applying them.`,
}

func init() {
	rootCmd.AddCommand(planCmd)
}
