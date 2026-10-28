#!/usr/bin/env bash
# Integration checks against the pinned OSV-Scanner and its real license data.
set -euo pipefail

cd "$(dirname "$0")/.."
mkdir -p tmp
fixture=$(mktemp -d "$PWD/tmp/license-check.XXXXXX")
trap 'rm -rf "$fixture"' EXIT

write_lockfile() {
  local directory="$fixture/$1"
  mkdir -p "$directory"
  cat > "$directory/package-lock.json" <<EOF
{"name":"license-check-fixture","lockfileVersion":3,"packages":{"":{"name":"license-check-fixture"},"node_modules/$2":{"version":"$3"}}}
EOF
}

expect_status() {
  local name="$1" expected="$2" lockfile="$3" status=0
  bash scripts/license-check.sh "$lockfile" > "$fixture/output" 2>&1 || status=$?
  if [ "$status" -ne "$expected" ]; then
    echo "FAIL $name: expected exit $expected, got $status" >&2
    cat "$fixture/output" >&2
    exit 1
  fi
  echo "PASS $name"
}

# Lodash 4.17.20 has known vulnerabilities, but its MIT license is permitted.
write_lockfile vulnerable lodash 4.17.20
expect_status "vulnerabilities do not fail the license check" 0 "$fixture/vulnerable/package-lock.json"

write_lockfile forbidden jschardet 3.1.4
expect_status "LGPL license fails the license check" 1 "$fixture/forbidden/package-lock.json"
if ! grep -q 'LGPL' "$fixture/output"; then
  cat "$fixture/output" >&2
  exit 1
fi

write_lockfile unknown superplane-license-check-nonexistent 0.0.0
expect_status "unknown license fails the license check" 1 "$fixture/unknown/package-lock.json"
if ! grep -q 'UNKNOWN' "$fixture/output"; then
  cat "$fixture/output" >&2
  exit 1
fi

write_lockfile override khroma 2.0.2
expect_status "package license override applies to nested lockfiles" 0 "$fixture/override/package-lock.json"

expect_status "missing lockfile fails the scan" 127 "$fixture/missing/package-lock.json"
