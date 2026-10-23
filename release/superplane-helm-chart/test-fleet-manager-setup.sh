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

email_manifest="${TMP_DIR}/api-email.yaml"
helm template superplane "${CHART_DIR}" \
  "${common_args[@]}" \
  --show-only templates/api.yaml \
  --set api.ownerSetupEnabled=false \
  --set email.secretName=test-email \
  --set email.fromName="Invitation Sender" \
  --set email.fromAddress=invites@example.com >"${email_manifest}"
grep -q 'name: test-email' "${email_manifest}"
grep -A1 'name: EMAIL_FROM_NAME' "${email_manifest}" | grep -q 'value: "Invitation Sender"'
grep -A1 'name: EMAIL_FROM_ADDRESS' "${email_manifest}" | grep -q 'value: "invites@example.com"'

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

billing_off="$(grep -A1 'name: SUPERPLANE_BILLING_ENABLED' "${manifest}" | grep -c 'value: "no"' || true)"
if [ "${billing_off}" -lt 3 ]; then
  echo "expected SUPERPLANE_BILLING_ENABLED=no on API, workers, and websocket" >&2
  exit 1
fi

polar_manifest="${TMP_DIR}/polar-billing.yaml"
helm template superplane "${CHART_DIR}" \
  "${common_args[@]}" \
  --set polar.enabled=true \
  --set polar.secretName=test-polar >"${polar_manifest}"
billing_on="$(grep -A1 'name: SUPERPLANE_BILLING_ENABLED' "${polar_manifest}" | grep -c 'value: "yes"' || true)"
if [ "${billing_on}" -lt 3 ]; then
  echo "expected SUPERPLANE_BILLING_ENABLED=yes when Polar is enabled" >&2
  exit 1
fi

forced_off="${TMP_DIR}/billing-off.yaml"
helm template superplane "${CHART_DIR}" \
  "${common_args[@]}" \
  --set polar.enabled=true \
  --set polar.secretName=test-polar \
  --set billing.enabled=no >"${forced_off}"
if grep -A1 'name: SUPERPLANE_BILLING_ENABLED' "${forced_off}" | grep -q 'value: "yes"'; then
  echo "operator billing.enabled=no must stay off when Polar is enabled" >&2
  exit 1
fi

forced_bool_off="${TMP_DIR}/billing-bool-off.yaml"
helm template superplane "${CHART_DIR}" \
  "${common_args[@]}" \
  --set polar.enabled=true \
  --set polar.secretName=test-polar \
  --set billing.enabled=false >"${forced_bool_off}"
if grep -A1 'name: SUPERPLANE_BILLING_ENABLED' "${forced_bool_off}" | grep -q 'value: "yes"'; then
  echo "operator billing.enabled=false must stay off when Polar is enabled" >&2
  exit 1
fi
bool_off="$(grep -A1 'name: SUPERPLANE_BILLING_ENABLED' "${forced_bool_off}" | grep -c 'value: "no"' || true)"
if [ "${bool_off}" -lt 3 ]; then
  echo "expected SUPERPLANE_BILLING_ENABLED=no when billing.enabled is boolean false" >&2
  exit 1
fi

runner_manifest="${TMP_DIR}/runner-billing.yaml"
helm template superplane "${CHART_DIR}" \
  "${common_args[@]}" \
  --set blobStorage.provider=gcs \
  --set runner.api.enabled=true >"${runner_manifest}"
runner_billing="$(grep -A1 'name: SUPERPLANE_BILLING_ENABLED' "${runner_manifest}" | grep -c 'value: "no"' || true)"
if [ "${runner_billing}" -lt 4 ]; then
  echo "expected SUPERPLANE_BILLING_ENABLED=no on the runner API" >&2
  exit 1
fi
