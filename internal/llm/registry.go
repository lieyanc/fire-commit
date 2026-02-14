package llm

import (
	"fmt"

	"github.com/lieyanc/fire-commit/internal/config"
)

// Known provider base URLs for OpenAI-compatible services.
var providerBaseURLs = map[string]string{
	"gemini":      "https://generativelanguage.googleapis.com/v1beta/openai/",
	"cerebras":    "https://api.cerebras.ai/v1",
	"siliconflow": "https://api.siliconflow.cn/v1",
}

// Default models for each provider.
var defaultModels = map[string]string{
	"openai":      "gpt-5-nano",
	"anthropic":   "claude-haiku-4-5",
	"gemini":      "gemini-2.5-flash-lite",
	"cerebras":    "gpt-oss-120b",
	"siliconflow": "Qwen/Qwen3-Next-80B-A3B-Instruct",
}

// ProviderNames returns the list of supported provider names.
func ProviderNames() []string {
	return []string{"openai", "anthropic", "gemini", "cerebras", "siliconflow", "custom"}
}

// ProviderDisplayNames returns human-readable names for providers.
func ProviderDisplayNames() map[string]string {
	return map[string]string{
		"openai":      "OpenAI",
		"anthropic":   "Anthropic (Claude)",
		"gemini":      "Google Gemini",
		"cerebras":    "Cerebras",
		"siliconflow": "SiliconFlow",
		"custom":      "Custom (OpenAI-compatible)",
	}
}

// DefaultModel returns the default model for a given provider.
func DefaultModel(provider string) string {
	if m, ok := defaultModels[provider]; ok {
		return m
	}
	return ""
}

// NewProvider creates a Provider from the given configuration.
func NewProvider(cfg *config.Config) (Provider, error) {
	name := cfg.DefaultProvider
	provCfg, ok := cfg.Providers[name]
	if !ok {
		return nil, fmt.Errorf("provider %q not configured", name)
	}

	if provCfg.APIKey == "" {
		return nil, fmt.Errorf("API key not set for provider %q", name)
	}

	model := provCfg.Model
	if model == "" {
		model = DefaultModel(name)
	}

	switch name {
	case "openai":
		return NewOpenAIProvider(provCfg.APIKey, model), nil
	case "anthropic":
		return NewAnthropicProvider(provCfg.APIKey, model), nil
	case "gemini", "cerebras", "siliconflow":
		baseURL := providerBaseURLs[name]
		return NewOpenAICompatProvider(provCfg.APIKey, model, baseURL), nil
	case "custom":
		if provCfg.BaseURL == "" {
			return nil, fmt.Errorf("custom provider requires a base_url")
		}
		return NewOpenAICompatProvider(provCfg.APIKey, model, provCfg.BaseURL), nil
	default:
		return nil, fmt.Errorf("unknown provider: %q", name)
	}
}
