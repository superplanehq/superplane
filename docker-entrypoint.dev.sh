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
    # Matches both the `go run` parent and the compiled binary it execs.
    pkill -9 -f 'dev-broker' 2>/dev/null || true
    sleep 1
  done
}
stop_watchers

# Runner components need a task broker, which self-hosted installations don't
# have. Run one in this container so localhost resolves the same way for both
# sides: the API reaches the broker, the broker reaches the API's webhooks.
#
# Built before air starts — two concurrent `go build` runs race on the shared
# build cache and make air fail with "exit status 1". The binary goes outside
# ./tmp because air deletes that directory when it exits.
if [[ "${START_DEV_BROKER:-yes}" == "yes" ]]; then
  go build -o /tmp/dev-broker ./cmd/dev-broker
  /tmp/dev-broker >/tmp/dev-broker.log 2>&1 &
fi

air &
air_pid=$!

cd web_src
npm run dev &
vite_pid=$!
cd ..

# Wait on the API and UI only. Waiting on every job would let a broker crash —
# a development convenience — take the whole stack down with it.
wait -n "$air_pid" "$vite_pid"
exit $?
