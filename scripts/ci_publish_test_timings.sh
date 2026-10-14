#!/usr/bin/env bash
set -euo pipefail

if [[ "${SEMAPHORE_GIT_BRANCH:-}" != main || "${SEMAPHORE_PIPELINE_RESULT:-}" != passed ]]; then
  exit 0
fi

if ! artifact pull workflow ci-test-timings; then
  echo "No backend test reports in this workflow."
  exit 0
fi
python3 scripts/ci_test_weights.py collect \
  --reports ci-test-timings \
  --output "ci-test-timings/${SEMAPHORE_PIPELINE_CREATED_AT}-${SEMAPHORE_WORKFLOW_ID}.json" \
  --sha "$SEMAPHORE_GIT_SHA"
artifact push project "ci-test-timings/${SEMAPHORE_PIPELINE_CREATED_AT}-${SEMAPHORE_WORKFLOW_ID}.json" --force
