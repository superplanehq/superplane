# SuperPlane Agents

Built with [Pydantic AI](https://ai.pydantic.dev/).

Starting points:

- System prompt is in `agent/src/ai/agent.py`
- Evals are in `agent/evals/runner.py`
- Patterns are in `agent/patterns`

## Canvas agent evals

Evals run the agent against a live SuperPlane app and assert on the resulting canvas.

### First-time eval setup (automated)

With the dev stack up (`docker compose -f docker-compose.dev.yml up -d`), provision org, eval canvas, and API token:

```bash
make agent.evals.bootstrap
```

The Make target uses `docker compose run --user $$(id -u):$$(id -g) --env-from-file agent/.env` so merged **`agent/.env` stays owned by your host user** (avoids root-owned files from the default container user). `UV_CACHE_DIR=/tmp/uv-cache` is set so `uv sync` works without writing to `/root/.cache`.

The script prints those values to stdout and, by default, **merges** them into `agent/.env` inside the container (`--merge-env /app/agent/.env`): existing keys are updated in place; missing keys are appended under a short comment. Duplicate keys are collapsed to a single updated line.

To **only print** (no file write), run the script yourself without `--merge-env`, for example:

```bash
docker compose -f docker-compose.dev.yml run --rm -T --no-deps --env-from-file agent/.env agent \
  bash -lc "cd /app/agent && uv sync --group dev && uv run --group dev python scripts/bootstrap_eval_env.py"
```

To **append** a block instead of merging, run the script with `-o /path` only (no `--merge-env`); the default `make agent.evals.bootstrap` always merges, so use a one-off `docker compose run ... python scripts/bootstrap_eval_env.py -o /app/agent/.env` if you need append behavior.

Configure bootstrap credentials in `agent/.env` (see `agent/.env.example`):

- **Fresh database (owner setup not done yet):** set `EVAL_BOOTSTRAP_OWNER_*` (email, password, first/last name). The script calls `POST /api/v1/setup-owner`.
- **Instance already has users (typical):** set **`EVAL_BOOTSTRAP_EMAIL`** and **`EVAL_BOOTSTRAP_PASSWORD`** in `agent/.env` (SuperPlane user with permission to manage canvases and service accounts). Required whenever setup-owner is not used (409).

The script reuses a canvas named **Agent evals** and a service account **agent-evals** when they already exist (regenerates the service account token if the account exists).

Requires dev dependencies: the Make target runs `uv sync --group dev` (includes `httpx`).

### Run evals

- Full suite: `make test.agent.evals`
- See `agent/.env.example` for optional `EVAL_CASE_NAMES`, `EVAL_OUTPUT_DIR`, etc.