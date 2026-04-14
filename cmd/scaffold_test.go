package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ishuar/tfskel/internal/fs"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// scaffoldTestSetup creates a temporary directory with a valid .tfskel.yaml config,
// changes into it, resets viper, and saves/restores all package-level flag state.
// Returns the temporary directory path.
func scaffoldTestSetup(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()
	writeTestConfig(t, tmpDir, "valid_config.yaml")
	chdirTemp(t, tmpDir)
	saveAndRestoreScaffoldFlags(t)

	viper.Reset()
	initConfig()
	t.Cleanup(func() { viper.Reset() })

	useColor = false
	return tmpDir
}

func TestValidateScaffoldParams(t *testing.T) {
	tests := []struct {
		name          string
		env           string
		region        string
		appDir        string
		wantEnv       string
		wantRegion    string
		wantAppDir    string
		wantError     bool
		errorContains string
	}{
		{
			name:       "all parameters valid",
			env:        "dev",
			region:     "us-east-1",
			appDir:     "myapp",
			wantEnv:    "dev",
			wantRegion: "us-east-1",
			wantAppDir: "myapp",
			wantError:  false,
		},
		{
			name:       "valid with different environment",
			env:        "prd",
			region:     "eu-central-1",
			appDir:     "webapp",
			wantEnv:    "prd",
			wantRegion: "eu-central-1",
			wantAppDir: "webapp",
			wantError:  false,
		},
		{
			name:       "valid with complex app directory name",
			env:        "stg",
			region:     "ap-south-1",
			appDir:     "my-complex-app-name",
			wantEnv:    "stg",
			wantRegion: "ap-south-1",
			wantAppDir: "my-complex-app-name",
			wantError:  false,
		},
		{
			name:          "missing environment",
			env:           "",
			region:        "us-east-1",
			appDir:        "myapp",
			wantError:     true,
			errorContains: "environment",
		},
		{
			name:          "missing region",
			env:           "dev",
			region:        "",
			appDir:        "myapp",
			wantError:     true,
			errorContains: "region",
		},
		{
			name:          "missing app directory",
			env:           "dev",
			region:        "us-east-1",
			appDir:        "",
			wantError:     true,
			errorContains: "app directory",
		},
		{
			name:          "all parameters missing",
			env:           "",
			region:        "",
			appDir:        "",
			wantError:     true,
			errorContains: "environment", // Should fail on first check
		},
		{
			name:          "only env provided",
			env:           "dev",
			region:        "",
			appDir:        "",
			wantError:     true,
			errorContains: "region", // Should fail on region check
		},
		{
			name:          "env and region provided, missing appdir",
			env:           "dev",
			region:        "us-east-1",
			appDir:        "",
			wantError:     true,
			errorContains: "app directory",
		},
		{
			name:       "trim leading whitespace from env",
			env:        "  dev",
			region:     "us-east-1",
			appDir:     "myapp",
			wantEnv:    "dev",
			wantRegion: "us-east-1",
			wantAppDir: "myapp",
			wantError:  false,
		},
		{
			name:       "trim trailing whitespace from region",
			env:        "dev",
			region:     "us-east-1  ",
			appDir:     "myapp",
			wantEnv:    "dev",
			wantRegion: "us-east-1",
			wantAppDir: "myapp",
			wantError:  false,
		},
		{
			name:       "trim both sides of appDir",
			env:        "dev",
			region:     "us-east-1",
			appDir:     "  myapp  ",
			wantEnv:    "dev",
			wantRegion: "us-east-1",
			wantAppDir: "myapp",
			wantError:  false,
		},
		{
			name:       "trim all parameters",
			env:        "  dev  ",
			region:     "  us-east-1  ",
			appDir:     "  myapp  ",
			wantEnv:    "dev",
			wantRegion: "us-east-1",
			wantAppDir: "myapp",
			wantError:  false,
		},
		{
			name:          "whitespace-only env is rejected",
			env:           "   ",
			region:        "us-east-1",
			appDir:        "myapp",
			wantError:     true,
			errorContains: "environment",
		},
		{
			name:          "whitespace-only region is rejected",
			env:           "dev",
			region:        "   ",
			appDir:        "myapp",
			wantError:     true,
			errorContains: "region",
		},
		{
			name:          "whitespace-only appDir is rejected",
			env:           "dev",
			region:        "us-east-1",
			appDir:        "   ",
			wantError:     true,
			errorContains: "app directory",
		},
		{
			name:       "replaces internal spaces with hyphens in appDir",
			env:        "dev",
			region:     "us-east-1",
			appDir:     "  my app  ",
			wantEnv:    "dev",
			wantRegion: "us-east-1",
			wantAppDir: "my-app",
			wantError:  false,
		},
		{
			name:       "replaces multiple spaces with hyphens",
			env:        "dev",
			region:     "us-east-1",
			appDir:     "my  complex   app",
			wantEnv:    "dev",
			wantRegion: "us-east-1",
			wantAppDir: "my-complex-app",
			wantError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotEnv, gotRegion, gotAppDir, err := validateScaffoldParams(tt.env, tt.region, tt.appDir)

			if tt.wantError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains, "error message should contain expected text")
				}
				assert.Empty(t, gotEnv, "env should be empty on error")
				assert.Empty(t, gotRegion, "region should be empty on error")
				assert.Empty(t, gotAppDir, "appDir should be empty on error")
			} else {
				assert.NoError(t, err, "should not return error for valid parameters")
				assert.Equal(t, tt.wantEnv, gotEnv, "env should match expected")
				assert.Equal(t, tt.wantRegion, gotRegion, "region should match expected")
				assert.Equal(t, tt.wantAppDir, gotAppDir, "appDir should match expected")
			}
		})
	}
}

func TestValidateScaffoldParams_ErrorMessages(t *testing.T) {
	t.Run("environment error has helpful message", func(t *testing.T) {
		_, _, _, err := validateScaffoldParams("", "us-east-1", "myapp")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "environment")
		assert.Contains(t, err.Error(), "use --env flag")
	})

	t.Run("region error has helpful message", func(t *testing.T) {
		_, _, _, err := validateScaffoldParams("dev", "", "myapp")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "region")
		assert.Contains(t, err.Error(), "use --region flag")
	})

	t.Run("appdir error has helpful message", func(t *testing.T) {
		_, _, _, err := validateScaffoldParams("dev", "us-east-1", "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "app directory")
		assert.Contains(t, err.Error(), "provide as argument")
	})
}

func TestValidateScaffoldParams_EdgeCases(t *testing.T) {
	t.Run("whitespace-only parameters are rejected", func(t *testing.T) {
		// New implementation trims and validates, so these should fail
		_, _, _, err := validateScaffoldParams("   ", "us-east-1", "myapp")
		assert.Error(t, err, "whitespace-only env should be rejected")

		_, _, _, err = validateScaffoldParams("dev", "   ", "myapp")
		assert.Error(t, err, "whitespace-only region should be rejected")

		_, _, _, err = validateScaffoldParams("dev", "us-east-1", "   ")
		assert.Error(t, err, "whitespace-only appdir should be rejected")
	})

	t.Run("parameters with special characters pass validation", func(t *testing.T) {
		// Validation only checks for empty strings, not format
		env, region, appDir, err := validateScaffoldParams("dev-v2", "us-east-1", "my_app/test")
		assert.NoError(t, err, "special characters in appdir should pass basic validation")
		assert.Equal(t, "dev-v2", env)
		assert.Equal(t, "us-east-1", region)
		assert.Equal(t, "my_app/test", appDir)

		env, region, appDir, err = validateScaffoldParams("dev", "eu-central-1", "app-with-dashes")
		assert.NoError(t, err, "dashes in appdir should pass")
		assert.Equal(t, "dev", env)
		assert.Equal(t, "eu-central-1", region)
		assert.Equal(t, "app-with-dashes", appDir)

		env, region, appDir, err = validateScaffoldParams("dev", "eu-central-1", "app_with_underscores")
		assert.NoError(t, err, "underscores in appdir should pass")
		assert.Equal(t, "dev", env)
		assert.Equal(t, "eu-central-1", region)
		assert.Equal(t, "app_with_underscores", appDir)
	})
}

func TestValidateScaffoldParams_ValidationOrder(t *testing.T) {
	t.Run("validates in order: env, region, appdir", func(t *testing.T) {
		// When all are missing, should fail on environment first
		_, _, _, err := validateScaffoldParams("", "", "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "environment", "should check env first")

		// When env is present but region missing
		_, _, _, err = validateScaffoldParams("dev", "", "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "region", "should check region second")

		// When env and region present but appdir missing
		_, _, _, err = validateScaffoldParams("dev", "us-east-1", "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "app directory", "should check appdir last")
	})
}

func TestScaffoldCommand_CommandSetup(t *testing.T) {
	t.Run("command is properly registered", func(t *testing.T) {
		assert.NotNil(t, scaffoldCmd, "scaffoldCmd should be initialized")
		assert.Equal(t, "scaffold [app-dir]", scaffoldCmd.Use, "command use pattern should be correct")
		assert.Equal(t, "main", scaffoldCmd.GroupID, "command should be in main group")
		assert.Contains(t, scaffoldCmd.Aliases, "sc", "command should have 'sc' alias")
	})

	t.Run("command has required flags", func(t *testing.T) {
		assert.NotNil(t, scaffoldCmd.Flags(), "command should have flags")

		// Check required flags exist
		envFlag := scaffoldCmd.Flags().Lookup("env")
		assert.NotNil(t, envFlag, "--env flag should exist")

		regionFlag := scaffoldCmd.Flags().Lookup("region")
		assert.NotNil(t, regionFlag, "--region flag should exist")

		// Check optional flags exist
		templatesFlag := scaffoldCmd.Flags().Lookup("templates-dir")
		assert.NotNil(t, templatesFlag, "--templates-dir flag should exist")

		s3Flag := scaffoldCmd.Flags().Lookup("s3-bucket-name")
		assert.NotNil(t, s3Flag, "--s3-bucket-name flag should exist")

		// --workflows has moved to 'tfskel init'; scaffold workflows subcommand has --env
		workflowsSubCmd, _, err := scaffoldCmd.Find([]string{"workflows"})
		assert.NoError(t, err, "scaffold workflows subcommand should be found")
		assert.NotNil(t, workflowsSubCmd, "scaffold workflows subcommand should exist")
		if workflowsSubCmd != nil {
			envFlag := workflowsSubCmd.Flags().Lookup("env")
			assert.NotNil(t, envFlag, "scaffold workflows --env flag should exist")
		}
	})

	t.Run("command requires exactly one argument", func(t *testing.T) {
		// The Args field should enforce exactly one argument
		assert.NotNil(t, scaffoldCmd.Args, "command should have Args validator")
		// cobra.ExactArgs(1) is set in the command definition
	})

	t.Run("command has help text", func(t *testing.T) {
		assert.NotEmpty(t, scaffoldCmd.Short, "command should have short description")
		assert.NotEmpty(t, scaffoldCmd.Long, "command should have long description")
		assert.NotEmpty(t, scaffoldCmd.Example, "command should have examples")

		// Verify help text references the correct command name
		assert.Contains(t, scaffoldCmd.Short, "Scaffold", "short description should mention 'Scaffold'")
		assert.Contains(t, scaffoldCmd.Long, "scaffold command", "long description should reference 'scaffold command'")
		assert.Contains(t, scaffoldCmd.Example, "tfskel scaffold", "examples should use 'tfskel scaffold'")
	})

	t.Run("upgrade-all and skip flags exist", func(t *testing.T) {
		upgradeAllFlag := scaffoldCmd.Flags().Lookup("upgrade-all")
		assert.NotNil(t, upgradeAllFlag, "--upgrade-all flag should exist")
		assert.Equal(t, "false", upgradeAllFlag.DefValue, "--upgrade-all should default to false")

		skipFlag := scaffoldCmd.Flags().Lookup("skip")
		assert.NotNil(t, skipFlag, "--skip flag should exist")
		assert.Equal(t, "", skipFlag.DefValue, "--skip should default to empty")
	})
}

func TestDirWord(t *testing.T) {
	tests := []struct {
		name string
		n    int
		want string
	}{
		{name: "singular", n: 1, want: "directory"},
		{name: "zero", n: 0, want: "directories"},
		{name: "plural", n: 2, want: "directories"},
		{name: "large number", n: 100, want: "directories"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, dirWord(tt.n))
		})
	}
}

func TestRunScaffold_FlagValidation(t *testing.T) {
	// saveAndRestore captures current package-level flag state and returns
	// a cleanup function that restores it. Call via t.Cleanup.
	saveAndRestore := func(t *testing.T) {
		t.Helper()
		origUpgradeAll := scaffoldUpgradeAll
		origUpgrade := scaffoldUpgrade
		origSkip := scaffoldSkip
		origForce := scaffoldForce
		t.Cleanup(func() {
			scaffoldUpgradeAll = origUpgradeAll
			scaffoldUpgrade = origUpgrade
			scaffoldSkip = origSkip
			scaffoldForce = origForce
		})
	}

	tests := []struct {
		name       string
		upgradeAll bool
		upgrade    bool
		skip       string
		force      bool
		wantErr    error
	}{
		{
			name:       "upgrade-all and upgrade are mutually exclusive",
			upgradeAll: true,
			upgrade:    true,
			wantErr:    ErrUpgradeAllWithUpgrade,
		},
		{
			name:    "skip requires upgrade-all",
			skip:    "foo,bar",
			wantErr: ErrSkipRequiresUpgradeAll,
		},
		{
			name:    "force requires upgrade or upgrade-all",
			force:   true,
			wantErr: ErrScaffoldForceRequiresUpgrade,
		},
		{
			name:       "force with upgrade-all is allowed (passes flag validation)",
			upgradeAll: true,
			force:      true,
			// This passes flag validation but will fail later at config.Load;
			// we only assert it doesn't return a flag-validation error.
		},
		{
			name:    "force with upgrade is allowed (passes flag validation)",
			upgrade: true,
			force:   true,
			// Passes flag validation, fails later at parameter or config stage.
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			saveAndRestore(t)
			scaffoldUpgradeAll = tt.upgradeAll
			scaffoldUpgrade = tt.upgrade
			scaffoldSkip = tt.skip
			scaffoldForce = tt.force

			// Provide a minimal cobra command (flags already live in package vars)
			cmd := &cobra.Command{}
			err := runScaffold(cmd, []string{"myapp"})

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				// Should pass flag validation; any subsequent error is fine
				assert.NotErrorIs(t, err, ErrUpgradeAllWithUpgrade)
				assert.NotErrorIs(t, err, ErrSkipRequiresUpgradeAll)
				assert.NotErrorIs(t, err, ErrScaffoldForceRequiresUpgrade)
			}
		})
	}
}

func TestScaffoldCmd_ArgsValidator(t *testing.T) {
	tests := []struct {
		name    string
		flags   map[string]string
		args    []string
		wantErr error
	}{
		{
			name:  "accepts one positional arg without upgrade-all",
			flags: map[string]string{"upgrade-all": "false"},
			args:  []string{"myapp"},
		},
		{
			name:    "rejects zero args without upgrade-all",
			flags:   map[string]string{"upgrade-all": "false"},
			args:    []string{},
			wantErr: nil, // cobra.ExactArgs returns a generic error
		},
		{
			name:  "accepts zero args with upgrade-all",
			flags: map[string]string{"upgrade-all": "true"},
			args:  []string{},
		},
		{
			name:    "rejects args with upgrade-all",
			flags:   map[string]string{"upgrade-all": "true"},
			args:    []string{"myapp"},
			wantErr: ErrUpgradeAllWithAppDir,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Build a fresh command with the upgrade-all flag
			cmd := &cobra.Command{}
			cmd.Flags().Bool("upgrade-all", false, "")
			for k, v := range tt.flags {
				require.NoError(t, cmd.Flags().Set(k, v))
			}

			err := scaffoldCmd.Args(cmd, tt.args)

			switch {
			case tt.wantErr != nil:
				assert.ErrorIs(t, err, tt.wantErr)
			case tt.name == "rejects zero args without upgrade-all":
				assert.Error(t, err, "should reject zero args")
			default:
				assert.NoError(t, err)
			}
		})
	}
}

func TestRunScaffoldUpgradeAll_ParameterValidation(t *testing.T) {
	saveAndRestore := func(t *testing.T) {
		t.Helper()
		origEnv := env
		origRegion := region
		origSkip := scaffoldSkip
		origForce := scaffoldForce
		origUpgradeAll := scaffoldUpgradeAll
		t.Cleanup(func() {
			env = origEnv
			region = origRegion
			scaffoldSkip = origSkip
			scaffoldForce = origForce
			scaffoldUpgradeAll = origUpgradeAll
		})
	}

	tests := []struct {
		name          string
		env           string
		region        string
		errorContains string
	}{
		{
			name:          "empty env returns error",
			env:           "",
			region:        "us-east-1",
			errorContains: "environment",
		},
		{
			name:          "whitespace-only env returns error",
			env:           "   ",
			region:        "us-east-1",
			errorContains: "environment",
		},
		{
			name:          "empty region returns error",
			env:           "dev",
			region:        "",
			errorContains: "region",
		},
		{
			name:          "whitespace-only region returns error",
			env:           "dev",
			region:        "   ",
			errorContains: "region",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			saveAndRestore(t)
			env = tt.env
			region = tt.region

			cmd := &cobra.Command{}
			err := runScaffoldUpgradeAll(cmd)

			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.errorContains)
		})
	}
}

func TestParseSkipList(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want map[string]bool
	}{
		{name: "empty string", raw: "", want: nil},
		{name: "single value", raw: "foo", want: map[string]bool{"foo": true}},
		{name: "multiple values", raw: "foo,bar,baz", want: map[string]bool{"foo": true, "bar": true, "baz": true}},
		{name: "whitespace trimmed", raw: " foo , bar ", want: map[string]bool{"foo": true, "bar": true}},
		{name: "trailing comma ignored", raw: "foo,bar,", want: map[string]bool{"foo": true, "bar": true}},
		{name: "empty segments ignored", raw: "foo,,bar", want: map[string]bool{"foo": true, "bar": true}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseSkipList(tt.raw)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestDiscoverAppDirs(t *testing.T) {
	t.Run("returns sorted directory names", func(t *testing.T) {
		memFS := fs.NewMemoryFileSystem()
		base := filepath.Join("envs", "prd", "eu-central-1")
		require.NoError(t, memFS.MkdirAll(base, 0755))
		require.NoError(t, memFS.MkdirAll(filepath.Join(base, "charlie"), 0755))
		require.NoError(t, memFS.MkdirAll(filepath.Join(base, "alpha"), 0755))
		require.NoError(t, memFS.MkdirAll(filepath.Join(base, "bravo"), 0755))

		dirs, err := discoverAppDirs(memFS, base, nil)
		require.NoError(t, err)
		assert.Equal(t, []string{"alpha", "bravo", "charlie"}, dirs)
	})

	t.Run("skips excluded directories", func(t *testing.T) {
		memFS := fs.NewMemoryFileSystem()
		base := filepath.Join("envs", "prd", "eu-central-1")
		require.NoError(t, memFS.MkdirAll(base, 0755))
		require.NoError(t, memFS.MkdirAll(filepath.Join(base, "app1"), 0755))
		require.NoError(t, memFS.MkdirAll(filepath.Join(base, "base-infra"), 0755))
		require.NoError(t, memFS.MkdirAll(filepath.Join(base, "app2"), 0755))

		skip := map[string]bool{"base-infra": true}
		dirs, err := discoverAppDirs(memFS, base, skip)
		require.NoError(t, err)
		assert.Equal(t, []string{"app1", "app2"}, dirs)
	})

	t.Run("skips hidden directories", func(t *testing.T) {
		memFS := fs.NewMemoryFileSystem()
		base := filepath.Join("envs", "dev", "us-east-1")
		require.NoError(t, memFS.MkdirAll(base, 0755))
		require.NoError(t, memFS.MkdirAll(filepath.Join(base, ".terraform"), 0755))
		require.NoError(t, memFS.MkdirAll(filepath.Join(base, "myapp"), 0755))

		dirs, err := discoverAppDirs(memFS, base, nil)
		require.NoError(t, err)
		assert.Equal(t, []string{"myapp"}, dirs)
	})

	t.Run("returns empty slice when no directories", func(t *testing.T) {
		memFS := fs.NewMemoryFileSystem()
		base := filepath.Join("envs", "dev", "us-east-1")
		require.NoError(t, memFS.MkdirAll(base, 0755))

		dirs, err := discoverAppDirs(memFS, base, nil)
		require.NoError(t, err)
		assert.Empty(t, dirs)
	})

	t.Run("errors when path does not exist", func(t *testing.T) {
		memFS := fs.NewMemoryFileSystem()
		_, err := discoverAppDirs(memFS, "nonexistent", nil)
		assert.Error(t, err)
	})

	t.Run("works with OSFileSystem", func(t *testing.T) {
		tmpDir := t.TempDir()
		base := filepath.Join(tmpDir, "envs", "dev", "us-east-1")
		require.NoError(t, os.MkdirAll(filepath.Join(base, "app1"), 0755))
		require.NoError(t, os.MkdirAll(filepath.Join(base, "app2"), 0755))
		// Create a file (should be skipped)
		require.NoError(t, os.WriteFile(filepath.Join(base, "somefile.txt"), []byte("hi"), 0644))

		osFS := fs.NewOSFileSystem()
		dirs, err := discoverAppDirs(osFS, base, nil)
		require.NoError(t, err)
		assert.Equal(t, []string{"app1", "app2"}, dirs)
	})
}

func TestRunScaffold_HappyPath(t *testing.T) {
	t.Run("creates directory structure and files", func(t *testing.T) {
		tmpDir := scaffoldTestSetup(t)
		env = "dev"
		region = "eu-central-1"

		cmd := newTestCmd(t)
		err := runScaffold(cmd, []string{"myapp"})
		require.NoError(t, err)

		appDir := filepath.Join(tmpDir, "envs", "dev", "eu-central-1", "myapp")
		assert.DirExists(t, appDir)
		assert.FileExists(t, filepath.Join(appDir, "backend.tf"))
		assert.FileExists(t, filepath.Join(appDir, "versions.tf"))
	})

	t.Run("with upgrade flag passes through to generator", func(t *testing.T) {
		tmpDir := scaffoldTestSetup(t)
		env = "dev"
		region = "eu-central-1"

		cmd := newTestCmd(t)
		// First scaffold to create files
		err := runScaffold(cmd, []string{"myapp"})
		require.NoError(t, err)

		appDir := filepath.Join(tmpDir, "envs", "dev", "eu-central-1", "myapp")
		assert.DirExists(t, appDir)

		// Now run with upgrade — should succeed (files already up to date)
		scaffoldUpgrade = true
		err = runScaffold(cmd, []string{"myapp"})
		require.NoError(t, err)
	})

	t.Run("dry-run does not write files", func(t *testing.T) {
		tmpDir := scaffoldTestSetup(t)
		env = "dev"
		region = "eu-central-1"
		dryRun = true

		cmd := newTestCmd(t)
		err := runScaffold(cmd, []string{"myapp"})
		require.NoError(t, err)

		appDir := filepath.Join(tmpDir, "envs", "dev", "eu-central-1", "myapp")
		assert.NoDirExists(t, appDir, "dry-run should not create directories")
	})
}

func TestRunScaffoldUpgradeAll_Integration(t *testing.T) {
	t.Run("upgrades all app dirs in a region", func(t *testing.T) {
		tmpDir := scaffoldTestSetup(t)
		env = "dev"
		region = "eu-central-1"
		scaffoldUpgradeAll = true

		// Pre-create app directories with scaffold files
		cmd := newTestCmd(t)
		scaffoldUpgradeAll = false
		require.NoError(t, runScaffold(cmd, []string{"app1"}))
		require.NoError(t, runScaffold(cmd, []string{"app2"}))

		// Now run upgrade-all
		scaffoldUpgradeAll = true
		err := runScaffoldUpgradeAll(cmd)
		require.NoError(t, err)

		// Both app dirs should still exist with files
		for _, app := range []string{"app1", "app2"} {
			appDir := filepath.Join(tmpDir, "envs", "dev", "eu-central-1", app)
			assert.DirExists(t, appDir)
			assert.FileExists(t, filepath.Join(appDir, "backend.tf"))
		}
	})

	t.Run("skips directories in skip list", func(t *testing.T) {
		scaffoldTestSetup(t)
		env = "dev"
		region = "eu-central-1"

		cmd := newTestCmd(t)
		require.NoError(t, runScaffold(cmd, []string{"app1"}))
		require.NoError(t, runScaffold(cmd, []string{"app2"}))
		require.NoError(t, runScaffold(cmd, []string{"app3"}))

		scaffoldUpgradeAll = true
		scaffoldSkip = "app2"
		err := runScaffoldUpgradeAll(cmd)
		require.NoError(t, err)
	})

	t.Run("base dir does not exist", func(t *testing.T) {
		scaffoldTestSetup(t)
		env = "dev"
		region = "nonexistent-region-1"
		scaffoldUpgradeAll = true

		cmd := newTestCmd(t)
		err := runScaffoldUpgradeAll(cmd)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrBaseDirNotExist)
	})

	t.Run("no app dirs found", func(t *testing.T) {
		tmpDir := scaffoldTestSetup(t)
		env = "dev"
		region = "eu-central-1"
		scaffoldUpgradeAll = true

		// Create the region dir but leave it empty
		require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "envs", "dev", "eu-central-1"), 0755))

		cmd := newTestCmd(t)
		err := runScaffoldUpgradeAll(cmd)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrNoAppDirsFound)
	})

	t.Run("dry-run mode", func(t *testing.T) {
		scaffoldTestSetup(t)
		env = "dev"
		region = "eu-central-1"
		dryRun = true
		scaffoldUpgradeAll = true

		// Pre-create app dirs (need real dirs for discovery)
		cmd := newTestCmd(t)
		dryRun = false
		scaffoldUpgradeAll = false
		require.NoError(t, runScaffold(cmd, []string{"app1"}))

		dryRun = true
		scaffoldUpgradeAll = true
		err := runScaffoldUpgradeAll(cmd)
		require.NoError(t, err)
	})
}

func TestRunScaffoldWorkflows(t *testing.T) {
	t.Run("generates workflow files", func(t *testing.T) {
		tmpDir := scaffoldTestSetup(t)
		workflowsEnv = "dev"

		cmd := newTestCmd(t)
		err := runScaffoldWorkflows(cmd, []string{})
		require.NoError(t, err)

		// Check that workflow files were created
		workflowDir := filepath.Join(tmpDir, ".github", "workflows")
		assert.DirExists(t, workflowDir)
	})

	t.Run("force without upgrade returns error", func(t *testing.T) {
		scaffoldTestSetup(t)
		workflowsEnv = "dev"
		workflowForce = true
		workflowUpgrade = false

		cmd := newTestCmd(t)
		err := runScaffoldWorkflows(cmd, []string{})
		assert.ErrorIs(t, err, ErrForceRequiresUpgrade)
	})

	t.Run("empty env returns error", func(t *testing.T) {
		scaffoldTestSetup(t)
		workflowsEnv = ""

		cmd := newTestCmd(t)
		err := runScaffoldWorkflows(cmd, []string{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "environment")
	})

	t.Run("whitespace-only env returns error", func(t *testing.T) {
		scaffoldTestSetup(t)
		workflowsEnv = "   "

		cmd := newTestCmd(t)
		err := runScaffoldWorkflows(cmd, []string{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "environment")
	})

	t.Run("dry-run mode", func(t *testing.T) {
		scaffoldTestSetup(t)
		workflowsEnv = "dev"
		dryRun = true

		cmd := newTestCmd(t)
		err := runScaffoldWorkflows(cmd, []string{})
		require.NoError(t, err)
	})
}
