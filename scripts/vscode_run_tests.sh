#!/usr/bin/env bash
set -euo pipefail

# VSCode test runner for Go
# Usage:
#   vscode_run_tests.sh all
#   vscode_run_tests.sh file <relative-filepath>
#   vscode_run_tests.sh line <relative-filepath> <line-number>

MODE=${1:-}
FILE_RELATIVE=${2:-}
LINE_NUMBER=${3:-}

# Resolve repo root based on this script's location
ROOT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
cd "$ROOT_DIR"

echo "[vscode] mode=$MODE file=$FILE_RELATIVE line=$LINE_NUMBER"

# Docker Compose config (assumes compose is already up)
DOCKER_COMPOSE_OPTS=${DOCKER_COMPOSE_OPTS:-"-f docker-compose.dev.yml"}

docker_exec() {
  # Pass DB_NAME=superplane_test to match Makefile test environment
  docker compose ${DOCKER_COMPOSE_OPTS} exec -e DB_NAME=superplane_test app bash -lc "$*"
}

# Utility: determine package import path for a given file or directory
pkg_for_path() {
  local path="$1"
  if [ -d "$path" ]; then
    pkg_dir="$path"
  else
    pkg_dir="$(dirname "$path")"
  fi
  # Convert to import path relative to module
  local module_path
  module_path=$(awk '/^module /{print $2; exit}' go.mod)
  # Trim leading ./ if present
  pkg_rel="${pkg_dir#./}"
  echo "$module_path/${pkg_rel}"
}

# Utility: extract all Test* names from a _test.go file
extract_tests_in_file() {
  local file="$1"
  # Match both free funcs and methods: func (r *R) TestXxx(t *testing.T)
  awk '/^func\s*(\([^)]*\)\s*)?Test[[:alnum:]_]*\s*\(/ { \
        name=$0; \
        sub(/^func\s*/, "", name); \
        sub(/^\([^)]*\)\s*/, "", name); \
        sub(/\(.*/, "", name); \
        gsub(/\s+/, "", name); \
        print name; \
      }' "$file"
}

# Utility: find Test* name at or above specific line in a _test.go file
test_name_for_line() {
  local file="$1"; local line="$2"
  awk -v target="$line" '
    /^func\s*(\([^)]*\)\s*)?Test[[:alnum:]_]*\s*\(/ {
      name=$0
      sub(/^func\s*/, "", name)
      sub(/^\([^)]*\)\s*/, "", name)
      sub(/\(.*/, "", name)
      gsub(/\s+/, "", name)
      start=NR
    }
    NR==target { current=name }
    END { if (length(current)>0) print current }
  ' "$file"
}

run_all() {
  echo "Running in docker: go test ./..."
  docker_exec "cd /app && GOFLAGS= go test -count=1 ./..."
}

run_file() {
  local file="$1"
  if [[ "$file" != *_test.go ]]; then
    # Not a test file: run the package containing the file
    local pkg
    pkg=$(pkg_for_path "$file")
    echo "Running package tests in docker: go test $pkg"
    docker_exec "cd /app && GOFLAGS= go test -count=1 '$pkg'"
    return
  fi

  # Test file: try to run only tests in this file
  local tests regex pkg
  mapfile -t tests < <(extract_tests_in_file "$file") || true
  pkg=$(pkg_for_path "$file")

  if [ ${#tests[@]} -eq 0 ]; then
    echo "No tests found in $file; running package in docker: $pkg"
    docker_exec "cd /app && GOFLAGS= go test -count=1 '$pkg'"
    return
  fi

  regex="^($(IFS='|'; echo "${tests[*]}")$)"
  echo "Running tests in file: $file"
  echo "docker exec: go test $pkg -run '$regex'"
  docker_exec "cd /app && GOFLAGS= go test -count=1 '$pkg' -run '$regex'"
}

run_line() {
  local file="$1"; local line="$2"
  local pkg test
  pkg=$(pkg_for_path "$file")

  if [[ "$file" != *_test.go ]]; then
    echo "File is not a test file; running package in docker: $pkg"
    docker_exec "cd /app && GOFLAGS= go test -count=1 '$pkg'"
    return
  fi

  test=$(test_name_for_line "$file" "$line" || true)
  if [ -z "$test" ]; then
    echo "No enclosing test found at $file:$line; running tests in file"
    run_file "$file"
    return
  fi

  echo "Running single test in docker: $test in $pkg"
  docker_exec "cd /app && GOFLAGS= go test -count=1 '$pkg' -run '^$test$'"
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
