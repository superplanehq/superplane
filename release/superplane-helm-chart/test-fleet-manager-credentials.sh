#!/usr/bin/env bash

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
CHART_DIR="${REPO_ROOT}/release/superplane-helm-chart/helm"
TMP_DIR="$(mktemp -d)"
trap 'rm -rf "${TMP_DIR}"' EXIT

default_manifest="${TMP_DIR}/default.yaml"
external_manifest="${TMP_DIR}/external.yaml"
disabled_manifest="${TMP_DIR}/disabled.yaml"
common_args=(
  --namespace superplane
  --set domain.name=superplane.example.com
  --set ingress.enabled=false
  --set authentication.secretName=test-authentication
  --set encryption.secretName=test-encryption
  --set jwt.secretName=test-jwt
  --set oidc.secretName=test-oidc
  --set session.secretName=test-session
  --set telemetry.secretName=test-telemetry
)

helm template superplane "${CHART_DIR}" \
  "${common_args[@]}" \
  --set fleetManagerSetup.managerId=test-cluster >"${default_manifest}"

grep -q 'kind: Job' "${default_manifest}"
grep -q 'name: superplane-fleet-manager-credential' "${default_manifest}"
grep -q 'name: FLEET_MANAGER_TOKEN_PATH' "${default_manifest}"
grep -q 'name: FLEET_MANAGER_SETUP_CONFIG_PATH' "${default_manifest}"
grep -q 'secretName: superplane-fleet-manager-token' "${default_manifest}"
grep -q 'name: superplane-fleet-manager-setup' "${default_manifest}"
grep -q '"existingSecret":"superplane-fleet-manager-token"' "${default_manifest}"
grep -q '"managerId":"test-cluster"' "${default_manifest}"
grep -q '"source":"manual"' "${default_manifest}"
if grep -q 'fleet-manager-service-token' "${default_manifest}"; then
  echo "rendered manifest contains a credential value" >&2
  exit 1
fi

helm template superplane "${CHART_DIR}" \
  "${common_args[@]}" \
  --set fleetManagerSetup.credentials.provision=false \
  --set fleetManagerSetup.credentials.existingSecret=externally-managed >"${external_manifest}"

if grep -q 'name: superplane-fleet-manager-credential' "${external_manifest}"; then
  echo "external credentials must disable the provisioning hook" >&2
  exit 1
fi
grep -q 'secretName: externally-managed' "${external_manifest}"

helm template superplane "${CHART_DIR}" \
  "${common_args[@]}" \
  --set fleetManagerSetup.enabled=false >"${disabled_manifest}"

if grep -Eq 'FLEET_MANAGER_(TOKEN|SETUP_CONFIG)_PATH|fleet-manager-(credential|setup|token)' "${disabled_manifest}"; then
  echo "disabled Fleet Manager setup rendered credential resources" >&2
  exit 1
fi
