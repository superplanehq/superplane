#!/usr/bin/env bash
# Exercises the fast security scripts against the real tree and temp fixtures.

set -euo pipefail

cd "$(dirname "$0")/.."
failed=0

expect_pass() {
	local name="$1"
	shift
	if "$@" >/tmp/fast-security-out.txt 2>/tmp/fast-security-err.txt; then
		echo "PASS $name"
	else
		echo "FAIL $name (expected pass)" >&2
		cat /tmp/fast-security-err.txt >&2
		failed=1
	fi
}

expect_fail() {
	local name="$1"
	shift
	if "$@" >/tmp/fast-security-out.txt 2>/tmp/fast-security-err.txt; then
		echo "FAIL $name (expected fail)" >&2
		failed=1
	else
		echo "PASS $name"
	fi
}

expect_pass "clean repo" bash ./scripts/check_fast_security.sh .

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT
mkdir -p "$tmp/web_src"
printf '%s\n' 'export default [];' >"$tmp/web_src/eslint.config.js"
printf '%s\n' '{"scripts":{"dev":"vite"}}' >"$tmp/web_src/package.json"

expect_pass "clean fixture" bash ./scripts/check_fast_security.sh "$tmp"

printf '%s\n' "import { createRequire } from 'node:module'" >"$tmp/web_src/eslint.config.js"
expect_fail "tool config createRequire" bash ./scripts/check_fast_security.sh "$tmp"
printf '%s\n' 'export default [];' >"$tmp/web_src/eslint.config.js"

printf '%s\n' "campaign 8-$(printf '%s' 15418)" >"$tmp/web_src/note.txt"
expect_fail "ioc campaign tag" bash ./scripts/check_ioc_markers.sh "$tmp"
rm -f "$tmp/web_src/note.txt"

printf '%s\n' '{"scripts":{"postinstall":"node pwn.js"}}' >"$tmp/web_src/package.json"
expect_fail "postinstall hook" bash ./scripts/check_install_hooks.sh "$tmp"

python3 - <<'PY'
from pathlib import Path

commands = []
for line in Path(".semaphore/semaphore.yml").read_text().splitlines():
    stripped = line.strip()
    if stripped.startswith("- "):
        commands.append(stripped[2:])

hook = next((i for i, cmd in enumerate(commands) if "check_install_hooks.sh" in cmd), None)
setup = next((i for i, cmd in enumerate(commands) if cmd.startswith("make dev.setup")), None)
if hook is None:
    raise SystemExit("check_install_hooks.sh missing from semaphore.yml")
if setup is None:
    raise SystemExit("make dev.setup missing from semaphore.yml")
if hook > setup:
    raise SystemExit("install-hook scan must appear before make dev.setup")
print("PASS semaphore install-hook order")
PY

if [ "$failed" -ne 0 ]; then
	exit 1
fi

echo "All fast security tests passed."
