#!/usr/bin/env bash
# Deploy a personal Forge app against a local tunnel.
# Run this script in a shell where `forge whoami` works.
#
# Usage:
#   BASE_URL=https://<tunnel> SUPERPLANE_BITBUCKET_FORGE_APP_ID=ari:... bash scripts/forge-deploy-dev.sh
#   FORGE_ENV=production bash scripts/forge-deploy-dev.sh
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
tunnel_url="${SUPERPLANE_BITBUCKET_FORGE_REMOTE_URL:-${FORGE_TUNNEL_URL:-${BASE_URL:-}}}"
app_id="${FORGE_APP_ID:-${SUPERPLANE_BITBUCKET_FORGE_APP_ID:-}}"
forge_environment="${FORGE_ENV:-development}"

if [[ -z "${tunnel_url}" ]]; then
  echo "Set SUPERPLANE_BITBUCKET_FORGE_REMOTE_URL or BASE_URL to the personal tunnel URL." >&2
  exit 1
fi
if [[ -z "${app_id}" ]]; then
  echo "Set SUPERPLANE_BITBUCKET_FORGE_APP_ID to the personal Forge app ID." >&2
  exit 1
fi
if ! command -v forge >/dev/null 2>&1; then
  echo "Install the Forge CLI first: npm install -g @forge/cli" >&2
  exit 1
fi

export SUPERPLANE_BITBUCKET_FORGE_REMOTE_URL="${tunnel_url}"
export SUPERPLANE_BITBUCKET_FORGE_APP_ID="${app_id}"
cd "${repo_root}/forge/bitbucket"
if [[ ! -d "node_modules/@forge/api" ]]; then
  npm ci --no-audit --no-fund
fi

printf 'Deploying personal app %s (%s) -> %s\n' "${app_id}" "${forge_environment}" "${tunnel_url}"
forge lint
forge deploy --environment "${forge_environment}"
printf 'Next: forge install --environment %s, then open Apps -> SuperPlane in the test workspace.\n' "${forge_environment}"
