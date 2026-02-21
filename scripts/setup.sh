#!/bin/bash
# Apple Dev Docs - Auto-install App Store Connect CLI
set -e

SKILL_DIR="$(cd "$(dirname "$0")/.." && pwd)"
BIN_DIR="$SKILL_DIR/bin"
TARGET_BIN="$BIN_DIR/appstore"
VERSION="0.31.3"

# Check if already installed
if [ -f "$TARGET_BIN" ]; then
  echo "App Store Connect CLI already installed at $TARGET_BIN"
  exit 0
fi

# Also check system PATH
if command -v appstore &>/dev/null; then
  echo "App Store Connect CLI found in PATH: $(which appstore)"
  exit 0
fi

echo "Installing App Store Connect CLI v${VERSION}..."

# Detect platform
OS="$(uname -s)"
ARCH="$(uname -m)"

case "$OS" in
  Darwin) PLATFORM="macOS" ;;
  Linux)  PLATFORM="linux" ;;
  *)      echo "Unsupported OS: $OS"; exit 1 ;;
esac

case "$ARCH" in
  arm64|aarch64) ARCH="arm64" ;;
  x86_64)        ARCH="amd64" ;;
  *)             echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

FILENAME="asc_${VERSION}_${PLATFORM}_${ARCH}"
URL="https://github.com/rudrankriyam/App-Store-Connect-CLI/releases/download/${VERSION}/${FILENAME}"

mkdir -p "$BIN_DIR"

echo "Downloading ${FILENAME}..."
curl -fsSL "$URL" -o "$TARGET_BIN"
chmod +x "$TARGET_BIN"

echo "Installed App Store Connect CLI v${VERSION} to $TARGET_BIN"
echo "Add to PATH: export PATH=\"$BIN_DIR:\$PATH\""
