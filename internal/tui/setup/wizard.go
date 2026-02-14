package setup

import (
	"fmt"

	"github.com/charmbracelet/huh"
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

	// Step 1: Select provider
	var providerName string
	names := llm.ProviderNames()
	displayNames := llm.ProviderDisplayNames()

	options := make([]huh.Option[string], len(names))
	for i, name := range names {
		options[i] = huh.NewOption(displayNames[name], name)
	}

	providerSelect := huh.NewSelect[string]().
		Title("Select your LLM provider").
		Options(options...).
		Value(&providerName)

	if err := huh.NewForm(huh.NewGroup(providerSelect)).Run(); err != nil {
		return nil, err
	}

	// Step 2: API Key
	var apiKey string
	apiKeyInput := huh.NewInput().
		Title(fmt.Sprintf("Enter your %s API key", displayNames[providerName])).
		Value(&apiKey).
		EchoMode(huh.EchoModePassword).
		Validate(func(s string) error {
			if s == "" {
				return fmt.Errorf("API key is required")
			}
			return nil
		})

	fields := []huh.Field{apiKeyInput}

	// Step 3: Custom base URL (only for custom provider)
	var baseURL string
	if providerName == "custom" {
		baseURLInput := huh.NewInput().
			Title("Enter the OpenAI-compatible API base URL").
			Placeholder("https://api.example.com/v1").
			Value(&baseURL).
			Validate(func(s string) error {
				if s == "" {
					return fmt.Errorf("base URL is required for custom provider")
				}
				return nil
			})
		fields = append(fields, baseURLInput)
	}

	// Step 4: Model
	var model string
	defaultModel := llm.DefaultModel(providerName)
	modelInput := huh.NewInput().
		Title("Model name").
		Placeholder(defaultModel).
		Value(&model)
	fields = append(fields, modelInput)

	if err := huh.NewForm(huh.NewGroup(fields...)).Run(); err != nil {
		return nil, err
	}

	if model == "" {
		model = defaultModel
	}

	// Build and save config
	provCfg := config.ProviderConfig{
		APIKey: apiKey,
		Model:  model,
	}
	if providerName == "custom" {
		provCfg.BaseURL = baseURL
	}

	cfg.DefaultProvider = providerName
	cfg.Providers[providerName] = provCfg
	cfg.ConfigVersion = config.CurrentConfigVersion

	if err := config.Save(cfg); err != nil {
		return nil, fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Println()
	fmt.Println(titleStyle.Render("✓ Configuration saved!"))
	fmt.Println(subtitleStyle.Render(fmt.Sprintf("  Provider: %s | Model: %s", displayNames[providerName], model)))
	fmt.Println()

	return cfg, nil
}
