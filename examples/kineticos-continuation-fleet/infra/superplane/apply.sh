#!/usr/bin/env bash
# Retarget the KineticOS fleet canvas at a KineticOS base URL and (optionally)
# load it into SuperPlane.
#
#   infra/superplane/apply.sh [BASE_URL]
#
#   BASE_URL   KineticOS base the http nodes call (default: the prod Render URL).
#   APPLY=1    also run `superplane canvas create -f` on the substituted file.
#
# Examples:
#   infra/superplane/apply.sh http://localhost:3000          # just substitute
#   APPLY=1 infra/superplane/apply.sh https://kos.example.com  # substitute + create
set -euo pipefail

DEFAULT_HOST="https://kineticos.onrender.com"
BASE_URL="${1:-$DEFAULT_HOST}"
BASE_URL="${BASE_URL%/}" # strip any trailing slash

SRC="$(cd "$(dirname "$0")" && pwd)/kineticos-fleet.canvas.yaml"
OUT="${TMPDIR:-/tmp}/kineticos-fleet.canvas.yaml"

if [[ ! -f "$SRC" ]]; then
  echo "✗ canvas not found: $SRC" >&2
  exit 1
fi

# Replace the baked-in host with the target base URL.
sed "s#${DEFAULT_HOST}#${BASE_URL}#g" "$SRC" > "$OUT"
COUNT="$(grep -c "${BASE_URL}/api/" "$OUT" || true)"
echo "✓ wrote $OUT"
echo "  → ${COUNT} http executor URLs now point at ${BASE_URL}"

if [[ "${APPLY:-0}" == "1" ]]; then
  if ! command -v superplane >/dev/null 2>&1; then
    echo "✗ APPLY=1 but the 'superplane' CLI is not installed / on PATH." >&2
    echo "  Install it, authenticate to your org, then re-run — or import $OUT in the UI." >&2
    exit 1
  fi
  echo "→ superplane canvas create -f $OUT"
  superplane canvas create -f "$OUT"
  echo "✓ canvas created. Bind the ingest trigger to a Webhook event source,"
  echo "  then set SP_INGEST_WEBHOOK + SP_WEBHOOK_TOKEN in KineticOS."
else
  echo
  echo "Next:"
  echo "  • CLI:  superplane canvas create -f $OUT"
  echo "  • UI:   New canvas → Import YAML → paste $OUT"
  echo "  • Then bind the ingest trigger to a Webhook event source and set"
  echo "    SP_INGEST_WEBHOOK + SP_WEBHOOK_TOKEN in KineticOS (see README.md)."
fi
