package cmd

import (
	"fmt"
	"os"

	"github.com/charmbracelet/lipgloss"
	"github.com/ishuar/tfskel/internal/format"
	"github.com/muesli/termenv"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// Commit and Date are injected via -ldflags at build time. They stay as
// package-level vars because the build toolchain targets them by name.
//
//nolint:gochecknoglobals // set by ldflags at build time
var (
	Commit = "unknown"
	Date   = "unknown"
)

// rootOpts holds persistent flag state for the root command, shared with
// every subcommand via *rootOpts pointer. Keeping state in a struct (rather
// than package-level vars) makes the command tree safe to construct per-test.
type rootOpts struct {
	cfgFile  string
	verbose  bool
	noColor  bool
	dryRun   bool
	useColor bool
}

// NewRootCmd builds a fresh root command tree with all subcommands attached.
// Every call returns an independent tree — nothing is shared between instances,
// which lets tests construct their own command trees without global leakage.
func NewRootCmd() *cobra.Command {
	opts := &rootOpts{}

	cmd := &cobra.Command{
		Use:   "tfskel <command> <subcommand>",
		Short: "Simplify Terraform operations and project structure",
		Long: `tfskel simplifies Terraform operations so teams can focus on building infrastructure
not managing folder structures, drift, or plan reviews. It provides clean, consistent,
and scalable Terraform layouts with built-in best practices.`,
		PersistentPreRunE: func(_ *cobra.Command, _ []string) error {
			opts.useColor = format.ShouldUseColor(opts.noColor)
			if opts.useColor {
				lipgloss.SetColorProfile(termenv.TrueColor)
			} else {
				lipgloss.SetColorProfile(termenv.Ascii)
			}
			opts.initConfig()
			return nil
		},
		RunE: func(c *cobra.Command, _ []string) error {
			return c.Help()
		},
	}

	// --version should print the same rich output as `tfskel version`. Cobra
	// only registers the --version flag when Version is non-empty, and the
	// custom template prints it verbatim instead of cobra's default wrapper.
	cmd.Version = buildVersionInfo()
	cmd.SetVersionTemplate("{{.Version}}")

	// Users who trigger a flag error clearly know the interface; dumping the
	// full usage block buries the actual error message.
	cmd.SetFlagErrorFunc(func(c *cobra.Command, err error) error {
		c.SilenceUsage = true
		return err
	})

	cmd.AddGroup(&cobra.Group{ID: "main", Title: "Main Commands:"})

	cmd.PersistentFlags().StringVarP(&opts.cfgFile, "config", "c", "", "config file (default is .tfskel.yaml in current directory)")
	cmd.PersistentFlags().BoolVarP(&opts.verbose, "verbose", "v", false, "enable verbose output")
	cmd.PersistentFlags().BoolVar(&opts.noColor, "no-color", false, "disable colored output (respects NO_COLOR and FORCE_COLOR env vars)")
	cmd.PersistentFlags().BoolVar(&opts.dryRun, "dry-run", false, "show what would happen without writing files")

	cmd.AddCommand(
		newScaffoldCmd(opts),
		newInitCmd(opts),
		newValidateCmd(opts),
		newReviewCmd(opts),
		newVersionCmd(),
	)

	return cmd
}

// Execute builds the root command tree and runs it. Called by main.main().
func Execute() error {
	return NewRootCmd().Execute()
}

// initConfig loads the .tfskel.yaml config via viper. Follows the Trivy pattern:
// --config/-c takes precedence if given, otherwise search the current directory.
func (o *rootOpts) initConfig() {
	if o.cfgFile != "" {
		viper.SetConfigFile(o.cfgFile)
	} else {
		viper.AddConfigPath(".")
		viper.SetConfigType("yaml")
		viper.SetConfigName(".tfskel")
	}

	if err := viper.ReadInConfig(); err == nil && o.verbose {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}
