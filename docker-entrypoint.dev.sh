#!/usr/bin/env bash
#
# Hot-reload stack for local development (run via `make dev.server`).
# Install JS deps with `make dev.setup` first so `npm run dev` can start quickly.
#
set -euo pipefail

# Best-effort: allow re-running `make dev.server` without recreating the container.
pkill -x air 2>/dev/null || true
pkill -f '/app/web_src/node_modules/.bin/vite' 2>/dev/null || true
sleep 1

air &

cd web_src
npm run dev &
cd ..

wait -n
exit $?
