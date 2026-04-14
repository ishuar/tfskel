package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/ishuar/tfskel/internal/config"
	"github.com/ishuar/tfskel/internal/logger"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// ErrConfigNotFound indicates the .tfskel.yaml config file was not discovered in cwd
var ErrConfigNotFound = errors.New(".tfskel.yaml not found")

// loadAndValidateConfig loads the configuration from the command context and validates it.
func loadAndValidateConfig(cmd *cobra.Command, log *logger.Logger) (*config.Config, error) {
	if viper.ConfigFileUsed() == "" {
		cwd, err := os.Getwd()
		if err != nil {
			cwd = "current directory"
		}
		return nil, fmt.Errorf("%w in %s; run from the project root, pass --config/-c, or run 'tfskel init' to create one", ErrConfigNotFound, cwd)
	}
	cfg, err := config.Load(cmd, viper.GetViper(), log)
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}
	return cfg, nil
}
