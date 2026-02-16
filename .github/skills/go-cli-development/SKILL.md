---
name: go-cli-development
description: Best practices for building Go CLI applications using cobra and viper. Use this when creating new commands, handling flags, or implementing CLI workflows.
---

# Go CLI Development with Cobra

This skill guides the development of CLI commands for tfskel using cobra and viper.

## Command Structure

Each command should be in its own file under `cmd/`:

```go
package cmd

import (
    "github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
    Use:   "init [flags]",
    Short: "Initialize a new Terraform project structure",
    Long: `Initialize a new Terraform project with best-practice directory structure.

Creates environment directories (dev, stg, prd) with regional subdirectories,
and sets up linting configuration files.`,
    Example: `  # Initialize with default settings
  tfskel init

  # Initialize with custom environments and regions
  tfskel init --envs dev,stg,prd --regions eu-central-1,us-east-1`,
    RunE: runInit,
}

func init() {
    rootCmd.AddCommand(initCmd)

    initCmd.Flags().StringP("project-name", "p", "", "Project name")
    initCmd.Flags().StringSliceP("envs", "e", []string{"dev", "stg", "prd"}, "Environments to create")
    initCmd.Flags().StringSliceP("regions", "r", []string{"eu-central-1"}, "AWS regions")
}

func runInit(cmd *cobra.Command, args []string) error {
    // Implementation
    return nil
}
```

## Flag Handling
- Use viper for configuration hierarchy (flags > env vars > config file > defaults)
- Always provide sensible defaults
- Validate flags early in RunE function

```go
func runInit(cmd *cobra.Command, args []string) error {
    // Bind flags to viper
    viper.BindPFlag("project-name", cmd.Flags().Lookup("project-name"))

    // Get values with fallback
    projectName := viper.GetString("project-name")
    if projectName == "" {
        projectName = filepath.Base(getCurrentDir())
    }

    // Validate
    if err := validateProjectName(projectName); err != nil {
        return fmt.Errorf("invalid project name: %w", err)
    }

    // Execute logic
    return createProjectStructure(projectName, ...)
}
```

## Error Handling
1. Return errors from RunE, don't call os.Exit()
2. Provide actionable error messages
3. Use error wrapping for context

```go
if err := os.MkdirAll(path, 0755); err != nil {
    return fmt.Errorf("failed to create directory %s: %w", path, err)
}
```

## Output Formatting
- Use consistent output formatting:

```go
import "github.com/fatih/color"

var (
    successMsg = color.New(color.FgGreen).SprintFunc()
    errorMsg   = color.New(color.FgRed).SprintFunc()
    infoMsg    = color.New(color.FgCyan).SprintFunc()
)

fmt.Printf("%s Project initialized successfully\n", successMsg("✓"))
fmt.Printf("%s Created directory: %s\n", infoMsg("→"), path)
```
## Testing Commands
- Test commands using buffer capture:
```go
func TestInitCommand(t *testing.T) {
    cmd := &cobra.Command{}
    buf := new(bytes.Buffer)
    cmd.SetOut(buf)

    err := runInit(cmd, []string{})
    assert.NoError(t, err)
    assert.Contains(t, buf.String(), "initialized successfully")
}
```
