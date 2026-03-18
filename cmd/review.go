package cmd

import (
	"github.com/spf13/cobra"
)

// reviewCmd represents the review command
var reviewCmd = &cobra.Command{
	Use:     "review",
	GroupID: "main",
	Short:   "Review and analyze Terraform plan changes",
	Long: `The review command helps you analyze terraform plan output
to understand the impact of planned changes before applying them.`,
}

func init() {
	rootCmd.AddCommand(reviewCmd)
}
