package setup

import (
	"fmt"
	"strconv"

	"github.com/charmbracelet/huh"
	"github.com/lieyanc/fire-commit/internal/config"
	"github.com/lieyanc/fire-commit/internal/llm"
)

// RunConfigEditor opens an interactive TUI for editing the current configuration.
// All fields are pre-filled with current values. Returns the updated config (already saved).
func RunConfigEditor(cfg *config.Config) (*config.Config, error) {
	fmt.Println()
	fmt.Println(titleStyle.Render("🔥 fire-commit configuration"))
	fmt.Println(subtitleStyle.Render("   Edit your settings below."))
	fmt.Println()

	// --- Provider settings ---
	providerName := cfg.DefaultProvider
	names := llm.ProviderNames()
	displayNames := llm.ProviderDisplayNames()

	providerOptions := make([]huh.Option[string], len(names))
	for i, name := range names {
		providerOptions[i] = huh.NewOption(displayNames[name], name)
	}

	var apiKey string
	var model string
	var baseURL string
	if p, ok := cfg.Providers[providerName]; ok {
		apiKey = p.APIKey
		model = p.Model
		baseURL = p.BaseURL
	}

	providerSelect := huh.NewSelect[string]().
		Title("LLM Provider").
		Options(providerOptions...).
		Value(&providerName)

	apiKeyInput := huh.NewInput().
		Title("API Key").
		Value(&apiKey).
		EchoMode(huh.EchoModePassword).
		Validate(func(s string) error {
			if s == "" {
				return fmt.Errorf("API key is required")
			}
			return nil
		})

	modelInput := huh.NewInput().
		Title("Model name").
		Placeholder(llm.DefaultModel(providerName)).
		Value(&model)

	baseURLInput := huh.NewInput().
		Title("API Base URL (for custom provider)").
		Placeholder("https://api.example.com/v1").
		Value(&baseURL)

	providerGroup := huh.NewGroup(providerSelect, apiKeyInput, modelInput, baseURLInput).
		Title("Provider Settings")

	// --- Generation settings ---
	language := cfg.Generation.Language
	numSugStr := strconv.Itoa(cfg.Generation.NumSuggestions)
	maxDiffStr := strconv.Itoa(cfg.Generation.MaxDiffLines)

	languageInput := huh.NewInput().
		Title("Commit message language").
		Placeholder("en").
		Value(&language)

	numSugInput := huh.NewInput().
		Title("Number of suggestions").
		Placeholder("3").
		Value(&numSugStr).
		Validate(func(s string) error {
			n, err := strconv.Atoi(s)
			if err != nil || n < 1 {
				return fmt.Errorf("must be a positive integer")
			}
			return nil
		})

	maxDiffInput := huh.NewInput().
		Title("Max diff lines").
		Placeholder("4096").
		Value(&maxDiffStr).
		Validate(func(s string) error {
			n, err := strconv.Atoi(s)
			if err != nil || n <= 0 {
				return fmt.Errorf("must be a positive integer")
			}
			return nil
		})

	genGroup := huh.NewGroup(languageInput, numSugInput, maxDiffInput).
		Title("Generation Settings")

	// --- Update settings ---
	updateChannel := cfg.UpdateChannel
	if updateChannel == "" {
		updateChannel = "latest"
	}
	autoUpdate := cfg.AutoUpdate
	if autoUpdate == "" {
		autoUpdate = "y"
	}

	channelSelect := huh.NewSelect[string]().
		Title("Update channel").
		Options(
			huh.NewOption("Latest (includes pre-releases)", "latest"),
			huh.NewOption("Stable (releases only)", "stable"),
		).
		Value(&updateChannel)

	autoUpdateSelect := huh.NewSelect[string]().
		Title("Auto-update mode (dev builds only)").
		Options(
			huh.NewOption("Show update notice", "y"),
			huh.NewOption("Auto-update without asking", "a"),
			huh.NewOption("Don't check for updates", "n"),
		).
		Value(&autoUpdate)

	updateGroup := huh.NewGroup(channelSelect, autoUpdateSelect).
		Title("Update Settings")

	// Run the form
	if err := huh.NewForm(providerGroup, genGroup, updateGroup).Run(); err != nil {
		return cfg, err
	}

	// Apply values
	if model == "" {
		model = llm.DefaultModel(providerName)
	}
	if language == "" {
		language = "en"
	}

	provCfg := config.ProviderConfig{
		APIKey: apiKey,
		Model:  model,
	}
	if providerName == "custom" {
		provCfg.BaseURL = baseURL
	}

	cfg.DefaultProvider = providerName
	if cfg.Providers == nil {
		cfg.Providers = make(map[string]config.ProviderConfig)
	}
	cfg.Providers[providerName] = provCfg
	cfg.Generation.Language = language
	if n, err := strconv.Atoi(numSugStr); err == nil {
		cfg.Generation.NumSuggestions = n
	}
	if n, err := strconv.Atoi(maxDiffStr); err == nil {
		cfg.Generation.MaxDiffLines = n
	}
	cfg.UpdateChannel = updateChannel
	cfg.AutoUpdate = autoUpdate
	cfg.ConfigVersion = config.CurrentConfigVersion

	if err := config.Save(cfg); err != nil {
		return cfg, fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Println()
	fmt.Println(titleStyle.Render("✓ Configuration saved!"))
	fmt.Println()

	return cfg, nil
}
