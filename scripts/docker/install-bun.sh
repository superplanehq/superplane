#!/bin/sh

set -e
set -o pipefail

BUN_VERSION="${BUN_VERSION:-1.4.2}"

echo "Installing Bun ${BUN_VERSION}"

apt-get update -y
apt-get install --no-install-recommends -y curl unzip ca-certificates

# Retry the installer too: it downloads the Bun binary in a separate curl call.
attempt=1
until curl --retry 5 --retry-delay 1 --retry-max-time 60 --retry-connrefused -fsSL https://bun.sh/install \
  | BUN_INSTALL=/usr/local bash -s "bun-v${BUN_VERSION}"; do
  if [ "$attempt" -ge 3 ]; then
    echo "Bun installation failed after ${attempt} attempts" >&2
    exit 1
  fi
  echo "Retrying Bun installation after failed attempt ${attempt}" >&2
  attempt=$((attempt + 1))
  sleep 5
done

apt-get clean
rm -rf /var/lib/apt/lists/* /var/cache/apt/archives/*

bun --version
