package updater

import (
	"os"
	"path/filepath"
	"time"

	"github.com/adrg/xdg"
)

const (
	checkIntervalStable = 24 * time.Hour
	checkIntervalLatest = 3 * time.Hour
)

func lastCheckPath() string {
	return filepath.Join(xdg.CacheHome, "firecommit", "last-update-check")
}

// shouldCheck returns true if enough time has passed since the last update
// check. This is a simple file-mtime-based throttle with no cached version.
func shouldCheck(channel string) bool {
	info, err := os.Stat(lastCheckPath())
	if err != nil {
		return true
	}
	interval := checkIntervalStable
	if channel == ChannelLatest {
		interval = checkIntervalLatest
	}
	return time.Since(info.ModTime()) > interval
}

// markChecked touches the last-check file to record that a check just happened.
func markChecked() {
	p := lastCheckPath()
	_ = os.MkdirAll(filepath.Dir(p), 0o755)
	_ = os.WriteFile(p, []byte{}, 0o644)
}
