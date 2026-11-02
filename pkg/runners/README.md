# SuperPlane runner

This package contains the integrated runner implementation in the root Go
module.

The runner:

1. Exchanges a single-use generic or task-specific registration token at
   `POST /runner/v1/register`.
2. Connects to `GET /runner/v1/connect` with the returned runner credential.
3. Executes tasks with the copied host or Docker execution engine.
4. Writes bounded log chunks to disk and uploads them in sequence with
   `PUT /runner/v1/tasks/{task_id}/logs/chunks/{sequence}`.
5. Sends task completion only after all retained chunks receive `204 No
   Content`.
6. Waits for the WebSocket completion acknowledgement.
7. Requests a controlled shutdown when it receives `SIGINT` or `SIGTERM`.

An ephemeral runner exits after one acknowledged task. A reusable runner waits
for another task.

## Local development

Start the development environment and complete owner setup. Enable the
`organization_runners` feature for your organization and select its CLI context.
Create an organization fleet with `superplane fleets create --file fleet.yaml`.
Use `runnerVersion: dev` in the fleet definition.

Create registration credentials for that fleet:

```sh
superplane runners create --fleet my-local-fleet --ephemeral=false
```

Run the local source in the development container with the returned token:

```sh
docker compose -f docker-compose.dev.yml exec app go run ./cmd/runner \
  --url http://localhost:8000 \
  --registration-token "$RUNNER_REGISTRATION_TOKEN"
```

Select the organization fleet when you start tasks from another terminal.
Press Ctrl-C to test a controlled shutdown. Organization fleets let you test
runner changes without pausing the installation Fleet Manager.

The registration token determines whether the runner is generic or bound to a
specific task. The runner does not receive an installation administrator
credential. Development builds use version `dev`. Release builds set
`main.Version` with the Go linker.

Run focused checks with:

```sh
docker compose -f docker-compose.dev.yml exec app go test ./pkg/runners/...
docker compose -f docker-compose.dev.yml exec app go build ./cmd/runner
```

## Direct runner setup

Create a fleet and registration credentials through the main CLI:

```sh
superplane fleets create --file fleet.yaml
superplane runners create --fleet my-fleet
superplane runners create --fleet my-fleet --ephemeral=false
```

Organization creation and configuration require the `organization_runners`
feature. Authorized inspection and runner termination remain available when
the feature is disabled. Fleet lists include disabled managed fleets.

For installation fleets, set both `SUPERPLANE_URL` and
`SUPERPLANE_TOKEN` to the server URL and an installation administrator token.
No organization context is required:

```sh
superplane fleets create --installation --file fleet.yaml
superplane runners create --installation --fleet my-fleet --ephemeral
superplane fleets list-tasks --installation --fleet my-fleet --state queued --limit 200
```

Installation runners are ephemeral-only. Organization runners can be ephemeral
or reusable. The default is ephemeral. Use `--task` with `--idempotency-key` to
bind registration to one task. Lists support repeatable `--state`, `--limit`
from 1 to 1000, and `--after-id`.

Creation returns a runner record, a short-lived registration token, its
expiration, and the runner server URL. It does not provision or start a machine.
Use the returned URL and registration token to start the runner:

```sh
runner --url <returned-server-root> --registration-token <registration-token>
```

Use the returned URL even if it differs from the management URL.
Do not append `/runner/v1`. The CLI does not save registration tokens.
Use `--output json` or `--output yaml` for scripts. Structured output preserves
credentials, timestamps, and list pagination metadata.

The main CLI replaces the `admin` binary. Use `.superplane.yaml` or main CLI
environment credentials instead of `.superplane-admin.yaml`, `--url`, and `--token`.
Set both `SUPERPLANE_URL` and `SUPERPLANE_TOKEN` when you use environment credentials.
Replace `admin tasks list` with `superplane fleets list-tasks --installation`.
Fleet updates now require `superplane fleets update <fleet-id> --file fleet.yaml`.
Fleet Manager still accepts `INSTALLATION_ADMIN_TOKEN` for installation fleet
API access. Local development generates this token unless you supply an override.
Runner processes receive only their registration token.

## Controlled shutdown

The runner sends a `shutdown_request` WebSocket message after the first signal.
If the runner is idle, SuperPlane terminates the runner and sends `shutdown`.
If a task is running, SuperPlane records a cancellation and sends `cancel`.
The runner stops the task, uploads its retained logs, and sends `complete`.
SuperPlane acknowledges the completion and then sends `shutdown`.

The runner waits up to 30 seconds for this sequence. A second signal or the
timeout stops the runner immediately.
