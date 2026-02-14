package cli

import (
	"context"
	"fmt"

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
	if err := updater.SelfUpdate(context.Background(), appVersion); err != nil {
		return fmt.Errorf("update failed: %w", err)
	}
	return nil
}
