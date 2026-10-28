---
title: Local Development Guide
---

# Local Development Guide

For first-time setup, follow [CONTRIBUTING.md](../../CONTRIBUTING.md#setting-up-the-development-environment).

## Everyday Workflow

If the containers are stopped, start them:

```sh
make dev.up
```

Start the development server:

```sh
make dev.server
```

For foreground logs, use `make dev.server.fg` instead of `make dev.server`.
To stop the containers, run `make dev.down`.

### When to Rerun Setup

Run `make dev.setup` when changes require updated dependencies, generated code,
or database migrations. This includes changes to:

- Go dependencies in `go.mod` or `go.sum`.
- Frontend dependencies in `web_src/package.json` or `web_src/package-lock.json`.
- Protocol Buffer definitions in `protos/`.
- Migrations in `db/migrations/` or `db/data_migrations/`.

You do not need to run setup for every development session.

## Runner Fleets and Worker Tools

The Docker Fleet Manager starts an ephemeral runner for each integrated task.
Both `make dev.setup` and `make dev.server` create or update the local
`e1-large-amd64` fleet. After owner setup, the Fleet Manager creates its local
API credentials automatically. Standard local development requires no manual
fleet creation or `INSTALLATION_ADMIN_TOKEN` configuration.

During local owner setup, select **Skip for now** on the SMTP and fleet setup
screens. The local fleet already exists, and owner password login works without
SMTP.

The runner image includes Claude Code, Codex, OpenCode, Git, `gh`, `jq`, and
media tools. Do not install these tools on the host for factory tasks.
Connect GitHub and Claude integrations in the organization before you dispatch
a factory line. Factory nodes use these integrations, not `.env`
`ANTHROPIC_API_KEY`. OpenCode must be on the runner `PATH` for Run OpenRouter
Agent.

After changes to `release/runner/Dockerfile`, rebuild the images with
`make dev.up`. Use `make doctor-local` to check the bundled runner tools and
model files. This check does not verify integration credentials or task
execution.

## Advanced Configuration

### Manual Runners and Token Overrides

Use `make runner.new` to start a runner manually for debugging.
This command requires a running app, an existing fleet, and an installation
administrator personal API token. Set `INSTALLATION_ADMIN_TOKEN` in your shell
or local `.env` before you run the command. The command pauses the local Fleet
Manager and restarts it on exit if it was running.

For the Fleet Manager, `INSTALLATION_ADMIN_TOKEN` overrides automatic token
creation. See [the environment examples](../../.env.example).

### Historical Runner Logs

Set `TASK_BROKER_*` in `.env` only when historical execution logs remain on the
legacy broker. New tasks use the integrated runner API.
See [the environment examples](../../.env.example).

### Snapshots and External Services

- [Local database snapshot](local-database-snapshot.md): restore an environment without repeating owner setup or GitHub connection.
- [Connecting to third-party services](connecting-to-3rdparty-services-from-development.md): configure external integrations, webhook tunnels, and optional hosted OpenRouter.
