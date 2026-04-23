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

// scaffoldTestSetup creates a temporary directory with a valid .tfskel.yaml
// config, changes into it, and resets viper. Returns the temporary directory.
func scaffoldTestSetup(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()
	writeTestConfig(t, tmpDir, "valid_config.yaml")
	chdirTemp(t, tmpDir)

	viper.Reset()
	(&rootOpts{}).initConfig()
	t.Cleanup(func() { viper.Reset() })

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
		{name: "all parameters valid", env: "dev", region: "us-east-1", appDir: "myapp", wantEnv: "dev", wantRegion: "us-east-1", wantAppDir: "myapp"},
		{name: "valid with different environment", env: "prd", region: "eu-central-1", appDir: "webapp", wantEnv: "prd", wantRegion: "eu-central-1", wantAppDir: "webapp"},
		{name: "valid with complex app directory name", env: "stg", region: "ap-south-1", appDir: "my-complex-app-name", wantEnv: "stg", wantRegion: "ap-south-1", wantAppDir: "my-complex-app-name"},
		{name: "missing environment", env: "", region: "us-east-1", appDir: "myapp", wantError: true, errorContains: "environment"},
		{name: "missing region", env: "dev", region: "", appDir: "myapp", wantError: true, errorContains: "region"},
		{name: "missing app directory", env: "dev", region: "us-east-1", appDir: "", wantError: true, errorContains: "app directory"},
		{name: "all parameters missing", env: "", region: "", appDir: "", wantError: true, errorContains: "environment"},
		{name: "only env provided", env: "dev", region: "", appDir: "", wantError: true, errorContains: "region"},
		{name: "env and region provided, missing appdir", env: "dev", region: "us-east-1", appDir: "", wantError: true, errorContains: "app directory"},
		{name: "trim leading whitespace from env", env: "  dev", region: "us-east-1", appDir: "myapp", wantEnv: "dev", wantRegion: "us-east-1", wantAppDir: "myapp"},
		{name: "trim trailing whitespace from region", env: "dev", region: "us-east-1  ", appDir: "myapp", wantEnv: "dev", wantRegion: "us-east-1", wantAppDir: "myapp"},
		{name: "trim both sides of appDir", env: "dev", region: "us-east-1", appDir: "  myapp  ", wantEnv: "dev", wantRegion: "us-east-1", wantAppDir: "myapp"},
		{name: "trim all parameters", env: "  dev  ", region: "  us-east-1  ", appDir: "  myapp  ", wantEnv: "dev", wantRegion: "us-east-1", wantAppDir: "myapp"},
		{name: "whitespace-only env is rejected", env: "   ", region: "us-east-1", appDir: "myapp", wantError: true, errorContains: "environment"},
		{name: "whitespace-only region is rejected", env: "dev", region: "   ", appDir: "myapp", wantError: true, errorContains: "region"},
		{name: "whitespace-only appDir is rejected", env: "dev", region: "us-east-1", appDir: "   ", wantError: true, errorContains: "app directory"},
		{name: "replaces internal spaces with hyphens in appDir", env: "dev", region: "us-east-1", appDir: "  my app  ", wantEnv: "dev", wantRegion: "us-east-1", wantAppDir: "my-app"},
		{name: "replaces multiple spaces with hyphens", env: "dev", region: "us-east-1", appDir: "my  complex   app", wantEnv: "dev", wantRegion: "us-east-1", wantAppDir: "my-complex-app"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotEnv, gotRegion, gotAppDir, err := validateScaffoldParams(tt.env, tt.region, tt.appDir)

			if tt.wantError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
				assert.Empty(t, gotEnv)
				assert.Empty(t, gotRegion)
				assert.Empty(t, gotAppDir)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantEnv, gotEnv)
				assert.Equal(t, tt.wantRegion, gotRegion)
				assert.Equal(t, tt.wantAppDir, gotAppDir)
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

func TestScaffoldCommand_CommandSetup(t *testing.T) {
	root := NewRootCmd()
	scaffoldCmd, _, err := root.Find([]string{"scaffold"})
	require.NoError(t, err)

	t.Run("command is properly registered", func(t *testing.T) {
		assert.Equal(t, "scaffold [app-dir]", scaffoldCmd.Use)
		assert.Equal(t, "main", scaffoldCmd.GroupID)
		assert.Contains(t, scaffoldCmd.Aliases, "sc")
	})

	t.Run("command has required flags", func(t *testing.T) {
		assert.NotNil(t, scaffoldCmd.Flags().Lookup("env"))
		assert.NotNil(t, scaffoldCmd.Flags().Lookup("region"))
		assert.NotNil(t, scaffoldCmd.Flags().Lookup("templates-dir"))
		assert.NotNil(t, scaffoldCmd.Flags().Lookup("s3-bucket-name"))

		workflowsSubCmd, _, werr := scaffoldCmd.Find([]string{"workflows"})
		require.NoError(t, werr)
		require.NotNil(t, workflowsSubCmd)
		assert.NotNil(t, workflowsSubCmd.Flags().Lookup("env"))
	})

	t.Run("command has args validator", func(t *testing.T) {
		assert.NotNil(t, scaffoldCmd.Args)
	})

	t.Run("command has help text", func(t *testing.T) {
		assert.NotEmpty(t, scaffoldCmd.Short)
		assert.NotEmpty(t, scaffoldCmd.Long)
		assert.NotEmpty(t, scaffoldCmd.Example)
		assert.Contains(t, scaffoldCmd.Short, "Scaffold")
	})

	t.Run("upgrade-all and skip flags exist", func(t *testing.T) {
		upgradeAllFlag := scaffoldCmd.Flags().Lookup("upgrade-all")
		require.NotNil(t, upgradeAllFlag)
		assert.Equal(t, "false", upgradeAllFlag.DefValue)

		skipFlag := scaffoldCmd.Flags().Lookup("skip")
		require.NotNil(t, skipFlag)
		assert.Equal(t, "", skipFlag.DefValue)
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
	tests := []struct {
		name       string
		upgradeAll bool
		upgrade    bool
		skip       string
		force      bool
		wantErr    error
	}{
		{name: "upgrade-all and upgrade are mutually exclusive", upgradeAll: true, upgrade: true, wantErr: ErrUpgradeAllWithUpgrade},
		{name: "skip requires upgrade-all", skip: "foo,bar", wantErr: ErrSkipRequiresUpgradeAll},
		{name: "force requires upgrade or upgrade-all", force: true, wantErr: ErrScaffoldForceRequiresUpgrade},
		{name: "force with upgrade-all is allowed (passes flag validation)", upgradeAll: true, force: true},
		{name: "force with upgrade is allowed (passes flag validation)", upgrade: true, force: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := &scaffoldOpts{
				root:       &rootOpts{},
				upgradeAll: tt.upgradeAll,
				upgrade:    tt.upgrade,
				skip:       tt.skip,
				force:      tt.force,
			}
			err := opts.run(&cobra.Command{}, []string{"myapp"})

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				// Should pass flag validation; any subsequent error is fine.
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
		{name: "accepts one positional arg without upgrade-all", flags: map[string]string{"upgrade-all": "false"}, args: []string{"myapp"}},
		{name: "rejects zero args without upgrade-all", flags: map[string]string{"upgrade-all": "false"}, args: []string{}, wantErr: nil},
		{name: "accepts zero args with upgrade-all", flags: map[string]string{"upgrade-all": "true"}, args: []string{}},
		{name: "rejects args with upgrade-all", flags: map[string]string{"upgrade-all": "true"}, args: []string{"myapp"}, wantErr: ErrUpgradeAllWithAppDir},
	}

	opts := &scaffoldOpts{root: &rootOpts{}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &cobra.Command{}
			cmd.Flags().Bool("upgrade-all", false, "")
			for k, v := range tt.flags {
				require.NoError(t, cmd.Flags().Set(k, v))
			}

			err := opts.args(cmd, tt.args)

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
	tests := []struct {
		name          string
		env           string
		region        string
		errorContains string
	}{
		{name: "empty env returns error", env: "", region: "us-east-1", errorContains: "environment"},
		{name: "whitespace-only env returns error", env: "   ", region: "us-east-1", errorContains: "environment"},
		{name: "empty region returns error", env: "dev", region: "", errorContains: "region"},
		{name: "whitespace-only region returns error", env: "dev", region: "   ", errorContains: "region"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := &scaffoldOpts{root: &rootOpts{}, env: tt.env, region: tt.region, upgradeAll: true}
			err := opts.runUpgradeAll(&cobra.Command{})
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
			assert.Equal(t, tt.want, parseSkipList(tt.raw))
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
		opts := &scaffoldOpts{root: &rootOpts{}, env: "dev", region: "eu-central-1"}

		err := opts.run(newTestCmd(t), []string{"myapp"})
		require.NoError(t, err)

		appDir := filepath.Join(tmpDir, "envs", "dev", "eu-central-1", "myapp")
		assert.DirExists(t, appDir)
		assert.FileExists(t, filepath.Join(appDir, "backend.tf"))
		assert.FileExists(t, filepath.Join(appDir, "versions.tf"))
	})

	t.Run("with upgrade flag passes through to generator", func(t *testing.T) {
		tmpDir := scaffoldTestSetup(t)
		opts := &scaffoldOpts{root: &rootOpts{}, env: "dev", region: "eu-central-1"}

		cmd := newTestCmd(t)
		require.NoError(t, opts.run(cmd, []string{"myapp"}))

		appDir := filepath.Join(tmpDir, "envs", "dev", "eu-central-1", "myapp")
		assert.DirExists(t, appDir)

		opts.upgrade = true
		require.NoError(t, opts.run(cmd, []string{"myapp"}))
	})

	t.Run("dry-run does not write files", func(t *testing.T) {
		tmpDir := scaffoldTestSetup(t)
		opts := &scaffoldOpts{root: &rootOpts{dryRun: true}, env: "dev", region: "eu-central-1"}

		err := opts.run(newTestCmd(t), []string{"myapp"})
		require.NoError(t, err)

		appDir := filepath.Join(tmpDir, "envs", "dev", "eu-central-1", "myapp")
		assert.NoDirExists(t, appDir, "dry-run should not create directories")
	})
}

func TestRunScaffoldUpgradeAll_Integration(t *testing.T) {
	t.Run("upgrades all app dirs in a region", func(t *testing.T) {
		tmpDir := scaffoldTestSetup(t)
		scaffold := &scaffoldOpts{root: &rootOpts{}, env: "dev", region: "eu-central-1"}

		cmd := newTestCmd(t)
		require.NoError(t, scaffold.run(cmd, []string{"app1"}))
		require.NoError(t, scaffold.run(cmd, []string{"app2"}))

		scaffold.upgradeAll = true
		require.NoError(t, scaffold.runUpgradeAll(cmd))

		for _, app := range []string{"app1", "app2"} {
			appDir := filepath.Join(tmpDir, "envs", "dev", "eu-central-1", app)
			assert.DirExists(t, appDir)
			assert.FileExists(t, filepath.Join(appDir, "backend.tf"))
		}
	})

	t.Run("skips directories in skip list", func(t *testing.T) {
		scaffoldTestSetup(t)
		scaffold := &scaffoldOpts{root: &rootOpts{}, env: "dev", region: "eu-central-1"}

		cmd := newTestCmd(t)
		require.NoError(t, scaffold.run(cmd, []string{"app1"}))
		require.NoError(t, scaffold.run(cmd, []string{"app2"}))
		require.NoError(t, scaffold.run(cmd, []string{"app3"}))

		scaffold.upgradeAll = true
		scaffold.skip = "app2"
		require.NoError(t, scaffold.runUpgradeAll(cmd))
	})

	t.Run("base dir does not exist", func(t *testing.T) {
		scaffoldTestSetup(t)
		opts := &scaffoldOpts{root: &rootOpts{}, env: "dev", region: "nonexistent-region-1", upgradeAll: true}

		err := opts.runUpgradeAll(newTestCmd(t))
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrBaseDirNotExist)
	})

	t.Run("no app dirs found", func(t *testing.T) {
		tmpDir := scaffoldTestSetup(t)
		require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "envs", "dev", "eu-central-1"), 0755))

		opts := &scaffoldOpts{root: &rootOpts{}, env: "dev", region: "eu-central-1", upgradeAll: true}
		err := opts.runUpgradeAll(newTestCmd(t))
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrNoAppDirsFound)
	})

	t.Run("dry-run mode", func(t *testing.T) {
		scaffoldTestSetup(t)
		scaffold := &scaffoldOpts{root: &rootOpts{}, env: "dev", region: "eu-central-1"}

		cmd := newTestCmd(t)
		require.NoError(t, scaffold.run(cmd, []string{"app1"}))

		dryRunScaffold := &scaffoldOpts{root: &rootOpts{dryRun: true}, env: "dev", region: "eu-central-1", upgradeAll: true}
		require.NoError(t, dryRunScaffold.runUpgradeAll(cmd))
	})
}

func TestRunScaffoldWorkflows(t *testing.T) {
	t.Run("generates workflow files", func(t *testing.T) {
		tmpDir := scaffoldTestSetup(t)
		opts := &scaffoldWorkflowsOpts{root: &rootOpts{}, env: "dev"}

		err := opts.run(newTestCmd(t), []string{})
		require.NoError(t, err)

		assert.DirExists(t, filepath.Join(tmpDir, ".github", "workflows"))
	})

	t.Run("force without upgrade returns error", func(t *testing.T) {
		scaffoldTestSetup(t)
		opts := &scaffoldWorkflowsOpts{root: &rootOpts{}, env: "dev", force: true}

		err := opts.run(newTestCmd(t), []string{})
		assert.ErrorIs(t, err, ErrForceRequiresUpgrade)
	})

	t.Run("empty env returns error", func(t *testing.T) {
		scaffoldTestSetup(t)
		opts := &scaffoldWorkflowsOpts{root: &rootOpts{}}

		err := opts.run(newTestCmd(t), []string{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "environment")
	})

	t.Run("whitespace-only env returns error", func(t *testing.T) {
		scaffoldTestSetup(t)
		opts := &scaffoldWorkflowsOpts{root: &rootOpts{}, env: "   "}

		err := opts.run(newTestCmd(t), []string{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "environment")
	})

	t.Run("dry-run mode", func(t *testing.T) {
		scaffoldTestSetup(t)
		opts := &scaffoldWorkflowsOpts{root: &rootOpts{dryRun: true}, env: "dev"}

		err := opts.run(newTestCmd(t), []string{})
		require.NoError(t, err)
	})
}
