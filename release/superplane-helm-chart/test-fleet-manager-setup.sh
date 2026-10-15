#!/usr/bin/env bash

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
CHART_DIR="${REPO_ROOT}/release/superplane-helm-chart/helm"
TMP_DIR="$(mktemp -d)"
trap 'rm -rf "${TMP_DIR}"' EXIT

manifest="${TMP_DIR}/manifest.yaml"
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
  --set fleetManagerSetup.managerId=test-cluster >"${manifest}"

grep -q 'name: FLEET_MANAGER_SETUP_CONFIG_PATH' "${manifest}"
grep -q 'name: superplane-fleet-manager-setup' "${manifest}"
grep -q 'name: RUNNER_SUPPORTED_VERSIONS' "${manifest}"
grep -q '"managerId":"test-cluster"' "${manifest}"
grep -q '"source":"manual"' "${manifest}"

if grep -Eq 'FLEET_MANAGER_TOKEN_PATH|fleet-manager-credential|superplane-fleet-manager-token|"credentials"' "${manifest}"; then
  echo "Fleet Manager setup rendered removed credential resources" >&2
  exit 1
fi

helm template superplane "${CHART_DIR}" \
  "${common_args[@]}" \
  --set fleetManagerSetup.enabled=false >"${disabled_manifest}"

if grep -Eq 'FLEET_MANAGER_SETUP_CONFIG_PATH|fleet-manager-setup' "${disabled_manifest}"; then
  echo "disabled Fleet Manager setup rendered metadata resources" >&2
  exit 1
fi

for provider in aws gcp azure; do
  provider_manifest="${TMP_DIR}/${provider}.yaml"
  helm template superplane "${CHART_DIR}" \
    "${common_args[@]}" \
    --set "fleetManagerSetup.provider=${provider}" \
    --set "fleetManagerSetup.infrastructure.testField=${provider}-value" >"${provider_manifest}"

  grep -q "\"provider\":\"${provider}\"" "${provider_manifest}"
  grep -q '"source":"terraform"' "${provider_manifest}"
  grep -q "\"testField\":\"${provider}-value\"" "${provider_manifest}"
done
