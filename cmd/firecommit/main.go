package main

import (
	"context"
	"fmt"
	"os"
	"strings"

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

	// Determine update timing: "before" or "after" (default)
	timing := updateTiming(cfg, cfgErr)

	// Don't auto-check when running explicit self-management commands.
	// This prevents duplicate updates for "firecommit update", and avoids
	// immediately re-upgrading after "firecommit rollback".
	skipAutoCheck := shouldSkipAutoCheck(os.Args[1:])

	// Start background update check unless disabled
	var checker *updater.BackgroundChecker
	if mode != "n" && !skipAutoCheck {
		checker = updater.StartBackgroundCheck(version, channel)
	}

	// Handle "before" timing: wait for check result before running the command
	if timing == "before" && checker != nil {
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
		// Mark checker as consumed so we don't handle it again after exit
		checker = nil
	}

	err := cli.Execute()

	// Handle update after command exits (default timing)
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
//   - Dev builds: always "a" (force auto-update).
//   - Non-dev builds: respect config.AutoUpdate, default "y" (notify).
func autoUpdateMode(version string, cfg *config.Config, cfgErr error) string {
	if updater.IsDevVersion(version) {
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

// updateTiming returns "before" or "after" based on config.
func updateTiming(cfg *config.Config, cfgErr error) string {
	if cfgErr != nil || cfg == nil {
		return "after"
	}
	if cfg.UpdateTiming == "before" {
		return "before"
	}
	return "after"
}

// shouldSkipAutoCheck returns true for commands that manage versions directly.
func shouldSkipAutoCheck(args []string) bool {
	subcmd := firstSubcommand(args)
	return subcmd == "update" || subcmd == "rollback" || subcmd == "tag"
}

func firstSubcommand(args []string) string {
	for _, a := range args {
		if a == "--" {
			break
		}
		if strings.HasPrefix(a, "-") {
			continue
		}
		return a
	}
	return ""
}
