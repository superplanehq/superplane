# Running the Agent Locally

The SuperPlane agent (AI chat sidebar) uses Anthropic's managed-agents API.
To enable it in local dev, you need to set a few env vars before starting the containers.

## Prerequisites

- Docker + Make (see CONTRIBUTING.md)
- An Anthropic API key with managed-agents access
- The agent ID and environment ID from the Anthropic console

## 1. Create `.env` in the repo root

```bash
# .env (repo root — gitignored)

export ANTHROPIC_API_KEY=sk-ant-api03-...
export ANTHROPIC_AGENT_ID=agent_01XXXX...
export ANTHROPIC_ENVIRONMENT_ID=env_01XXXX...

# Required: tells the frontend to show the Agent tab
export AGENT_ENABLED=yes

# If you're exposing the app externally (e.g. ngrok, Caddy reverse proxy):
# export BASE_URL=https://your-domain.com
# export ALLOWED_WS_ORIGINS=https://your-domain.com
```

> **Where to get these values:**
> - `ANTHROPIC_API_KEY` — from console.anthropic.com → API Keys
> - `ANTHROPIC_AGENT_ID` — from the managed-agents dashboard (ask Igor/Alex)
> - `ANTHROPIC_ENVIRONMENT_ID` — same dashboard, under Environments

## 2. Source the env before starting containers

```bash
source .env
make dev.up
make dev.setup   # first time only
make dev.server
```

The `docker-compose.dev.yml` passes these vars into the `app` container:
- `AGENT_ENABLED` — controls whether the Agent tab appears in the UI
- `ANTHROPIC_API_KEY` / `ANTHROPIC_AGENT_ID` / `ANTHROPIC_ENVIRONMENT_ID` — the backend uses these to authenticate with Anthropic's API

## 3. Verify it works

1. Open `http://localhost:8000` (or your `BASE_URL`)
2. Navigate to any canvas
3. The sidebar should show the **Agent** tab
4. Type a message — you should see "Thinking..." followed by a response

## How it works

```
Browser (WebSocket) → Go server → Anthropic managed-agents API
                    ↕
              PostgreSQL (chat history)
              RabbitMQ (stream worker)
```

- `pkg/agents/anthropic/provider.go` — Anthropic client that calls the managed-agents API
- `pkg/workers/agent_stream_worker.go` — processes streaming responses from Anthropic
- `pkg/web/index_template.go` — injects `AGENT_ENABLED` flag into the frontend HTML
- `pkg/config/config.go` → `LoadAnthropicAgentConfig()` — reads the env vars

## Env var reference

| Variable | Required | Description |
|----------|----------|-------------|
| `AGENT_ENABLED` | Yes | Set to `yes` to show the Agent tab |
| `ANTHROPIC_API_KEY` | Yes | Anthropic API key |
| `ANTHROPIC_AGENT_ID` | Yes | Managed-agent ID |
| `ANTHROPIC_ENVIRONMENT_ID` | Yes | Agent environment ID |
| `BASE_URL` | No | Public URL if not localhost:8000 |
| `ALLOWED_WS_ORIGINS` | No | WebSocket CORS origins (match BASE_URL) |

## Troubleshooting

- **Agent tab missing:** Check `AGENT_ENABLED=yes` is set and you re-ran `make dev.up` after sourcing `.env`
- **"Thinking..." but no response:** Check `ANTHROPIC_API_KEY` is valid. Look at container logs: `docker compose -f docker-compose.dev.yml logs app -f`
- **WebSocket errors:** Make sure `ALLOWED_WS_ORIGINS` matches the URL you're accessing the app from
