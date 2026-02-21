#!/bin/bash
# Apple Developer Toolkit - Auto-install binaries via Homebrew
set -e

# Check if Homebrew is available
if ! command -v brew &>/dev/null; then
  echo "Homebrew required. Install: https://brew.sh"
  exit 1
fi

# Add tap if not already added
if ! brew tap | grep -q "abdullah4ai/tap"; then
  echo "Adding tap..."
  brew tap Abdullah4AI/tap
fi

# Install appstore CLI
if ! command -v appstore &>/dev/null; then
  echo "Installing appstore CLI..."
  brew install Abdullah4AI/tap/appstore
else
  echo "appstore CLI already installed: $(which appstore)"
fi

# Install swiftship CLI
if ! command -v swiftship &>/dev/null; then
  echo "Installing swiftship CLI..."
  brew install Abdullah4AI/tap/swiftship
else
  echo "swiftship CLI already installed: $(which swiftship)"
fi

echo ""
echo "Ready. Run 'appstore --help' or 'swiftship --help' to get started."
