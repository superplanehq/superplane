# Local development

The SuperPlane repo root starts the task-broker and runner workers. Use the
same three commands as the app.

## Start

From the SuperPlane repo root:

```bash
make dev.up
make dev.setup
make dev.server
```

| Command | App | Runner stack |
| --- | --- | --- |
| `make dev.up` | Build the app image. Start `db`, `rabbitmq`, and the idle app shell. | Build the task-broker and worker images. |
| `make dev.setup` | Install npm and Go modules. Generate protos. Create and migrate `superplane_dev`. Create database `broker` on that Postgres. | Start task-broker so GORM migrates `broker`. Register fleets `local` and `e1-*`. |
| `make dev.server` | Start air and Vite. | Start 10 runner workers. Override the count with `N=1 make dev.server`. |

The broker listens on **http://127.0.0.1:8091**. SuperPlane pgweb uses host
port **8081**.

Stop both stacks with `make dev.down`. After you change
`runner/Dockerfile.local`, run `make dev.up` again so Compose rebuilds the
worker image.

If a sibling `../runner` Compose project still holds port `8091`, stop it
first. Do not run two brokers at the same time.

## Connect SuperPlane

`docker-compose.dev.yml` already defaults to this stack:

```
TASK_BROKER_BASE_URL=http://task-broker:8081
TASK_BROKER_PUBLIC_URL=http://localhost:8091
TASK_BROKER_AUTH_TOKEN=dev-local-token
WEBHOOKS_BASE_URL=http://app:8000
```

Do not set `TASK_BROKER_*` in SuperPlane `.env` unless you want a remote
broker.

If SuperPlane `.env` sets `WEBHOOKS_BASE_URL` to a tunnel, the broker posts
there instead. GitHub needs that tunnel. The local worker needs a URL it can
reach; keep the compose default for local Runner nodes.

## Test a Runner node

1. Open SuperPlane at http://localhost:8000.
2. Add a Runner, Run Bash, Run Python, Run JavaScript, Run Claude Code, Run
   OpenRouter Agent, or Run Codex node.
3. Run the canvas. A local worker claims the task.

Live logs in the UI stream from `http://localhost:8091`. The local worker does
not use CloudWatch. Host-mode tasks get a fresh `HOME` each run
(`RUNNER_RESET_TASK_HOME`). Leftover `repo/` dirs from a prior factory
clone cannot leak into the next task.

## Tools on the local worker

The Compose worker image (`runner/Dockerfile.local`) includes bash, Python 3,
Node.js 22, git, GitHub CLI (`gh`), jq, openssl, Docker CLI, Claude Code,
OpenCode, and Codex. Factory line apps can run on this worker. Do not install
those CLIs on the host. Host NVM binaries are not on the container PATH.

Check the tools after `make dev.server`:

```bash
make doctor-local
```

Factory agent nodes use a SuperPlane integration, not a host API key. Connect
GitHub and the applicable agent integration before you dispatch a factory line.

## Manual start

Use these targets from this directory when you debug the broker and workers
without the Compose task-broker service. Postgres must already be running
(`make dev.up` from the repo root). `make task-broker` creates database
`broker` when that database is missing.

```bash
make task-broker
make register-local-fleet
make register-superplane-fleets
make runner
```

`make task-broker` runs on the Compose network so it can open Postgres at
`db:5432`. That port stays unpublished. The broker listens on
**127.0.0.1:8081**. Compose `make dev.server` uses **8091**. Do not run both
brokers at the same time. They share database `broker`. Enqueue with
`Authorization: Bearer dev-local-token` and `"fleet_id":"local"`.

Host `make runner` inherits the shell PATH. Tasks run `bash --norc
--noprofile`, so NVM hooks in `.bashrc` do not load. Prefer the root
`make dev.server` path.
