#!/usr/bin/env bash
set -euo pipefail

if [[ "${SEMAPHORE_GIT_BRANCH:-}" != main || "${SEMAPHORE_BLOCK_NAME:-}" != "Backend Tests" || ! -f junit-report.xml ]]; then
  exit 0
fi

case "${SEMAPHORE_JOB_NAME:-}" in
  "Unit Tests"*) kind=unit ;;
  "E2E Tests"*) kind=e2e ;;
  *) exit 0 ;;
esac

if [[ "${SEMAPHORE_JOB_RESULT:-}" != passed ]]; then
  exit 0
fi

path="ci-test-timings/${kind}/${SEMAPHORE_JOB_INDEX}-of-${SEMAPHORE_JOB_COUNT}.xml"
mkdir -p "$(dirname "$path")"
cp junit-report.xml "$path"
artifact push workflow "$path" --force
