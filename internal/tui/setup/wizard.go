package setup

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/lieyanc/fire-commit/internal/config"
	"github.com/lieyanc/fire-commit/internal/llm"
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FF6B35"))

	subtitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFB347"))
)

// RunWizard runs the first-time setup wizard and returns the configured Config.
func RunWizard() (*config.Config, error) {
	fmt.Println()
	fmt.Println(titleStyle.Render("🔥 Welcome to fire-commit!"))
	fmt.Println(subtitleStyle.Render("   Let's set up your LLM provider."))
	fmt.Println()

	cfg := config.DefaultConfig()

	if err := editProviderSettings(cfg); err != nil {
		return nil, err
	}

	cfg.ConfigVersion = config.CurrentConfigVersion

	if err := config.Save(cfg); err != nil {
		return nil, fmt.Errorf("failed to save config: %w", err)
	}

	displayNames := llm.ProviderDisplayNames()
	p := cfg.Providers[cfg.DefaultProvider]

	fmt.Println()
	fmt.Println(titleStyle.Render("✓ Configuration saved!"))
	fmt.Println(subtitleStyle.Render(fmt.Sprintf("  Provider: %s | Model: %s", displayNames[cfg.DefaultProvider], p.Model)))
	fmt.Println()

	return cfg, nil
}
