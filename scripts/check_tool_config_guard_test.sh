#!/usr/bin/env bash
# Exercises check_tool_config_guard.sh against the real tree and temp fixtures.

set -euo pipefail

cd "$(dirname "$0")/.."
guard=./scripts/check_tool_config_guard.sh
failed=0

expect_pass() {
	local name="$1"
	shift
	if "$@" >/tmp/tool-config-guard-out.txt 2>/tmp/tool-config-guard-err.txt; then
		echo "PASS $name"
	else
		echo "FAIL $name (expected pass)" >&2
		cat /tmp/tool-config-guard-err.txt >&2
		failed=1
	fi
}

expect_fail() {
	local name="$1"
	shift
	if "$@" >/tmp/tool-config-guard-out.txt 2>/tmp/tool-config-guard-err.txt; then
		echo "FAIL $name (expected fail)" >&2
		failed=1
	else
		echo "PASS $name"
	fi
}

expect_pass "clean repo" "$guard" .

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT
mkdir -p "$tmp/web_src"

printf '%s\n' 'export default [];' >"$tmp/web_src/eslint.config.js"
expect_pass "minimal clean fixture" "$guard" "$tmp"

printf '%s\n' "import { createRequire } from 'node:module'" >"$tmp/web_src/eslint.config.js"
expect_fail "createRequire" "$guard" "$tmp"

printf '%s\n' 'const x = eval("1")' >"$tmp/web_src/eslint.config.js"
expect_fail "eval" "$guard" "$tmp"

printf '%s\n' "globalThis['eval']('1')" >"$tmp/web_src/eslint.config.js"
expect_fail "globalThis eval alias" "$guard" "$tmp"

printf '%s\n' "require('child' + '_process')" >"$tmp/web_src/eslint.config.js"
expect_fail "concat child_process" "$guard" "$tmp"

python3 -c "print('export default [];' + (' ' * 250) + 'hidden')" >"$tmp/web_src/eslint.config.js"
expect_fail "long whitespace run" "$guard" "$tmp"

python3 -c "print('x = ' + ('a' * 520))" >"$tmp/web_src/eslint.config.js"
expect_fail "long line" "$guard" "$tmp"

if [ "$failed" -ne 0 ]; then
	exit 1
fi

echo "All tool config guard tests passed."
