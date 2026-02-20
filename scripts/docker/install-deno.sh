#!/bin/sh

set -e
set -o pipefail

DENO_VERSION="$1"

apt-get update -y
apt-get install --no-install-recommends -y ca-certificates curl unzip

if [ -n "$DENO_VERSION" ]; then
  echo "Installing Deno version $DENO_VERSION"
  curl --retry 5 --retry-delay 1 --retry-max-time 60 --retry-connrefused -fsSL https://deno.land/install.sh | DENO_INSTALL=/usr/local sh -s "$DENO_VERSION"
else
  echo "Installing latest Deno version"
  curl --retry 5 --retry-delay 1 --retry-max-time 60 --retry-connrefused -fsSL https://deno.land/install.sh | DENO_INSTALL=/usr/local sh
fi

apt-get clean && rm -f /var/lib/apt/lists/*_*

deno --version
