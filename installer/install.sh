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

# Choose an install dir. Prefer one that's already on PATH and writable (so the
# binary works immediately with no PATH edits) — e.g. Homebrew's bin. Fall back
# to /usr/local/bin via sudo, then to ~/.local/bin.
SUDO=""
dest=""
for d in "$HOME/.local/bin" /opt/homebrew/bin /usr/local/bin; do
  case ":$PATH:" in
    *":$d:"*)
      if [ -d "$d" ] && [ -w "$d" ]; then dest="$d"; break; fi
      ;;
  esac
done
if [ -z "$dest" ]; then
  if [ "$(id -u)" = "0" ] || [ -w /usr/local/bin ]; then
    dest="/usr/local/bin"
  elif command -v sudo >/dev/null 2>&1; then
    dest="/usr/local/bin"; SUDO="sudo"
  else
    dest="$HOME/.local/bin"; mkdir -p "$dest"
  fi
fi
$SUDO install -m 0755 "$tmp/$BINARY" "$dest/$BINARY"

echo ""
echo "Installed $BINARY to $dest/$BINARY"

# If the install dir isn't on PATH, persist it to the user's shell profile and
# print the line to use it in the current shell (a piped installer can't change
# the parent shell's environment).
case ":$PATH:" in
  *":$dest:"*)
    : # already on PATH, nothing to do
    ;;
  *)
    line="export PATH=\"$dest:\$PATH\""
    case "$(basename "${SHELL:-sh}")" in
      zsh)  rc="$HOME/.zshrc" ;;
      bash) rc="$HOME/.bashrc" ;;
      *)    rc="$HOME/.profile" ;;
    esac
    if [ -f "$rc" ] && grep -qF "$dest" "$rc" 2>/dev/null; then
      echo "$dest is already configured in $rc."
    else
      printf '\n# Added by agentbox installer\n%s\n' "$line" >> "$rc"
      echo "Added $dest to your PATH in $rc (effective in new shells)."
    fi
    echo "To use box in THIS shell now, run:"
    echo "  $line"
    ;;
esac

echo ""
echo "Next:"
echo "  On your always-on box:   box host init"
echo "  On a client:             box pair <code>   then   box"
