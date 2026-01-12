#!/bin/bash
# PAL Kit Installer
# Usage: curl -fsSL https://raw.githubusercontent.com/n0roo/pal-kit/main/install.sh | bash

set -e

REPO="n0roo/pal-kit"
BINARY_NAME="pal"
INSTALL_DIR="${PAL_INSTALL_DIR:-$HOME/.local/bin}"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

info() { echo -e "${GREEN}[INFO]${NC} $1"; }
warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
error() { echo -e "${RED}[ERROR]${NC} $1"; exit 1; }

# Detect OS and architecture
detect_platform() {
    OS=$(uname -s | tr '[:upper:]' '[:lower:]')
    ARCH=$(uname -m)

    case "$OS" in
        darwin) OS="darwin" ;;
        linux) OS="linux" ;;
        mingw*|msys*|cygwin*) OS="windows" ;;
        *) error "Unsupported OS: $OS" ;;
    esac

    case "$ARCH" in
        x86_64|amd64) ARCH="amd64" ;;
        arm64|aarch64) ARCH="arm64" ;;
        *) error "Unsupported architecture: $ARCH" ;;
    esac

    echo "${OS}_${ARCH}"
}

# Get latest release version
get_latest_version() {
    curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | \
        grep '"tag_name":' | \
        sed -E 's/.*"([^"]+)".*/\1/'
}

# Download and install
install_pal() {
    PLATFORM=$(detect_platform)
    VERSION=$(get_latest_version)

    if [ -z "$VERSION" ]; then
        error "Failed to get latest version"
    fi

    info "Installing PAL Kit ${VERSION} for ${PLATFORM}..."

    # Determine file extension
    EXT="tar.gz"
    if [ "$OS" = "windows" ]; then
        EXT="zip"
    fi

    # Download URL
    DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${VERSION}/${BINARY_NAME}_${VERSION#v}_${PLATFORM}.${EXT}"

    info "Downloading from ${DOWNLOAD_URL}..."

    # Create temp directory
    TMP_DIR=$(mktemp -d)
    trap "rm -rf $TMP_DIR" EXIT

    # Download
    if ! curl -fsSL "$DOWNLOAD_URL" -o "$TMP_DIR/pal.${EXT}"; then
        error "Failed to download PAL Kit"
    fi

    # Extract
    cd "$TMP_DIR"
    if [ "$EXT" = "tar.gz" ]; then
        tar -xzf "pal.${EXT}"
    else
        unzip -q "pal.${EXT}"
    fi

    # Create install directory
    mkdir -p "$INSTALL_DIR"

    # Install binary
    if [ -f "$BINARY_NAME" ]; then
        mv "$BINARY_NAME" "$INSTALL_DIR/"
        chmod +x "$INSTALL_DIR/$BINARY_NAME"
    else
        error "Binary not found in archive"
    fi

    # Remove quarantine attribute on macOS
    if [ "$OS" = "darwin" ]; then
        xattr -d com.apple.quarantine "$INSTALL_DIR/$BINARY_NAME" 2>/dev/null || true
    fi

    info "PAL Kit installed to $INSTALL_DIR/$BINARY_NAME"

    # Check if in PATH
    if ! command -v "$BINARY_NAME" &> /dev/null; then
        warn "$INSTALL_DIR is not in your PATH"
        echo ""
        echo "Add to your shell profile:"
        echo ""
        if [ -f "$HOME/.zshrc" ]; then
            echo "  echo 'export PATH=\"$INSTALL_DIR:\$PATH\"' >> ~/.zshrc"
            echo "  source ~/.zshrc"
        elif [ -f "$HOME/.bashrc" ]; then
            echo "  echo 'export PATH=\"$INSTALL_DIR:\$PATH\"' >> ~/.bashrc"
            echo "  source ~/.bashrc"
        else
            echo "  export PATH=\"$INSTALL_DIR:\$PATH\""
        fi
        echo ""
    fi

    info "Installation complete!"
    echo ""
    echo "Run 'pal --help' to get started"
    echo ""
}

# Main
main() {
    echo ""
    echo "  ____   _    _       _  ___ _   "
    echo " |  _ \\ / \\  | |     | |/ (_) |_ "
    echo " | |_) / _ \\ | |     | ' /| | __|"
    echo " |  __/ ___ \\| |___  | . \\| | |_ "
    echo " |_| /_/   \\_\\_____| |_|\\_\\_|\\__|"
    echo ""
    echo " Claude Code Project Management CLI"
    echo ""

    install_pal
}

main
