# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Fire-commit is an AI-powered conventional commit message generator with a TUI (Terminal User Interface). It analyzes staged git changes, streams multiple commit message suggestions via LLM APIs, and lets users select/edit/commit/push interactively. Written in Go.

Binary aliases: `firecommit`, `fcmt`, `git fire-commit`.

## Build & Run Commands

```bash
make build      # Build to ./bin/ (creates firecommit + symlinks fcmt, git-fire-commit)
make install    # Build and install to ~/.fire-commit/bin/
make dist       # Cross-compile for Linux/macOS/Windows (amd64, arm64)
make clean      # Remove build artifacts
```

Version is injected via ldflags: `-X main.version=$(VERSION)`.

There are no tests in this repository currently.

## Architecture

### Entry Point & CLI Layer

- `cmd/firecommit/main.go` — Entry point. Sets version, loads config, runs background update check, executes CLI.
- `internal/cli/` — Cobra command handlers. `root.go` is the main flow: verify git repo → check staged changes (auto-stage if needed) → get diff → launch TUI.

### TUI (Bubble Tea State Machine)

`internal/tui/app.go` orchestrates a 6-phase state machine:

```
Loading → Select → Edit (optional) → Confirm → Committing → Done
```

Each phase is in its own file (`phase_loading.go`, `phase_select.go`, etc.). The `setup/` subdirectory contains the first-time setup wizard and config editor.

Styling uses Lipgloss with a brand blue palette defined in `styles.go`.

### LLM Provider System

`internal/llm/provider.go` defines the `Provider` interface:
```go
type Provider interface {
    GenerateCommitMessages(ctx context.Context, diff string, opts GenerateOptions) (<-chan StreamChunk, error)
}
```

Three implementations:
- `OpenAIProvider` (`openai.go`) — Direct OpenAI SDK
- `AnthropicProvider` (`anthropic.go`) — Direct Anthropic SDK
- `OpenAICompatProvider` (`openai_compat.go`) — Generic wrapper for any OpenAI-compatible API (used by Gemini, Cerebras, SiliconFlow, custom endpoints)

`registry.go` holds the provider registry with default models/endpoints. `prompt.go` builds language-aware system/user prompts for conventional commit generation. Multiple suggestions are streamed in parallel via `GenerateMultiple`.

### Git Operations

`internal/git/` wraps git commands: `diff.go` (staged diff with truncation), `commit.go` (commit, push, tag), `status.go` (repo/staging checks).

### Configuration

`internal/config/config.go` — YAML config at `~/.config/firecommit/config.yaml` (XDG-compliant). Stores provider selection, API keys, model overrides, generation settings (num_suggestions, language, max_diff_lines), and update preferences.

### Auto-Update

`internal/updater/updater.go` — Fetches GitHub releases, supports `latest` (dev + stable) and `stable` channels, version-aware update checks (dev `date-build-hash`), and self-update via binary replacement.

## Key Patterns

- **Streaming via channels**: LLM responses stream chunk-by-chunk through Go channels
- **Parallel generation**: Multiple suggestions generated concurrently, first error cancels remaining
- **Provider factory**: `NewProvider()` in `provider.go` routes config to the correct implementation
- **Phase-based TUI**: Each phase handles its own Update/View logic; `app.go` dispatches based on current phase

## Commit Style

This project uses conventional commits (e.g., `feat(tui):`, `fix(llm):`, `refactor(config):`). Scopes typically match the package name under `internal/`.
