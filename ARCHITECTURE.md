# Architecture

SuperPlane runs tasks through **task-broker** (queue + API), **runners** (execution), and optionally **fleet-manager** (EC2 capacity + health).

## Roles

| Role | Responsibility |
|------|----------------|
| **task-broker** | Postgres queue, fleet registry (labels for routing), runner WebSocket/HTTP APIs, caller webhooks, cancel, optional live-log proxy. SuperPlane talks here. |
| **runner** | Claims tasks from task-broker (`TASK_BROKER_URL`, `RUNNER_FLEET_ID`), executes host or Docker workloads, reports completion. Exposes `GET /healthz`. |
| **fleet-manager** | EC2 hot pool only: launch/terminate VMs, reconcile capacity, health-sweep runners via private IP. No task queue. |
| **Callers** | Submit tasks and receive webhooks at the URL provided with each task. |

## Data flow

```
Caller → task-broker (GET /v1/fleets, POST /v1/tasks with fleet_id, cancel, GET status, webhook)
Runner → task-broker (WS stream or claim/complete; scoped by fleet_id)
fleet-manager → EC2 (reconcile: health probe + RunInstances/TerminateInstances, per pool)
fleet-manager → runner private IP:9090/healthz
fleet-manager → task-broker (POST /v1/fleets on startup; GET /v1/fleets/{id}/task-counts for pools with headroom>0)
```

There is **no** task-broker → fleet-manager path, no runner → fleet-manager shutdown callback, and no `fleet_task_id` correlation field. The optional `fleet-manager → task-broker` direction is pull-only (fleet registration + task counts).

### Machine profiles (amd64 vs arm64, instance size)

Run **separate broker fleets** — one per homogeneous runner pool. Each pool has catalog metadata (`provisioner`, `arch`, `size`) registered on the broker. SuperPlane lists fleets via **`GET /v1/fleets`** and submits tasks with an explicit **`fleet_id`**. Use an x86_64 AMI + `t3.*` sizes for amd64 pools and an arm64 AMI + `t4g.*` Graviton sizes for arm64 pools. Fleet-manager auto-registers each `pools[]` entry on startup; local dev can register manually via `make register-local-fleet`.

## task-broker

- **Queue** — Postgres with `FOR UPDATE SKIP LOCKED` claims scoped by `fleet_id`.
- **Fleets** — Registered with `id` plus optional catalog fields `provisioner`, `arch`, `size` (EC2 instance type, e.g. `t3.micro`). Callers choose a fleet with **`fleet_id`** on `POST /v1/tasks`.
- **Runner transport** — Default WebSocket (`GET /v1/runners/stream`); optional HTTP claim/complete.
- **Webhooks** — Delivered directly to the caller `webhook_url` on terminal status.
- **Lease reap** — Background loop requeues or finalizes expired leases.

## fleet-manager (EC2)

- **Configuration** — One JSON file at `FM_CONFIG_FILE` (default `/etc/fleet-manager/config.json`) is the sole runtime input. The file has a global section (AWS region, broker URL, subnet, security groups, IAM profile, CloudWatch, …) plus a `pools[]` array — one entry per VM pool. See `fleet-manager/internal/config` and the example at `scripts/deploy/fleet-manager.config.example.json`.
- **Multi-pool** — One fleet-manager process can manage multiple pools (e.g. one amd64 pool, one arm64 pool). Each pool is a separate `Launcher` keyed by `fleet_id`, with its own AMI / instance type / runner-binary S3 URI / hot count / headroom. Pools are reconciled **serially** per tick (predictable EC2 Describe rate).
- **Broker registration** — On startup, registers each pool on task-broker (`POST /v1/fleets`) with `provisioner` (`aws`), `arch`, and `size` (from `pools[].instance_type`).
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
