#!/usr/bin/env bash
#
# Static hunt for known PolinRider / BeaverTail markers.
# Build IoC strings at runtime so this script does not store them as literals.
# Do not contact C2 hosts. Do not execute matched files.

set -euo pipefail

cd "$(dirname "$0")/.."
root="${1:-.}"

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

echo "==> IoC marker scan"
echo "    Looks for known dropper campaign tags and C2 hostnames."

failed=0
checked=0
for marker in "${markers[@]}"; do
	checked=$((checked + 1))
	hits=$(mktemp)
	if grep -R -n -I "${exclude_dirs[@]}" -e "$marker" "$root" \
		--exclude='check_ioc_markers.sh' \
		--exclude='check_fast_security_test.sh' \
		>"$hits" 2>/dev/null; then
		echo "FAIL  marker $marker"
		sed 's/^/      /' "$hits"
		failed=1
	else
		echo "OK    no match for $marker"
	fi
	rm -f "$hits"
done

if [ "$failed" -ne 0 ]; then
	echo "==> IoC marker scan: FAIL"
	echo "    Do not run ESLint or Node on matched files."
	exit 1
fi

echo "==> IoC marker scan: PASS ($checked markers)"
