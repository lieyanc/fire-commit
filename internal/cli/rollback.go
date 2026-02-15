package cli

import (
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/lieyanc/fire-commit/internal/updater"
	"github.com/spf13/cobra"
)

var rollbackCmd = &cobra.Command{
	Use:   "rollback",
	Short: "Restore a previously archived version",
	RunE:  runRollback,
}

func init() {
	rootCmd.AddCommand(rollbackCmd)
}

func runRollback(cmd *cobra.Command, args []string) error {
	versions, err := updater.ListArchive()
	if err != nil {
		return fmt.Errorf("failed to read version archive: %w", err)
	}
	if len(versions) == 0 {
		fmt.Println("No archived versions available.")
		fmt.Println("Versions are archived automatically when you run 'firecommit update'.")
		return nil
	}

	// Build options newest-first for the selector.
	options := make([]huh.Option[string], 0, len(versions))
	for i := len(versions) - 1; i >= 0; i-- {
		v := versions[i]
		label := fmt.Sprintf("%s  (archived %s)", v.Version, v.ArchivedAt.Format("2006-01-02 15:04"))
		options = append(options, huh.NewOption(label, v.Version))
	}

	var selected string
	sel := huh.NewSelect[string]().
		Title("Select a version to restore").
		Options(options...).
		Value(&selected)

	if err := huh.NewForm(huh.NewGroup(sel)).Run(); err != nil {
		return err
	}

	fmt.Printf("Restoring %s...\n", selected)
	if err := updater.RestoreBinary(selected); err != nil {
		return fmt.Errorf("rollback failed: %w", err)
	}

	fmt.Printf("Successfully restored %s\n", selected)
	return nil
}
