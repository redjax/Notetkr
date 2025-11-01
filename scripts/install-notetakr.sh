#!/usr/bin/env bash

set -uo pipefail

AUTO_MODE=false


if ! command -v curl &>/dev/null; then
  echo "curl is not installed."
  exit 1
fi

if ! command -v tar &>/dev/null; then
  echo "tar is not installed."
  exit 1
fi

## Parse arguments
for arg in "$@"; do
  if [[ "$arg" == "--auto" ]]; then
    AUTO_MODE=true
  fi
done

if command -v nt &>/dev/null; then
  if [[ "$AUTO_MODE" == true ]]; then
    CONFIRM="y"
  else
    read -p "Notetkr is already installed. Download anyway? (y/N) " CONFIRM
  fi

  if [[ "$CONFIRM" != "y" ]]; then
    echo "Aborting."
    exit 0
  fi
fi

## Detect OS
OS="$(uname -s)"
ARCH="$(uname -m)"

## Detect distro using /etc/os-release
if [ -f /etc/os-release ]; then
  . /etc/os-release
  DISTRO_ID=$ID
else
  echo "Cannot detect Linux distribution."
  exit 1
fi

## Get latest version from GitHub API
## For private repos, you need a GitHub token
if [[ -n "${GITHUB_TOKEN:-}" ]]; then
  NOTETKR_VERSION=$(curl -s -H "Authorization: token $GITHUB_TOKEN" https://api.github.com/repos/redjax/notetkr/releases/latest | grep -Po '"tag_name": "\K.*?(?=")')
else
  NOTETKR_VERSION=$(curl -s https://api.github.com/repos/redjax/notetkr/releases/latest | grep -Po '"tag_name": "\K.*?(?=")')
fi

if [[ -z "$NOTETKR_VERSION" ]]; then
  echo "Error: No releases found for notetkr."
  echo "If this is a private repository, set the GITHUB_TOKEN environment variable:"
  echo "  export GITHUB_TOKEN=your_github_token"
  echo "  ./scripts/install-notetakr.sh"
  echo ""
  echo "Or make the repository public, then create a release by running the Release workflow on GitHub."
  exit 1
fi

## Remove leading 'v' if present (e.g., 'v${NOTETKR_VERSION}' -> '${NOTETKR_VERSION}')
NOTETKR_VERSION="${NOTETKR_VERSION#v}"

echo "Installing notetkr v${NOTETKR_VERSION}"

## Normalize architecture names for your release naming
case "$ARCH" in
  x86_64)
    ARCH_NORM="x86_64"
    ;;
  aarch64|arm64)
    ARCH_NORM="arm64"
    ;;
  *)
    echo "Unsupported architecture: $ARCH"
    exit 1
    ;;
esac

## Map asset names (goreleaser default naming without version in filename)
case "$OS" in
  Linux)
    FILE="notetkr_Linux_${ARCH_NORM}.tar.gz"
    ;;
  Darwin)
    FILE="notetkr_Darwin_${ARCH_NORM}.tar.gz"
    ;;
  *)
    echo "Unsupported OS: $OS"
    exit 1
    ;;
esac

## Create a temporary directory
TMPDIR=$(mktemp -d)

## Download the release
ARCHIVE="$TMPDIR/notetkr.tar.gz"

## For private repos, need to use the GitHub API to get the asset download URL
if [[ -n "${GITHUB_TOKEN:-}" ]]; then
  echo "Fetching asset download URL for $FILE..."
  
  # Get the API url for the asset (NOT browser_download_url, which doesn't work with token auth)
  # We need to find the asset with matching name and extract its "url" field
  RELEASE_JSON=$(curl -s -H "Authorization: token $GITHUB_TOKEN" \
    "https://api.github.com/repos/redjax/notetkr/releases/tags/v${NOTETKR_VERSION}")
  
  # Extract the "url" field for the matching asset name
  # This looks for: "name": "$FILE", then finds the next "url": "..." field
  ASSET_URL=$(echo "$RELEASE_JSON" | grep -B 5 "\"name\": \"$FILE\"" | grep "\"url\":" | head -1 | sed -E 's/.*"url": "([^"]+)".*/\1/')

  if [[ -z "$ASSET_URL" ]]; then
    echo "Error: Could not find asset $FILE in release v${NOTETKR_VERSION}"
    exit 1
  fi

  echo "Downloading $FILE from GitHub API..."
  curl -sL -H "Authorization: token $GITHUB_TOKEN" \
    -H "Accept: application/octet-stream" \
    -o "$ARCHIVE" \
    "$ASSET_URL"
else
  # Public repo - use direct download URL
  URL="https://github.com/redjax/notetkr/releases/download/v${NOTETKR_VERSION}/$FILE"
  echo "Downloading $FILE from $URL"
  curl -L -o "$ARCHIVE" "$URL"
fi

## Extract the archive into the temp directory
tar -xzf "$ARCHIVE" -C "$TMPDIR"
if [ $? -ne 0 ]; then
  echo "Failed to extract $ARCHIVE to $TMPDIR"
  exit 1
fi

if [[ ! -d "$HOME/.local/bin" ]]; then
  mkdir -p "$HOME/.local/bin"
fi

if [ "$OS" = "Darwin" ]; then
  ## macOS: install to $HOME/.local/bin/
  install -m 755 "$TMPDIR/nt" $HOME/.local/bin/
else
  ## Linux: install to $HOME/.local/bin/
  install -m 755 "$TMPDIR/nt" $HOME/.local/bin/
fi

## Cleanup
rm -rf "$TMPDIR"

echo "notetkr installed successfully as 'nt'!"
echo "You may need to add this to your ~/.bashrc:"
echo "export PATH=\$PATH:\$HOME/.local/bin"

exit 0
