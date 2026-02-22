#!/bin/bash
# Apple Developer Toolkit - Build from Source
#
# Builds the unified appledev binary from this repo.
# No external Go dependencies needed - everything is in-tree.
#
# Source: https://github.com/Abdullah4AI/apple-developer-toolkit

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_DIR="$(dirname "$SCRIPT_DIR")"
BUILD_DIR="$REPO_DIR/bin"
INSTALL_DIR="/opt/homebrew/bin"

if ! command -v go &>/dev/null; then
  echo "Go required. Install: https://go.dev/dl/"
  exit 1
fi

echo "Building appledev..."

mkdir -p "$BUILD_DIR"
cd "$REPO_DIR"

VERSION=$(git describe --tags --always 2>/dev/null || echo "dev")
COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE=$(date -u +%Y-%m-%dT%H:%M:%SZ)

go build -ldflags "-s -w -X main.version=$VERSION -X main.commit=$COMMIT -X main.date=$DATE" \
  -o "$BUILD_DIR/appledev" ./cmd/appledev/

echo "Built: $BUILD_DIR/appledev"

if [ -d "$INSTALL_DIR" ]; then
  cp "$BUILD_DIR/appledev" "$INSTALL_DIR/appledev"
  ln -sf appledev "$INSTALL_DIR/swiftship"
  ln -sf appledev "$INSTALL_DIR/appstore"
  echo "Installed to $INSTALL_DIR"
else
  echo "Binary at: $BUILD_DIR/appledev"
fi

echo ""
echo "Done."
echo "  appledev --help         Unified CLI"
echo "  appledev store --help   App Store Connect"
echo "  appledev build --help   iOS App Builder"
