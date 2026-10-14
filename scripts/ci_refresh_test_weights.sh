#!/usr/bin/env bash
set -euo pipefail

if [[ "${SEMAPHORE_GIT_BRANCH:-}" != main ]]; then
  echo "Refresh task must run on main" >&2
  exit 1
fi

python3 scripts/ci_test_weights.py refresh --history ci-test-timings

if git diff --quiet -- scripts/ci_e2e_test_weights.tsv scripts/ci_unit_package_weights.tsv; then
  echo "Measured weights remain within the five-second and 20% change limits."
  exit 0
fi

python3 scripts/ci_open_test_weights_pr.py
