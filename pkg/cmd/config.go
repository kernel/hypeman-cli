package cmd

import (
	"os"
	"path/filepath"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
	"github.com/urfave/cli/v3"
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

// resolveBaseURL returns the effective base URL with precedence:
// CLI flag > env var > config file > default.
func resolveBaseURL(cmd *cli.Command) string {
	if u := cmd.Root().String("base-url"); u != "" {
		return u
	}
	if u := os.Getenv("HYPEMAN_BASE_URL"); u != "" {
		return u
	}
	cfg := loadCLIConfig()
	if cfg.BaseURL != "" {
		return cfg.BaseURL
	}
	return "http://localhost:8080"
}

// resolveAPIKey returns the effective API key with precedence:
// HYPEMAN_API_KEY env var > HYPEMAN_BEARER_TOKEN env var > config file.
func resolveAPIKey() string {
	if k := os.Getenv("HYPEMAN_API_KEY"); k != "" {
		return k
	}
	if k := os.Getenv("HYPEMAN_BEARER_TOKEN"); k != "" {
		return k
	}
	cfg := loadCLIConfig()
	return cfg.APIKey
}
