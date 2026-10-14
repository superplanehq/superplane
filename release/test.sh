#!/usr/bin/env bash

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${REPO_ROOT}"

node --test release/component-release.test.js
bash release/component_release_artifact_test.sh
bash release/fetch_component_release_history_test.sh
bash release/runner/upload_test.sh
bash release/fleet-manager/publish_image_test.sh
bash release/fleet-manager/verify_image_test.sh

bash -n \
  release/generate-component-sbom.sh \
  release/component-release-artifact.sh \
  release/component_release_artifact_test.sh \
  release/fetch-component-release-history.sh \
  release/fetch_component_release_history_test.sh \
  release/verify-component-release-source.sh \
  release/verify-public-assets.sh \
  release/runner/build.sh \
  release/runner/upload.sh \
  release/fleet-manager/build.sh \
  release/fleet-manager/build-release.sh \
  release/fleet-manager/publish-image.sh \
  release/fleet-manager/publish_image_test.sh \
  release/fleet-manager/run-skopeo.sh \
  release/fleet-manager/verify-oci-archive.sh \
  release/fleet-manager/verify-image.sh \
  release/fleet-manager/verify_image_test.sh

node --check release/component-release.js
