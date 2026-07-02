#!/usr/bin/env sh
set -eu

INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/bin}"
DRY_RUN="${SUPERPLANE_CLI_INSTALL_DRY_RUN:-0}"

require_command() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "Missing required command: $1" >&2
    exit 1
  fi
}

require_command uname
require_command curl
require_command mktemp
require_command chmod
require_command mv

os="$(uname -s)"
arch="$(uname -m)"

case "$os" in
  Darwin) os="darwin" ;;
  Linux) os="linux" ;;
  *)
    echo "Unsupported OS: $os" >&2
    exit 1
    ;;
esac

case "$arch" in
  x86_64 | amd64) arch="amd64" ;;
  arm64 | aarch64) arch="arm64" ;;
  *)
    echo "Unsupported architecture: $arch" >&2
    exit 1
    ;;
esac

platform="${os}-${arch}"
url="https://install.superplane.com/superplane-cli-${platform}"
target="${INSTALL_DIR}/superplane"
tmp_file=""

if [ "$DRY_RUN" = "1" ]; then
  echo "platform=${platform}"
  echo "url=${url}"
  echo "target=${target}"
  exit 0
fi

mkdir -p "$INSTALL_DIR"

tmp_file="$(mktemp "${INSTALL_DIR}/.superplane.tmp.XXXXXX")"
cleanup() {
  if [ -n "$tmp_file" ] && [ -f "$tmp_file" ]; then
    rm -f "$tmp_file"
  fi
}
trap cleanup EXIT INT TERM HUP

curl -fsSL "$url" -o "$tmp_file"
chmod +x "$tmp_file"
mv "$tmp_file" "$target"
tmp_file=""
trap - EXIT INT TERM HUP

case ":$PATH:" in
  *":$INSTALL_DIR:"*) ;;
  *)
    echo "Warning: ${INSTALL_DIR} is not in PATH." >&2
    echo "Add to PATH: export PATH=\"${INSTALL_DIR}:\$PATH\"" >&2
    ;;
esac

echo "Installed SuperPlane CLI to ${target}"
