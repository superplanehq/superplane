#!/bin/sh

set -e
set -o pipefail

echo "Installing Protobuf Compiler (protoc) 3.15.8"

ARCH=$(dpkg --print-architecture)

apt-get update -y
apt-get install --no-install-recommends -y unzip
apt-get clean
rm -rf /var/lib/apt/lists/* /var/cache/apt/archives/*

if [ "$ARCH" = "arm64" ]; then
  curl --retry 5 --retry-delay 1 --retry-max-time 60 --retry-connrefused -sL https://github.com/protocolbuffers/protobuf/releases/download/v3.15.8/protoc-3.15.8-linux-aarch_64.zip -o protoc.zip;
else
  curl --retry 5 --retry-delay 1 --retry-max-time 60 --retry-connrefused -sL https://github.com/protocolbuffers/protobuf/releases/download/v3.15.8/protoc-3.15.8-linux-x86_64.zip -o protoc.zip;
fi

unzip protoc.zip
mv bin/protoc /usr/local/bin/protoc
rm -rf protoc.zip bin include readme.txt