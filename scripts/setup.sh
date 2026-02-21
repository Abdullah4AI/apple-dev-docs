#!/bin/bash
# Apple Developer Toolkit - Auto-build custom binaries
set -e

SKILL_DIR="$(cd "$(dirname "$0")/.." && pwd)"
BIN_DIR="$SKILL_DIR/bin"

mkdir -p "$BIN_DIR"

# Check if Go is available
if ! command -v go &>/dev/null; then
  echo "Go is required. Install: brew install go"
  exit 1
fi

# Build appstore CLI (App Store Connect)
if [ ! -f "$BIN_DIR/appstore" ]; then
  echo "Building App Store Connect CLI..."
  TMPDIR=$(mktemp -d)
  git clone --depth 1 https://github.com/rudrankriyam/App-Store-Connect-CLI.git "$TMPDIR/src" 2>/dev/null
  cd "$TMPDIR/src"
  sed -i '' 's/Name:      "asc"/Name:      "appstore"/g' cmd/run.go 2>/dev/null || true
  sed -i '' 's/Name:        "asc"/Name:        "appstore"/g' cmd/root.go 2>/dev/null || true
  sed -i '' 's/"asc <subcommand>/"appstore <subcommand>/g' cmd/root.go 2>/dev/null || true
  sed -i '' 's/Unofficial\. asc is a fast/A fast/g' cmd/root.go 2>/dev/null || true
  CGO_ENABLED=0 go build -ldflags="-s -w" -o "$BIN_DIR/appstore" . 2>/dev/null
  rm -rf "$TMPDIR"
  echo "Built appstore CLI"
fi

# Build swiftship CLI (iOS App Builder)
if [ ! -f "$BIN_DIR/swiftship" ]; then
  echo "Building iOS App Builder CLI..."
  TMPDIR=$(mktemp -d)
  git clone --depth 1 https://github.com/moasq/nanowave.git "$TMPDIR/src" 2>/dev/null
  cd "$TMPDIR/src"
  find . -name "*.go" -not -path "./.git/*" -exec sed -i '' -e 's/nanowave/swiftship/g' -e 's/Nanowave/SwiftShip/g' {} + 2>/dev/null
  mv cmd/nanowave cmd/swiftship 2>/dev/null || true
  sed -i '' 's|module github.com/moasq/nanowave|module github.com/moasq/swiftship|g' go.mod
  CGO_ENABLED=0 go build -ldflags="-s -w" -o "$BIN_DIR/swiftship" ./cmd/swiftship/ 2>/dev/null
  rm -rf "$TMPDIR"
  echo "Built swiftship CLI"
fi

echo ""
echo "Binaries ready at: $BIN_DIR"
echo "Add to PATH: export PATH=\"$BIN_DIR:\$PATH\""
