#!/bin/sh
# fire-commit installer
# Usage: curl -fsSL https://raw.githubusercontent.com/lieyanc/fire-commit/master/install.sh | bash
#   --latest   Install latest (dev + stable) channel (default)
#   --stable   Install stable channel only
set -e

REPO="lieyanc/fire-commit"
INSTALL_DIR="$HOME/.fire-commit/bin"
CHANNEL=""

info() { printf "\033[1;34m==>\033[0m %s\n" "$1"; }
error() { printf "\033[1;31mError:\033[0m %s\n" "$1" >&2; exit 1; }

# Parse arguments
for arg in "$@"; do
    case "$arg" in
        --stable) CHANNEL="stable";;
        --latest) CHANNEL="latest";;
    esac
done

# If no flag, show interactive menu or default to latest
if [ -z "$CHANNEL" ]; then
    if [ -t 0 ] && [ -t 1 ]; then
        printf "\n\033[1mSelect update channel:\033[0m\n"
        printf "  1) latest  — includes dev builds and stable releases (default)\n"
        printf "  2) stable  — only stable releases\n"
        printf "\n"
        printf "Choice [1]: "
        read -r choice
        case "$choice" in
            2) CHANNEL="stable";;
            *) CHANNEL="latest";;
        esac
    else
        CHANNEL="latest"
    fi
fi

# Detect OS
case "$(uname -s)" in
    Linux*)  OS="linux";;
    Darwin*) OS="darwin";;
    *)       error "Unsupported OS: $(uname -s). Only Linux and macOS are supported.";;
esac

# Detect architecture
case "$(uname -m)" in
    x86_64|amd64)  ARCH="amd64";;
    aarch64|arm64) ARCH="arm64";;
    *)             error "Unsupported architecture: $(uname -m). Only amd64 and arm64 are supported.";;
esac

info "Detected platform: ${OS}/${ARCH}"
info "Channel: ${CHANNEL}"

# Helper: fetch URL content
fetch() {
    if command -v curl >/dev/null 2>&1; then
        curl -fsSL "$1"
    elif command -v wget >/dev/null 2>&1; then
        wget -qO- "$1"
    else
        error "Neither curl nor wget found. Please install one and try again."
    fi
}

# Helper: download file
download() {
    if command -v curl >/dev/null 2>&1; then
        curl -fsSL -o "$2" "$1"
    else
        wget -qO "$2" "$1"
    fi
}

if [ "$CHANNEL" = "stable" ]; then
    # Stable: fetch /releases/latest
    info "Fetching latest stable release..."
    RELEASE_JSON=$(fetch "https://api.github.com/repos/${REPO}/releases/latest")

    VERSION=$(printf '%s' "$RELEASE_JSON" | grep '"tag_name"' | head -1 | sed 's/.*"tag_name": *"//;s/".*//')
    if [ -z "$VERSION" ]; then
        error "Failed to determine latest stable version. Check your internet connection."
    fi

    VERSION_NUM=$(printf '%s' "$VERSION" | sed 's/^v//')
    ARCHIVE="fire-commit_${VERSION_NUM}_${OS}_${ARCH}.tar.gz"
    DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${VERSION}/${ARCHIVE}"
    CHECKSUMS_URL="https://github.com/${REPO}/releases/download/${VERSION}/checksums.txt"
    DISPLAY_VERSION="$VERSION"
else
    # Latest: fetch /releases/tags/dev (dev pre-release)
    info "Fetching latest dev release..."
    RELEASE_JSON=$(fetch "https://api.github.com/repos/${REPO}/releases/tags/dev")

    VERSION_NAME=$(printf '%s' "$RELEASE_JSON" | grep '"name"' | head -1 | sed 's/.*"name": *"//;s/".*//')
    if [ -z "$VERSION_NAME" ]; then
        error "Failed to determine latest dev version. Check your internet connection."
    fi

    ARCHIVE="fire-commit_dev_${OS}_${ARCH}.tar.gz"
    DOWNLOAD_URL="https://github.com/${REPO}/releases/download/dev/${ARCHIVE}"
    CHECKSUMS_URL="https://github.com/${REPO}/releases/download/dev/checksums.txt"
    DISPLAY_VERSION="$VERSION_NAME"
fi

info "Version: ${DISPLAY_VERSION}"

# Download to temp directory
TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT

info "Downloading ${ARCHIVE}..."
download "$DOWNLOAD_URL" "${TMPDIR}/${ARCHIVE}"
download "$CHECKSUMS_URL" "${TMPDIR}/checksums.txt" 2>/dev/null || true

# Verify checksum if checksums.txt was downloaded
if [ -f "${TMPDIR}/checksums.txt" ]; then
    EXPECTED=$(grep "${ARCHIVE}" "${TMPDIR}/checksums.txt" | awk '{print $1}')
    if [ -n "$EXPECTED" ]; then
        if command -v sha256sum >/dev/null 2>&1; then
            ACTUAL=$(sha256sum "${TMPDIR}/${ARCHIVE}" | awk '{print $1}')
        elif command -v shasum >/dev/null 2>&1; then
            ACTUAL=$(shasum -a 256 "${TMPDIR}/${ARCHIVE}" | awk '{print $1}')
        else
            ACTUAL=""
        fi
        if [ -n "$ACTUAL" ]; then
            if [ "$ACTUAL" != "$EXPECTED" ]; then
                error "Checksum verification failed!\n  Expected: ${EXPECTED}\n  Got:      ${ACTUAL}"
            fi
            info "Checksum verified."
        fi
    fi
fi

# Extract
info "Installing to ${INSTALL_DIR}..."
mkdir -p "$INSTALL_DIR"
tar xzf "${TMPDIR}/${ARCHIVE}" -C "$TMPDIR"

# Copy binary and create symlinks
cp "${TMPDIR}/firecommit" "${INSTALL_DIR}/firecommit"
chmod +x "${INSTALL_DIR}/firecommit"
ln -sf firecommit "${INSTALL_DIR}/fcmt"
ln -sf firecommit "${INSTALL_DIR}/git-fire-commit"

# Write update channel to config
CONFIG_DIR="${XDG_CONFIG_HOME:-$HOME/.config}/firecommit"
CONFIG_FILE="${CONFIG_DIR}/config.yaml"
mkdir -p "$CONFIG_DIR"

if [ -f "$CONFIG_FILE" ]; then
    if grep -q '^update_channel:' "$CONFIG_FILE"; then
        # Replace existing line
        sed -i.bak "s/^update_channel:.*$/update_channel: ${CHANNEL}/" "$CONFIG_FILE"
        rm -f "${CONFIG_FILE}.bak"
    else
        # Append to existing file
        printf 'update_channel: %s\n' "$CHANNEL" >> "$CONFIG_FILE"
    fi
else
    printf 'update_channel: %s\n' "$CHANNEL" > "$CONFIG_FILE"
fi
info "Update channel set to '${CHANNEL}' in ${CONFIG_FILE}"

# Configure PATH
add_to_path() {
    local rc_file="$1"
    local line="$2"

    if [ -f "$rc_file" ] && grep -qF '.fire-commit/bin' "$rc_file"; then
        return 0
    fi

    printf '\n# fire-commit\n%s\n' "$line" >> "$rc_file"
    info "Added fire-commit to PATH in ${rc_file}"
}

SHELL_NAME=$(basename "$SHELL" 2>/dev/null || echo "")

case "$SHELL_NAME" in
    zsh)
        add_to_path "$HOME/.zshrc" 'export PATH="$HOME/.fire-commit/bin:$PATH"'
        ;;
    bash)
        if [ "$OS" = "darwin" ] && [ ! -f "$HOME/.bashrc" ]; then
            add_to_path "$HOME/.bash_profile" 'export PATH="$HOME/.fire-commit/bin:$PATH"'
        else
            add_to_path "$HOME/.bashrc" 'export PATH="$HOME/.fire-commit/bin:$PATH"'
        fi
        ;;
    fish)
        mkdir -p "$HOME/.config/fish"
        add_to_path "$HOME/.config/fish/config.fish" 'fish_add_path $HOME/.fire-commit/bin'
        ;;
    *)
        info "Could not detect shell. Please add ${INSTALL_DIR} to your PATH manually."
        ;;
esac

printf '\n\033[1;32mfire-commit %s installed successfully!\033[0m\n' "$DISPLAY_VERSION"
printf 'Channel: %s\n' "$CHANNEL"
printf 'Restart your shell or run:\n  export PATH="%s:$PATH"\n' "$INSTALL_DIR"
printf 'Then run: firecommit --help\n'
