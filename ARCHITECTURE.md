# Architecture

SuperPlane runs tasks through **task-broker** (queue + API), **runners** (execution), and optionally **fleet-manager** (EC2 capacity + health).

## Roles

| Role | Responsibility |
|------|----------------|
| **task-broker** | Postgres queue, fleet registry, runner WebSocket/HTTP APIs, caller webhooks. SuperPlane talks here. |
| **runner** | Claims tasks from task-broker (`TASK_BROKER_URL`, `RUNNER_FLEET_ID`), executes bash (host or Docker), reports completion. Exposes `GET /healthz`. |
| **fleet-manager** | EC2 hot pool only: launch/terminate VMs, reconcile capacity, health-sweep runners via private IP. No task queue. |
| **Callers** | Submit tasks and receive webhooks at the URL provided with each task. |

## Data flow

```
Caller → task-broker (POST /v1/tasks, cancel, GET status, webhook)
Runner → task-broker (WS stream or claim/complete; scoped by fleet_id)
fleet-manager → EC2 (reconcile: health probe + RunInstances/TerminateInstances, per pool)
fleet-manager → runner private IP:9090/healthz
fleet-manager → task-broker (GET /v1/fleets/{id}/task-counts, only for pools with headroom>0)
```

There is **no** task-broker → fleet-manager path, no runner → fleet-manager shutdown callback, and no `fleet_task_id` correlation field. The optional `fleet-manager → task-broker` direction is pull-only.

## task-broker

- **Queue** — Postgres with `FOR UPDATE SKIP LOCKED` claims scoped by `fleet_id`.
- **Fleets** — Registered with `id` + `labels` (routing via `fleet_id` or `fleet_labels` on create).
- **Runner transport** — Default WebSocket (`GET /v1/runners/stream`); optional HTTP claim/complete.
- **Webhooks** — Delivered directly to the caller `webhook_url` on terminal status.
- **Lease reap** — Background loop requeues or finalizes expired leases.

## fleet-manager (EC2)

- **Configuration** — One JSON file at `FM_CONFIG_FILE` (default `/etc/fleet-manager/config.json`) is the sole runtime input. The file has a global section (AWS region, broker URL, subnet, security groups, IAM profile, CloudWatch, …) plus a `pools[]` array — one entry per VM pool. See `fleet-manager/internal/config` and the example at `scripts/deploy/fleet-manager.config.example.json`.
- **Multi-pool** — One fleet-manager process can manage multiple pools (e.g. one amd64 pool, one arm64 pool). Each pool is a separate `Launcher` keyed by `fleet_id`, with its own AMI / instance type / runner-binary S3 URI / hot count / headroom. Pools are reconciled **serially** per tick (predictable EC2 Describe rate).
- **Reconcile loop** — Per pool, per tick: health sweep (`GET /healthz` after boot grace) then scale toward a target. Target is `pools[].hot_instance_count` by default, or `queued + claimed + pools[].headroom` when headroom is set (counts pulled from task-broker per fleet). On broker failure, that pool's tick falls back to its static count.
- **Instance partitioning** — Every managed EC2 instance is tagged with `superplane_managed_runner=true` *and* `superplane_fleet_id=<pool.fleet_id>`. All Describe calls (reconcile, sweep, admin listing) filter by both tags, so pools sharing one AWS account never terminate each other's instances.
- **User-data** — Installs runner from S3, sets `TASK_BROKER_URL`, `RUNNER_FLEET_ID`, `RUNNER_HEALTH_ADDR`, optional `RUNNER_TERMINATE_AFTER_EACH_TASK`.
- **Admin** — Optional `/v1/admin/*` diagnostics (managed instances across all pools, EC2 console output). Gated by the `diagnostics_token` field.

## Runner

See existing sections below for executor abstraction, Docker lifecycle, and structured results (`SUPERPLANE_RESULT_FILE`).

## Repository layout

| Path | Concern |
|------|---------|
| `task-broker/` | Queue store (Postgres), HTTP + WebSocket handlers, webhooks |
| `fleet-manager/` | EC2 provisioning + reconcile (`internal/ec2provision`) |
| `runner/` | Worker agent + `/healthz` |
| `shared/` | API types, models, webhook client, WebSocket messages |
