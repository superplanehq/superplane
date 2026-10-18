#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "${script_dir}/.." && pwd)"
port="${STORYBOOK_PORT:-6016}"
url="http://127.0.0.1:${port}"
log="${TMPDIR:-/tmp}/backlog-create-tabs-storybook.log"

cd "${repo_root}/web_src"
npm run storybook -- --host 127.0.0.1 --port "${port}" --ci --no-open >"${log}" 2>&1 &
server_pid=$!
cleanup() {
  kill "${server_pid}" >/dev/null 2>&1 || true
  wait "${server_pid}" 2>/dev/null || true
}
trap cleanup EXIT

ready=0
for _ in $(seq 1 90); do
  if node -e 'fetch(process.argv[1]).then((r) => process.exit(r.ok ? 0 : 1)).catch(() => process.exit(1))' "${url}/iframe.html"; then
    ready=1
    break
  fi
  if ! kill -0 "${server_pid}" >/dev/null 2>&1; then
    echo "Storybook exited before it was ready. Log:" >&2
    cat "${log}" >&2
    exit 1
  fi
  sleep 1
done
if [[ "${ready}" -ne 1 ]]; then
  echo "Storybook did not start on ${url}. Log:" >&2
  cat "${log}" >&2
  exit 1
fi

cd "${repo_root}"
STORYBOOK_URL="${url}" go test ./test/browser -count=1 -timeout 5m -run 'TestBacklogCreateMenuScrollsHiddenSourceTabIntoView|TestMCPClientRevokeStaysInsideNarrowCard'
