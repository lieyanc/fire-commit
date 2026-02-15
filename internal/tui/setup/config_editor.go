package setup

import (
	"fmt"
	"strconv"

	"github.com/charmbracelet/huh"
	"github.com/lieyanc/fire-commit/internal/config"
	"github.com/lieyanc/fire-commit/internal/llm"
)

// RunConfigEditor opens an interactive TUI for editing the current configuration.
// It presents a main menu where the user can freely choose which section to edit.
// Changes are only saved when the user selects "Save & Exit".
func RunConfigEditor(cfg *config.Config) (*config.Config, error) {
	fmt.Println()
	fmt.Println(titleStyle.Render("🔥 fire-commit configuration"))
	fmt.Println()

	for {
		var choice string

		displayNames := llm.ProviderDisplayNames()
		providerSummary := "not configured"
		if cfg.DefaultProvider != "" {
			if p, ok := cfg.Providers[cfg.DefaultProvider]; ok {
				providerSummary = fmt.Sprintf("%s / %s", displayNames[cfg.DefaultProvider], p.Model)
			} else {
				providerSummary = displayNames[cfg.DefaultProvider]
			}
		}
		genSummary := fmt.Sprintf("%s, %d suggestions, %d lines",
			cfg.Generation.Language, cfg.Generation.NumSuggestions, cfg.Generation.MaxDiffLines)
		ch := cfg.UpdateChannel
		if ch == "" {
			ch = "latest"
		}
		timing := cfg.UpdateTiming
		if timing == "" {
			timing = "after"
		}
		cache := "off"
		if cfg.UpdateCache {
			cache = "on"
		}
		updateSummary := fmt.Sprintf("channel: %s, timing: %s, cache: %s", ch, timing, cache)

		menu := huh.NewSelect[string]().
			Title("What would you like to configure?").
			Options(
				huh.NewOption(fmt.Sprintf("Provider Settings   (%s)", providerSummary), "provider"),
				huh.NewOption(fmt.Sprintf("Generation Settings  (%s)", genSummary), "generation"),
				huh.NewOption(fmt.Sprintf("Update Settings     (%s)", updateSummary), "update"),
				huh.NewOption("Save & Exit", "save"),
			).
			Value(&choice)

		if err := huh.NewForm(huh.NewGroup(menu)).Run(); err != nil {
			return cfg, err
		}

		switch choice {
		case "provider":
			if err := editProviderSettings(cfg); err != nil {
				return cfg, err
			}
		case "generation":
			if err := editGenerationSettings(cfg); err != nil {
				return cfg, err
			}
		case "update":
			if err := editUpdateSettings(cfg); err != nil {
				return cfg, err
			}
		case "save":
			cfg.ConfigVersion = config.CurrentConfigVersion
			if err := config.Save(cfg); err != nil {
				return cfg, fmt.Errorf("failed to save config: %w", err)
			}
			fmt.Println()
			fmt.Println(titleStyle.Render("✓ Configuration saved!"))
			fmt.Println()
			return cfg, nil
		}
	}
}

// editProviderSettings runs the provider selection and details forms.
// It modifies cfg in-place. Used by both RunConfigEditor and RunWizard.
func editProviderSettings(cfg *config.Config) error {
	providerName := cfg.DefaultProvider
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
		return err
	}

	// Pre-fill from existing provider config
	var apiKey, model, baseURL string
	if p, ok := cfg.Providers[providerName]; ok {
		apiKey = p.APIKey
		model = p.Model
		baseURL = p.BaseURL
	}

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

	defaultModel := llm.DefaultModel(providerName)
	modelInput := huh.NewInput().
		Title("Model name").
		Placeholder(defaultModel).
		Value(&model)
	fields = append(fields, modelInput)

	if err := huh.NewForm(huh.NewGroup(fields...)).Run(); err != nil {
		return err
	}

	if model == "" {
		model = defaultModel
	}

	// Apply to cfg
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
	return nil
}

// editGenerationSettings runs the generation settings form.
func editGenerationSettings(cfg *config.Config) error {
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
		return err
	}

	cfg.Generation.Language = language
	cfg.Generation.NumSuggestions = numSuggestions
	if n, err := strconv.Atoi(maxDiffStr); err == nil {
		cfg.Generation.MaxDiffLines = n
	}
	return nil
}

// editUpdateSettings runs the update settings form.
func editUpdateSettings(cfg *config.Config) error {
	updateChannel := cfg.UpdateChannel
	if updateChannel == "" {
		updateChannel = "latest"
	}
	autoUpdate := cfg.AutoUpdate
	if autoUpdate == "" {
		autoUpdate = "y"
	}
	updateTiming := cfg.UpdateTiming
	if updateTiming == "" {
		updateTiming = "after"
	}
	updateCache := cfg.UpdateCache

	channelSelect := huh.NewSelect[string]().
		Title("Update channel").
		Options(
			huh.NewOption("Latest (includes pre-releases)", "latest"),
			huh.NewOption("Stable (releases only)", "stable"),
		).
		Value(&updateChannel)

	autoUpdateSelect := huh.NewSelect[string]().
		Title("Auto-update mode (non-dev builds only)").
		Options(
			huh.NewOption("Show update notice", "y"),
			huh.NewOption("Auto-update without asking", "a"),
			huh.NewOption("Don't check for updates", "n"),
		).
		Value(&autoUpdate)

	timingSelect := huh.NewSelect[string]().
		Title("Update timing").
		Options(
			huh.NewOption("After exit (default)", "after"),
			huh.NewOption("Before startup", "before"),
		).
		Value(&updateTiming)

	cacheConfirm := huh.NewConfirm().
		Title("Enable update-check cache").
		Description("Yes: ETag + adaptive interval. No: check every run (default).").
		Value(&updateCache)

	if err := huh.NewForm(huh.NewGroup(channelSelect, autoUpdateSelect, timingSelect, cacheConfirm)).Run(); err != nil {
		return err
	}

	cfg.UpdateChannel = updateChannel
	cfg.AutoUpdate = autoUpdate
	cfg.UpdateTiming = updateTiming
	cfg.UpdateCache = updateCache
	return nil
}
