package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile string
	verbose bool
	// Commit is the git commit hash of the build
	Commit = "unknown"
	// Date is the build date
	Date = "unknown"
	// BuildTime is the build timestamp
	BuildTime = "unknown"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "tfskel <command> <subcommand>",
	Short: "Simplify Terraform operations and project structure",
	Long: `tfskel simplifies Terraform operations so teams can focus on building infrastructure
not managing folder structures, drift, or plan reviews. It provides clean, consistent,
and scalable Terraform layouts with built-in best practices.`,

	Version: Version,
	RunE: func(cmd *cobra.Command, _ []string) error {
		return cmd.Help()
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	// Define command groups
	rootCmd.AddGroup(&cobra.Group{
		ID:    "main",
		Title: "Main Commands:",
	})

	// Global flags
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file (default is .tfskel.yaml in current directory)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "enable verbose output")

	// Bind flags to viper
	if err := viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose")); err != nil {
		// This should never fail, but handle it anyway
		panic(fmt.Sprintf("failed to bind verbose flag: %v", err))
	}
	rootCmd.SetHelpTemplate(`Usage: tfskel [global options] <subcommand> [args]

tfskel simplifies Terraform operations so teams can focus on building infrastructure — not managing folder structures, drift, or plan reviews.
It provides clean, consistent, and scalable Terraform layouts with built-in best practices.

Main workflow commands:
  init       Initialize a tfskel project with environment-based structure
  generate   Generate terraform target directory using subcommand as input
  drift      Analyze versions drift and plan changes analysis
  help       Show help about any command

drift subcommands:
  version  Drift check between configured terraform, provider versions & the actual versions in use
  plan     Analyze Terraform plan json file & output a human-readable terraform plan summary

Other commands:
  completion  Generate shell autocompletion scripts

Global options (use these before the subcommand):
  -c, --config string   Load configuration from the specified file (default is .tfskel.yaml in the current directory)
  -h, --help            Show this help output
  -v, --verbose         Enable verbose output
      --version         Show the current tfskel version

For more details on a specific command, use:
  tfskel <command> --help
`)
}

// initConfig reads in config file and ENV variables if set.
// Similar to Trivy's approach: checks current directory by default,
// --config | -c flag takes precedence if specified.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag (takes precedence).
		viper.SetConfigFile(cfgFile)
	} else {
		// Search config in current directory first (Trivy-like behavior)
		viper.AddConfigPath(".")
		viper.SetConfigType("yaml")
		viper.SetConfigName(".tfskel")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil && verbose {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}
