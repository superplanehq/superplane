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

Start the development environment and complete owner setup. Use this command
to test runner or runner API changes. Set `INSTALLATION_ADMIN_TOKEN` to a
personal API token for an installation administrator:

```sh
make runner.new
```

The target rebuilds the local runner image and starts one attached ephemeral
runner. It pauses the local Fleet Manager and restores it after the runner
exits. `make dev.setup` creates the default `e1-large-amd64` fleet. To use a
different fleet, use `superplane fleets create --installation --file fleet.yaml` and set `RUNNER_FLEET`:

```sh
make runner.new RUNNER_FLEET=my-local-fleet
```

Use a different terminal to start tasks and inspect runner behavior. Press
Ctrl-C to test a controlled shutdown.

To run the runner directly from the development container, use:

```sh
docker compose -f docker-compose.dev.yml exec app go run ./cmd/runner \
  --url http://localhost:8000 \
  --registration-token "$RUNNER_REGISTRATION_TOKEN"
```

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

For installation fleets, set both `SUPERPLANE_API_URL` and
`SUPERPLANE_API_TOKEN` to the server URL and an installation administrator token.
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
Run the startup command from the text output on the runner machine:

```sh
runner --url <returned-server-root> --registration-token <registration-token>
```

Use the returned URL even if it differs from the management URL.
Do not append `/runner/v1`. The CLI does not save registration tokens.
Use `--output json` or `--output yaml` for scripts. Structured output preserves
credentials, timestamps, and list pagination metadata.

The main CLI replaces the `admin` binary. Use `.superplane.yaml` or main CLI
environment credentials instead of `.superplane-admin.yaml`, `--url`, and `--token`.
The API-prefixed environment pair takes precedence over the legacy
`SUPERPLANE_URL` and `SUPERPLANE_TOKEN` pair. Each pair must be complete.
Replace `admin tasks list` with `superplane fleets list-tasks --installation`.
Fleet updates now require `superplane fleets update <fleet-id> --file fleet.yaml`.
Keep `INSTALLATION_ADMIN_TOKEN` for the local script and Fleet Manager.

## Controlled shutdown

The runner sends a `shutdown_request` WebSocket message after the first signal.
If the runner is idle, SuperPlane terminates the runner and sends `shutdown`.
If a task is running, SuperPlane records a cancellation and sends `cancel`.
The runner stops the task, uploads its retained logs, and sends `complete`.
SuperPlane acknowledges the completion and then sends `shutdown`.

The runner waits up to 30 seconds for this sequence. A second signal or the
timeout stops the runner immediately.
