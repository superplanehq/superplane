#!/usr/bin/env bash
# Runs bootstrap in the already-running agent container (reuses its Python env; talks to app over HTTP).
# Usage (from repository root): bash agent/scripts/run_bootstrap_eval_env.sh [args passed to Python script]
set -euo pipefail

script_dir=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
repo_root=$(cd "$script_dir/../.." && pwd)

cd "$repo_root"

docker compose -f docker-compose.dev.yml exec -T agent \
  bash /app/agent/scripts/bootstrap_eval_env_in_container.sh "$@"
