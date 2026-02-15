package cli

import (
	"fmt"
	"strings"

	"github.com/lieyanc/fire-commit/internal/git"
	"github.com/spf13/cobra"
)

var tagCmd = &cobra.Command{
	Use:   "tag <version>",
	Short: "Create and push a version tag",
	Args:  cobra.ExactArgs(1),
	RunE:  runTag,
}

func init() {
	rootCmd.AddCommand(tagCmd)
}

func runTag(cmd *cobra.Command, args []string) error {
	if !git.IsGitRepo() {
		return fmt.Errorf("not a git repository")
	}

	tag := strings.TrimSpace(args[0])
	if tag == "" {
		return fmt.Errorf("tag cannot be empty")
	}
	if !strings.HasPrefix(tag, "v") {
		return fmt.Errorf("tag must start with 'v' to trigger release workflow, e.g. v1.2.3")
	}

	if err := git.Tag(tag); err != nil {
		return err
	}
	fmt.Printf("Created tag: %s\n", tag)

	if err := git.PushTag(tag); err != nil {
		return err
	}
	fmt.Printf("Pushed tag: %s\n", tag)
	fmt.Println("Release workflow should start shortly.")
	return nil
}
