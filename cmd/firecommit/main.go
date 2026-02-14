package main

import (
	"context"
	"fmt"
	"os"

	"github.com/lieyanc/fire-commit/internal/cli"
	"github.com/lieyanc/fire-commit/internal/config"
	"github.com/lieyanc/fire-commit/internal/updater"
)

var version = "dev"

func main() {
	cli.SetVersion(version)

	channel := updater.ChannelLatest
	cfg, cfgErr := config.Load()
	if cfgErr == nil && cfg.UpdateChannel != "" {
		channel = cfg.UpdateChannel
	}

	// Determine auto-update mode
	mode := autoUpdateMode(version, cfg, cfgErr)

	// Start background update check unless disabled
	var checker *updater.BackgroundChecker
	if mode != "n" {
		checker = updater.StartBackgroundCheck(version, channel)
	}

	err := cli.Execute()

	// Handle update after command exits
	if checker != nil {
		r := checker.Result()
		if r.HasUpdate && r.Err == nil {
			if mode == "a" {
				fmt.Fprintf(os.Stderr, "\nAuto-updating fire-commit: %s → %s\n", r.CurrentVersion, r.LatestVersion)
				if updateErr := updater.SelfUpdate(context.Background(), version, channel); updateErr != nil {
					fmt.Fprintf(os.Stderr, "Auto-update failed: %v\n", updateErr)
				}
			} else {
				if notice := checker.NoticeString(); notice != "" {
					fmt.Fprint(os.Stderr, notice)
				}
			}
		}
	}

	if err != nil {
		os.Exit(1)
	}
}

// autoUpdateMode returns "a" (auto-update), "y" (notify only), or "n" (skip).
//   - Non-dev builds: always "a" (force auto-update).
//   - Dev builds: respect config.AutoUpdate, default "y" (notify).
func autoUpdateMode(version string, cfg *config.Config, cfgErr error) string {
	if version != "dev" {
		return "a"
	}
	if cfgErr != nil || cfg == nil {
		return "y"
	}
	switch cfg.AutoUpdate {
	case "a", "always":
		return "a"
	case "n", "no":
		return "n"
	case "y", "yes":
		return "y"
	default:
		return "y"
	}
}
