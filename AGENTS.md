# Repository Guidelines

Guidance for AI coding agents and human contributors working in this repository.
Read this top-to-bottom: what the project is, how to set it up, how to build and
test, the conventions to follow, what not to touch, and how to submit changes.
For UI-specific work, also read [web_src/AGENTS.md](web_src/AGENTS.md).

## Project Overview

SuperPlane is an open-source automation engine for AI-driven engineering. It
orchestrates workflows across the tools teams already use (Git, LLMs, CI/CD,
observability, incident, and infrastructure tools) with durable execution,
approvals, and an operational UI.

- **Backend**: Go, exposing a gRPC API with a REST/OpenAPI gateway.
- **Frontend**: TypeScript + React, built with Vite.
- **Infrastructure**: PostgreSQL (state) and RabbitMQ (messaging), run via Docker.
- The application name is **SuperPlane** (not "Superplane") in all user-facing text.

## Repository Layout

- `cmd/` — Go entrypoints (server, workers, CLI).
- `pkg/` — Go application code. Notable packages:
  - `pkg/grpc/actions` — gRPC API implementation.
  - `pkg/models` — database models.
  - `pkg/workers` — background workers.
  - `pkg/integrations/<integration>/` — integration component implementations.
- `web_src/` — TypeScript/React frontend (Vite). UI component mappers live in
  `web_src/src/pages/app/mappers/<integration>/`.
- `protos/` — protobuf definitions for the API.
- `db/` — database structure and migrations.
- `scripts/` — codegen, DB, and CI helper scripts.
- `test/` — backend and end-to-end tests.
- `docs/` — Markdown documentation (see `docs/contributing/`).
- `Makefile` — the entrypoint for all common tasks.

## Prerequisites & Setup

The development environment is **entirely Docker-based**: build, lint, test, and
run commands execute inside containers via `docker compose`. You do not need Go
or Node installed on the host, only Docker.

- Go `1.26.2` (pinned in `go.mod`; provided by the dev container).
- Node.js + npm (provided by the dev container; used by Vite/frontend).
- Docker with a working `docker compose`.

Run these three steps once, in order:

1. `make dev.up` — builds the dev-base image and starts containers (app, db,
   rabbitmq). The first run builds the image (~3-5 min); later runs reuse it.
2. `make dev.setup` — installs npm deps, downloads Go modules, runs protobuf
   codegen, and creates + migrates the databases. Re-run when protos, Go
   modules, or frontend deps change. By default only `superplane_dev` is
   migrated; use `DEV_SETUP_DBS="superplane_dev superplane_test"` when you also
   need `superplane_test` (E2E; backend CI sets this via the environment).
3. `make dev.server` — starts the API (Go hot-reload via `air`) and the Vite dev
   server. UI at http://localhost:8000; health check at
   http://localhost:8000/health. Use `make dev.server.fg` for foreground logs.

On first UI load, owner setup is enabled (`OWNER_SETUP_ENABLED=yes`), so you are
prompted to create an admin account. Open registration is disabled by default
(`BLOCK_SIGNUP=yes`).

If `go mod download` / `go build` fail with missing files under
`tmp/go/pkg/mod` (often after a disk-full or interrupted download), run
`make dev.clean.go.cache` then `make dev.setup.go`.

## Build, Test & Lint Commands

- One-shot backend tests: `make test` (Go).
- Targeted backend tests: `make test PKG_TEST_PACKAGES=./pkg/workers`
- Targeted E2E tests: `E2E_TEST_PACKAGES=./test/e2e/workflows make test.e2e`
  (or `make test.e2e FILE=test/e2e/foo_test.go LINE=19` for a single test).
- After editing Go code: `make format.go`, then `make lint && make check.build.app`.
- After editing JS/TS code: `make format.js`, then `make check.build.ui`.
- After updating `protos/`: regenerate protos, the OpenAPI spec, and the CLI/UI
  SDKs with `make pb.gen` (requires a running `app` container from `make dev.up`).
  After removing proto fields, renumber remaining fields so message field numbers
  stay contiguous (no gaps or `reserved` markers — these protos are used for JSON
  conversion, not wire compatibility), then run `make check.proto.field.numbers`.
- **NEVER MANUALLY CREATE MIGRATION FILES.** Use `make db.migration.create NAME=<name>`
  (dashes, not underscores). We do not write rollbacks, so leave `*.down.sql`
  empty. After adding a migration, run `make db.migrate DB_NAME=<DB_NAME>` where
  `DB_NAME` is `superplane_dev` or `superplane_test` (requires a running app
  container).

Cross-cutting rules when extending the backend:

- When validating enum fields in protobuf requests, ensure enums are mapped to
  constants in `pkg/models`. Check the `Proto*` and `*ToProto` functions in
  `pkg/grpc/actions/common.go`.
- When adding a new worker in `pkg/workers`, add its startup to
  `cmd/server/main.go` and update the docker compose files with any new
  environment variables.
- After adding new API endpoints, ensure they are covered in
  `pkg/authorization/interceptor.go`.

Further reading:

- E2E test authoring: [docs/contributing/e2e-tests.md](docs/contributing/e2e-tests.md)
- Dev server profiling: [docs/contributing/profiling.md](docs/contributing/profiling.md)
- New components/triggers: [docs/contributing/component-implementations.md](docs/contributing/component-implementations.md)
- Component design & quality: [docs/contributing/component-design.md](docs/contributing/component-design.md)
- UI component workflow: [web_src/AGENTS.md](web_src/AGENTS.md)

## Coding Style & Naming Conventions

- Always write clean code: work test-first by default, then keep names clear,
  functions focused, side effects explicit, control flow shallow, and error
  handling useful.
- Tests end with `_test.go`.
- Always prefer early returns over else blocks when possible.
- Go: prefer `any` over `interface{}`.
- Go: to check membership in a slice, use `slices.Contains` or `slices.ContainsFunc`.
- Avoid variable names like `*Str` or `*UUID`; Go is typed, so types don't belong
  in variable names.
- In tests needing specific timestamps, base them off `time.Now()` rather than
  absolute times from `time.Date`.
- The name of the application is "SuperPlane", not "Superplane", in all
  user-facing text (UIs, emails, notifications, documentation).
- Frontend: do not create or use `web_src/src/utils/*` or `utils.ts` files. Put
  shared non-React helpers in `web_src/src/lib/`, and React-specific reusable
  logic in `web_src/src/hooks/`.

## Database Transaction Guidelines

We are moving away from `database.Conn()` inside `pkg/models` and from the
`FindX` / `FindXInTransaction` dual API. CI tracks remaining legacy usage via
`make check.models.tx.debt`; do not add new `*InTransaction` definitions or
`database.Conn()` call sites in `pkg/models`.

**Why:**

- Calling `database.Conn()` inside model code breaks transaction isolation when
  the caller already holds a `tx`.
- Conn wrappers plus `*InTransaction` methods duplicate API surface without
  adding behavior.

**Preferred pattern:** pass an explicit `*gorm.DB` as the first parameter.
Callers outside `pkg/models` obtain it with `database.DB(ctx)` (request-scoped,
attaches OpenTelemetry trace context).

```go
func FindCanvas(tx *gorm.DB, orgID, id uuid.UUID) (*Canvas, error) {
    var canvas Canvas
    err := tx.Where("organization_id = ? AND id = ?", orgID, id).First(&canvas).Error
    if err != nil {
        return nil, err
    }
    return &canvas, nil
}

// Handler (no surrounding transaction):
canvas, err := models.FindCanvas(database.DB(ctx), orgID, canvasID)

// Inside an existing transaction:
err := database.DB(ctx).Transaction(func(tx *gorm.DB) error {
    canvas, err := models.FindCanvas(tx, orgID, canvasID)
    return err
})
```

Rules:

- **NEVER** call `database.Conn()` inside `pkg/models` — pass the `*gorm.DB` from
  the caller instead.
- **NEVER** call a model function that uses `database.Conn()` internally while you
  already hold a `tx`.
- **Always propagate** the `*gorm.DB` through the entire call chain — pass it as
  the first parameter to functions that need database access.
- **Do not add** new `FindX` + `FindXInTransaction` pairs or conn wrappers; use a
  single function with an explicit `*gorm.DB` parameter.
- **Context constructors** that perform database queries must accept `tx *gorm.DB`
  as their first parameter.

When touching legacy `*InTransaction` or conn-wrapper code, migrate to the
explicit-parameter pattern when practical and update the debt baseline with
`make check.models.tx.debt.baseline.update`.

### Model file layout (`pkg/models`)

Order declarations in each model file as follows:

1. **Struct** — package constants used by the model, then the struct type.
2. **Constructors** — `New…` functions that build values for the model (including
   name/ID helpers).
3. **Getters** — methods on the struct (e.g. `TableName()`, computed accessors).
4. **Database access** — functions whose first parameter is `tx *gorm.DB` (or
   `db *gorm.DB`).

Place private helpers after the public API in the file.

### Models API shape (`pkg/models`)

Choose one style per concern and stick to it. Prefer object style when you
already have a model handle; do not invent free functions that re-take IDs you
already hold.

| Situation | Prefer | Example |
| --- | --- | --- |
| Operation on a loaded model | Method on the struct | `node.HardDelete(tx)` |
| Multi-step / configurable DB work for a model | Package constructor + collaborator/builder | `NewNodeResourceCleaner(tx, node).ForUnreferenced().WithLimit(n).Run()` |
| Lookup / list when you do **not** have a handle | Package function with `tx` first | `ListDeletedCanvasNodes(tx, …)`, `FindCanvas(tx, …)` |

Rules:

- **Do not** add `models.HardDeleteCanvasNode(tx, orgID, nodeID)` (or similar)
  when the caller already has `*CanvasNode` — that forces an extra find and mixes
  procedural style with OO for the same concern.
- **Do not** hang multi-step cleanup/publish logic as a thick method chain on the
  aggregate when a dedicated collaborator is clearer (`NodeResourceCleaner`,
  canvas publisher patterns).
- Keep SQL / GORM deletes and queries in `pkg/models`. Workers and gRPC actions
  **orchestrate** (lock → clean → hard-delete); they do not own batched delete
  queries.
- Receivers on model methods should use a short name consistent with the type
  (`c` for `*CanvasNode`, etc.), matching nearby code in the file.

```go
// Good: handle already loaded
if err := node.HardDelete(tx); err != nil {
    return err
}

// Good: multi-step cleanup as a collaborator
n, err := NewNodeResourceCleaner(tx, node).ForUnreferenced().WithLimit(batchSize).Run()

// Good: no handle yet — package function
nodes, err := ListDeletedCanvasNodes(tx, before, limit)

// Avoid: free function that re-keys a node you already have
_ = HardDeleteCanvasNode(tx, node.OrganizationID, node.ID)
```

## Files & Directories Not to Modify by Hand

- **Generated code — regenerate, never hand-edit** (all produced by `make pb.gen`):
  - `pkg/protos/` — Go generated from `protos/*.proto`.
  - `pkg/openapi_client/` — generated Go SDK for the CLI.
  - `web_src/src/api-client/` — generated TypeScript SDK for the UI.
  - `api/` — generated OpenAPI/swagger spec.
  Edit the source (`protos/`) and rerun `make pb.gen` instead.
- **Database migrations** — never create or edit migration files by hand; use
  `make db.migration.create NAME=<name>` (see the Build section).
- **Secrets & local config** — never commit real secrets. `.env.example` and
  `.env.multi-instance.example` are templates; do not check in a populated
  `.env`.
- **Vendored / cached dependencies** — do not edit `web_src/node_modules/` or the
  Go module cache under `tmp/go/`.

## Commit & Pull Request Guidelines

- PR titles must follow Conventional Commits with a release-type prefix that CI
  enforces: `feat:`, `fix:`, `chore:`, or `docs:`.
- All commits must include a DCO sign-off trailer
  (`Signed-off-by: Name <email>`). Use `git commit -s` (or `git commit --amend -s`).
- Before submitting, run the checks relevant to your change:
  - Backend: `make format.go`, `make lint`, `make check.build.app`, `make test`.
  - Frontend: `make format.js`, `make check.build.ui`.
  - Protos: `make pb.gen` and `make check.proto.field.numbers`.

## Cursor Cloud Environment Notes

These apply only to Docker-in-Docker cloud VMs (e.g. Cursor Cloud); skip them on
a normal workstation with Docker already running.

- Run all `make` commands from the repository root.
- The Docker daemon must be started manually:
  `sudo dockerd &>/tmp/dockerd.log &` — wait ~3-4 seconds before issuing Docker
  commands, then make the socket accessible with
  `sudo chmod 666 /var/run/docker.sock`.
- Docker needs the `fuse-overlayfs` storage driver and `iptables-legacy` for
  nested-container support.
- The `app` container starts with `sleep infinity`; you must explicitly run
  `make dev.server` to start the API + UI stack.
