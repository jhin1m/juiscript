#!/usr/bin/env bash
# juiscript installer
# Usage: curl -sSL https://raw.githubusercontent.com/jhin1m/juiscript/main/install.sh | sudo bash
#
# Downloads the latest juiscript binary from GitHub Releases
# and installs it to /usr/local/bin/juiscript.

set -euo pipefail

# --- Configuration ---
REPO="jhin1m/juiscript"
BINARY_NAME="juiscript"
INSTALL_DIR="/usr/local/bin"

# --- Colors for output ---
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

info()  { echo -e "${GREEN}[INFO]${NC} $*"; }
warn()  { echo -e "${YELLOW}[WARN]${NC} $*"; }
error() { echo -e "${RED}[ERROR]${NC} $*" >&2; exit 1; }

# --- Pre-flight checks ---

# Must run as root because juiscript manages system services
if [ "$(id -u)" -ne 0 ]; then
    error "This installer must be run as root. Use: curl -sSL ... | sudo bash"
fi

# Only support Linux (juiscript manages LEMP on Ubuntu)
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
if [ "$OS" != "linux" ]; then
    error "juiscript only supports Linux. Detected: $OS"
fi

# Detect architecture and map to Go naming convention
ARCH=$(uname -m)
case "$ARCH" in
    x86_64)  ARCH="amd64" ;;
    aarch64) ARCH="arm64" ;;
    *)       error "Unsupported architecture: $ARCH. Supported: x86_64, aarch64" ;;
esac

# Need curl or wget to download
if command -v curl &>/dev/null; then
    DOWNLOAD="curl -fsSL"
elif command -v wget &>/dev/null; then
    DOWNLOAD="wget -qO-"
else
    error "curl or wget is required but not found"
fi

# --- Resolve latest version ---
info "Detecting latest version..."
LATEST_URL="https://api.github.com/repos/${REPO}/releases/latest"

# GitHub API returns JSON - extract tag_name with grep+sed (no jq dependency)
VERSION=$($DOWNLOAD "$LATEST_URL" | grep '"tag_name"' | sed -E 's/.*"tag_name": *"([^"]+)".*/\1/')

if [ -z "$VERSION" ]; then
    error "Failed to detect latest version. Check https://github.com/${REPO}/releases"
fi

info "Latest version: $VERSION"

# --- Download binary ---
DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${VERSION}/${BINARY_NAME}-linux-${ARCH}"
info "Downloading ${BINARY_NAME}-linux-${ARCH}..."

TMP_FILE=$(mktemp)
trap 'rm -f "$TMP_FILE"' EXIT

if command -v curl &>/dev/null; then
    curl -fsSL -o "$TMP_FILE" "$DOWNLOAD_URL"
else
    wget -qO "$TMP_FILE" "$DOWNLOAD_URL"
fi

# --- Install ---
install -m 755 "$TMP_FILE" "${INSTALL_DIR}/${BINARY_NAME}"

# --- Verify ---
INSTALLED_VERSION=$("${INSTALL_DIR}/${BINARY_NAME}" version 2>/dev/null || echo "unknown")
info "Installed: $INSTALLED_VERSION"
info "Binary location: ${INSTALL_DIR}/${BINARY_NAME}"
echo ""
info "Run 'sudo juiscript' to start managing your LEMP server."
