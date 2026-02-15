package cmd

import (
	"os"
	"path/filepath"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

// CLIConfig holds CLI configuration loaded from cli.yaml
type CLIConfig struct {
	BaseURL string `koanf:"base_url"`
	APIKey  string `koanf:"api_key"`
}

// getCLIConfigPath returns the path to the CLI config file.
// The CLI uses ~/.config/hypeman/cli.yaml on all platforms.
func getCLIConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "hypeman", "cli.yaml")
}

// loadCLIConfig loads CLI configuration from the config file.
// Returns an empty config if the file doesn't exist or can't be parsed.
func loadCLIConfig() *CLIConfig {
	cfg := &CLIConfig{}

	configPath := getCLIConfigPath()
	if configPath == "" {
		return cfg
	}

	k := koanf.New(".")
	if err := k.Load(file.Provider(configPath), yaml.Parser()); err != nil {
		// File doesn't exist or can't be parsed - return empty config
		return cfg
	}

	_ = k.Unmarshal("", cfg)
	return cfg
}
