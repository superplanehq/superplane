#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "${script_dir}/.." && pwd)"
log="${TMPDIR:-/tmp}/monaco-worker-startup.log"
ready="${TMPDIR:-/tmp}/monaco-worker-startup.json"
rm -f "${ready}"

cd "${repo_root}/web_src"
node scripts/serve-monaco-worker-check.mjs "${ready}" >"${log}" 2>&1 &
server_pid=$!
cleanup() {
  kill "${server_pid}" >/dev/null 2>&1 || true
  wait "${server_pid}" 2>/dev/null || true
}
trap cleanup EXIT

page_url=""
worker_url=""
for _ in $(seq 1 300); do
  if [[ -s "${ready}" ]]; then
    if page_url="$(node -e 'const ready = JSON.parse(require("fs").readFileSync(process.argv[1], "utf8")); if (!ready.pageUrl || !ready.workerUrl) process.exit(1); process.stdout.write(ready.pageUrl)' "${ready}" 2>/dev/null)"; then
      worker_url="$(node -e 'const ready = JSON.parse(require("fs").readFileSync(process.argv[1], "utf8")); process.stdout.write(ready.workerUrl)' "${ready}")"
      break
    fi
  fi
  if ! kill -0 "${server_pid}" >/dev/null 2>&1; then
    echo "Monaco worker check exited before it was ready. Log:" >&2
    cat "${log}" >&2
    exit 1
  fi
  sleep 1
done

if [[ -z "${page_url}" || -z "${worker_url}" ]]; then
  echo "Monaco worker check did not become ready. Log:" >&2
  cat "${log}" >&2
  exit 1
fi

cd "${repo_root}"
MONACO_PAGE_URL="${page_url}" MONACO_WORKER_URL="${worker_url}" \
  go test ./test/browser -count=1 -timeout 2m -run TestMonacoProductionWorkerStarts
