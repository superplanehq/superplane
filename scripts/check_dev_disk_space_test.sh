#!/usr/bin/env bash
# Exercise disk recovery without deleting real Docker resources.
set -euo pipefail

repo_root="$(cd "$(dirname "$0")/.." && pwd)"
fixture="$(mktemp -d)"
trap 'rm -rf "$fixture"' EXIT
mkdir "$fixture/bin"

# Reject volume operations and any cleanup other than build-cache pruning.
cat >"$fixture/bin/docker" <<'DOCKER'
#!/usr/bin/env bash
set -euo pipefail
for argument in "$@"; do
  case "$argument" in
    -v*|--volume*|--mount*) echo 'Unexpected volume access' >&2; exit 90 ;;
  esac
done
case "$1" in
  run)
    error="${BEFORE_ERROR:-}"
    [ ! -f "$STATE/pruned" ] || error="${AFTER_ERROR:-}"
    if [ -n "$error" ]; then
      echo "$error" >&2
      exit 1
    fi
    [ "${PULL_PROGRESS:-0}" -eq 0 ] || echo 'Pull complete' >&2
    available="$BEFORE_KB"
    [ ! -f "$STATE/pruned" ] || available="$AFTER_KB"
    printf 'Filesystem 1024-blocks Used Available Capacity Mounted on\n'
    printf '/dev/vda1 60000000 40000000 %s 70%% /\n' "$available"
    ;;
  context)
    [ "$2" = show ] || exit 90
    echo disk-test-context
    ;;
  buildx)
    [ "$2" = prune ] || exit 90
    # A separately selected builder must not redirect recovery to another disk.
    case " $* " in *' --builder disk-test-context '*) ;; *) exit 90 ;; esac
    echo prune >> "$STATE/prune_attempts"
    [ "${PRUNE_FAIL:-0}" -eq 0 ] || exit 1
    touch "$STATE/pruned"
    ;;
  compose)
    touch "$STATE/started"
    ;;
  *) echo "Unexpected Docker operation: $*" >&2; exit 90 ;;
esac
DOCKER
chmod +x "$fixture/bin/docker"
export PATH="$fixture/bin:$PATH" STATE="$fixture" BUILDX_BUILDER=unrelated-builder

run_case() {
  local name="$1" expected_status="$2" expected_cleanup="$3"
  shift 3
  rm -f "$STATE/pruned" "$STATE/started" "$STATE/prune_attempts"
  local status=0
  env "$@" bash "$repo_root/scripts/check_dev_disk_space.sh" >"$STATE/output" 2>&1 || status=$?
  if [ "$status" -ne "$expected_status" ]; then
    cat "$STATE/output" >&2
    echo "FAIL $name: expected status $expected_status, got $status" >&2
    exit 1
  fi
  if [ "$expected_cleanup" = yes ]; then
    test -f "$STATE/pruned"
    test "$(wc -l < "$STATE/prune_attempts")" -eq 1
  else
    test ! -f "$STATE/pruned"
  fi
  echo "PASS $name"
}

# Values are in 1 KiB blocks; the threshold is 15 decimal GB.
run_case 'enough space skips cleanup' 0 no BEFORE_KB=20000000 AFTER_KB=20000000
run_case 'low space recovers' 0 yes BEFORE_KB=100000 AFTER_KB=20000000
run_case 'insufficient recovery stops startup' 1 yes BEFORE_KB=100000 AFTER_KB=200000
grep -q 'Increase Docker disk space' "$STATE/output"
run_case 'failed measurement does not prune' 1 no BEFORE_KB=0 AFTER_KB=0 BEFORE_ERROR='Cannot connect to the Docker daemon'
grep -q 'Increase Docker disk space' "$STATE/output"
run_case 'full disk during image pull recovers' 0 yes BEFORE_KB=0 AFTER_KB=20000000 BEFORE_ERROR='failed to register layer: no space left on device'
run_case 'persistent full disk stops after one cleanup' 1 yes BEFORE_KB=0 AFTER_KB=0 BEFORE_ERROR='no space left on device' AFTER_ERROR='no space left on device'
grep -q 'Increase Docker disk space' "$STATE/output"
run_case 'registry errors do not prune' 1 no BEFORE_KB=0 AFTER_KB=0 BEFORE_ERROR='pull access denied'
run_case 'pull progress does not affect measurement' 0 no BEFORE_KB=20000000 AFTER_KB=20000000 PULL_PROGRESS=1
run_case 'invalid measurement does not prune' 1 no BEFORE_KB=unknown AFTER_KB=0
run_case 'failed cleanup stops startup' 1 no BEFORE_KB=100000 AFTER_KB=20000000 PRUNE_FAIL=1

# Check the real Make entry point, including its local-only recovery rule.
rm -f "$STATE/pruned" "$STATE/started"
if BEFORE_KB=100000 AFTER_KB=200000 CI= make -C "$repo_root" dev.up >"$STATE/output" 2>&1; then
  echo 'FAIL dev.up started with insufficient space' >&2
  exit 1
fi
test -f "$STATE/pruned"
test ! -f "$STATE/started"
echo 'PASS dev.up checks space before startup'

rm -f "$STATE/pruned" "$STATE/started"
BEFORE_KB=100000 AFTER_KB=200000 CI=true make -C "$repo_root" dev.up >"$STATE/output" 2>&1
test ! -f "$STATE/pruned"
test -f "$STATE/started"
echo 'PASS CI does not prune host cache'
