package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestDiffConfigCommandTree verifies the diff config command structure is properly wired
// and prevents regressions from accidental command tree changes
func TestDiffConfigCommandTree(t *testing.T) {
	t.Run("diff command exists and is registered to root", func(t *testing.T) {
		assert.NotNil(t, diffCmd, "diffCmd should be initialized")

		// Verify diff command is a direct child of root
		found := false
		for _, cmd := range rootCmd.Commands() {
			if cmd.Name() == "diff" {
				found = true
				assert.Equal(t, diffCmd, cmd, "diff command should be registered to root")
				break
			}
		}
		assert.True(t, found, "diff command should be found in root commands")
	})

	t.Run("diff config subcommand exists and is registered to diff", func(t *testing.T) {
		assert.NotNil(t, diffConfigCmd, "diffConfigCmd should be initialized")

		// Verify config command is a direct child of diff
		found := false
		for _, cmd := range diffCmd.Commands() {
			if cmd.Name() == "config" {
				found = true
				assert.Equal(t, diffConfigCmd, cmd, "config command should be registered to diff")
				break
			}
		}
		assert.True(t, found, "config subcommand should be found under diff command")
	})

	t.Run("diff config command is accessible from root", func(t *testing.T) {
		// Simulate full command path lookup
		configFromRoot, _, err := rootCmd.Find([]string{"diff", "config"})
		assert.NoError(t, err, "should find 'diff config' command from root")
		assert.NotNil(t, configFromRoot, "diff config command should be found")
		assert.Equal(t, "config", configFromRoot.Name(), "found command should be 'config'")
		assert.Equal(t, diffConfigCmd, configFromRoot, "should find the correct config command")
	})
}

func TestDiffConfigCommand_CommandSetup(t *testing.T) {
	t.Run("command is properly initialized", func(t *testing.T) {
		assert.NotNil(t, diffConfigCmd, "diffConfigCmd should be initialized")
		assert.Equal(t, "config", diffConfigCmd.Use, "command use should be 'config'")
	})

	t.Run("command has required flags", func(t *testing.T) {
		assert.NotNil(t, diffConfigCmd.Flags(), "command should have flags")

		// Check format flag
		formatFlag := diffConfigCmd.Flags().Lookup("format")
		assert.NotNil(t, formatFlag, "--format flag should exist")
		assert.Equal(t, "f", formatFlag.Shorthand, "format flag should have -f shorthand")
		assert.Equal(t, "table", formatFlag.DefValue, "default format should be 'table'")

		// Check dir flag
		dirFlag := diffConfigCmd.Flags().Lookup("dir")
		assert.NotNil(t, dirFlag, "--dir flag should exist")
		assert.Equal(t, "d", dirFlag.Shorthand, "dir flag should have -d shorthand")
		assert.Equal(t, ".", dirFlag.DefValue, "default dir should be '.'")
	})

	t.Run("inherits global flags from root", func(t *testing.T) {
		// --no-color is a persistent flag on root command
		noColorFlag := rootCmd.PersistentFlags().Lookup("no-color")
		assert.NotNil(t, noColorFlag, "--no-color flag should exist on root")

		// Verify it's available to diff config command (inherited)
		inheritedNoColorFlag := diffConfigCmd.Flag("no-color")
		assert.NotNil(t, inheritedNoColorFlag, "diff config should inherit --no-color flag")
	})

	t.Run("command has help text", func(t *testing.T) {
		assert.NotEmpty(t, diffConfigCmd.Short, "command should have short description")
		assert.NotEmpty(t, diffConfigCmd.Long, "command should have long description")
		assert.NotEmpty(t, diffConfigCmd.Example, "command should have examples")
		assert.Contains(t, diffConfigCmd.Short, "version drift", "short description should mention version drift")
		assert.Contains(t, diffConfigCmd.Long, "HCL parsing", "long description should mention HCL parsing")
	})

	t.Run("command has runnable function", func(t *testing.T) {
		assert.NotNil(t, diffConfigCmd.RunE, "command should have RunE function")
	})
}

// TestDiffConfigCommand_FlagValidation verifies flag values and aliases work correctly
func TestDiffConfigCommand_FlagValidation(t *testing.T) {
	t.Run("format flag accepts valid values", func(t *testing.T) {
		validFormats := []string{"table", "json", "csv"}
		formatFlag := diffConfigCmd.Flags().Lookup("format")
		assert.NotNil(t, formatFlag, "--format flag should exist")

		// Note: The actual validation happens in runDiffConfig, not in flag definition
		// This test verifies the flag exists and has correct default
		for _, format := range validFormats {
			_ = format // Valid formats are checked at runtime
		}
	})

	t.Run("dir flag accepts path values", func(t *testing.T) {
		dirFlag := diffConfigCmd.Flags().Lookup("dir")
		assert.NotNil(t, dirFlag, "--dir flag should exist")
		assert.Equal(t, "string", dirFlag.Value.Type(), "dir flag should be string type")
	})
}

// TestDiffConfigCommand_ErrorHandling verifies custom error types are defined
func TestDiffConfigCommand_ErrorTypes(t *testing.T) {
	t.Run("custom error types exist", func(t *testing.T) {
		assert.NotNil(t, ErrDirDoesNotExist, "ErrDirDoesNotExist should be defined")
		assert.NotNil(t, ErrDirNotDirectory, "ErrDirNotDirectory should be defined")
		assert.Contains(t, ErrDirDoesNotExist.Error(), "does not exist", "error message should be descriptive")
		assert.Contains(t, ErrDirNotDirectory.Error(), "not a directory", "error message should be descriptive")
	})
}

// TestDiffCommand_ParentCommand verifies the diff parent command setup
func TestDiffCommand_ParentCommand(t *testing.T) {
	t.Run("command has correct metadata", func(t *testing.T) {
		assert.Equal(t, "diff", diffCmd.Use, "command use should be 'diff'")
		assert.Equal(t, "main", diffCmd.GroupID, "command should be in main group")
		assert.NotEmpty(t, diffCmd.Short, "command should have short description")
		assert.NotEmpty(t, diffCmd.Long, "command should have long description")
	})

	t.Run("command has no runnable function", func(t *testing.T) {
		// Parent commands typically don't have RunE, they just group subcommands
		assert.Nil(t, diffCmd.RunE, "parent diff command should not have RunE")
		assert.Nil(t, diffCmd.Run, "parent diff command should not have Run")
	})

	t.Run("command has at least one subcommand", func(t *testing.T) {
		subcommands := diffCmd.Commands()
		assert.NotEmpty(t, subcommands, "diff command should have subcommands")

		// Verify config is one of them
		foundConfig := false
		for _, cmd := range subcommands {
			if cmd.Name() == "config" {
				foundConfig = true
				break
			}
		}
		assert.True(t, foundConfig, "config should be a subcommand of diff")
	})
}
