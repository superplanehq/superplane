#!/usr/bin/env bash

set -euo pipefail

if [ "$#" -ne 1 ]; then
  echo "Usage: release/verify-superplane-release-branch.sh <branch>" >&2
  exit 1
fi

BRANCH="$1"
if [ "${BRANCH}" = "main" ]; then
  exit 0
fi
if [[ "${BRANCH}" =~ ^release/v[0-9]+\.[0-9]+\.x$ ]]; then
  exit 0
fi

echo "Error: SuperPlane releases must originate from main or release/vMAJOR.MINOR.x" >&2
exit 1
