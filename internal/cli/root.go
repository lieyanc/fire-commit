package cli

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"github.com/lieyanc/fire-commit/internal/config"
	"github.com/lieyanc/fire-commit/internal/git"
	"github.com/lieyanc/fire-commit/internal/tui"
	"github.com/lieyanc/fire-commit/internal/tui/setup"
)

var rootCmd = &cobra.Command{
	Use:   "firecommit",
	Short: "Generate beautiful commit messages with AI",
	Long:  "🔥 fire-commit — AI-powered conventional commit message generator with a beautiful TUI.",
	RunE:  runDefault,
}

func Execute() error {
	return rootCmd.Execute()
}

func runDefault(cmd *cobra.Command, args []string) error {
	// Show version
	versionStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FF6B35"))
	fmt.Println(versionStyle.Render(fmt.Sprintf("🔥 fire-commit %s", appVersion)))
	fmt.Println()

	// Step 1: Check or create config
	var cfg *config.Config
	if !config.Exists() {
		var err error
		cfg, err = setup.RunWizard()
		if err != nil {
			return fmt.Errorf("setup failed: %w", err)
		}
	} else {
		var err error
		cfg, err = config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
	}

	// Step 1.5: Check if config needs migration
	if config.NeedsMigration(cfg) {
		var err error
		cfg, err = setup.RunMigration(cfg)
		if err != nil {
			return fmt.Errorf("config migration failed: %w", err)
		}
	}

	// Step 2: Check git repo
	if !git.IsGitRepo() {
		return fmt.Errorf("not a git repository")
	}

	// Step 3: Get diff
	staged, err := git.HasStagedChanges()
	if err != nil {
		return fmt.Errorf("failed to check staged changes: %w", err)
	}

	if !staged {
		unstaged, err := git.HasUnstagedChanges()
		if err != nil {
			return fmt.Errorf("failed to check unstaged changes: %w", err)
		}
		untracked, err := git.HasUntrackedFiles()
		if err != nil {
			return fmt.Errorf("failed to check untracked files: %w", err)
		}

		if !unstaged && !untracked {
			return fmt.Errorf("no changes to commit")
		}

		// Auto-stage all changes
		if err := git.StageAll(); err != nil {
			return fmt.Errorf("failed to stage changes: %w", err)
		}
	}

	diff, err := git.StagedDiff(cfg.Generation.MaxDiffLines)
	if err != nil {
		return fmt.Errorf("failed to get diff: %w", err)
	}

	if diff == "" {
		return fmt.Errorf("empty diff — nothing to commit")
	}

	stat, _ := git.DiffStat()

	// Step 4: Launch TUI
	return tui.Run(cfg, diff, stat)
}
