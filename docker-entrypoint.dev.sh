#!/usr/bin/env bash
#
# Hot-reload stack for local development (run via `make dev.server`).
# Install JS deps with `make dev.setup` first so `npm run dev` can start quickly.
#
set -euo pipefail

# Best-effort: allow re-running `make dev.server` without recreating the container.
# Multiple `air` processes (e.g. after interrupted `make dev.server.fg`) race on the same
# ./tmp/superplane binary and (historically) shared module cache — `go build` then fails with
# "failed to build, error: exit status 1" while Vite may already show "ready". Force-stop prior
# watchers a few times (pgrep is unreliable here because old `air` PIDs can linger as zombies).
stop_watchers() {
  local i
  for i in 1 2 3; do
    pkill -9 -x air 2>/dev/null || true
    pkill -9 -f 'node_modules/.bin/vite' 2>/dev/null || true
    sleep 1
  done
}
stop_watchers

air &

cd web_src
npm run dev &
cd ..

wait -n
exit $?
