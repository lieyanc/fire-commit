# fire-commit

AI-powered conventional commit message generator with a beautiful TUI.

Analyzes your staged git diff, streams multiple commit message suggestions via LLM, and lets you pick, edit, commit, and push — all without leaving the terminal.

## Install

**One-line install** (Linux / macOS):

```sh
curl -fsSL https://raw.githubusercontent.com/lieyanc/fire-commit/master/install.sh | bash
```

This downloads the latest release to `~/.fire-commit/bin/` and configures your shell PATH. An interactive menu lets you choose between the **latest** (default, includes dev builds) and **stable** channels.

For non-interactive use:

```sh
# Install from dev channel (default)
curl -fsSL https://raw.githubusercontent.com/lieyanc/fire-commit/master/install.sh | bash -s -- --latest

# Install from stable channel only
curl -fsSL https://raw.githubusercontent.com/lieyanc/fire-commit/master/install.sh | bash -s -- --stable
```

**From source** (requires Go 1.25+):

```sh
git clone https://github.com/lieyanc/fire-commit.git
cd fire-commit
make install
```

**Binaries**: Download pre-built archives from the [Releases](https://github.com/lieyanc/fire-commit/releases) page.

## Usage

```sh
# Run in any git repo with changes
firecommit

# Also available as
fcmt
git fire-commit   # works as a git subcommand: git fire-commit
```

On first run, an interactive setup wizard will ask you to choose an LLM provider and enter your API key.

### Workflow

1. **Stage** — if nothing is staged, fire-commit auto-stages all changes
2. **Generate** — streams commit message suggestions from your configured LLM
3. **Select** — pick a suggestion with `j`/`k` and `Enter`
4. **Edit** — press `e` to customize the message
5. **Commit** — confirm and optionally push with `p`

### Keyboard Shortcuts

| Key | Action |
|-----|--------|
| `↑` / `k` | Move up |
| `↓` / `j` | Move down |
| `Enter` | Confirm |
| `e` | Edit message |
| `r` | Regenerate suggestions |
| `p` | Toggle push |
| `Tab` | Switch |
| `Esc` | Back |
| `q` | Quit |

### Commands

```sh
firecommit              # default — generate & commit
firecommit version      # print version
firecommit update       # self-update to latest release
firecommit config       # show current configuration
firecommit config setup # re-run the setup wizard
```

## Supported Providers

| Provider | Default Model | Notes |
|----------|--------------|-------|
| OpenAI | `gpt-5-nano` | |
| Anthropic | `claude-haiku-4-5` | |
| Google Gemini | `gemini-2.5-flash-lite` | OpenAI-compatible endpoint |
| Cerebras | `gpt-oss-120b` | |
| SiliconFlow | `Qwen/Qwen3-Next-80B-A3B-Instruct` | |
| Custom | — | Any OpenAI-compatible API |

## Configuration

Config is stored at `~/.config/firecommit/config.yaml` (follows XDG). Override with `FIRECOMMIT_CONFIG` env var.

```yaml
default_provider: openai
providers:
  openai:
    api_key: sk-...
    model: gpt-5-nano        # optional, uses default if omitted
  custom:
    api_key: your-key
    model: your-model
    base_url: https://your-endpoint/v1
generation:
  num_suggestions: 3          # number of suggestions to generate
  language: en                # commit message language (en, zh, ja, ko, es, fr, de, ru)
  max_diff_lines: 4096        # truncate diff beyond this
update_channel: latest        # "latest" (dev + stable) or "stable"
auto_update: y                # non-dev builds: y(notify), a(auto-update), n(skip checks)
update_timing: after          # "after" (default) or "before"
```

## Auto-Update

fire-commit checks for updates in the background (unless `auto_update: n`):

- `latest` channel: at most once every 3 hours
- `stable` channel: at most once every 24 hours

Update behavior:

- Dev builds (`dev-*`) always auto-update when a newer version is found.
- Non-dev builds default to notice-only (`auto_update: y`) and can be configured to auto-update (`a`) or skip checks (`n`).

If a newer version is found in notice mode, a message is printed after the command exits:

```
A new version of fire-commit is available: v0.1.0 -> v0.2.0
Run `firecommit update` to upgrade.
```

Run `firecommit update` to download and replace the binary in-place.

### Update Channels

fire-commit supports two update channels:

| Channel | Description |
|---------|-------------|
| `latest` | Includes dev builds and stable releases (default) |
| `stable` | Only stable tagged releases |

The channel is set during installation and stored in `update_channel` in your config file. Both `firecommit update` and the background update check respect this setting.

### Dev Builds

Every push to `master` triggers an automated dev build. These are published as pre-releases under the rolling `dev` tag with version strings like `dev-20260214-1234-abc1234` (`date-build-hash`).

## Building

```sh
make build     # build to ./bin/
make install   # build and install to ~/.fire-commit/bin/
make dist      # cross-compile for all platforms
make clean     # remove build artifacts
```

Version is injected at build time via `-ldflags`:

```sh
go build -ldflags "-s -w -X main.version=v1.0.0" -o firecommit ./cmd/firecommit
```

## License

MIT
