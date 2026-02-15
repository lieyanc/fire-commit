package updater

import (
	"context"
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

// BackgroundChecker runs an update check in a background goroutine.
type BackgroundChecker struct {
	done   chan struct{}
	result CheckResult
}

// StartBackgroundCheck launches a background goroutine that checks for updates.
// It respects a time-based throttle to avoid hitting the API on every run.
// The channel parameter determines which releases to consider ("latest" or "stable").
func StartBackgroundCheck(currentVersion, channel string) *BackgroundChecker {
	bc := &BackgroundChecker{
		done: make(chan struct{}),
	}

	go func() {
		defer close(bc.done)

		bc.result.CurrentVersion = currentVersion

		if !shouldCheck(channel) {
			return // checked recently, skip
		}

		release, err := FetchLatestRelease(context.Background(), channel)
		if err != nil {
			bc.result.Err = err
			return
		}

		latestVersion := release.Version()
		bc.result.LatestVersion = latestVersion
		bc.result.HasUpdate = HasNewerVersion(currentVersion, latestVersion, channel)

		markChecked()
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
