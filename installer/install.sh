#!/bin/sh
# agentbox installer. Downloads the `box` binary for this OS/arch and installs
# it. Usage:
#   curl -fsSL https://raw.githubusercontent.com/shariqnaiyer/agentbox/main/installer/install.sh | sh
set -eu

REPO="shariqnaiyer/agentbox"
BINARY="box"

os="$(uname -s | tr '[:upper:]' '[:lower:]')"
arch="$(uname -m)"
case "$arch" in
  x86_64|amd64) arch="amd64" ;;
  aarch64|arm64) arch="arm64" ;;
  *) echo "unsupported architecture: $arch" >&2; exit 1 ;;
esac
case "$os" in
  linux|darwin) ;;
  *) echo "agentbox v1 supports Linux and macOS only (got: $os)" >&2; exit 1 ;;
esac

need() { command -v "$1" >/dev/null 2>&1 || { echo "missing required tool: $1" >&2; exit 1; }; }
need curl
need tar

echo "Finding the latest agentbox release..."
tag="$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" \
  | grep '"tag_name"' | head -1 | cut -d'"' -f4)"
if [ -z "${tag:-}" ]; then
  echo "Could not find a release. Build from source instead: make build" >&2
  exit 1
fi

asset="box_${tag}_${os}_${arch}.tar.gz"
url="https://github.com/$REPO/releases/download/$tag/$asset"

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
echo "Downloading $asset ..."
curl -fsSL "$url" -o "$tmp/box.tgz"
tar -xzf "$tmp/box.tgz" -C "$tmp"

# Choose an install dir we can write to.
if [ -w /usr/local/bin ] || [ "$(id -u)" = "0" ]; then
  dest="/usr/local/bin"
else
  dest="$HOME/.local/bin"
  mkdir -p "$dest"
fi
install -m 0755 "$tmp/$BINARY" "$dest/$BINARY" 2>/dev/null || {
  echo "Installing to $dest needs sudo..."
  sudo install -m 0755 "$tmp/$BINARY" "$dest/$BINARY"
}

echo ""
echo "Installed $BINARY to $dest/$BINARY"
case ":$PATH:" in
  *":$dest:"*) ;;
  *) echo "NOTE: add $dest to your PATH." ;;
esac
echo ""
echo "Next:"
echo "  On your always-on box:   box host init"
echo "  On a client:             box pair <code>   then   box"
