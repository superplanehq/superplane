# SuperPlane runner

Monorepo (single Go module) for **task-broker**, **fleet-manager**, and the **runner** worker.

- **Task-broker** — SuperPlane front door: registers runner fleets, owns the Postgres task queue, serves runner WebSocket/HTTP APIs, and delivers completion webhooks to callers.
- **Fleet-manager** — EC2-only: maintains a hot pool of runner VMs, health-sweeps `GET /healthz` on private IPs, and reconciles capacity. No task queue.
- **Runner** — worker: connects to **task-broker** (`TASK_BROKER_URL`, `RUNNER_FLEET_ID`), claims tasks, executes `command` or **`commands`**, completes results. Exposes **`GET /healthz`** on `RUNNER_HEALTH_ADDR` (default `:9090`).

Shared JSON types live under **`shared/`**; webhook retries use **`shared/webhook`**.

See [ARCHITECTURE.md](./ARCHITECTURE.md) for design detail.

## Layout

```
.
├── task-broker/            # fleet registry + Postgres queue + runner API + webhooks
│   ├── cmd/task-broker/
│   └── internal/…
├── fleet-manager/          # EC2 hot pool + health reconcile (optional)
│   ├── cmd/fleet-manager/
│   └── internal/…
├── runner/                 # worker agent
│   ├── cmd/runner/
│   └── internal/agent/
├── shared/
│   ├── api/               # REST DTOs (incl. broker types)
│   ├── models/
│   └── webhook/           # retrying POST client
├── test/                  # e2e tests (task-broker + runner)
├── go.mod
└── Makefile
```

## Requirements

- Go 1.22+
- For Docker tasks: Docker CLI **and a reachable Docker daemon** on the runner host. The runner uses a pull → long-lived named container → `docker exec` → `docker stop`/`rm` lifecycle (see **Docker** below and [ARCHITECTURE.md](./ARCHITECTURE.md)). The image must include `sleep` (alpine, debian, ubuntu, python:*, node:* all satisfy this). Multi-line **`commands`** are bundled into one `sh -c` script with `set -e`, so env/cwd persist across directives and the script fails fast on the first non-zero exit. Task **`environment`** entries are passed to the `docker exec` process, not to the idle `docker run` container. **Quoting:** each directive is a line inside a single-quoted `sh -c` argument; a raw **`'`** in a line is a classic shell-quoting footgun—avoid it in `commands` or use argv **`command`** for tricky literals. **`docker exec` is invoked without `-t`**, so the task runs in a non-TTY context: tools that detect `isatty()` (color output, progress bars, interactive prompts) will see stdout/stderr as a pipe. This is intentional — matches `docker run` without `-t`, more predictable for CI / batch workloads, and lets stdout and stderr stay distinct in captures.

### Upgrade note: Docker multi-line `commands` (breaking if you relied on the old runner)

Older runner builds ran multi-line **`commands`** through **`docker run` with a PTY** and **interactive bash** in the container. Current runners use **`docker pull` → `docker run -d` → `docker exec` … `sh -c '…'`** with **no PTY**. Anything that depended on a **TTY**, **bash-only** syntax (e.g. `[[ ]]`, bashisms not in POSIX `sh`), or **interactive** behavior may break or change. Prefer argv **`command`** for strict control, or adjust scripts for **`sh`**. When **CloudWatch live** logging is enabled, **`docker pull` / `docker run -d` diagnostics** are copied to the live stream on success; they are **not** duplicated in completion payloads. Task stdout/stderr are streamed to CloudWatch; completion webhooks and **`GET /v1/tasks/{id}`** expose **`task_log`** (not inline **`output`**).

## Build

```bash
make build
# or:
go build -o bin/fleet-manager ./fleet-manager/cmd/fleet-manager
go build -o bin/runner ./runner/cmd/runner
go build -o bin/task-broker ./task-broker/cmd/task-broker
```

### Local dev

Use **separate terminals**: **`make task-broker`**, then **`make register-local-fleet`**, then **`make runner`** (optional **`N=3`** runner processes; default **`N=1`**). **`make fleet-manager`** is only needed when testing the EC2 provisioner locally. **`make local-dev-help`** lists this. Defaults (`LOCAL_*`, **`LOCAL_STACK_AUTH_TOKEN`**, **`LOCAL_RUNNER_*`**, **`N`**) are in the **`Makefile`**; override on the command line when needed. Enqueue on the broker with **`Authorization: Bearer dev-local-token`** and **`"fleet_id":"local"`** (unless you changed **`LOCAL_STACK_AUTH_TOKEN`** / **`LOCAL_FLEET_ID`**).

## Run task-broker

| Environment variable | Default       | Description                                                                                                              |
| -------------------- | ------------- | ------------------------------------------------------------------------------------------------------------------------ |
| `LISTEN_ADDR`        | `:8081`       | HTTP listen address                                                                                                      |
| `DATABASE_URL`       | —             | **Required.** PostgreSQL connection string (fleets + task queue)                                                       |
| `AUTH_TOKEN`         | —             | **Required.** `Authorization: Bearer …` for all **`/v1/*`** routes (including runner claim/complete and live-logs JWT issuance) |
| `REAP_INTERVAL_SEC`  | `15`          | How often to requeue expired leases or finalize canceled tasks                                                           |
| `TASK_CLOUDWATCH_LOG_GROUP` | (empty) | When set, `GET /v1/tasks/{id}` and completion webhooks include **`task_log`** (and legacy `cloudwatch_log_*` fields) pointing at the stream the runner writes to |
| `TASK_CLOUDWATCH_LOG_STREAM_PREFIX` | (empty) | Optional; stream name is `{prefix}/{task_id}` (see `shared/cwstream`). Must match **`RUNNER_CLOUDWATCH_LOG_STREAM_PREFIX`** on runners. |
| `TASK_CLOUDWATCH_REGION` | (empty) | Optional AWS region in **`task_log.cloudwatch.region`** |
| `TASK_BROKER_LIVE_LOGS_CORS_ORIGINS` | (empty) | Optional comma-separated origins for **`GET /v1/tasks/{id}/live-logs`** CORS |

**Logging:** stdout emits JSON **`http_access`** per request (**method**, **path**, **dur**, **status**, **bytes**, **remote**, optional **request_id** / **ua**). **`GET /healthz`** is skipped to reduce load-balancer noise. Outbound caller webhooks log **`webhook_delivery`** per attempt (**attempt**, **task_id**, **status_outcome**, **url_host**, **dur**, **http_status** or **err**).

**HTTP (`/v1`, Bearer auth unless noted)**

| Method   | Path                                | Notes                                                                                                                                                                                                                                           |
| -------- | ----------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `GET`    | `/fleets`                           | List registered runner pools (`id`, `provisioner`, `arch`, `size`, …)                                                                                                                                                                           |
| `POST`   | `/fleets`                           | Register/upsert a fleet (`id`, optional `provisioner`, `arch`, `size`)                                                                                                                                                                          |
| `GET`    | `/fleets/{id}/task-counts`          | Queued/claimed counts for a fleet (fleet-manager dynamic scaling)                                                                                                                                                                               |
| `DELETE` | `/fleets/{id}`                      | Remove a fleet                                                                                                                                                                                                                                  |
| `POST`   | `/tasks`                            | Body: `BrokerCreateTaskRequest` — task fields (`command` xor `commands`, `webhook_url`, execution mode…) plus required **`fleet_id`**                                                                                                           |

```bash
export DATABASE_URL='postgres://broker:broker@127.0.0.1:5432/broker?sslmode=disable'
export LISTEN_ADDR=:8081
export AUTH_TOKEN=your-secret
./bin/task-broker
```

Local dev expects Postgres on `127.0.0.1:5432` with database `broker` (see `LOCAL_BROKER_DATABASE_URL` in the `Makefile`). GORM auto-migrates schema on startup.

**Tests** that touch the broker store require `TEST_DATABASE_URL` (same format as `DATABASE_URL`). CI starts Postgres via `sem-service`; locally run e.g. `docker run -d --name broker-pg -e POSTGRES_USER=broker -e POSTGRES_PASSWORD=broker -e POSTGRES_DB=broker -p 5432:5432 postgres:16-alpine` and export `TEST_DATABASE_URL=postgres://broker:broker@127.0.0.1:5432/broker?sslmode=disable`. Use `go test ./... -p 1` when sharing one test database.

**Inspect upstream task status** (uses `AUTH_TOKEN` and broker base from **`scripts/deploy/task-broker.env`** unless you export overrides): `./scripts/check-broker-task.sh <broker_task_id>`

**Inspect recent broker queue rows** (requires local `psql`): `TASK_BROKER_DATABASE_URL='postgres://…' ./scripts/show-runner-queue-state.sh`

## Run fleet-manager

Without **`EC2_PROVISION_HOT_INSTANCE_COUNT`**, fleet-manager starts but only serves **`GET /healthz`** (and optional **`/v1/admin/*`** when **`FLEET_DIAGNOSTICS_TOKEN`** is set). The EC2 hot pool is enabled when **`EC2_PROVISION_*`** is configured (see **`scripts/deploy/fleet-manager.env.example`**).

| Environment variable | Default      | Description                                                  |
| -------------------- | ------------ | ------------------------------------------------------------ |
| `LISTEN_ADDR`        | `:8080`      | HTTP listen address                                          |
| `FLEET_DIAGNOSTICS_TOKEN` | (empty) | When set, protects **`GET /v1/admin/managed-runners`** and **`GET /v1/admin/ec2-console-output`** |

**Task log descriptor:** When **`TASK_CLOUDWATCH_LOG_GROUP`** is set, `GET /v1/tasks/{id}` and completion webhooks include **`task_log`** with `{"type":"cloudwatch","cloudwatch":{"log_group_name","log_stream_name","region"}}`. Otherwise **`task_log`** is omitted. Legacy **`cloudwatch_log_group`** / **`cloudwatch_log_stream`** fields are still present when CloudWatch is enabled.

Optional **EC2 hot runner pool** — set **`AWS_REGION`** (also used as **`AWS_DEFAULT_REGION`** inside user-data for **`aws s3 cp`**), **`EC2_PROVISION_HOT_INSTANCE_COUNT`**, **`EC2_PROVISION_AMI_ID`**, **`EC2_PROVISION_SUBNET_ID`**, **`EC2_PROVISION_SECURITY_GROUP_IDS`**, **`EC2_PROVISION_FLEET_MANAGER_URL`**, **`EC2_PROVISION_RUNNER_S3_URI`**, and **`EC2_PROVISION_RUNNER_INSTANCE_PROFILE`**. **`EC2_PROVISION_RUNNER_S3_URI`** points to the static **`runner`** binary for this fleet-manager's architecture, and the runner instance profile needs **`s3:GetObject`** on that object.

Optional: **`EC2_PROVISION_RUNNER_CLOUDWATCH_LOG_GROUP`**, **`EC2_PROVISION_RUNNER_CLOUDWATCH_LOG_STREAM_PREFIX`** — written into **`/etc/default/superplane-runner`** as **`RUNNER_CLOUDWATCH_*`** (requires the runner instance profile to allow **`logs:CreateLogGroup`**, **`logs:CreateLogStream`**, **`logs:PutLogEvents`**, **`logs:DescribeLogStreams`** on that log group).

Fleet-manager **reconciles in the background** (default **60** s, **`EC2_PROVISION_RECONCILE_INTERVAL_SEC`**, minimum **15**). Optional: **`EC2_PROVISION_ARCH`** (`amd64` default, or `arm64`), **`EC2_PROVISION_FLEET_ID`** (stable name for this deployment; defaults to the host's OS hostname — **set this explicitly** when running multiple fleet-managers in the same AWS account to prevent cross-fleet reconcile interference), **`EC2_PROVISION_INSTANCE_TYPE`**, **`EC2_PROVISION_RUNNER_AUTH_TOKEN`**, **`EC2_PROVISION_KEY_NAME`**.

**Dynamic scaling (optional)** — set **`EC2_PROVISION_RUNNER_HEADROOM=N`** to make fleet-manager target **`want = queued + claimed + N`** for its **`EC2_PROVISION_RUNNER_FLEET_ID`** each reconcile tick (counts pulled from task-broker via **`GET /v1/fleets/{id}/task-counts`** using **`EC2_PROVISION_TASK_BROKER_URL`** + **`EC2_PROVISION_RUNNER_AUTH_TOKEN`**). Counting **queued** tasks (not only **claimed**) pre-warms capacity for a burst — when several tasks arrive at once, fleet-manager launches VMs in parallel instead of waiting for each one to be claimed first. Scale-down is automatic — when claimed/queued drops, want drops, and the existing oldest-first terminate logic removes excess VMs. When the broker call fails, the tick falls back to **`EC2_PROVISION_HOT_INSTANCE_COUNT`**. Leave **`EC2_PROVISION_RUNNER_HEADROOM`** unset for the previous static behavior. task-broker never initiates HTTP toward fleet-manager; communication is fleet-manager pull only.

Provisioner **user-data** installs **`/usr/local/bin/runner`**, runs **`superplane-runner.service`** as the **`ubuntu`** user (host tasks start in **`/home/ubuntu`**), and adds **`ubuntu`** to the **`docker`** group on **Ubuntu** AMIs.

#### Runner binary via S3 (typical setup)

1. **Bucket** (same account/region as runners is simplest). Upload the static binaries, e.g. **`runner-linux-amd64`** at **`s3://my-runner-binaries/release/runner-linux-amd64`** and **`runner-linux-arm64`** at **`s3://my-runner-binaries/release/runner-linux-arm64`** (`make runner-linux-all` builds both).

2. **IAM role for runners** (**instance profile** name = **`EC2_PROVISION_RUNNER_INSTANCE_PROFILE`**): attach an inline policy allowing **`s3:GetObject`** on **`arn:aws:s3:::my-runner-binaries/release/*`** (tighten to the exact key).

3. **Fleet-manager env**: run one fleet-manager per architecture. For amd64, use **`EC2_PROVISION_ARCH=amd64`**, an x86_64 Ubuntu AMI, an amd64 instance type such as **`t3.micro`**, and **`EC2_PROVISION_RUNNER_S3_URI=s3://my-runner-binaries/release/runner-linux-amd64`**. For arm64, use **`EC2_PROVISION_ARCH=arm64`**, an arm64 Ubuntu AMI, a Graviton instance type such as **`t4g.micro`**, and **`EC2_PROVISION_RUNNER_S3_URI=s3://my-runner-binaries/release/runner-linux-arm64`**. Both need **`AWS_REGION=us-east-1`** (or your region) and **`EC2_PROVISION_RUNNER_INSTANCE_PROFILE=…`**.

4. **CI**: on each release, upload both built runner artifacts to their architecture-specific keys. Semaphore uses **`EC2_PROVISION_RUNNER_AMD64_S3_URI`** and **`EC2_PROVISION_RUNNER_ARM64_S3_URI`** for this publish step.

By default **`EC2_PROVISION_RUNNER_TERMINATE_AFTER_TASK`** is **on** (`true`): **`runner_id`** is the EC2 instance id from IMDS; after **one** successful task, **fleet-manager** calls **`TerminateInstances`** and the systemd unit **`Restart=no`** stops respawn before shutdown. Set **`false`** for long-lived workers (**`Restart=always`**). Runner VMs do **not** need **`TerminateInstances`** on their profile for that flow; **fleet-manager’s** role must **`TerminateInstances`** (reconcile + disposable runners).

Fleet-manager still needs **`ec2:RunInstances`**, **`ec2:DescribeInstances`**, **`ec2:CreateTags`**, **`ec2:TerminateInstances`**, and **`iam:PassRole`** when using an instance profile on runners.

```bash
# EC2 pool disabled without EC2_PROVISION_HOT_INSTANCE_COUNT and related vars
./bin/fleet-manager
```

## Run the runner

| Environment variable | Description                                                                                            |
| -------------------- | ------------------------------------------------------------------------------------------------------ |
| `TASK_BROKER_URL`      | **Required.** Base URL of **task-broker**                                                              |
| `RUNNER_FLEET_ID`      | **Required.** Fleet id registered on the broker (`POST /v1/fleets`)                                    |
| `AUTH_TOKEN`           | **Required.** Same bearer token as task-broker **`AUTH_TOKEN`**                                          |
| `RUNNER_TRANSPORT`     | Default **WebSocket** (`GET /v1/runners/stream`). Set **`http`**, **`polling`**, or **`legacy`** for **`POST /v1/tasks/claim`** / **`complete`**. |
| `RUNNER_ID`            | Optional; defaults to host name or a random id (EC2 user-data sets instance id from IMDS)              |
| `POLL_EMPTY_MS`        | Sleep when no work (default ~1000 ms)                                                                  |
| `RUNNER_MAX_EXECUTION_SECONDS` | Optional. Hard cap on run wall clock on **this** runner. Does **not** change broker `lease_until`, which uses `execution_timeout_seconds` from the task (or the 1h default) plus buffer. |
| `RUNNER_TERMINATE_AFTER_EACH_TASK` | If `true`/`1`/`yes`, exit after **one** successful task (off by default locally; **on** for EC2 user-data unless disabled). **`runner_id`** should be the EC2 instance id (`i-…`) so fleet-manager can terminate the VM after the process exits. |
| `RUNNER_CLOUDWATCH_LOG_GROUP` | When set, task stdout/stderr stream to **CloudWatch Logs** (`PutLogEvents`) per task (see `shared/cwstream`). |
| `RUNNER_CLOUDWATCH_REGION` | Optional AWS region for the CloudWatch Logs client. |
| `RUNNER_CLOUDWATCH_LOG_STREAM_PREFIX` | Optional; must match **`TASK_CLOUDWATCH_LOG_STREAM_PREFIX`** on task-broker. |

```bash
export TASK_BROKER_URL=http://127.0.0.1:8081
export RUNNER_FLEET_ID=local
export AUTH_TOKEN=dev-local-token
./bin/runner
```

**Logging:** WebSocket transport logs **`task_broker_ws`**; HTTP transport logs **`task_broker_http`** for claim/complete (**`op`**, **`http_status`**, **`dur`**, **`runner_id`**, **`task_id`**).

**Structured task result:** The runner exports **`SUPERPLANE_RESULT_FILE`** to each task pointing at a host temp file (`superplane-result-<task_id>.json`). Write valid JSON there before exit; the runner reads it after execution and sends **`result`** on **`POST /v1/tasks/{id}/complete`**. **`GET /v1/tasks/{id}`**, completion webhooks, and broker **`GET /v1/tasks/{id}`** include **`result`** when present. Missing, empty, invalid JSON, or payload over **`MaxOutputBytes`** → **`result`** omitted. **`execution_mode: docker`:** the same variable inside the container is **`/mnt/superplane-result.json`** (bind-mounted from that host path).

## HTTP API — fleet-manager

- `GET /healthz` — liveness
- `GET /v1/admin/managed-runners` — EC2 instances tagged `superplane_managed_runner` (requires **`FLEET_DIAGNOSTICS_TOKEN`**)
- `GET /v1/admin/ec2-console-output?instance_id=i-…` — boot console output (same auth)

## End-to-end

## End-to-end with the broker

1. Start **task-broker** (Postgres + `AUTH_TOKEN`).
2. Start **fleet-manager** with a JSON config (`pools[]`); it registers each pool on the broker at startup. For local dev without fleet-manager, run **`make register-local-fleet`** after the broker is up.
3. Start **runner(s)** with `TASK_BROKER_URL` and `RUNNER_FLEET_ID` matching a registered fleet.
4. SuperPlane (or curl) calls **`GET /v1/fleets`** to list machine profiles, then **`POST /v1/tasks`** with **`fleet_id`** and the caller **`webhook_url`**.
5. Runner claims from the broker, executes, completes; broker delivers the webhook to the caller.

## Example: enqueue directly on fleet-manager (curl)

```bash
curl -X POST http://127.0.0.1:8080/v1/tasks \
  -H 'Content-Type: application/json' \
  -d '{
    "commands": ["echo hello", "echo \"$COMMIT_AUTHOR\""],
    "environment": [{"name": "COMMIT_AUTHOR", "value": "alice@example.com"}],
    "webhook_url": "https://example.com/your-hook"
  }'
```

## Example: fleet catalog and enqueue via task-broker (curl)

```bash
# Same token as task-broker AUTH_TOKEN
BROKER_TOKEN=your-secret

# List machine profiles (fleet-manager registers pools on startup in production).
curl -s http://127.0.0.1:8081/v1/fleets \
  -H "Authorization: Bearer ${BROKER_TOKEN}"

# Manual registration (local dev or ops); fleet-manager does this automatically on startup.
curl -X POST http://127.0.0.1:8081/v1/fleets \
  -H 'Content-Type: application/json' \
  -H "Authorization: Bearer ${BROKER_TOKEN}" \
  -d '{
    "id": "aws-standard-amd64",
    "provisioner": "aws",
    "arch": "amd64",
    "size": "t3.micro"
  }'

curl -X POST http://127.0.0.1:8081/v1/fleets \
  -H 'Content-Type: application/json' \
  -H "Authorization: Bearer ${BROKER_TOKEN}" \
  -d '{
    "id": "aws-standard-arm64",
    "provisioner": "aws",
    "arch": "arm64",
    "size": "t4g.micro"
  }'

curl -X POST http://127.0.0.1:8081/v1/tasks \
  -H 'Content-Type: application/json' \
  -H "Authorization: Bearer ${BROKER_TOKEN}" \
  -d '{
    "fleet_id": "aws-standard-arm64",
    "commands": ["uname -m", "echo \"$COMMIT_AUTHOR\""],
    "environment": [{"name": "COMMIT_AUTHOR", "value": "alice@example.com"}],
    "webhook_url": "https://example.com/your-hook"
  }'
```

Pick **`fleet_id`** from **`GET /v1/fleets`** (or your SuperPlane machine picker). Each fleet is a homogeneous runner pool — do not mix architectures behind one `fleet_id`.

## AWS validation checklist

Use this checklist before rolling architecture-specific fleets into production:

1. Build and upload both `runner-linux-amd64` and `runner-linux-arm64` to S3.
2. Configure fleet-manager JSON with separate `pools[]` entries (amd64 + arm64 AMIs, instance types, runner S3 URIs, distinct `fleet_id` values). Set `"arch": "arm64"` on Graviton pools.
3. Deploy fleet-manager; confirm startup logs show each pool registered on the broker.
4. **`GET /v1/fleets`** lists both pools with correct `arch` and `size`.
5. Submit one task with `"fleet_id": "<amd64-pool-id>"` and one with `"fleet_id": "<arm64-pool-id>"`; verify `uname -m` reports `x86_64` and `aarch64`.
6. Check EC2 console output and `superplane-runner.service` logs for clean user-data startup on both architectures.
7. Run a simple Docker task on both fleets to confirm Docker and the architecture-specific CloudWatch agent install correctly.
8. If `runner_terminate_after_each_task` is enabled, confirm completed runner instances terminate and fleet-manager reconciles replacement capacity.

## Tests

```bash
go test ./...
# e2e (subprocess task-broker + runners):
go test ./test/... -v
```

## CI (Semaphore)

The pipeline definition is [.semaphore/semaphore.yml](.semaphore/semaphore.yml).

In [Semaphore](https://semaphoreci.com/), create a **new project from this Git repository**. Semaphore 2.x picks up `.semaphore/semaphore.yml` on the default branch. Each push runs Go **1.22** on Ubuntu **24.04**: module cache restore/store, **`gofmt` check**, **`go vet`**, **`make build`**, **`go test ./...`**.

### Container images → GitHub Container Registry (GHCR)

Publishing runs from [.semaphore/docker-publish.yml](.semaphore/docker-publish.yml) (promoted via `pipeline_file: docker-publish.yml` next to [.semaphore/semaphore.yml](.semaphore/semaphore.yml)). After `docker login`, it runs **`make docker-publish-ghcr`** ([Makefile](./Makefile)); Semaphore fills **`IMAGE_PREFIX`** / **`IMAGE_TAG`**, and each of **`fleet-manager`**, **`task-broker`**, and **`runner`** is pushed under **`ghcr.io/<owner>/<repo>/`** with both the commit tag and **`latest`**. **[Auto-promote](https://docs.semaphoreci.com/using-semaphore/promotions)** after a green **Build and test** is limited to **`main`**.

**Semaphore setup**

1. In GitHub, create a [**personal access token (classic)**](https://docs.github.com/en/packages/learn-github-packages/publishing-and-managing-packages/publishing-docker-images) (or organization-level bot PAT) with at least **`read:packages`** and **`write:packages`**. SSO-enabled orgs must **authorize** the token for that org.

2. In Semaphore: **Secrets** → create a secret named exactly **`ghcr`** with **`GHCR_TOKEN`** (a PAT with **`read:packages`** and **`write:packages`**). Non-interactive `docker login --password-stdin` still requires a username, so the workflow uses the **`owner`** segment of **`owner/repo`** from `SEMAPHORE_GIT_REPO_SLUG` as **`-u`**, matching your **`ghcr.io/owner/...`** image paths. Ensure the PAT is for an account allowed to push to that namespace (often the same **`owner`** or a **`write:packages`** bot).

After the first successful push, configure each package under **GitHub → Packages** → package → **Package settings**. **Public** packages can be **pulled** without `docker login` ([visibility](https://docs.github.com/en/packages/learn-github-packages/configuring-a-packages-access-control-and-visibility)); that avoids **ECS/Fargate `repositoryCredentials`** for pulls. Publishing from CI still needs the PAT above.

### Runner binary → S3 (EC2 host workers)

When **main** is green, [.semaphore/runner-binary-s3.yml](.semaphore/runner-binary-s3.yml) builds static **linux/amd64** and **linux/arm64** `runner` binaries and uploads them with the AWS CLI. Create a Semaphore secret named **`runner-s3`** with **`AWS_ACCESS_KEY_ID`**, **`AWS_SECRET_ACCESS_KEY`**, **`AWS_DEFAULT_REGION`**, **`EC2_PROVISION_RUNNER_AMD64_S3_URI`**, and **`EC2_PROVISION_RUNNER_ARM64_S3_URI`**. Optional: **`AWS_SESSION_TOKEN`**.

## License

(Add your license.)
