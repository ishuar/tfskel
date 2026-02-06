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
	// Version information (set by goreleaser)
	Version   = "dev"
	Commit    = "unknown"
	Date      = "unknown"
	BuildTime = "unknown"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "tfskel",
	Short: "Opinionated Terraform scaffolding for real-world teams",
	Long: `tfskel helps you bootstrap Terraform projects the *right* way — fast, consistent, and scalable.

It creates production-ready Terraform layouts with built-in best practices,
so teams can focus on infrastructure instead of structure.

What you get:
  - Clean, environment-first project layouts (dev, stg, prd) with region separation
  - Pre-wired backend and provider configuration using reusable templates
  - Terraform and AWS provider version drift detection across repos and monorepos
  - Simple, declarative customization via .tfskel.yaml

Configuration:
  tfskel automatically loads .tfskel.yaml from the current directory.
  Use --config to point to a different file (this always takes precedence).`,

	Version: Version,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is .tfskel.yaml in current directory)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "enable verbose output")

	// Bind flags to viper
	if err := viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose")); err != nil {
		// This should never fail, but handle it anyway
		panic(fmt.Sprintf("failed to bind verbose flag: %v", err))
	}
}

// initConfig reads in config file and ENV variables if set.
// Similar to Trivy's approach: checks current directory by default,
// --config flag takes precedence if specified.
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
