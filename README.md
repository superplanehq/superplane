# SuperPlane runner

Monorepo (single Go module) for **task-broker**, **fleet-manager**, and the **runner** worker.

- **Task-broker** — optional front door: registers downstream fleet-manager instances (per region/cloud), routes `POST /v1/tasks`, and relays completion webhooks to callers (`task_id` is broker-scoped; `fleet_task_id` identifies the upstream task).
- **Fleet-manager** — durable queue per fleet, runners claim work via HTTP, completions POST to either the caller webhook or the broker relay URL.
- **Runner** — long-lived worker: claims tasks, executes `command` argv or multi-line **`commands`** shell scripts, completes results.

Shared JSON types live under **`shared/`**; webhook retries use **`shared/webhook`**.

See [ARCHITECTURE.md](./ARCHITECTURE.md) for design detail.

## Layout

```
.
├── task-broker/            # proxy + fleet registry + webhook relay
│   ├── cmd/task-broker/
│   └── internal/…
├── fleet-manager/          # queue + HTTP API (per fleet)
│   ├── cmd/fleet-manager/
│   └── internal/…
├── runner/                 # worker agent
│   ├── cmd/runner/
│   └── internal/agent/
├── shared/
│   ├── api/               # REST DTOs (incl. broker types)
│   ├── models/
│   └── webhook/           # retrying POST client
├── test/                  # e2e tests (fleet-only and broker+fleet stacks)
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
go build -o bin/task-broker ./task-broker/cmd/task-broker
```

## Run task-broker

The broker needs a URL that **downstream fleet-manager** instances can POST to when a task completes (typically your public/load-balanced origin).

| Environment variable | Default       | Description                                                                                                              |
| -------------------- | ------------- | ------------------------------------------------------------------------------------------------------------------------ |
| `LISTEN_ADDR`        | `:8081`       | HTTP listen address                                                                                                      |
| `DATABASE_PATH`      | `./broker.db` | SQLite (fleets + broker-scoped tasks)                                                                                    |
| `BROKER_PUBLIC_URL`  | (empty)       | Base URL reachable by fleet-manager(s), used to build completion relay URLs (**set in real deployments**)                |
| `AUTH_TOKEN`         | (empty)       | If set, required for **`/v1/fleets` and `/v1/tasks`** (`Authorization: Bearer …`). Webhook callbacks are unauthenticated |

**HTTP (`/v1`)**

| Method   | Path                                | Notes                                                                                                                                                                                                                                           |
| -------- | ----------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `GET`    | `/fleets`                           | List registered fleet-managers                                                                                                                                                                                                                  |
| `POST`   | `/fleets`                           | Register/replace a fleet (`id`, `base_url`, optional `auth_token`, `labels`)                                                                                                                                                                    |
| `DELETE` | `/fleets/{id}`                      | Remove a fleet                                                                                                                                                                                                                                  |
| `POST`   | `/tasks`                            | Body: `BrokerCreateTaskRequest` — embedded task fields (`command` xor `commands`, `webhook_url`, execution mode…) plus **`fleet_id`** xor **`fleet_labels`**: selected fleet must have **every** listed label (see `POST /v1/fleets` `labels`). |
| `POST`   | `/webhooks/complete/{brokerTaskId}` | Called by fleet-manager; forwards JSON to the original caller                                                                                                                                                                                   |

```bash
export DATABASE_PATH=./broker.db
export LISTEN_ADDR=:8081
export BROKER_PUBLIC_URL=http://127.0.0.1:8081   # fleet-manager must reach this
./bin/task-broker
```

## Run fleet-manager

| Environment variable | Default      | Description                                                  |
| -------------------- | ------------ | ------------------------------------------------------------ |
| `LISTEN_ADDR`        | `:8080`      | HTTP listen address                                          |
| `DATABASE_PATH`      | `./fleet.db` | SQLite database file                                         |
| `AUTH_TOKEN`         | (empty)      | If set, requires `Authorization: Bearer <token>` for `/v1/*` |
| `REAP_INTERVAL_SEC`  | `15`         | How often to return expired leases to the queue              |

```bash
export DATABASE_PATH=./fleet.db
./bin/fleet-manager
```

## Run the runner

| Environment variable | Description                                                                                            |
| -------------------- | ------------------------------------------------------------------------------------------------------ |
| `FLEET_MANAGER_URL`  | **Required.** Base URL of **that fleet’s** fleet-manager (not the broker unless you bypass the broker) |
| `RUNNER_ID`          | Optional; defaults to host name or a random id                                                         |
| `AUTH_TOKEN`         | Optional; must match fleet-manager if set                                                              |
| `POLL_EMPTY_MS`      | Sleep when no work (default ~1000 ms)                                                                  |

```bash
export FLEET_MANAGER_URL=http://127.0.0.1:8080
export AUTH_TOKEN= # if fleet-manager uses it
./bin/runner
```

## HTTP API — fleet-manager (v1)

- `GET /healthz` — liveness
- `POST /v1/tasks` — enqueue: **`command`** (argv for one process) **or** **`commands`** (string lines concatenated into one `sh -c` script so `export` / `cd` persist), **`webhook_url`**, optional `execution_mode` / `docker_image`
- `POST /v1/tasks/claim` — runner pulls the next task
- `POST /v1/tasks/{id}/complete` — runner reports result

When the broker is **not** in the path, fleet-manager POSTs the completion **webhook** to `webhook_url` with `task_id`, `status`, `exit_code`, `output`, optional `error` (no `fleet_task_id`).

## End-to-end with the broker

1. Start **fleet-manager** and **runner** for that fleet (runner points at fleet-manager URL).
2. Start **task-broker** with `BROKER_PUBLIC_URL` reachable from fleet-manager.
3. `POST /v1/fleets` on the broker with this fleet-manager’s **`base_url`** (and labels for routing).
4. **`POST /v1/tasks`** on the broker — use caller `webhook_url` as today; broker substitutes an internal relay URL when talking to fleet-manager.
5. Caller receives a webhook whose **`task_id` is the broker task id**, with **`fleet_task_id`** set to the fleet-managed id.

## Example: enqueue directly on fleet-manager (curl)

```bash
curl -X POST http://127.0.0.1:8080/v1/tasks \
  -H 'Content-Type: application/json' \
  -d '{
    "commands": ["echo hello", "echo done"],
    "webhook_url": "https://example.com/your-hook"
  }'
```

## Example: register fleet and enqueue via task-broker (curl)

```bash
# Register a fleet-manager (repeat per region/environment)
curl -X POST http://127.0.0.1:8081/v1/fleets \
  -H 'Content-Type: application/json' \
  -d '{
    "id": "aws-standard-1",
    "base_url": "http://127.0.0.1:8080",
    "labels": ["aws", "standard"]
  }'

curl -X POST http://127.0.0.1:8081/v1/tasks \
  -H 'Content-Type: application/json' \
  -d '{
    "fleet_id": "aws-standard-1",
    "commands": ["export A=1", "echo $A"],
    "webhook_url": "https://example.com/your-hook"
  }'
```

## Tests

```bash
go test ./...
# e2e (subprocess fleets + brokers):
go test ./test/... -v
```

## CI (Semaphore)

The pipeline definition is [.semaphore/semaphore.yml](.semaphore/semaphore.yml).

In [Semaphore](https://semaphoreci.com/), create a **new project from this Git repository**. Semaphore 2.x picks up `.semaphore/semaphore.yml` on the default branch. Each push runs Go **1.22** on Ubuntu **24.04**: module cache restore/store, **`gofmt` check**, **`go vet`**, **`make build`**, **`go test ./...`**.

### Container images → GitHub Container Registry (GHCR)

Publishing runs in [.semaphore/docker-publish.yml](.semaphore/docker-publish.yml); [.semaphore/semaphore.yml](.semaphore/semaphore.yml) promotes it via `pipeline_file: docker-publish.yml` (path relative to the `.semaphore/` directory).

When **Build and test** completes with **`passed`**, Semaphore **[auto-promotes](https://docs.semaphoreci.com/using-semaphore/promotions)** only for the **`main`** branch.

**Tags** (`v1.2.3`, etc.) and other branches do **not** auto-publish — run promotion **Publish Docker images to GHCR** manually from a finished pipeline in the Semaphore UI when you want images for those refs.

Images are tagged as **`ghcr.io/<github-owner>/<repo>/<service>`** with:

- **`fleet-manager`**, **`task-broker`**, **`runner`** as the `<service>` name  
- A **12-character commit SHA** tag on every qualifying run  
- **`latest`** only when publishing a **`main`** branch workflow  
- The **literal git tag name** when the workflow is triggered by **`v*`** tags  

**Semaphore setup**

1. In GitHub, create a [**personal access token (classic)**](https://docs.github.com/en/packages/learn-github-packages/publishing-and-managing-packages/publishing-docker-images) (or organization-level bot PAT) with at least **`read:packages`** and **`write:packages`**. SSO-enabled orgs must **authorize** the token for that org.

2. In Semaphore: **Secrets** → create a secret named exactly **`ghcr`** with environment variables **`GHCR_USERNAME`** (your GitHub username or the PAT owner account) and **`GHCR_TOKEN`** (the PAT value).

For private packages, configure visibility under **GitHub → Packages** after the first successful push.

## License

(Add your license.)
