#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "${BASH_SOURCE[0]}")/.."

uncovered=()
while IFS= read -r -d '' file; do
  case "$file" in
    pkg/*|ee/*|cmd/*|test/support/*|test/e2e/*) ;;
    test/browser/*)
      [[ "${file#test/browser/}" == */* ]] && uncovered+=("$file")
      ;;
    *) uncovered+=("$file") ;;
  esac
done < <(git ls-files --cached --others --exclude-standard -z -- '*_test.go')

if (( ${#uncovered[@]} > 0 )); then
  printf 'Go test files outside CI test jobs:\n' >&2
  printf '  %s\n' "${uncovered[@]}" >&2
  printf 'Add the package to a CI test job before merging.\n' >&2
  exit 1
fi

browser_tests="$(awk '
  /^func Test[[:alnum:]_]*[[:space:]]*\(/ {
    name = $2
    sub(/\(.*/, "", name)
    print name
  }
' test/browser/*_test.go | LC_ALL=C sort -u)"
browser_runners="$(awk '
  {
    while (match($0, /Test[A-Z][[:alnum:]_]*/)) {
      print substr($0, RSTART, RLENGTH)
      $0 = substr($0, RSTART + RLENGTH)
    }
  }
' scripts/check_monaco_worker_startup.sh scripts/check_backlog_create_tabs.sh | LC_ALL=C sort -u)"
unrouted_browser_tests="$(comm -23 <(printf '%s\n' "$browser_tests") <(printf '%s\n' "$browser_runners"))"

if [[ -n "$unrouted_browser_tests" ]]; then
  printf 'Browser tests missing from CI runners:\n%s\n' "$unrouted_browser_tests" >&2
  exit 1
fi

echo 'Go test discovery checks passed.'
