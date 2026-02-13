package config

import (
	"os"
	"path/filepath"

	"github.com/adrg/xdg"
)

const (
	appName    = "firecommit"
	configFile = "config.yaml"
)

// ConfigPath returns the full path to the config file.
// It respects the FIRECOMMIT_CONFIG env var override.
func ConfigPath() string {
	if p := os.Getenv("FIRECOMMIT_CONFIG"); p != "" {
		return p
	}
	return filepath.Join(xdg.ConfigHome, appName, configFile)
}

// ConfigDir returns the directory containing the config file.
func ConfigDir() string {
	return filepath.Dir(ConfigPath())
}
