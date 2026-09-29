# SuperPlane runner

This package contains the runner implementation in the root Go module. It does
not import code from the legacy `runner/` module.

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

An ephemeral runner exits after one acknowledged task. A reusable runner waits
for another task.

From the repository root, run it with:

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
