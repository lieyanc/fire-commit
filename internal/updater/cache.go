package updater

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/adrg/xdg"
)

const (
	cacheDir      = "firecommit"
	cacheFile     = "update-check.json"
	stableCheckInterval = 24 * time.Hour
	latestCheckInterval = 3 * time.Hour
)

// CacheFile stores the last update check state.
type CacheFile struct {
	LastCheck     time.Time `json:"last_check"`
	LatestVersion string    `json:"latest_version"`
	Channel       string    `json:"channel,omitempty"`
}

func cachePath() string {
	return filepath.Join(xdg.CacheHome, cacheDir, cacheFile)
}

// LoadCache reads the update check cache from disk.
func LoadCache() (*CacheFile, error) {
	data, err := os.ReadFile(cachePath())
	if err != nil {
		return &CacheFile{}, nil
	}
	var cf CacheFile
	if err := json.Unmarshal(data, &cf); err != nil {
		return &CacheFile{}, nil
	}
	return &cf, nil
}

// SaveCache writes the update check cache to disk.
func SaveCache(cf *CacheFile) error {
	p := cachePath()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	data, err := json.Marshal(cf)
	if err != nil {
		return err
	}
	return os.WriteFile(p, data, 0o644)
}

// ShouldCheck returns true if enough time has passed since the last check,
// or if the channel has changed since the last check.
func ShouldCheck(cf *CacheFile, currentChannel string) bool {
	if cf.Channel != currentChannel {
		return true
	}
	interval := stableCheckInterval
	if currentChannel == ChannelLatest {
		interval = latestCheckInterval
	}
	return time.Since(cf.LastCheck) > interval
}
