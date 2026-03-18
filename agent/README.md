# PydanticAI Bootstrap Repository

Bootstrap template for Python services built with PydanticAI.
All development and quality checks run inside Docker and are orchestrated via `make`.

## Prerequisites

- Docker
- Docker Compose
- GNU Make

## Quick Start

1. Copy environment defaults:

   ```bash
   cp .env.example .env
   ```

2. Build containers and install dependencies:

   ```bash
   make dev.setup
   ```

3. Run tests:

   ```bash
   make test
   ```

## Development Workflow

- Start containerized dev environment: `make dev.start`
- Follow logs: `make dev.logs`
- Open shell in app container: `make dev.shell`
- Stop and remove services: `make dev.down`

## Canvas Q&A CLI

Run a question against a real canvas (requires API access and a non-test model):

```bash
python -m ai.main \
  --question "What triggers this flow?" \
  --canvas-id "<canvas-id>" \
  --base-url "<https://your-superplane>" \
  --org-id "<organization-id>" \
  --token "<api-token>" \
  --model "openai:gpt-5-mini"
```

Environment variables are also supported:

- `SUPERPLANE_BASE_URL`
- `SUPERPLANE_API_TOKEN`
- `SUPERPLANE_ORG_ID`
- `SUPERPLANE_CANVAS_ID`
- `CANVAS_ID` (alias for `SUPERPLANE_CANVAS_ID`)

### Interactive Console

Start a REPL-like console inside Docker:

```bash
make dev.console
```

`dev.console` supports manual reload while running: type `/reload` in the REPL.
`dev.console` always prints tool invocations before each assistant answer.
It also prints timing:

- question lifecycle (`<elapsed> Started` / `<elapsed> Completed`)
- per tool call prefixed with elapsed (`<elapsed> [tool] ...`)
- per-tool duration (`tool_elapsed_ms=...`)
- tool payload size (`response_size=10.9 KiB (11115 bytes)`)
- post-tool status (`<elapsed> [status] Tools completed. Generating final answer...`)
- pre-answer status (`<elapsed> [status] Final answer ready.`)

Console logs are colorized (timestamps dim, tool/status labels highlighted) when output is a TTY.

You can pass a canvas id directly from the command line:

```bash
make dev.console CANVAS_ID=<canvas-id>
```

One-off runs also print tool invocations by default:

```bash
python -m ai.main --question "What is on this canvas?"
```

Run a non-reloading console once (no `/reload` restart loop):

```bash
make dev.console.once CANVAS_ID=<canvas-id>
```

By default, this runs with `AI_MODEL=test`. To ask real canvas questions, set:

```bash
export AI_MODEL="openai:gpt-5-mini"
export SUPERPLANE_BASE_URL="https://your-superplane"
export SUPERPLANE_API_TOKEN="your-token"
export SUPERPLANE_ORG_ID="your-org-id"
export SUPERPLANE_CANVAS_ID="your-canvas-id"
export OPENAI_API_KEY="your-openai-key"
make dev.console
```

## Quality Checks

- Lint: `make lint`
- Format: `make format`
- Type check: `make typecheck`

## Python Version

The repository pins Python `3.13` in `Dockerfile`, chosen as the latest stable release
with strong support in current Pydantic versions.
