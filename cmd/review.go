package cmd

import (
	"github.com/spf13/cobra"
)

// newReviewCmd returns the `review` parent command. It groups subcommands
// (currently just `plan`) and has no RunE of its own — invoking `tfskel review`
// alone shows help.
func newReviewCmd(root *rootOpts) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "review",
		GroupID: "main",
		Short:   "Review and analyze Terraform plan changes",
		Long: `The review command helps you analyze terraform plan output
to understand the impact of planned changes before applying them.`,
	}
	cmd.AddCommand(newReviewPlanCmd(root))
	return cmd
}
