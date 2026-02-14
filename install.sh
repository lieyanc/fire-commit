#!/bin/sh
# fire-commit installer
# Usage: curl -fsSL https://raw.githubusercontent.com/lieyanc/fire-commit/master/install.sh | bash
set -e

REPO="lieyanc/fire-commit"
INSTALL_DIR="$HOME/.fire-commit/bin"

info() { printf "\033[1;34m==>\033[0m %s\n" "$1"; }
error() { printf "\033[1;31mError:\033[0m %s\n" "$1" >&2; exit 1; }

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

# Fetch latest release tag
info "Fetching latest release..."
if command -v curl >/dev/null 2>&1; then
    RELEASE_JSON=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest")
elif command -v wget >/dev/null 2>&1; then
    RELEASE_JSON=$(wget -qO- "https://api.github.com/repos/${REPO}/releases/latest")
else
    error "Neither curl nor wget found. Please install one and try again."
fi

VERSION=$(printf '%s' "$RELEASE_JSON" | grep '"tag_name"' | head -1 | sed 's/.*"tag_name": *"//;s/".*//')
if [ -z "$VERSION" ]; then
    error "Failed to determine latest version. Check your internet connection."
fi

VERSION_NUM=$(printf '%s' "$VERSION" | sed 's/^v//')
info "Latest version: ${VERSION}"

# Build download URL
ARCHIVE="fire-commit_${VERSION_NUM}_${OS}_${ARCH}.tar.gz"
DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${VERSION}/${ARCHIVE}"
CHECKSUMS_URL="https://github.com/${REPO}/releases/download/${VERSION}/checksums.txt"

# Download to temp directory
TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT

info "Downloading ${ARCHIVE}..."
if command -v curl >/dev/null 2>&1; then
    curl -fsSL -o "${TMPDIR}/${ARCHIVE}" "$DOWNLOAD_URL"
    curl -fsSL -o "${TMPDIR}/checksums.txt" "$CHECKSUMS_URL" 2>/dev/null || true
else
    wget -qO "${TMPDIR}/${ARCHIVE}" "$DOWNLOAD_URL"
    wget -qO "${TMPDIR}/checksums.txt" "$CHECKSUMS_URL" 2>/dev/null || true
fi

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

printf '\n\033[1;32mfire-commit %s installed successfully!\033[0m\n' "$VERSION"
printf 'Restart your shell or run:\n  export PATH="%s:$PATH"\n' "$INSTALL_DIR"
printf 'Then run: firecommit --help\n'
