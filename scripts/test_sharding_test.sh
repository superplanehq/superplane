#!/usr/bin/env bash
# Exercise the shard launchers at their command boundary. A failed discovery
# command must never turn a backend test job green.
set -euo pipefail

repo_root="$(cd "$(dirname "$0")/.." && pwd)"
fixture="$(mktemp -d)"
trap 'rm -rf "$fixture"' EXIT
mkdir -p "$fixture/bin" "$fixture/repo/scripts/lib" "$fixture/repo/pkg/example"
cp "$repo_root/scripts/test_unit_autoparallel.sh" "$fixture/repo/scripts/"
cp "$repo_root/scripts/lib/shard_args.sh" "$fixture/repo/scripts/lib/"
cat > "$fixture/bin/go" <<'MOCK'
#!/usr/bin/env bash
if [[ "$*" == "list -m" ]]; then
  echo example.com/project
  exit 0
fi
if [[ "$DISCOVERY" == failure ]]; then
  # Partial output must not hide a discovery failure either.
  echo example.com/project/pkg/example
  echo 'package discovery failed' >&2
  exit 128
fi
if [[ "$DISCOVERY" == success ]]; then
  case "$1" in
    list) echo example.com/project/pkg/example ;;
    tool) echo 'total: 100.0%' ;;
  esac
fi
MOCK
cat > "$fixture/bin/gotestsum" <<'MOCK'
#!/usr/bin/env bash
touch "$TESTS_STARTED"
case "${REPORT:-missing}" in
  empty) echo '<testsuites><testsuite tests="0"/></testsuites>' > junit-report.xml ;;
  populated) echo '<testsuites><testsuite tests="1"><testcase name="TestExample"/></testsuite></testsuites>' > junit-report.xml ;;
esac
MOCK
chmod +x "$fixture/bin/"*
export PATH="$fixture/bin:$PATH" TESTS_STARTED="$fixture/tests-started"
cd "$fixture/repo"
for DISCOVERY in failure empty; do
  export DISCOVERY
  if SHARD_INDEX=1 SHARD_COUNT=4 bash scripts/test_unit_autoparallel.sh >"$fixture/output" 2>&1; then
    cat "$fixture/output"
    echo "Expected $DISCOVERY discovery to fail the shard" >&2
    exit 1
  fi
  if [[ -e "$TESTS_STARTED" ]]; then
    cat "$fixture/output"
    echo 'Tests started after failed package discovery' >&2
    exit 1
  fi
done
export DISCOVERY=success
for REPORT in missing empty; do
  export REPORT
  # An old report must not let a run with no test cases succeed.
  echo '<testcase name="TestStale"/>' > junit-report.xml
  if SHARD_INDEX=1 SHARD_COUNT=1 bash scripts/test_unit_autoparallel.sh >"$fixture/output" 2>&1; then
    echo "Expected $REPORT test report to fail the shard" >&2
    exit 1
  fi
done
REPORT=populated SHARD_INDEX=1 SHARD_COUNT=1 bash scripts/test_unit_autoparallel.sh >"$fixture/output" 2>&1
echo 'Unit shard discovery tests passed.'

cp "$repo_root/scripts/test_e2e_autoparallel.sh" scripts/
mkdir -p test/e2e
printf '%s\n' TestHeavy TestNew TestA TestB TestC TestD TestE TestF TestG TestH | sort > "$fixture/expected"
while read -r name; do
  printf 'func %s(t *testing.T) {}\n' "$name" >> test/e2e/example_test.go
done < "$fixture/expected"
printf 'TestHeavy\t120\nTestRemoved\t999\n' > scripts/e2e_test_durations.tsv
for name in TestA TestB TestC TestD TestE TestF TestG TestH; do
  printf '%s\t10\n' "$name" >> scripts/e2e_test_durations.tsv
done
cat > "$fixture/bin/gotestsum" <<'MOCK'
#!/usr/bin/env bash
while [[ "$1" != -run ]]; do shift; done
printf '%s\n' "$2" >> "$SHARD_SELECTIONS"
MOCK
export SHARD_SELECTIONS="$fixture/selections"
for SHARD_INDEX in {1..8}; do
  SHARD_INDEX="$SHARD_INDEX" SHARD_COUNT=8 bash scripts/test_e2e_autoparallel.sh > /dev/null
done
# Every discovered test, including one without a timing, runs exactly once.
# An obsolete timing must not introduce a nonexistent test.
tr -d '^()$' < "$SHARD_SELECTIONS" | tr '|' '\n' | sort > "$fixture/actual"
cmp "$fixture/expected" "$fixture/actual"
# The slowest test fills its own shard instead of sharing one by name order.
grep -Fxq '^(TestHeavy)$' "$SHARD_SELECTIONS"
echo 'E2E shard balancing tests passed.'
