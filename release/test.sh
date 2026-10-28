#!/usr/bin/env bash

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${REPO_ROOT}"

node --test release/component-release.test.js
bash release/component_release_artifact_test.sh
bash release/fetch_component_release_history_test.sh
bash release/verify_component_release_source_test.sh
bash release/runner/upload_test.sh
bash release/fleet-manager/publish_image_test.sh
bash release/fleet-manager/skopeo_image_pin_test.sh
bash release/fleet-manager/verify_image_test.sh
bash release/fleet-manager/publish_chart_test.sh
bash release/fleet-manager-chart/test-chart.sh
bash release/superplane-helm-chart/test-fleet-manager-setup.sh
bash release/superplane-helm-chart/update_component_release_defaults_test.sh
bash release/superplane/publish_architecture_test.sh
bash release/superplane/publish_chart_test.sh
bash release/superplane/finalize_assets_test.sh
bash release/cli/pipeline_test.sh
bash release/superplane/pipeline_test.sh
bash release/runner/pipeline_test.sh
bash release/fleet-manager/pipeline_test.sh
bash release/superplane-single-host-tarball/test.sh
bash release/superplane/verify_prepared_archive_test.sh
bash release/superplane/verify_image_test.sh

bash -n \
  release/generate-component-sbom.sh \
  release/component-release-artifact.sh \
  release/component_release_artifact_test.sh \
  release/fetch-component-release-history.sh \
  release/fetch_component_release_history_test.sh \
  release/verify-component-release-source.sh \
  release/verify-superplane-release-branch.sh \
  release/verify_component_release_source_test.sh \
  release/verify-public-assets.sh \
  release/runner/build.sh \
  release/runner/upload.sh \
  release/fleet-manager/build.sh \
  release/fleet-manager/build-release.sh \
  release/fleet-manager/publish-image.sh \
  release/fleet-manager/publish-chart.sh \
  release/fleet-manager/publish_chart_test.sh \
  release/fleet-manager/publish_image_test.sh \
  release/fleet-manager/skopeo_image_pin_test.sh \
  release/fleet-manager/run-skopeo.sh \
  release/fleet-manager/verify-oci-archive.sh \
  release/fleet-manager/verify-image.sh \
  release/fleet-manager/verify_image_test.sh \
  release/superplane-helm-chart/build.sh \
  release/superplane-helm-chart/test-fleet-manager-setup.sh \
  release/superplane-helm-chart/update_component_release_defaults_test.sh \
  release/cli/build.sh \
  release/cli/pipeline_test.sh \
  release/superplane/build-assets.sh \
  release/superplane/build-finalized-assets.sh \
  release/superplane/build-image.sh \
  release/superplane/finalize_assets_test.sh \
  release/superplane/publish-architecture.sh \
  release/superplane/publish-chart.sh \
  release/superplane/publish-manifest.sh \
  release/superplane/publish_architecture_test.sh \
  release/superplane/publish_chart_test.sh \
  release/superplane/pipeline_test.sh \
  release/superplane/run-skopeo.sh \
  release/superplane/verify-image.sh \
  release/superplane/verify-oci-archive.sh \
  release/superplane/verify-prepared-archive.sh \
  release/superplane/verify_image_test.sh \
  release/superplane/verify_prepared_archive_test.sh \
  release/superplane-single-host-tarball/build.sh \
  release/superplane-single-host-tarball/test.sh \
  release/superplane-single-host-tarball/smoke-test.sh \
  release/superplane-single-host-tarball/templates/install.sh \
  release/superplane-single-host-tarball/templates/uninstall.sh \
  release/fleet-manager/docker-entrypoint.sh \
  release/fleet-manager/pipeline_test.sh \
  release/runner/publish-image.sh \
  release/runner/pipeline_test.sh

node --check release/component-release.js
node --check release/superplane-helm-chart/update-component-release-defaults.js
