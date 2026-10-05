#!/bin/sh

set -eu

BUN_VERSION="${BUN_VERSION:-1.4.2}"

echo "Installing Bun ${BUN_VERSION}"

apt-get update -y
apt-get install --no-install-recommends -y curl unzip ca-certificates

arch=$(uname -m)
case "$arch" in
  x86_64)
    target=linux-x64
    if ! grep -q avx2 /proc/cpuinfo; then
      target=linux-x64-baseline
    fi
    ;;
  aarch64 | arm64)
    target=linux-aarch64
    ;;
  *)
    echo "Unsupported architecture: ${arch}" >&2
    exit 1
    ;;
esac

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

url="https://github.com/oven-sh/bun/releases/download/bun-v${BUN_VERSION}/bun-${target}.zip"

curl --retry 5 --retry-all-errors --retry-delay 2 --retry-max-time 180 \
  --connect-timeout 20 --max-time 180 \
  -fsSL -o "$tmp/bun.zip" "$url"

unzip -q "$tmp/bun.zip" -d "$tmp"
install -m 755 "$tmp/bun-${target}/bun" /usr/local/bin/bun

apt-get clean
rm -rf /var/lib/apt/lists/* /var/cache/apt/archives/*

bun --version
