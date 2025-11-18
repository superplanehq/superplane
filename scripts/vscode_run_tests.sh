#!/usr/bin/env sh
set -eu

# VSCode test runner for Go
# Usage:
#   vscode_run_tests.sh all
#   vscode_run_tests.sh file <relative-filepath>
#   vscode_run_tests.sh line <relative-filepath> <line-number>

MODE=${1:-}
FILE_RELATIVE=${2:-}
LINE_NUMBER=${3:-}

# Resolve repo root based on this script's location
ROOT_DIR=$(cd "$(dirname "$0")/.." && pwd)
cd "$ROOT_DIR"

# Docker Compose config (assumes compose is already up)
DOCKER_COMPOSE_OPTS=${DOCKER_COMPOSE_OPTS:-"-f docker-compose.dev.yml"}

docker_exec() {
  # Pass DB_NAME=superplane_test to match Makefile test environment
  # Also force Go build cache to a writable, persisted path in the repo
  echo "+ docker compose ${DOCKER_COMPOSE_OPTS} exec -e DB_NAME=superplane_test -e GOCACHE=/app/tmp/go-build -e XDG_CACHE_HOME=/app/tmp app bash -lc \"$*\""
  echo ""
  echo "--------------------------------------------------------------------------------"
  echo ""
  # Run command while capturing exit status even with set -e
  set +e
  docker compose ${DOCKER_COMPOSE_OPTS} exec -e DB_NAME=superplane_test -e GOCACHE=/app/tmp/go-build -e XDG_CACHE_HOME=/app/tmp app bash -lc "$*"
  status=$?
  set -e
  if [ "$status" -eq 0 ]; then
    # Green PASSED
    printf "\n\033[32mPASSED\033[0m\n"
  else
    # Red FAILED
    printf "\n\033[31mFAILED\033[0m\n"
  fi
  return "$status"
}

# Utility: determine package import path for a given file or directory
pkg_for_path() {
  path="$1"
  if [ -d "$path" ]; then
    pkg_dir="$path"
  else
    pkg_dir="$(dirname "$path")"
  fi
  module_path=$(awk '/^module /{print $2; exit}' go.mod)
  pkg_rel="${pkg_dir#./}"
  echo "$module_path/${pkg_rel}"
}

# Utility: extract all Test* names from a _test.go file
extract_tests_in_file() {
  file="$1"
  # Output Test* names one per line
  awk '/^func[[:space:]]*(\([^)]*\)[[:space:]]*)?Test[[:alnum:]_]*[[:space:]]*\(/ { \
        name=$0; \
        sub(/^func[[:space:]]*/, "", name); \
        sub(/^\([^)]*\)[[:space:]]*/, "", name); \
        sub(/\(.*/, "", name); \
        gsub(/[[:space:]]+/, "", name); \
        print name; \
      }' "$file"
}

# Utility: find Test* name at or above specific line in a _test.go file
test_name_for_line() {
  file="$1"; line="$2"
  awk -v target="$line" '
    /^func[[:space:]]*(\([^)]*\)[[:space:]]*)?Test[[:alnum:]_]*[[:space:]]*\(/ {
      name=$0
      sub(/^func[[:space:]]*/, "", name)
      sub(/^\([^)]*\)[[:space:]]*/, "", name)
      sub(/\(.*/, "", name)
      gsub(/[[:space:]]+/, "", name)
      current=name
    }
    NR==target { seen=1 }
    NR>target && seen && !printed { if (length(current)>0) { print current; printed=1 } }
    END { if (seen && !printed && length(current)>0) print current }
  ' "$file"
}

# Utility: find nearest subtest name (t.Run("<name>", ...)) at or above a specific line
subtest_name_for_line() {
  file="$1"; line="$2"
  awk -v target="$line" '
    NR>target { exit }
    {
      # Match t.Run("name", ...) or something.Run("name", ...)
      if ($0 ~ /Run\("[^"]*"[[:space:]]*,/) {
        line=$0
        sub(/.*Run\("/, "", line)
        sub(/".*/, "", line)
        current=line
      }
    }
    END {
      if (length(current)>0) print current
    }
  ' "$file"
}

run_all() {
  echo "Running in docker: go test -p 1 -v ./..."
  docker_exec "cd /app && GOFLAGS= go test -count=1 -p 1 -v ./..."
}

run_file() {
  file="$1"
  case "$file" in
    *_test.go) : ;;
    *)
    # Not a test file: run the package containing the file
    pkg=$(pkg_for_path "$file")
    echo "Running package tests in docker: go test -p 1 -v $pkg"
    docker_exec "cd /app && GOFLAGS= go test -count=1 -p 1 -v '$pkg'"
    return
    ;;
  esac

  # Test file: try to run only tests in this file
  pkg=$(pkg_for_path "$file")
  tests_list=$(extract_tests_in_file "$file" || true)
  if [ -z "${tests_list:-}" ]; then
    echo "No tests found in $file; running package in docker: $pkg"
    docker_exec "cd /app && GOFLAGS= go test -count=1 -p 1 -v '$pkg'"
    return
  fi

  # Join test names with | to build regex
  regex="^($(printf '%s' "$tests_list" | paste -sd '|' -))$"
  echo "Running tests in file: $file"
  echo "docker exec: go test -p 1 -v $pkg -run $regex"
  docker_exec "cd /app && GOFLAGS= go test -count=1 -p 1 -v '$pkg' -run '$regex'"
}

run_line() {
  file="$1"; line="$2"
  pkg=$(pkg_for_path "$file")

  case "$file" in
    *_test.go) : ;;
    *)
    echo "File is not a test file; running package in docker: $pkg"
    docker_exec "cd /app && GOFLAGS= go test -count=1 -p 1 -v '$pkg'"
    return
    ;;
  esac

  test_name=$(test_name_for_line "$file" "$line" || true)
  subtest_name=$(subtest_name_for_line "$file" "$line" || true)
  if [ -z "${test_name:-}" ]; then
    echo "No enclosing test found at $file:$line; running tests in file"
    run_file "$file"
    return
  fi

  if [ -n "${subtest_name:-}" ]; then
    echo "Running subtest: $test_name/$subtest_name"
    docker_exec "cd /app && GOFLAGS= go test -count=1 -p 1 -v '$pkg' -run '^$test_name$/^$subtest_name$'"
  else
    docker_exec "cd /app && GOFLAGS= go test -count=1 -p 1 -v '$pkg' -run '^$test_name$'"
  fi
}

case "$MODE" in
  all)
    run_all
    ;;
  file)
    if [ -z "${FILE_RELATIVE:-}" ]; then
      echo "Missing file argument" >&2; exit 2
    fi
    run_file "$FILE_RELATIVE"
    ;;
  line)
    if [ -z "${FILE_RELATIVE:-}" ] || [ -z "${LINE_NUMBER:-}" ]; then
      echo "Missing file or line argument" >&2; exit 2
    fi
    run_line "$FILE_RELATIVE" "$LINE_NUMBER"
    ;;
  *)
    echo "Usage: $0 {all|file <file>|line <file> <line>}" >&2
    exit 2
    ;;
esac
