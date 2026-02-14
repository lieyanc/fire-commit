package main

import (
	"fmt"
	"os"

	"github.com/lieyanc/fire-commit/internal/cli"
	"github.com/lieyanc/fire-commit/internal/updater"
)

var version = "dev"

func main() {
	cli.SetVersion(version)

	// Start background update check for non-dev builds
	var checker *updater.BackgroundChecker
	if version != "dev" {
		checker = updater.StartBackgroundCheck(version)
	}

	err := cli.Execute()

	// Show update notice after TUI exits, only for the default command
	if checker != nil && cli.RanDefault {
		if notice := checker.NoticeString(); notice != "" {
			fmt.Fprint(os.Stderr, notice)
		}
	}

	if err != nil {
		os.Exit(1)
	}
}
