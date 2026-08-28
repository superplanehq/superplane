# Local development

This repository owns the local task-broker and runner worker. SuperPlane does
not start these services.

## Start

```bash
make dev
```

That command starts Postgres, task-broker, SuperPlane fleets (`local` and
`e1-*`), and one runner worker.

The broker listens on **http://127.0.0.1:8091**. SuperPlane pgweb already uses
host port **8081**.

Stop the stack with `make dev.down`.

## Connect SuperPlane

SuperPlane `docker-compose.dev.yml` already defaults to this stack:

```
TASK_BROKER_BASE_URL=http://host.docker.internal:8091
TASK_BROKER_PUBLIC_URL=http://localhost:8091
TASK_BROKER_AUTH_TOKEN=dev-local-token
TASK_BROKER_FLEET_ID=local
WEBHOOKS_BASE_URL=http://host.docker.internal:8000
```

Start SuperPlane with `make dev.up` then `make dev.server`. Do not set
`TASK_BROKER_*` in SuperPlane `.env` unless you want a remote broker.

If SuperPlane `.env` sets `WEBHOOKS_BASE_URL` to a tunnel, the broker posts
there instead. GitHub needs that tunnel. The local worker needs a URL it can
reach; keep the compose default for local Runner nodes.

Do not run two brokers at the same time.

## Test a Runner node

1. Open SuperPlane at http://localhost:8000.
2. Add a Runner, Run Bash, Run Python, or Run JavaScript node.
3. Run the canvas. The local worker claims the task.

Live logs in the UI stream from `http://localhost:8091`. The local worker does
not use CloudWatch.

## Claude Code on the local worker

The local runner image includes bash, Python, Node.js, git, and Docker CLI.
It does not include the Claude CLI. To test Run Claude Code on your machine,
run a host worker:

```bash
make register-local-fleet
make runner
```

Keep `TASK_BROKER_FLEET_ID=local` on SuperPlane so tasks still go to that worker.

## Manual start

Use separate terminals:

```bash
make task-broker
make register-local-fleet
make register-superplane-fleets
make runner
```

Host `make task-broker` listens on **:8081**. Compose `make dev` uses **8091**.
Enqueue with `Authorization: Bearer dev-local-token` and `"fleet_id":"local"`.
