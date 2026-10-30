#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."

# Pin the release and digest so local checks and CI use the same scanner.
scanner_image="ghcr.io/google/osv-scanner:v2.7.0@sha256:4f3b7e2c64b9d38a5febc5a7e4a3dfef1d1362f726dea7a2ef4762b46469fcd0"
permitted_licenses=$(paste -sd, scripts/permitted-licenses.txt)

config_directory=$(mktemp -d)
trap 'rm -rf "$config_directory"' EXIT
cp osv-scanner.toml "$config_directory/osv-scanner.toml"

# Restrict this command to licenses. Keep vulnerability exceptions out of the
# shared config so future vulnerability scans can use it without this override.
cat >> "$config_directory/osv-scanner.toml" <<'TOML'

[[PackageOverrides]]
vulnerability.ignore = true
reason = "Vulnerability enforcement is outside this license check."
TOML

# Explicit lockfiles preserve the existing Go and frontend dependency scope.
# Optional lockfile arguments support focused checks and integration fixtures.
if [ "$#" -eq 0 ]; then
  set -- go.mod web_src/package-lock.json
fi
lockfiles=()
for lockfile in "$@"; do
  lockfiles+=(--lockfile="/src/${lockfile#"$PWD"/}")
done

status=0
echo "Checking dependency licenses only. Vulnerability findings are suppressed for this command."
docker run --rm \
  --mount "type=bind,source=$PWD,target=/src,readonly" \
  --mount "type=bind,source=$config_directory,target=/config,readonly" \
  --workdir /src \
  "$scanner_image" scan source \
  --config=/config/osv-scanner.toml \
  --licenses="$permitted_licenses" \
  "${lockfiles[@]}" || status=$?

if [ "$status" -ne 0 ]; then
  echo "License check failed (exit $status)." >&2
  echo "Review license violations before changing scripts/permitted-licenses.txt or osv-scanner.toml." >&2
fi
exit "$status"
