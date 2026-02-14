package cli

import (
	"context"
	"fmt"

	"github.com/lieyanc/fire-commit/internal/config"
	"github.com/lieyanc/fire-commit/internal/updater"
	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update fire-commit to the latest version",
	RunE:  runUpdate,
}

func init() {
	rootCmd.AddCommand(updateCmd)
}

func runUpdate(cmd *cobra.Command, args []string) error {
	fmt.Printf("Current version: %s\n", appVersion)

	channel := updater.ChannelLatest
	if cfg, err := config.Load(); err == nil && cfg.UpdateChannel != "" {
		channel = cfg.UpdateChannel
	}

	if err := updater.SelfUpdate(context.Background(), appVersion, channel); err != nil {
		return fmt.Errorf("update failed: %w", err)
	}
	return nil
}
