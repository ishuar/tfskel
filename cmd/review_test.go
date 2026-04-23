package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestReviewCommandTree verifies the review command tree is wired correctly.
func TestReviewCommandTree(t *testing.T) {
	root := NewRootCmd()

	t.Run("review is a direct child of root", func(t *testing.T) {
		reviewFromRoot, _, err := root.Find([]string{"review"})
		require.NoError(t, err, "should find review command from root")
		require.NotNil(t, reviewFromRoot)
		assert.Equal(t, "review", reviewFromRoot.Name())
	})

	t.Run("review plan is reachable from root", func(t *testing.T) {
		planFromRoot, _, err := root.Find([]string{"review", "plan"})
		require.NoError(t, err, "should find 'review plan' command from root")
		require.NotNil(t, planFromRoot)
		assert.Equal(t, "plan", planFromRoot.Name())
	})

	t.Run("no top-level 'plan' command", func(t *testing.T) {
		for _, c := range root.Commands() {
			assert.NotEqual(t, "plan", c.Name(),
				"root should not have a 'plan' command (breaking change: plan review -> review plan)")
		}
	})
}

func TestReviewCommand_CommandSetup(t *testing.T) {
	root := NewRootCmd()
	reviewCmd, _, err := root.Find([]string{"review"})
	require.NoError(t, err)

	t.Run("command has correct metadata", func(t *testing.T) {
		assert.Equal(t, "review", reviewCmd.Use)
		assert.Equal(t, "main", reviewCmd.GroupID)
		assert.NotEmpty(t, reviewCmd.Short)
		assert.NotEmpty(t, reviewCmd.Long)
	})

	t.Run("command has no runnable function", func(t *testing.T) {
		// Parent commands typically don't have RunE, they just group subcommands.
		assert.Nil(t, reviewCmd.RunE, "parent review command should not have RunE")
		assert.Nil(t, reviewCmd.Run, "parent review command should not have Run")
	})
}

func TestReviewPlanCommand_CommandSetup(t *testing.T) {
	root := NewRootCmd()
	planCmd, _, err := root.Find([]string{"review", "plan"})
	require.NoError(t, err)

	t.Run("command is properly initialized", func(t *testing.T) {
		assert.Equal(t, "plan", planCmd.Use)
	})

	t.Run("command has required flags", func(t *testing.T) {
		jsonFileFlag := planCmd.Flags().Lookup("json-file")
		assert.NotNil(t, jsonFileFlag, "--json-file flag should exist")

		formatFlag := planCmd.Flags().Lookup("format")
		assert.NotNil(t, formatFlag, "--format flag should exist")
		assert.Equal(t, "table", formatFlag.DefValue)

		topResourcesFlag := planCmd.Flags().Lookup("top-resources-count")
		assert.NotNil(t, topResourcesFlag, "--top-resources-count flag should exist")
	})

	t.Run("inherits global flags from root", func(t *testing.T) {
		noColorFlag := root.PersistentFlags().Lookup("no-color")
		assert.NotNil(t, noColorFlag, "--no-color flag should exist on root")

		inheritedNoColorFlag := planCmd.Flag("no-color")
		assert.NotNil(t, inheritedNoColorFlag, "review plan should inherit --no-color flag")
	})

	t.Run("command has help text", func(t *testing.T) {
		assert.NotEmpty(t, planCmd.Short)
		assert.NotEmpty(t, planCmd.Long)
		assert.NotEmpty(t, planCmd.Example)
	})

	t.Run("command has runnable function", func(t *testing.T) {
		assert.NotNil(t, planCmd.RunE)
	})
}

func TestReviewPlanCommand_FlagAliases(t *testing.T) {
	root := NewRootCmd()
	planCmd, _, err := root.Find([]string{"review", "plan"})
	require.NoError(t, err)

	formatFlag := planCmd.Flags().Lookup("format")
	require.NotNil(t, formatFlag)
	assert.Equal(t, "f", formatFlag.Shorthand)
}
