#!/usr/bin/env bash
#
# Static hunt for known PolinRider / BeaverTail markers.
# Build IoC strings at runtime so this script does not store them as literals.
# Do not contact C2 hosts. Do not execute matched files.

set -euo pipefail

cd "$(dirname "$0")/.."
root="${1:-.}"

# Campaign tags from public PolinRider notes (split so a self-scan stays clean).
markers=()
markers+=("8-$(printf '%s' 15418)")
markers+=("8-$(printf '%s' 10495)")
markers+=("8-$(printf '%s' 1638)")
markers+=("$(printf '%s' 33ff3edaf55a8e03)dcbc7cb40d498a49")
markers+=("eth.$(printf '%s' drpc).org")
markers+=("temp_auto_push.bat")
markers+=("global.o=")

exclude_dirs=(
	--exclude-dir=.git
	--exclude-dir=node_modules
	--exclude-dir=tmp
	--exclude-dir=dist
	--exclude-dir=dist-ssr
	--exclude-dir=storybook-static
	--exclude-dir=.cursor
	--exclude-dir=canvases
)

failed=0
for marker in "${markers[@]}"; do
	if grep -R -n -I "${exclude_dirs[@]}" -e "$marker" "$root" \
		--exclude='check_ioc_markers.sh' \
		--exclude='check_fast_security_test.sh' \
		>/tmp/ioc-marker-hits.txt 2>/dev/null; then
		echo "IoC marker found: $marker" >&2
		cat /tmp/ioc-marker-hits.txt >&2
		failed=1
	fi
done

if [ "$failed" -ne 0 ]; then
	echo "IoC marker scan failed. Do not run ESLint or Node on matched files." >&2
	exit 1
fi

echo "IoC marker scan passed."
