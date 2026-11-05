#!/bin/sh

set -e
set -o pipefail

BUN_VERSION="${BUN_VERSION:-1.4.2}"

echo "Installing Bun ${BUN_VERSION}"

apt-get update -y
apt-get install --no-install-recommends -y curl unzip ca-certificates

# The Bun installer downloads a release archive itself. Retry the whole
# installer when the install script or the archive download fails.
install_bun() {
  curl --retry 5 --retry-delay 5 --retry-max-time 120 --retry-all-errors -fsSL https://bun.sh/install \
    | BUN_INSTALL=/usr/local bash -s "bun-v${BUN_VERSION}"
}

attempt=1
until install_bun; do
  if [ "$attempt" -ge 5 ]; then
    echo "Bun install failed after ${attempt} attempts" >&2
    exit 1
  fi
  echo "Bun install attempt ${attempt} failed. Retrying." >&2
  attempt=$((attempt + 1))
  sleep 5
done

apt-get clean
rm -rf /var/lib/apt/lists/* /var/cache/apt/archives/*

bun --version
