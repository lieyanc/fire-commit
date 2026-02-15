package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

// ProviderConfig holds credentials and settings for a single LLM provider.
type ProviderConfig struct {
	APIKey  string `yaml:"api_key"`
	Model   string `yaml:"model"`
	BaseURL string `yaml:"base_url,omitempty"`
}

// GenerationConfig holds generation-related settings.
type GenerationConfig struct {
	NumSuggestions int    `yaml:"num_suggestions"`
	Language       string `yaml:"language"`
	MaxDiffLines   int    `yaml:"max_diff_lines"`
}

// CurrentConfigVersion is bumped when new config fields are added.
// Existing configs with a lower version will trigger a migration prompt.
const CurrentConfigVersion = 1

// Config is the top-level configuration.
type Config struct {
	ConfigVersion   int                       `yaml:"config_version"`
	DefaultProvider string                    `yaml:"default_provider"`
	Providers       map[string]ProviderConfig `yaml:"providers"`
	Generation      GenerationConfig          `yaml:"generation"`
	UpdateChannel   string                    `yaml:"update_channel"`
	// AutoUpdate controls automatic update behavior for non-dev builds.
	// "y" = show update notice (default for non-dev builds)
	// "n" = don't check for updates
	// "a" = always auto-update without asking
	// Dev builds ignore this and always auto-update.
	AutoUpdate   string `yaml:"auto_update,omitempty"`
	UpdateTiming string `yaml:"update_timing,omitempty"`
}

// NeedsMigration returns true if the config was created with an older version
// and new config fields need to be presented to the user.
func NeedsMigration(cfg *Config) bool {
	return cfg.ConfigVersion < CurrentConfigVersion
}

// DefaultConfig returns a config with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		DefaultProvider: "",
		Providers:       make(map[string]ProviderConfig),
		Generation: GenerationConfig{
			NumSuggestions: 3,
			Language:       "en",
			MaxDiffLines:   4096,
		},
		UpdateChannel: "latest",
	}
}

// Exists returns true if the config file exists on disk.
func Exists() bool {
	_, err := os.Stat(ConfigPath())
	return err == nil
}

// Load reads the config file from disk.
func Load() (*Config, error) {
	data, err := os.ReadFile(ConfigPath())
	if err != nil {
		return nil, err
	}
	cfg := DefaultConfig()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

// Save writes the config to disk, creating parent directories as needed.
func Save(cfg *Config) error {
	if err := os.MkdirAll(ConfigDir(), 0o755); err != nil {
		return err
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(ConfigPath(), data, 0o644)
}
