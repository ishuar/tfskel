package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestReviewCommandTree verifies the review command structure is properly wired
// and prevents regressions from accidental command tree changes
func TestReviewCommandTree(t *testing.T) {
	t.Run("review command exists and is registered to root", func(t *testing.T) {
		assert.NotNil(t, reviewCmd, "reviewCmd should be initialized")

		// Verify review command is a direct child of root
		found := false
		for _, cmd := range rootCmd.Commands() {
			if cmd.Name() == "review" {
				found = true
				assert.Equal(t, reviewCmd, cmd, "review command should be registered to root")
				break
			}
		}
		assert.True(t, found, "review command should be found in root commands")
	})

	t.Run("review plan subcommand exists and is registered to review", func(t *testing.T) {
		assert.NotNil(t, reviewPlanCmd, "reviewPlanCmd should be initialized")

		// Verify plan command is a direct child of review
		found := false
		for _, cmd := range reviewCmd.Commands() {
			if cmd.Name() == "plan" {
				found = true
				assert.Equal(t, reviewPlanCmd, cmd, "plan command should be registered to review")
				break
			}
		}
		assert.True(t, found, "plan subcommand should be found under review command")
	})

	t.Run("old 'plan review' command structure does not exist", func(t *testing.T) {
		// Verify there is NO 'plan' command at root level
		for _, cmd := range rootCmd.Commands() {
			assert.NotEqual(t, "plan", cmd.Name(),
				"root should not have a 'plan' command (breaking change: plan review -> review plan)")
		}
	})

	t.Run("review command is accessible from root", func(t *testing.T) {
		// Simulate command lookup as cobra does it
		reviewFromRoot, _, err := rootCmd.Find([]string{"review"})
		assert.NoError(t, err, "should find review command from root")
		assert.NotNil(t, reviewFromRoot, "review command should be found")
		assert.Equal(t, "review", reviewFromRoot.Name(), "found command should be 'review'")
	})

	t.Run("review plan subcommand is accessible from root", func(t *testing.T) {
		// Simulate full command path lookup
		planFromRoot, _, err := rootCmd.Find([]string{"review", "plan"})
		assert.NoError(t, err, "should find 'review plan' command from root")
		assert.NotNil(t, planFromRoot, "review plan command should be found")
		assert.Equal(t, "plan", planFromRoot.Name(), "found command should be 'plan'")
		assert.Equal(t, reviewPlanCmd, planFromRoot, "should find the correct plan command")
	})
}

func TestReviewCommand_CommandSetup(t *testing.T) {
	t.Run("command has correct metadata", func(t *testing.T) {
		assert.Equal(t, "review", reviewCmd.Use, "command use should be 'review'")
		assert.Equal(t, "main", reviewCmd.GroupID, "command should be in main group")
		assert.NotEmpty(t, reviewCmd.Short, "command should have short description")
		assert.NotEmpty(t, reviewCmd.Long, "command should have long description")
	})

	t.Run("command has no runnable function", func(t *testing.T) {
		// Parent commands typically don't have RunE, they just group subcommands
		assert.Nil(t, reviewCmd.RunE, "parent review command should not have RunE")
		assert.Nil(t, reviewCmd.Run, "parent review command should not have Run")
	})
}

func TestReviewPlanCommand_CommandSetup(t *testing.T) {
	t.Run("command is properly initialized", func(t *testing.T) {
		assert.NotNil(t, reviewPlanCmd, "reviewPlanCmd should be initialized")
		assert.Equal(t, "plan", reviewPlanCmd.Use, "command use should be 'plan'")
	})

	t.Run("command has required flags", func(t *testing.T) {
		assert.NotNil(t, reviewPlanCmd.Flags(), "command should have flags")

		// Check required flag
		jsonFileFlag := reviewPlanCmd.Flags().Lookup("json-file")
		assert.NotNil(t, jsonFileFlag, "--json-file flag should exist")
		// Note: We can't easily test MarkFlagRequired here, but we verify the flag exists

		// Check optional flags
		formatFlag := reviewPlanCmd.Flags().Lookup("format")
		assert.NotNil(t, formatFlag, "--format flag should exist")
		assert.Equal(t, "table", formatFlag.DefValue, "default format should be 'table'")

		topResourcesFlag := reviewPlanCmd.Flags().Lookup("top-resources-count")
		assert.NotNil(t, topResourcesFlag, "--top-resources-count flag should exist")
	})

	t.Run("inherits global flags from root", func(t *testing.T) {
		// --no-color is now a persistent flag on root command
		noColorFlag := rootCmd.PersistentFlags().Lookup("no-color")
		assert.NotNil(t, noColorFlag, "--no-color flag should exist on root")
	})

	t.Run("command has help text", func(t *testing.T) {
		assert.NotEmpty(t, reviewPlanCmd.Short, "command should have short description")
		assert.NotEmpty(t, reviewPlanCmd.Long, "command should have long description")
		assert.NotEmpty(t, reviewPlanCmd.Example, "command should have examples")
	})

	t.Run("command has runnable function", func(t *testing.T) {
		assert.NotNil(t, reviewPlanCmd.RunE, "command should have RunE function")
	})
}

// TestReviewPlanCommand_FlagAliases verifies short flags work correctly
func TestReviewPlanCommand_FlagAliases(t *testing.T) {
	t.Run("format flag has short alias", func(t *testing.T) {
		formatFlag := reviewPlanCmd.Flags().Lookup("format")
		assert.NotNil(t, formatFlag, "--format flag should exist")
		assert.Equal(t, "f", formatFlag.Shorthand, "format flag should have -f shorthand")
	})
}
