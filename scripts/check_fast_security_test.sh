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

import yaml


def cmd_list(node):
    if not isinstance(node, dict):
        return []
    commands = node.get("commands")
    if not isinstance(commands, list):
        return []
    return [str(cmd) for cmd in commands]


def is_hook(cmd):
    return "check_install_hooks.sh" in cmd


def is_setup(cmd):
    return cmd.startswith("make dev.setup")


def job_paths(data):
    global_cmds = cmd_list(((data.get("global_job_config") or {}).get("prologue") or {}))
    paths = []
    for block in data.get("blocks") or []:
        if not isinstance(block, dict):
            continue
        task = block.get("task") or {}
        prologue = cmd_list(task.get("prologue") or {})
        block_name = block.get("name") or "block"
        for job in task.get("jobs") or []:
            if not isinstance(job, dict):
                continue
            job_name = job.get("name") or "job"
            paths.append((f"{block_name}/{job_name}", global_cmds + prologue + cmd_list(job)))
    after = (data.get("after_pipeline") or {}).get("task") or {}
    after_prologue = cmd_list(after.get("prologue") or {})
    for job in after.get("jobs") or []:
        if not isinstance(job, dict):
            continue
        job_name = job.get("name") or "job"
        paths.append((f"after_pipeline/{job_name}", global_cmds + after_prologue + cmd_list(job)))
    return paths


data = yaml.safe_load(Path(".semaphore/semaphore.yml").read_text())
if not isinstance(data, dict):
    raise SystemExit("semaphore.yml did not parse as a mapping")

setup_jobs = 0
for label, commands in job_paths(data):
    seen_hook = False
    for cmd in commands:
        if is_hook(cmd):
            seen_hook = True
        if is_setup(cmd):
            setup_jobs += 1
            if not seen_hook:
                raise SystemExit(f"install-hook scan must run before {cmd} in {label}")

if setup_jobs == 0:
    raise SystemExit("make dev.setup missing from semaphore.yml")
print("PASS semaphore install-hook order")
PY

if [ "$failed" -ne 0 ]; then
	exit 1
fi

echo "All fast security tests passed."
