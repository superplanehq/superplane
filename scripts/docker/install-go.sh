#!/bin/sh

set -e
set -o pipefail

GO_VERSION=$1
ARCH=$(dpkg --print-architecture)

echo "Installing Go version: ${GO_VERSION} for architecture: ${ARCH}"

apt-get update
apt-get install -y wget ca-certificates

if [ "$ARCH" = "arm64" ]; then
  wget https://go.dev/dl/go${GO_VERSION}.linux-arm64.tar.gz -O go.tar.gz;
else
  wget https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz -O go.tar.gz;
fi

tar -C /usr/local -xzf go.tar.gz
rm go.tar.gz