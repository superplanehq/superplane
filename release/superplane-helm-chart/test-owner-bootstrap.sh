#!/usr/bin/env bash

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
CHART_DIR="${REPO_ROOT}/release/superplane-helm-chart/helm"
TMP_DIR="$(mktemp -d)"
trap 'rm -rf "${TMP_DIR}"' EXIT

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

PASSWORD_B64="$(printf 'Sup3rSecret!' | base64)"

#
# Disabled bootstrap renders no password variable and no Secret.
#

disabled_manifest="${TMP_DIR}/disabled.yaml"
helm template superplane "${CHART_DIR}" "${common_args[@]}" >"${disabled_manifest}"

if grep -q 'SUPERPLANE_OWNER_PASSWORD' "${disabled_manifest}"; then
  echo "disabled owner bootstrap rendered SUPERPLANE_OWNER_PASSWORD" >&2
  exit 1
fi

if grep -q 'owner-bootstrap' "${disabled_manifest}"; then
  echo "disabled owner bootstrap rendered an owner bootstrap Secret" >&2
  exit 1
fi

#
# Enabled bootstrap stores the password in a Secret and references it from
# the API Deployment. The Deployment and Pod objects must not carry the
# password value.
#

enabled_manifest="${TMP_DIR}/enabled.yaml"
helm template superplane "${CHART_DIR}" \
  "${common_args[@]}" \
  --set api.ownerBootstrap.email=owner@example.com \
  --set api.ownerBootstrap.password="${PASSWORD_B64}" \
  --set api.ownerBootstrap.firstName=Owner \
  --set api.ownerBootstrap.lastName=User >"${enabled_manifest}"

grep -q 'name: superplane-owner-bootstrap' "${enabled_manifest}"
grep -q "SUPERPLANE_OWNER_PASSWORD: ${PASSWORD_B64}" "${enabled_manifest}"
grep -q 'secretKeyRef' "${enabled_manifest}"

api_deployment="${TMP_DIR}/api-deployment.yaml"
awk '/^kind: Deployment$/,/^---$/' "${enabled_manifest}" >"${api_deployment}"

if grep -q "${PASSWORD_B64}" "${api_deployment}"; then
  echo "API Deployment carries the owner password value" >&2
  exit 1
fi

if ! awk '/SUPERPLANE_OWNER_PASSWORD/,/key: SUPERPLANE_OWNER_PASSWORD/' "${api_deployment}" | grep -q 'name: superplane-owner-bootstrap'; then
  echo "API Deployment does not reference the owner bootstrap Secret" >&2
  exit 1
fi

#
# An existing Secret is referenced without rendering a new one.
#

existing_manifest="${TMP_DIR}/existing.yaml"
helm template superplane "${CHART_DIR}" \
  "${common_args[@]}" \
  --set api.ownerBootstrap.email=owner@example.com \
  --set api.ownerBootstrap.secretName=my-owner-secret >"${existing_manifest}"

if grep -q 'name: superplane-owner-bootstrap' "${existing_manifest}"; then
  echo "chart rendered an owner bootstrap Secret for an existing secretName" >&2
  exit 1
fi

grep -q 'name: my-owner-secret' "${existing_manifest}"
