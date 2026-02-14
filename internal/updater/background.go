package updater

import (
	"context"
	"fmt"
	"time"
)

// BackgroundChecker runs an update check in a background goroutine.
type BackgroundChecker struct {
	done   chan struct{}
	result CheckResult
}

// StartBackgroundCheck launches a background goroutine that checks for updates.
// It respects the 24h cache interval to avoid hitting the API on every run.
// The channel parameter determines which releases to consider ("latest" or "stable").
func StartBackgroundCheck(currentVersion, channel string) *BackgroundChecker {
	bc := &BackgroundChecker{
		done: make(chan struct{}),
	}

	go func() {
		defer close(bc.done)

		bc.result.CurrentVersion = currentVersion

		cache, err := LoadCache()
		if err != nil {
			bc.result.Err = err
			return
		}

		if !ShouldCheck(cache, channel) {
			// Use cached result
			bc.result.LatestVersion = cache.LatestVersion
			bc.result.HasUpdate = HasNewerVersion(currentVersion, cache.LatestVersion, channel)
			return
		}

		release, err := FetchLatestRelease(context.Background(), channel)
		if err != nil {
			bc.result.Err = err
			return
		}

		latestVersion := release.Version()
		bc.result.LatestVersion = latestVersion
		bc.result.HasUpdate = HasNewerVersion(currentVersion, latestVersion, channel)

		// Update cache regardless of whether there's an update
		_ = SaveCache(&CacheFile{
			LastCheck:     time.Now(),
			LatestVersion: latestVersion,
			Channel:       channel,
		})
	}()

	return bc
}

// Result blocks until the background check is done and returns the result.
func (bc *BackgroundChecker) Result() CheckResult {
	<-bc.done
	return bc.result
}

// NoticeString returns a formatted update notice, or empty string if no update.
func (bc *BackgroundChecker) NoticeString() string {
	r := bc.Result()
	if r.Err != nil || !r.HasUpdate {
		return ""
	}
	return fmt.Sprintf(
		"\nA new version of fire-commit is available: %s -> %s\nRun `firecommit update` to upgrade.\n",
		r.CurrentVersion, r.LatestVersion,
	)
}
