#!/bin/sh

set -e
set -o pipefail

echo "Installing Protobuf Compiler (protoc) 3.15.8"

ARCH=$(dpkg --print-architecture)

if [ "$ARCH" = "arm64" ]; then
  curl -sL https://github.com/protocolbuffers/protobuf/releases/download/v3.15.8/protoc-3.15.8-linux-aarch_64.zip -o protoc.zip;
else
  curl -sL https://github.com/protocolbuffers/protobuf/releases/download/v3.15.8/protoc-3.15.8-linux-x86_64.zip -o protoc.zip;
fi

unzip protoc.zip
mv bin/protoc /usr/local/bin/protoc
rm -rf protoc.zip