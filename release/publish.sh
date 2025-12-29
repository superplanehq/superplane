#!/bin/bash

set -euo pipefail
IFS=$'\n\t'

if [ "${1-}" = "" ]; then
  echo "Usage: release/demo/build.sh <version>"
  exit 1
fi

VERSION="$1"

echo "Publishing superplane-demo image"
bash release/demo/build-and-push-image.sh $VERSION

echo "Publishing superplane image"
bash release/single-host/build-and-push-image.sh $VERSION

echo "Building superplane-single-host.tar.gz"
bash release/single-host/build-tar-gz.sh $VERSION

echo "Publishing GitHub Release"
node release/make-github-release.js $VERSION
