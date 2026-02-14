package setup

import (
	"fmt"
	"strconv"

	"github.com/charmbracelet/huh"
	"github.com/lieyanc/fire-commit/internal/config"
)

// RunMigration prompts the user about new config options added since their
// config was last saved. Returns the updated config (already saved to disk).
func RunMigration(cfg *config.Config) (*config.Config, error) {
	fromVersion := cfg.ConfigVersion

	fmt.Println()
	fmt.Println(titleStyle.Render("🔥 New configuration options available!"))
	fmt.Println(subtitleStyle.Render("   Your config will be updated to the latest version."))
	fmt.Println()

	var useDefaults bool
	confirm := huh.NewConfirm().
		Title("Use defaults for all new options?").
		Description("Select No to customize each new option individually.").
		Value(&useDefaults)

	if err := huh.NewForm(huh.NewGroup(confirm)).Run(); err != nil {
		return cfg, err
	}

	if useDefaults {
		applyDefaults(cfg, fromVersion)
	} else {
		if err := runMigrationWizard(cfg, fromVersion); err != nil {
			return cfg, err
		}
	}

	cfg.ConfigVersion = config.CurrentConfigVersion
	if err := config.Save(cfg); err != nil {
		return cfg, fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Println()
	fmt.Println(titleStyle.Render("✓ Configuration updated!"))
	fmt.Println()

	return cfg, nil
}

// applyDefaults sets new fields to their default values for all versions
// between fromVersion and CurrentConfigVersion.
func applyDefaults(cfg *config.Config, fromVersion int) {
	if fromVersion < 1 {
		// v0 -> v1: MaxDiffLines increased from 500 to 4096
		defaults := config.DefaultConfig()
		cfg.Generation.MaxDiffLines = defaults.Generation.MaxDiffLines
	}
}

// runMigrationWizard presents huh forms for each new field added since fromVersion.
func runMigrationWizard(cfg *config.Config, fromVersion int) error {
	if fromVersion < 1 {
		// v0 -> v1: MaxDiffLines
		maxDiffStr := strconv.Itoa(cfg.Generation.MaxDiffLines)
		maxDiffInput := huh.NewInput().
			Title("Max diff lines").
			Description("Maximum number of diff lines to send to the LLM (old default: 500, new default: 4096).").
			Placeholder("4096").
			Value(&maxDiffStr).
			Validate(func(s string) error {
				n, err := strconv.Atoi(s)
				if err != nil || n <= 0 {
					return fmt.Errorf("must be a positive integer")
				}
				return nil
			})

		if err := huh.NewForm(huh.NewGroup(maxDiffInput)).Run(); err != nil {
			return err
		}

		if n, err := strconv.Atoi(maxDiffStr); err == nil {
			cfg.Generation.MaxDiffLines = n
		}
	}

	return nil
}
