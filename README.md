# SuperPlane runner

Monorepo (single Go module) for **task-broker**, **fleet-manager**, and the **runner** worker.

- **Task-broker** — optional front door: registers downstream fleet-manager instances (per region/cloud), routes `POST /v1/tasks`, and relays completion webhooks to callers (`task_id` is broker-scoped; `fleet_task_id` identifies the upstream task).
- **Fleet-manager** — durable queue per fleet; runners use **WebSocket** by default (`GET /v1/runners/stream`) or optional HTTP claim/complete when **`RUNNER_TRANSPORT=http`**. Completions POST to either the caller webhook or the broker relay URL.
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
- For Docker tasks: Docker CLI **and a reachable Docker daemon** on the runner host. The runner uses a pull → long-lived named container → `docker exec` → `docker stop`/`rm` lifecycle (see [ARCHITECTURE.md](./ARCHITECTURE.md#docker-execution-lifecycle)). The image must include `sleep` (alpine, debian, ubuntu, python:*, node:* all satisfy this). Multi-line **`commands`** are bundled into one `sh -c` script with `set -e`, so env/cwd persist across directives and the script fails fast on the first non-zero exit. Task **`environment`** entries are passed to the `docker exec` process, not to the idle `docker run` container. **Quoting:** each directive is a line inside a single-quoted `sh -c` argument; a raw **`'`** in a line is a classic shell-quoting footgun—avoid it in `commands` or use argv **`command`** for tricky literals. **`docker exec` is invoked without `-t`**, so the task runs in a non-TTY context: tools that detect `isatty()` (color output, progress bars, interactive prompts) will see stdout/stderr as a pipe. This is intentional — matches `docker run` without `-t`, more predictable for CI / batch workloads, and lets stdout and stderr stay distinct in captures.

### Upgrade note: Docker multi-line `commands` (breaking if you relied on the old runner)

Older runner builds ran multi-line **`commands`** through **`docker run` with a PTY** and **interactive bash** in the container. Current runners use **`docker pull` → `docker run -d` → `docker exec` … `sh -c '…'`** with **no PTY**. Anything that depended on a **TTY**, **bash-only** syntax (e.g. `[[ ]]`, bashisms not in POSIX `sh`), or **interactive** behavior may break or change. Prefer argv **`command`** for strict control, or adjust scripts for **`sh`**. When **CloudWatch live** logging is enabled, **`docker pull` / `docker run -d` diagnostics** are copied to the live stream on success; they are **not** included in the task **completion `output`** / webhook body (that payload remains **exec-phase** stdout/stderr only). Putting pull/run logs into the webhook would be an explicit product change.

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
| `AUTH_TOKEN`         | —             | **Required.** Clients must send `Authorization: Bearer …` for **`/v1/fleets`** and **`/v1/tasks`** (and related routes). **`/v1/webhooks/complete/*`** stays unauthenticated for fleet-manager callbacks |

**Logging:** stdout emits JSON **`http_access`** per request (**method**, **path**, **dur**, **status**, **bytes**, **remote**, optional **request_id** / **ua**). **`GET /healthz`** is skipped to reduce load-balancer noise. Successful fleet-manager creates also emit **`fleet_upstream_http`** (**op** `create_task`, **http_status**, **dur**, **broker_task_id**, **fleet_id**, **fleet_task_id**, **upstream_host**). Outbound caller webhooks log **`webhook_delivery`** per attempt (**attempt**, **task_id**, **fleet_task_id**, **status_outcome**, **url_host**, **dur**, **http_status** or **err**).

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
export AUTH_TOKEN=your-secret                  # mandatory
./bin/task-broker
```

**Inspect upstream task status** (uses `AUTH_TOKEN` and broker base from **`scripts/deploy/task-broker.env`** unless you export overrides): `./scripts/check-broker-task.sh <broker_task_id>`

**Correlate broker + fleet SQLite over SSH** (default EC2 hosts match deploy scripts): `./scripts/show-runner-queue-state.sh`

## Run fleet-manager

| Environment variable | Default      | Description                                                  |
| -------------------- | ------------ | ------------------------------------------------------------ |
| `LISTEN_ADDR`        | `:8080`      | HTTP listen address                                          |
| `DATABASE_PATH`      | `./fleet.db` | SQLite database file                                         |
| `AUTH_TOKEN`         | (empty)      | If set, requires `Authorization: Bearer <token>` for `/v1/*` |
| `REAP_INTERVAL_SEC`  | `15`         | How often to return expired leases to the queue              |
| `TASK_CLOUDWATCH_LOG_GROUP` | (empty) | When set, `GET /v1/tasks/{id}` and completion webhooks include `cloudwatch_log_group` and `cloudwatch_log_stream` so clients can tail the same stream the runner writes to |
| `TASK_CLOUDWATCH_LOG_STREAM_PREFIX` | (empty) | Optional; stream name is `{prefix}/{task_id}` (see `shared/cwstream`). Must match `RUNNER_CLOUDWATCH_LOG_STREAM_PREFIX` on workers. |
| `TASK_CLOUDWATCH_REGION` | (empty) | Optional AWS region included in **`task_log.cloudwatch.region`** for clients (e.g. your log proxy). |

**Task log descriptor:** When **`TASK_CLOUDWATCH_LOG_GROUP`** is set, `GET /v1/tasks/{id}` and completion webhooks include **`task_log`** with `{"type":"cloudwatch","cloudwatch":{"log_group_name","log_stream_name","region"}}`. Otherwise **`task_log`** is omitted. Legacy **`cloudwatch_log_group`** / **`cloudwatch_log_stream`** fields are still present when CloudWatch is enabled.

Optional **EC2 hot runner pool** — set **`AWS_REGION`** (also used as **`AWS_DEFAULT_REGION`** inside user-data for **`aws s3 cp`**), **`EC2_PROVISION_HOT_INSTANCE_COUNT`**, **`EC2_PROVISION_AMI_ID`**, **`EC2_PROVISION_SUBNET_ID`**, **`EC2_PROVISION_SECURITY_GROUP_IDS`**, **`EC2_PROVISION_FLEET_MANAGER_URL`**, plus **either**:

- **`EC2_PROVISION_RUNNER_S3_URI`** — `s3://bucket/key` to a **linux/amd64** static **`runner`** binary (**recommended with private repos**). Requires **`EC2_PROVISION_RUNNER_INSTANCE_PROFILE`** on runners with **`s3:GetObject`** on that object; user-data runs **`aws s3 cp`**.
- **`EC2_PROVISION_RUNNER_BINARY_URL`** — public **http(s)** URL (**curl**).

Do **not** set both.

Optional: **`EC2_PROVISION_RUNNER_CLOUDWATCH_LOG_GROUP`**, **`EC2_PROVISION_RUNNER_CLOUDWATCH_LOG_STREAM_PREFIX`** — written into **`/etc/default/superplane-runner`** as **`RUNNER_CLOUDWATCH_*`** (requires the runner instance profile to allow **`logs:CreateLogGroup`**, **`logs:CreateLogStream`**, **`logs:PutLogEvents`**, **`logs:DescribeLogStreams`** on that log group).

Fleet-manager **reconciles in the background** (default **60** s, **`EC2_PROVISION_RECONCILE_INTERVAL_SEC`**, minimum **15**). Optional: **`EC2_PROVISION_INSTANCE_TYPE`**, **`EC2_PROVISION_RUNNER_AUTH_TOKEN`**, **`EC2_PROVISION_KEY_NAME`**, **`EC2_PROVISION_RUNNER_INSTANCE_PROFILE`**.

Provisioner **user-data** installs **`/usr/local/bin/runner`** and starts **`superplane-runner.service`** on **Ubuntu** (**`docker.io`** remains for **`execution_mode: docker`**).

#### Runner binary via S3 (typical setup)

1. **Bucket** (same account/region as runners is simplest). Upload the static binary, e.g. **`runner-linux-amd64`** at **`s3://my-runner-binaries/release/runner-linux-amd64`** (`aws s3 cp bin/runner …` after **`CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bin/runner ./runner/cmd/runner`**).

2. **IAM role for runners** (**instance profile** name = **`EC2_PROVISION_RUNNER_INSTANCE_PROFILE`**): attach an inline policy allowing **`s3:GetObject`** on **`arn:aws:s3:::my-runner-binaries/release/*`** (tighten to the exact key).

3. **Fleet-manager env**: **`EC2_PROVISION_RUNNER_S3_URI=s3://my-runner-binaries/release/runner-linux-amd64`**, **`AWS_REGION=us-east-1`** (or your region), **`EC2_PROVISION_RUNNER_INSTANCE_PROFILE=…`**.

4. **CI**: on each release, **`aws s3 cp`** / sync the built **`runner`** to that key (OIDC **`aws-actions/configure-aws-credentials`** or long-lived IAM user with **`s3:PutObject`** on that prefix).

By default **`EC2_PROVISION_RUNNER_TERMINATE_AFTER_TASK`** is **on** (`true`): **`runner_id`** is the EC2 instance id from IMDS; after **one** successful task, **fleet-manager** calls **`TerminateInstances`** and the systemd unit **`Restart=no`** stops respawn before shutdown. Set **`false`** for long-lived workers (**`Restart=always`**). Runner VMs do **not** need **`TerminateInstances`** on their profile for that flow; **fleet-manager’s** role must **`TerminateInstances`** (reconcile + disposable runners).

Fleet-manager still needs **`ec2:RunInstances`**, **`ec2:DescribeInstances`**, **`ec2:CreateTags`**, **`ec2:TerminateInstances`**, and **`iam:PassRole`** when using an instance profile on runners.

```bash
export DATABASE_PATH=./fleet.db
./bin/fleet-manager
```

## Run the runner

| Environment variable | Description                                                                                            |
| -------------------- | ------------------------------------------------------------------------------------------------------ |
| `FLEET_MANAGER_URL`  | **Required.** Base URL of **that fleet’s** fleet-manager (not the broker unless you bypass the broker) |
| `RUNNER_TRANSPORT`   | Default **WebSocket**. Set **`http`**, **`polling`**, or **`legacy`** to use **`POST /v1/tasks/claim`** / **`complete`** instead (e.g. old fleet-manager without the stream route). |
| `RUNNER_ID`          | Optional; defaults to host name or a random id                                                         |
| `AUTH_TOKEN`         | Optional; must match fleet-manager if set                                                              |
| `POLL_EMPTY_MS`      | Sleep when no work (default ~1000 ms)                                                                  |
| `RUNNER_MAX_EXECUTION_SECONDS` | Optional. Hard cap on run wall clock on **this** runner. Does **not** change fleet-manager `lease_until`, which still uses `execution_timeout_seconds` from the task (or the 9m default) plus buffer — so leases can be longer than the capped run when set. |
| `RUNNER_TERMINATE_AFTER_EACH_TASK` | If `true`/`1`/`yes`, exit the runner process after **one** successful task (off by default locally; **on** for fleet-manager EC2 user-data unless disabled). **`runner_id`** must be the EC2 instance id (`i-…`) for **fleet-manager** to terminate the VM; termination is done by fleet-manager, not the runner binary. |
| `RUNNER_CLOUDWATCH_LOG_GROUP` | When set, task stdout/stderr are streamed to **Amazon CloudWatch Logs** (`PutLogEvents`) on a per-task log stream (see `shared/cwstream`). Uses the default AWS credential chain (EC2 instance profile, env keys, etc.). |
| `RUNNER_CLOUDWATCH_REGION` | Optional AWS region for the CloudWatch Logs client (defaults to the SDK default chain). |
| `RUNNER_CLOUDWATCH_LOG_STREAM_PREFIX` | Optional; must match **`TASK_CLOUDWATCH_LOG_STREAM_PREFIX`** on fleet-manager so **`GET /v1/tasks/{id}`** reports the correct **`cloudwatch_log_stream`**. |

```bash
export FLEET_MANAGER_URL=http://127.0.0.1:8080
export AUTH_TOKEN= # if fleet-manager uses it
./bin/runner
```

**Logging:** Default transport logs **`fleet_manager_ws`** for stream lifecycle; HTTP transport logs **`fleet_manager_http`** for **`claim_task`**/**`complete_task`**: **`op`**, **`http_status`**, **`dur`**, **`runner_id`**, **`task_id`** (when known); failures use **`Warn`** with **`err`**.

## HTTP API — fleet-manager (v1)

- `GET /healthz` — liveness
- `POST /v1/tasks` — enqueue: **`command`** (argv for one process) **or** **`commands`** (string lines concatenated into one `sh -c` script so `export` / `cd` persist), **`webhook_url`**, optional `environment` (`[{ "name", "value" }]`), optional `execution_mode` / `docker_image`
- `GET /v1/runners/stream` — runner WebSocket (default worker transport)
- `POST /v1/tasks/claim` — runner pulls the next task (HTTP transport)
- `POST /v1/tasks/{id}/complete` — runner reports result (HTTP transport)

When the broker is **not** in the path, fleet-manager POSTs the completion **webhook** to `webhook_url` with `task_id`, `status`, `exit_code`, `output`, optional `error` (no `fleet_task_id`). Task `environment` values are execution-only and are not returned by status or webhook payloads.

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
    "commands": ["echo hello", "echo \"$COMMIT_AUTHOR\""],
    "environment": [{"name": "COMMIT_AUTHOR", "value": "alice@example.com"}],
    "webhook_url": "https://example.com/your-hook"
  }'
```

## Example: register fleet and enqueue via task-broker (curl)

```bash
# Same token as task-broker AUTH_TOKEN
BROKER_TOKEN=your-secret

# Register a fleet-manager (repeat per region/environment)
curl -X POST http://127.0.0.1:8081/v1/fleets \
  -H 'Content-Type: application/json' \
  -H "Authorization: Bearer ${BROKER_TOKEN}" \
  -d '{
    "id": "aws-standard-1",
    "base_url": "http://127.0.0.1:8080",
    "labels": ["aws", "standard"]
  }'

curl -X POST http://127.0.0.1:8081/v1/tasks \
  -H 'Content-Type: application/json' \
  -H "Authorization: Bearer ${BROKER_TOKEN}" \
  -d '{
    "fleet_id": "aws-standard-1",
    "commands": ["echo \"$COMMIT_AUTHOR\""],
    "environment": [{"name": "COMMIT_AUTHOR", "value": "alice@example.com"}],
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

Publishing runs from [.semaphore/docker-publish.yml](.semaphore/docker-publish.yml) (promoted via `pipeline_file: docker-publish.yml` next to [.semaphore/semaphore.yml](.semaphore/semaphore.yml)). After `docker login`, it runs **`make docker-publish-ghcr`** ([Makefile](./Makefile)); Semaphore fills **`IMAGE_PREFIX`** / **`IMAGE_TAG`**, and each of **`fleet-manager`**, **`task-broker`**, and **`runner`** is pushed under **`ghcr.io/<owner>/<repo>/`** with both the commit tag and **`latest`**. **[Auto-promote](https://docs.semaphoreci.com/using-semaphore/promotions)** after a green **Build and test** is limited to **`main`**.

**Semaphore setup**

1. In GitHub, create a [**personal access token (classic)**](https://docs.github.com/en/packages/learn-github-packages/publishing-and-managing-packages/publishing-docker-images) (or organization-level bot PAT) with at least **`read:packages`** and **`write:packages`**. SSO-enabled orgs must **authorize** the token for that org.

2. In Semaphore: **Secrets** → create a secret named exactly **`ghcr`** with **`GHCR_TOKEN`** (a PAT with **`read:packages`** and **`write:packages`**). Non-interactive `docker login --password-stdin` still requires a username, so the workflow uses the **`owner`** segment of **`owner/repo`** from `SEMAPHORE_GIT_REPO_SLUG` as **`-u`**, matching your **`ghcr.io/owner/...`** image paths. Ensure the PAT is for an account allowed to push to that namespace (often the same **`owner`** or a **`write:packages`** bot).

After the first successful push, configure each package under **GitHub → Packages** → package → **Package settings**. **Public** packages can be **pulled** without `docker login` ([visibility](https://docs.github.com/en/packages/learn-github-packages/configuring-a-packages-access-control-and-visibility)); that avoids **ECS/Fargate `repositoryCredentials`** for pulls. Publishing from CI still needs the PAT above.

### Runner binary → S3 (EC2 host workers)

When **main** is green, [.semaphore/runner-binary-s3.yml](.semaphore/runner-binary-s3.yml) builds a static **linux/amd64** `runner` and uploads it with the AWS CLI. Create a Semaphore secret named **`runner-s3`** with **`AWS_ACCESS_KEY_ID`**, **`AWS_SECRET_ACCESS_KEY`**, **`AWS_DEFAULT_REGION`**, and **`EC2_PROVISION_RUNNER_S3_URI`** (the same `s3://bucket/key` as **fleet-manager** — see **`EC2_PROVISION_RUNNER_S3_URI`** above). Optional: **`AWS_SESSION_TOKEN`**.

## License

(Add your license.)
