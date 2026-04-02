#!/usr/bin/env bash
# Run inside the dev compose `agent` service (repo at /app/agent). Reuses the same venv as uvicorn.
set -euo pipefail

cd /app/agent
exec uv run --group dev python scripts/bootstrap_eval_env.py "$@"
