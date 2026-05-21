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

### Executor abstraction

Both modes implement a small `Executor` interface (`runner/internal/agent/executor.go`) with a single `Execute(ctx, task, live)` method. The agent loop picks `HostExecutor` or `DockerExecutor` based on `task.execution_mode` and is otherwise oblivious to how the task is run; when a CloudWatch log group is configured it passes a `live io.Writer` so each executor streams stdout/stderr to CloudWatch. Completion webhooks and status responses expose **`task_log`** pointers, not inline log text. New backends (Kubernetes, Firecracker, …) plug in as additional implementations without touching the claim/complete code paths.

### Docker execution lifecycle

The `DockerExecutor` follows a `pull → run -d --name → exec → stop+rm` lifecycle modeled after the docker-compose executor in [`semaphoreci/agent`](https://github.com/semaphoreci/agent) (`pkg/executors/docker_compose_executor.go`). For exec/stop orchestration alongside other backends, see `pkg/executors/shell_executor.go`. Semaphore’s interactive shell and PTY plumbing live under `pkg/shell/` (for example `pkg/shell/shell.go`); Superplane does not reuse that stack—tasks run via `docker exec` without a PTY (see runner README).

1. **Pull** — `docker pull <image>` first, so a bad image or auth failure surfaces as a clean error before user code runs. When CloudWatch live streaming is configured, pull output is copied to the live log sink on **success** as well as on failure, so long pulls are visible before `docker exec` starts.
2. **Run** — `docker run -d --entrypoint sleep --name superplane-task-<runner_id>-<task_id> <image> infinity` starts a long-lived idle container. The deterministic name lets cancellation and the orphan sweep target it reliably; including `runner_id` keeps multiple runners on the same host from stomping on each other's containers. Successful `docker run -d` diagnostics are also copied to the live writer when present (same rationale as pull).
3. **Exec** — For argv `command`, the runner invokes `docker exec --env NAME=value <name> <argv...>` when task environment variables are present. For multi-line `commands`, all directives are bundled into a single `docker exec --env NAME=value <name> sh -c '<script>'` with `set -e` so env / cwd persist across lines and the script fails fast on the first non-zero exit. Task environment is intentionally applied to `docker exec`, not the idle `docker run` container config.
4. **Cleanup** — A deferred `docker stop --time 5 && docker rm -f` runs even when the task context is cancelled or `Execute` panics, fixing the "kill the docker CLI but the container keeps running" leak.

On startup the runner additionally runs an **orphan sweep**: any `superplane-task-<runner_id>-…` container left behind by a previous crashed run (where the per-task `defer` did not get to execute) is force-removed. The sweep is filtered by this runner's id, so it never touches another runner's containers.

If `docker` is not on PATH (or the daemon is unreachable) and a Docker task is claimed, the executor fails the task immediately with a `"docker not available on this runner"` error. Capability-based claim filtering (so Docker-less runners never claim Docker tasks in the first place) is a separate runner-labels concern.

**Migration (fleet already on a pre–DockerExecutor runner):** If production ever used the legacy path (PTY + interactive bash inside `docker run` for multi-line `commands`), upgrading changes TTY availability, shell dialect (`bash` → POSIX `sh`), and lifecycle (`run --rm` style vs pull / long-lived container / exec). Operators and task authors should read the README “Upgrade note: Docker multi-line `commands`” and validate representative Docker tasks after deploy.

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
- **Log handling** — CloudWatch via **`task_log`** on completion webhooks and status; optional live tail via task-broker.
- **Security** — Authentication for callers and runners (tokens, mTLS); whether arbitrary shell is acceptable or sandboxes/allowlists are required.
- **Fairness and isolation** — Single queue vs priorities; per-tenant quotas and rate limits.

## Repository layout (this monorepo)

| Path | Concern |
|------|---------|
| `task-broker/` | Proxies tasks to registered fleet-managers (`internal/broker`), fleet registry + routing SQLite (`internal/store`), `cmd/task-broker`; receives downstream completion webhooks and forwards to caller URLs |
| `fleet-manager/` | HTTP API (`internal/fleetmanager`), persistence (`internal/store`), `cmd/fleet-manager`; delivers completion webhooks (shared sender in `shared/webhook`) |
| `runner/` | Worker agent (`internal/agent`), `cmd/runner` |
| `shared/api`, `shared/models`, `shared/webhook` | Shared types and webhook retry client used by fleet-manager and task-broker |

**EC2 hot pool** (optional): when `EC2_PROVISION_HOT_INSTANCE_COUNT`, `EC2_PROVISION_*` network/AMI settings, **`EC2_PROVISION_RUNNER_S3_URI`** ( **`aws s3 cp`**) **or** **`EC2_PROVISION_RUNNER_BINARY_URL`** (**`curl`**), **`AWS_REGION`**, and (for S3) **`EC2_PROVISION_RUNNER_INSTANCE_PROFILE`**, fleet-manager **reconciles on a timer** toward that many `pending`+`running` instances tagged `superplane_managed_runner`; user-data installs Docker, installs the **`runner`** binary on the Ubuntu host, and runs **systemd** against `EC2_PROVISION_FLEET_MANAGER_URL`. There is **no HTTP API** for provisioning—capacity is entirely **environment-driven**.

This separation keeps **orchestration and callbacks** in fleet-manager, **execution** in runners, and **infrastructure scaling** as an optional plug-in.
