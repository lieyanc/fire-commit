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

	// ── Step 1: Select provider ──
	providerName := cfg.DefaultProvider
	names := llm.ProviderNames()
	displayNames := llm.ProviderDisplayNames()

	providerOptions := make([]huh.Option[string], len(names))
	for i, name := range names {
		providerOptions[i] = huh.NewOption(displayNames[name], name)
	}

	providerSelect := huh.NewSelect[string]().
		Title("LLM Provider").
		Options(providerOptions...).
		Value(&providerName)

	if err := huh.NewForm(huh.NewGroup(providerSelect)).Run(); err != nil {
		return cfg, err
	}

	// ── Step 2: Provider details (API key, model, base URL) ──
	var apiKey, model, baseURL string
	if p, ok := cfg.Providers[providerName]; ok {
		apiKey = p.APIKey
		model = p.Model
		baseURL = p.BaseURL
	}

	apiKeyInput := huh.NewInput().
		Title(fmt.Sprintf("%s API Key", displayNames[providerName])).
		Value(&apiKey).
		EchoMode(huh.EchoModePassword).
		Validate(func(s string) error {
			if s == "" {
				return fmt.Errorf("API key is required")
			}
			return nil
		})

	providerFields := []huh.Field{apiKeyInput}

	if providerName == "custom" {
		baseURLInput := huh.NewInput().
			Title("API Base URL").
			Placeholder("https://api.example.com/v1").
			Value(&baseURL).
			Validate(func(s string) error {
				if s == "" {
					return fmt.Errorf("base URL is required for custom provider")
				}
				return nil
			})
		providerFields = append(providerFields, baseURLInput)
	}

	defaultModel := llm.DefaultModel(providerName)
	modelInput := huh.NewInput().
		Title("Model name").
		Placeholder(defaultModel).
		Value(&model)
	providerFields = append(providerFields, modelInput)

	if err := huh.NewForm(huh.NewGroup(providerFields...)).Run(); err != nil {
		return cfg, err
	}

	// ── Step 3: Generation settings ──
	language := cfg.Generation.Language
	numSuggestions := cfg.Generation.NumSuggestions
	maxDiffStr := strconv.Itoa(cfg.Generation.MaxDiffLines)

	languageSelect := huh.NewSelect[string]().
		Title("Commit message language").
		Options(
			huh.NewOption("English", "en"),
			huh.NewOption("中文", "zh"),
			huh.NewOption("日本語", "ja"),
			huh.NewOption("한국어", "ko"),
		).
		Value(&language)

	numSugSelect := huh.NewSelect[int]().
		Title("Number of suggestions").
		Options(
			huh.NewOption("1", 1),
			huh.NewOption("2", 2),
			huh.NewOption("3 (default)", 3),
			huh.NewOption("4", 4),
			huh.NewOption("5", 5),
		).
		Value(&numSuggestions)

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

	if err := huh.NewForm(huh.NewGroup(languageSelect, numSugSelect, maxDiffInput)).Run(); err != nil {
		return cfg, err
	}

	// ── Step 4: Update settings ──
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

	if err := huh.NewForm(huh.NewGroup(channelSelect, autoUpdateSelect)).Run(); err != nil {
		return cfg, err
	}

	// ── Apply & save ──
	if model == "" {
		model = defaultModel
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
	cfg.Generation.NumSuggestions = numSuggestions
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
