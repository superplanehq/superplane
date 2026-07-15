#!/usr/bin/env bash
#
# Tests for scripts/go-mod-download. Mocks the `go` binary via a temp PATH shim
# so no network access is needed. Run: bash scripts/go-mod-download_test.sh
#
# Backoff delays are forced to 0 so the suite runs instantly.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TARGET="${SCRIPT_DIR}/go-mod-download"

fails=0

# make_go <fail_count> writes a `go` shim into a fresh temp dir that exits 1 for
# the first <fail_count> invocations and 0 afterwards. Echoes the temp dir.
make_go() {
  local fail_count="$1"
  local dir
  dir="$(mktemp -d)"
  cat >"${dir}/go" <<EOF
#!/usr/bin/env bash
# Only "go mod download" is counted/failed; the failure-path "go env" probe is
# a no-op here so it does not inflate the invocation count.
if [ "\$1" != "mod" ]; then
  exit 0
fi
count_file="${dir}/count"
n=\$(cat "\$count_file" 2>/dev/null || echo 0)
n=\$((n + 1))
echo "\$n" >"\$count_file"
if [ "\$n" -le "${fail_count}" ]; then
  echo "simulated transient failure #\$n" >&2
  exit 1
fi
exit 0
EOF
  chmod +x "${dir}/go"
  echo "$dir"
}

run() {
  local dir="$1"; shift
  env \
    PATH="${dir}:${PATH}" \
    GO_MOD_DOWNLOAD_INITIAL_DELAY=0 \
    GO_MOD_DOWNLOAD_MAX_DELAY=0 \
    "$@" bash "$TARGET"
}

assert() {
  local desc="$1" expected="$2" actual="$3"
  if [ "$expected" = "$actual" ]; then
    echo "ok - ${desc}"
  else
    echo "FAIL - ${desc}: expected '${expected}', got '${actual}'" >&2
    fails=$((fails + 1))
  fi
}

# 1) Succeeds on the first try -> exactly one `go` invocation.
dir="$(make_go 0)"
run "$dir" GO_MOD_DOWNLOAD_MAX_RETRIES=3 && rc=0 || rc=$?
assert "first-try success exits 0" 0 "$rc"
assert "first-try success calls go once" 1 "$(cat "${dir}/count")"
rm -rf "$dir"

# 2) Fails twice then succeeds within the retry budget.
dir="$(make_go 2)"
run "$dir" GO_MOD_DOWNLOAD_MAX_RETRIES=3 && rc=0 || rc=$?
assert "transient failure recovers exits 0" 0 "$rc"
assert "transient failure retries until success" 3 "$(cat "${dir}/count")"
rm -rf "$dir"

# 3) Exhausts retries -> non-zero exit after MAX_RETRIES attempts.
dir="$(make_go 99)"
run "$dir" GO_MOD_DOWNLOAD_MAX_RETRIES=3 && rc=0 || rc=$?
assert "exhausted retries exits non-zero" 1 "$rc"
assert "exhausted retries stops at MAX_RETRIES" 3 "$(cat "${dir}/count")"
rm -rf "$dir"

if [ "$fails" -ne 0 ]; then
  echo "${fails} test(s) failed" >&2
  exit 1
fi
echo "All tests passed"
