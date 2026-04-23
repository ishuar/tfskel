package cmd

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFlagErrorFunc_SuppressesUsage(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{
			name: "unknown flag on root command",
			args: []string{"--non-existent-flag"},
		},
		{
			name: "unknown flag on subcommand",
			args: []string{"init", "--bogus"},
		},
		{
			name: "unknown shorthand flag on subcommand",
			args: []string{"review", "plan", "-Z"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rootCmd := NewRootCmd()
			stdout := new(bytes.Buffer)
			stderr := new(bytes.Buffer)
			rootCmd.SetOut(stdout)
			rootCmd.SetErr(stderr)
			rootCmd.SetArgs(tt.args)

			err := rootCmd.Execute()
			require.Error(t, err, "command should fail with invalid flag")

			combined := stdout.String() + stderr.String()
			assert.NotContains(t, combined, "Usage:", "usage block should be suppressed on flag errors")
			assert.NotContains(t, combined, "Available Commands:", "available commands should be suppressed on flag errors")
		})
	}
}

func TestSilenceUsage_ArgsValidationError(t *testing.T) {
	// scaffold requires exactly 1 arg (cobra.ExactArgs(1)).
	// With SilenceUsage set at the struct level, usage is suppressed even for
	// argument validation errors — the user knows the command, they just forgot the arg.
	rootCmd := NewRootCmd()
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	rootCmd.SetOut(stdout)
	rootCmd.SetErr(stderr)
	rootCmd.SetArgs([]string{"scaffold"})

	err := rootCmd.Execute()
	require.Error(t, err, "scaffold without app-dir should fail")

	combined := stdout.String() + stderr.String()
	assert.NotContains(t, combined, "Usage:", "usage block should be suppressed on args validation errors")
	assert.NotContains(t, combined, "Main Commands:", "command list should be suppressed on args validation errors")
}

func TestRootCommand_ShowsUsageWithNoArgs(t *testing.T) {
	rootCmd := NewRootCmd()
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	rootCmd.SetOut(stdout)
	rootCmd.SetErr(stderr)
	rootCmd.SetArgs([]string{})

	err := rootCmd.Execute()
	require.NoError(t, err)

	combined := stdout.String() + stderr.String()
	assert.Contains(t, combined, "Main Commands:", "root command with no args should show usage with available commands")
}
