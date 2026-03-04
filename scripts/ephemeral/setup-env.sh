#!/usr/bin/env bash

set -euo pipefail

base_url="$1"

if [ -z "${base_url}" ]; then
  echo "base-url is required" >&2
  exit 1
fi

echo "BASE_URL=${base_url}" >> .env
echo "WEBHOOKS_BASE_URL=${base_url}" >> .env