# agent2 — Go Agent Service (Anthropic Managed Agents)

Drop-in replacement for the Python `agent/` service. Implements the same
`protos/private/agents.proto` gRPC interface but delegates all LLM work to
Anthropic's Managed Agents API.

## Architecture

```
Go Backend (gRPC client) → agent2 (gRPC server) → Anthropic Managed Agents API
                                ↓
                          PostgreSQL (session mapping)
                                ↓
                          SSE stream → Frontend AgentSidebar
```

## Configuration (env vars)

| Variable | Description |
|----------|-------------|
| `ANTHROPIC_API_KEY` | Anthropic API key |
| `ANTHROPIC_AGENT_ID` | Managed agent ID (e.g. `agent_01CZNjeiKsvJTfbZZpt5MMu6`) |
| `ANTHROPIC_ENVIRONMENT_ID` | Environment ID for sandboxed execution |
| `DB_URL` | PostgreSQL connection string |
| `GRPC_PORT` | gRPC listen port (default: 50061) |
| `HTTP_PORT` | HTTP listen port for SSE streams (default: 8090) |
| `JWT_SECRET` | Shared JWT secret for auth tokens |

## Running

```bash
go run ./cmd/agent2
```

## Development

```bash
make build.agent2
make test.agent2
```
