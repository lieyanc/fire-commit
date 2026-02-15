package updater

import (
	"context"
	"fmt"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// BackgroundChecker runs an update check in a background goroutine.
type BackgroundChecker struct {
	done   chan struct{}
	result CheckResult
}

// StartBackgroundCheck launches a background goroutine that checks for updates.
// If useCache is false, it checks on every run (no persisted state).
// If useCache is true, it uses adaptive scheduling + ETag conditional requests
// to avoid redundant network calls while still detecting frequent releases.
// The channel parameter determines which releases to consider ("latest" or "stable").
func StartBackgroundCheck(currentVersion, channel string, useCache bool) *BackgroundChecker {
	bc := &BackgroundChecker{
		done: make(chan struct{}),
	}

	go func() {
		defer close(bc.done)

		bc.result.CurrentVersion = currentVersion

		if !useCache {
			release, err := FetchLatestRelease(context.Background(), channel)
			if err != nil {
				bc.result.Err = err
				return
			}
			latestVersion := release.Version()
			bc.result.LatestVersion = latestVersion
			bc.result.HasUpdate = HasNewerVersion(currentVersion, latestVersion, channel)
			return
		}

		now := time.Now()

		state, err := loadCheckState()
		if err != nil {
			// Corrupt/missing cache should not block update checks; recover with a
			// fresh in-memory state and rewrite it after a successful check.
			state = &checkState{}
		}
		channelState := state.channel(channel)

		if !shouldCheckChannel(channelState, now) {
			return // checked recently, skip
		}

		release, newETag, notModified, err := FetchLatestReleaseConditional(
			context.Background(),
			channel,
			channelState.ETag,
		)
		if err != nil {
			bc.result.Err = err
			recordFetchError(channelState, channel, now)
			_ = saveCheckState(state)
			return
		}
		if newETag != "" {
			channelState.ETag = newETag
		}

		if notModified {
			bc.result.LatestVersion = channelState.LastSeenVersion
			if bc.result.LatestVersion != "" {
				bc.result.HasUpdate = HasNewerVersion(currentVersion, bc.result.LatestVersion, channel)
			}
			if bc.result.HasUpdate {
				recordHasUpdate(channelState, now)
			} else {
				recordNoUpdate(channelState, channel, now)
			}
			_ = saveCheckState(state)
			return
		}

		latestVersion := release.Version()
		bc.result.LatestVersion = latestVersion
		bc.result.HasUpdate = HasNewerVersion(currentVersion, latestVersion, channel)
		channelState.LastSeenVersion = latestVersion
		if bc.result.HasUpdate {
			recordHasUpdate(channelState, now)
		} else {
			recordNoUpdate(channelState, channel, now)
		}
		_ = saveCheckState(state)
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

	versionStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#52B0FF")).Bold(true)
	commandStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#7EC8FF")).Bold(true)

	content := fmt.Sprintf(
		"Update available: %s → %s\nRun %s to upgrade.",
		versionStyle.Render(r.CurrentVersion),
		versionStyle.Render(r.LatestVersion),
		commandStyle.Render("firecommit update"),
	)

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#018EEE")).
		Padding(0, 1)

	return "\n" + box.Render(content) + "\n"
}
