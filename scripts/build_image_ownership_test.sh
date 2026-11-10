#!/usr/bin/env bash
# Use the container's native filesystem. Docker Desktop bind mounts conceal
# the root/non-root ownership mismatch that occurs on Linux CI hosts.
set -euo pipefail

repo_root="$(cd "$(dirname "$0")/.." && pwd)"
fixture="$(mktemp -d /var/tmp/image-ownership.XXXXXX)"
trap 'rm -rf "$fixture"' EXIT
chown nobody:nogroup "$fixture"

for expected_status in 0 7; do
  output="$fixture/output-$expected_status"
  install -d -o nobody -g nogroup "$output"
  status=0
  bash "$repo_root/scripts/with-host-owned-output" "$output" bash -c '
    mkdir -p "$1/app/build" "$1/assets"
    printf binary > "$1/app/build/superplane"
    printf frontend > "$1/assets/index.html"
    exit "$2"
  ' _ "$output" "$expected_status" || status=$?
  test "$status" -eq "$expected_status"

  # A non-root caller must be able to extend, replace, and remove the output,
  # including partial files left by a failed build.
  runuser -u nobody -- bash -ec '
    mkdir "$1/app/db"
    printf replacement > "$1/app/build/superplane"
    rm -rf "$1"
  ' _ "$output"
done

echo 'Build output ownership tests passed.'
