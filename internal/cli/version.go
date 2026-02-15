package cli

import (
	"fmt"

	"github.com/lieyanc/fire-commit/internal/updater"
	"github.com/spf13/cobra"
)

var appVersion = "dev"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version",
	Run: func(cmd *cobra.Command, args []string) {
		versions, err := updater.ListArchive()
		if err != nil || len(versions) == 0 {
			fmt.Printf("fire-commit %s\n", appVersion)
			return
		}
		fmt.Printf("fire-commit %s (%d archived versions available, use 'firecommit rollback' to restore)\n",
			appVersion, len(versions))
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}

// SetVersion sets the application version (called from main with ldflags value).
func SetVersion(v string) {
	appVersion = v
}
