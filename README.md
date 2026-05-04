# runner

Monorepo (single Go module) for **fleet-manager** and the **runner** worker. Shared API types and domain models live under `shared/`.

See [ARCHITECTURE.md](./ARCHITECTURE.md) for the system design.

## Layout

```
.
├── fleet-manager/          # queue + HTTP API + webhooks
│   ├── cmd/fleet-manager/
│   └── internal/…
├── runner/                 # worker agent
│   ├── cmd/runner/
│   └── internal/agent/
├── shared/                 # shared: api (JSON DTOs), models
├── go.mod
└── Makefile
```

## Requirements

- Go 1.22+
- For Docker tasks: Docker CLI on the runner host

## Build

```bash
make build
# or:
go build -o bin/fleet-manager ./fleet-manager/cmd/fleet-manager
go build -o bin/runner ./runner/cmd/runner
```

## Run fleet-manager

| Environment variable | Default | Description |
|----------------------|---------|-------------|
| `LISTEN_ADDR` | `:8080` | HTTP listen address |
| `DATABASE_PATH` | `./fleet.db` | SQLite database file |
| `AUTH_TOKEN` | (empty) | If set, requires `Authorization: Bearer <token>` for `/v1/*` |
| `REAP_INTERVAL_SEC` | `15` | How often to return expired leases to the queue |

```bash
export DATABASE_PATH=./fleet.db
./bin/fleet-manager
```

## Run the runner

| Environment variable | Description |
|----------------------|-------------|
| `FLEET_MANAGER_URL` | **Required.** Base URL (e.g. `http://127.0.0.1:8080`) |
| `RUNNER_ID` | Optional; defaults to host name or a random id |
| `AUTH_TOKEN` | Optional; must match fleet-manager if set |
| `POLL_EMPTY_MS` | Sleep when no work (default ~1000 ms) |

```bash
export FLEET_MANAGER_URL=http://127.0.0.1:8080
export AUTH_TOKEN= # if fleet-manager uses it
./bin/runner
```

## HTTP API (v1)

- `GET /healthz` — liveness
- `POST /v1/tasks` — enqueue a task (`command`, `webhook_url`, optional `execution_mode` / `docker_image`)
- `POST /v1/tasks/claim` — runner pulls the next task (`runner_id`, `lease_seconds`)
- `POST /v1/tasks/{id}/complete` — runner reports result (`runner_id`, `exit_code`, `output`, optional `error`)

On completion, fleet-manager POSTs a JSON **webhook** to `webhook_url` with `task_id`, `status`, `exit_code`, `output`, and `error`.

## Example: enqueue with curl

```bash
curl -X POST http://127.0.0.1:8080/v1/tasks \
  -H 'Content-Type: application/json' \
  -d '{
    "command": ["sh", "-c", "echo hello; exit 0"],
    "webhook_url": "https://example.com/your-hook"
  }'
```

## License

(Add your license.)
