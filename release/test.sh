#!/usr/bin/env bash

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${REPO_ROOT}"

node --test release/component-release.test.js
bash release/runner/upload_test.sh
bash release/fleet-manager/publish_image_test.sh

bash -n \
  release/generate-component-sbom.sh \
  release/verify-component-release-source.sh \
  release/verify-public-assets.sh \
  release/runner/build.sh \
  release/runner/upload.sh \
  release/fleet-manager/build.sh \
  release/fleet-manager/build-release.sh \
  release/fleet-manager/publish-image.sh \
  release/fleet-manager/publish_image_test.sh \
  release/fleet-manager/verify-oci-archive.sh \
  release/fleet-manager/verify-image.sh

node --check release/component-release.js
node --check release/github-app-token.js
