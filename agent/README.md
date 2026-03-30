# SuperPlane Agents

Python agent built with [Pydantic AI](https://ai.pydantic.dev/). Prompt: `src/ai/system_prompt.txt`. Workflow eval cases: `evals/cases.py`. Reusable patterns: `patterns/`.

## Canvas evals

Evals talk to a running SuperPlane app, run the agent, and assert on the canvas.

**Prep (once):** Copy `agent/.env.example` to `agent/.env`. Set `AI_MODEL`, `ANTHROPIC_API_KEY`, `EVAL_BOOTSTRAP_EMAIL`, and `EVAL_BOOTSTRAP_PASSWORD` (user must be allowed to manage canvases and service accounts). Optional fields are in `agent/.env.example`.

**Typical flow**

1. Start SuperPlane (e.g. dev stack: `docker compose -f docker-compose.dev.yml up -d`). The **`agent`** service must be up — bootstrap runs inside it.
2. Run **`make agent.evals.bootstrap`** — it prints `EVAL_ORG_ID`, `EVAL_CANVAS_ID`, `SUPERPLANE_API_TOKEN`, and `SUPERPLANE_BASE_URL`. Paste those lines into `agent/.env` (or `ARGS='--merge-env /app/agent/.env'` to write the file from the container). Re-run when you need a new token or IDs.
3. Run **`make test.agent.evals`**. The runner reloads `agent/.env` from disk each time.
