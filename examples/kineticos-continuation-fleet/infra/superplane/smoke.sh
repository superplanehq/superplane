#!/usr/bin/env bash
# Drive the KineticOS agent fleet end-to-end over HTTP, in the EXACT order the
# SuperPlane canvas does — a faithful offline rehearsal of a canvas run. Runs at
# zero credentials (every agent has a deterministic fallback).
#
#   infra/superplane/smoke.sh [APP_BASE_URL]   (default http://localhost:3000)
#
# Start the app first:  npm run dev
set -euo pipefail
APP="${1:-http://localhost:3000}"

post() { # post <path> <json-body>
  curl -fsS -X POST "$APP$1" -H 'content-type: application/json' -d "$2"
}
field() { python3 -c 'import sys,json;d=json.load(sys.stdin);print(d.get(sys.argv[1],""))' "$1"; }
show() { python3 -m json.tool; }

echo "▸ ingest a broken-gearbox CAD image"
JOB=$(post /api/jobs '{"cadImageUris":["s3://demo/gearbox-stage2-cad.png"],"telemetryUri":"s3://demo/gearbox-vibration.csv","assemblyContext":"gearbox stage 2 — driven idler","failureNote":"sheared tooth on idler gear"}' | field jobId)
[ -n "$JOB" ] || { echo "✗ no jobId — is the app running at $APP?"; exit 1; }
echo "  job = $JOB"
B="{\"job_id\":\"$JOB\"}"

echo "▸ perception fleet (2A–2F fan out, then fuse)"
for a in conditioning classification dimensioning material-infer telemetry; do
  printf '  %-16s ' "$a"; post "/api/agents/$a" "$B" | python3 -c 'import sys,json;print(json.load(sys.stdin))'
done
printf '  %-16s ' reconstruction; post /api/agents/reconstruction "$B" | python3 -c 'import sys,json;print(json.load(sys.stdin))'
echo "▸ assemble PerceptionResult"; post /api/agents/perception-assemble "$B" | show

echo "▸ Gate 2.G · composite confidence"
post /api/evaluate-gate "{\"job_id\":\"$JOB\",\"gate\":\"composite_confidence\"}" | show

echo "▸ design fleet (3A sourcing → else 3B generative)"
MATCHED=$(post /api/agents/sourcing "$B" | field matched)
echo "  sourcing matched = $MATCHED"
if [ "$MATCHED" != "True" ] && [ "$MATCHED" != "true" ]; then
  echo "  → 3B generative continuation CAD (B1–B8)"
  post /api/agents/generative-cad "$B" | show
fi

echo "▸ Gate 3.G · continuation strategy"
post /api/evaluate-gate "{\"job_id\":\"$JOB\",\"gate\":\"continuation_strategy\"}" | show

echo "▸ Phase 4 · printability adaptation"; post /api/stages/material "$B" | show
echo "▸ Gate 4.3 · printability"; post /api/evaluate-gate "{\"job_id\":\"$JOB\",\"gate\":\"printability\"}" | show

echo "▸ Phases 5–7 · provision validator + emit CAD + validate"; post /api/stages/fabricate-report "$B" | show
echo "▸ Gate 8.1 · output acceptance"; post /api/evaluate-gate "{\"job_id\":\"$JOB\",\"gate\":\"output_acceptance\"}" | show

echo "▸ 8.3 · seal run + resume production"; post /api/agents/finalize "$B" | show
echo "✅ fleet complete — production resumed on the v1 continuation (job $JOB)"
