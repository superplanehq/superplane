# Architecture

This document describes the distributed task execution system: **fleet-manager** queues work, notifies callers, and coordinates a fleet of **runners**. Each **runner** pulls tasks and executes bash on the host or inside Docker. Runners can be deployed in many environments (for example EC2).

## Roles

| Role | Responsibility |
|------|------------------|
| **fleet-manager** | Accept tasks, persist them, expose queue semantics, track lifecycle, invoke completion webhooks, optionally provision additional runners. |
| **runner** | Long-lived worker that claims work, executes bash (host or Docker), reports stdout/stderr, exit code, and terminal state. |
| **Callers** | Clients that submit tasks and receive asynchronous notification at a URL provided with each task (webhook). |

## fleet-manager

### Responsibilities

1. **Task submission API** — Accept tasks (e.g. command(s), optional env/cwd, **webhook URL**, labels, timeout, execution mode: `host` vs `docker` plus image/reference).
2. **Durable queue** — Tasks move through explicit states: `queued` → `claimed` → `running` → `succeeded` | `failed` | `timeout` | `canceled`.
3. **Claim semantics** — Runners receive **at-most-one** task per claim (lease). Claims expire so stuck runners do not block work indefinitely.
4. **Completion path** — When a run finishes, persist the result, then **POST to the task’s webhook** with a defined payload (task id, status, exit code, log summary or references, timestamps). Retry with backoff on failure; optionally record webhook delivery failures.
5. **Runner registry (optional)** — Runners register with id, capabilities (e.g. Docker available, labels), and heartbeat; fleet-manager can use this for routing or capacity visibility.
6. **Runner provisioning (optional)** — Express “need N runners” or “scale pool X” as infrastructure actions (e.g. EC2 Auto Scaling, ECS, Kubernetes Jobs). Keep this as a separate module from core queue logic.

Implementation detail (storage): use a store that supports safe concurrent claiming — for example Postgres with `SKIP LOCKED`, Redis streams, or a message queue with explicit lease/visibility timeouts.

## Runner

### Responsibilities

1. **Connect / poll** — Claim the next task from fleet-manager (HTTP long poll or repeated claim calls; optional WebSocket for push).
2. **Execute** — Run bash:
   - **Host** — Subprocess with timeouts; honor cancellation signals; stream logs upstream as configured.
   - **Docker** — Run the command in a container (image from task spec); same timeout/cancel behavior.
3. **Report** — Send progress and final status back to fleet-manager (streaming logs, exit code, failure reason).

Runners should remain **stateless** with respect to queue policy: they execute assigned work and report results; they do not decide global ordering or webhook retries.

## End-to-end flows

### Submit

1. Caller submits a task to fleet-manager (including webhook URL).
2. fleet-manager enqueues the task and returns an identifier (e.g. `202 Accepted` with task id).

### Execute and notify

1. Runner claims a task (receives lease + task spec).
2. Runner executes on host or in Docker.
3. Runner reports completion to fleet-manager.
4. fleet-manager persists the outcome and **asynchronously** POSTs to the caller’s webhook (with retries as needed).

## Spinning up runners

Provisioning is **orthogonal** to task claiming:

- fleet-manager (or a separate operator) adjusts capacity — API such as `POST /pools/:id/scale`, or direct integration with cloud APIs.
- **EC2 pattern** — Auto Scaling Group + user data that installs the runner, injects fleet-manager’s base URL and auth credentials, and starts the runner (e.g. systemd). Instances self-register and begin claiming work.
- The same runner binary can run on developer machines, bare metal, or cloud VMs; only configuration and secrets differ.

## Design decisions to finalize

- **Webhook contract** — JSON schema, optional HMAC signing, idempotency keys for safe retries.
- **Log handling** — Inline in webhook payload vs URLs to large artifacts (e.g. object storage).
- **Security** — Authentication for callers and runners (tokens, mTLS); whether arbitrary shell is acceptable or sandboxes/allowlists are required.
- **Fairness and isolation** — Single queue vs priorities; per-tenant quotas and rate limits.

## Repository layout (this monorepo)

| Path | Concern |
|------|---------|
| `task-broker/` | Proxies tasks to registered fleet-managers (`internal/broker`), fleet registry + routing SQLite (`internal/store`), `cmd/task-broker`; receives downstream completion webhooks and forwards to caller URLs |
| `fleet-manager/` | HTTP API (`internal/fleetmanager`), persistence (`internal/store`), `cmd/fleet-manager`; delivers completion webhooks (shared sender in `shared/webhook`) |
| `runner/` | Worker agent (`internal/agent`), `cmd/runner` |
| `shared/api`, `shared/models`, `shared/webhook` | Shared types and webhook retry client used by fleet-manager and task-broker |

**EC2 hot pool** (optional): when `EC2_PROVISION_HOT_INSTANCE_COUNT`, `EC2_PROVISION_*` network/AMI settings, and `AWS_REGION` are set, fleet-manager **reconciles on a timer** toward that many `pending`+`running` instances tagged `superplane_managed_runner`; user-data installs Docker and starts the runner image against `EC2_PROVISION_FLEET_MANAGER_URL`. There is **no HTTP API** for provisioning—capacity is entirely **environment-driven**.

This separation keeps **orchestration and callbacks** in fleet-manager, **execution** in runners, and **infrastructure scaling** as an optional plug-in.
