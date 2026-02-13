package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var appVersion = "dev"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("fire-commit %s\n", appVersion)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}

// SetVersion sets the application version (called from main with ldflags value).
func SetVersion(v string) {
	appVersion = v
}
