# Repository Guidelines

## Project Structure & Module Organization

- Backend (GoLang): cmd/ with pkg/ (GoLang code), and test/.
- Frontend (TypeScript/React): web_src/ built with Vite.
- Tooling: Makefile (common tasks), protos/ (protobuf definitions for the API), scripts/ (protobuf generation), db/ (database structure and migrations).
- Documentation: Markdown files in docs/.
- gRPC API implementation in in pkg/grpc/actions
- Database models in pkg/models
- Integration component implementations: pkg/integrations/<integration>/
- UI component mappers: web_src/src/pages/app/mappers/<integration>/

## Pull Request Guidelines

- PR titles must follow Conventional Commits and include a release-type prefix: `feat:`, `fix:`, `chore:`, or `docs:` (CI enforces this).
- All commits must include a DCO sign-off trailer (`Signed-off-by: Name <email>`). Use `git commit -s` or `git commit --amend -s` when creating or updating commits.

## Build, Test, and Development Commands

- Bring up dev containers: `make dev.up`
- Install deps, codegen, DB: `make dev.setup` after `dev.up` (re-run when protos, Go modules, or frontend deps change). By default only `superplane_dev` is migrated; use `DEV_SETUP_DBS="superplane_dev superplane_test"` when you also need `superplane_test` (E2E; backend CI sets this via the environment). If `go mod download` / `go build` fail with missing files under `tmp/go/pkg/mod`, run `make dev.clean.go.cache` then `make dev.setup.go` (often after disk-full or interrupted downloads).
- Start API + Vite: `make dev.server` (after `make dev.up`) — UI at http://localhost:8000; use `make dev.server.fg` for foreground logs
- One-shot backend tests: `make test` (Golang).
- Targeted backend tests: `make test PKG_TEST_PACKAGES=./pkg/workers`
- Targeted E2E tests: `E2E_TEST_PACKAGES=./test/e2e/workflows make test.e2e` (or `make test.e2e FILE=test/e2e/foo_test.go LINE=19` for one test)
- For E2E test authoring, see [docs/contributing/e2e-tests.md](docs/contributing/e2e-tests.md)
- For performance profiling of the dev server, see [docs/contributing/profiling.md](docs/contributing/profiling.md)
- After updating UI code, always run `make check.build.ui` to verify everything is correct
- After editing JS code, always run `make format.js` to make sure that the files are consistently formatted
- After editing Golang code, always run `make format.go` to make sure that files are consistently formatted
- After updating GoLang code, always check it with `make lint && make check.build.app`
- **NEVER MANUALLY CREATE MIGRATION FILES**. ALWAYS use `make db.migration.create NAME=<name>` to generate DB migrations. Always use dashes instead of underscores in the name. We do not write migrations to rollback, so leave the `*.down.sql` files empty. After adding a migration, run `make db.migrate DB_NAME=<DB_NAME>` (requires a running app container from `make dev.up`), where DB_NAME can be `superplane_dev` or `superplane_test`
- When validating enum fields in protobuf requests, ensure that the enums are properly mapped to constants in the `pkg/models` package. Check the `Proto*` and `*ToProto` functions in pkg/grpc/actions/common.go.
- When adding a new worker in pkg/workers, always add its startup to `cmd/server/main.go`, and update the docker compose files with the new environment variables that are needed.
- After adding new API endpoints, ensure the new endpoints have their authorization covered in `pkg/authorization/interceptor.go`
- For UI component workflow, see [web_src/AGENTS.md](web_src/AGENTS.md)
- For new components or triggers, see [docs/contributing/component-implementations.md](docs/contributing/component-implementations.md)
- For component design guidelines and quality standards, see [docs/contributing/component-design.md](docs/contributing/component-design.md)
- After updating the proto definitions in protos/, always regenerate them, the OpenAPI spec for the API, and SDKs for the CLI and the UI with `make pb.gen`(requires a running `app` container from `make dev.up`)
- After removing proto fields, renumber the remaining fields so message field numbers stay contiguous (no gaps), then run `make check.proto.field.numbers`. These protos are used for JSON conversion, not wire compatibility, so do not leave holes or `reserved` markers.

## Coding Style & Naming Conventions

- Always write clean code: work test-first by default, then keep names clear, functions focused, side effects explicit, control flow shallow, and error handling useful.
- Tests end with \_test.go
- Always prefer early returns over else blocks when possible
- GoLang: prefer `any` over `interface{}` types
- GoLang: when checking for the existence of an item on a list, use `slice.Contains` or `slice.ContainsFunc`
- When naming variables, avoid names like `*Str` or `*UUID`; Go is a typed language, we don't need types in the variables names
- When writing tests that require specific timestamps to be used, always use timestamps based off of `time.Now()`, instead of absolute times created with `time.Date`
- The name of the application is "SuperPlane", not "Superplane" in all user-facing text (user interfaces, emails, notifications, documentation, etc.).
- Frontend: do not create or use `web_src/src/utils/*` or `utils.ts` files. Put shared non-React helpers in `web_src/src/lib/`, and put React-specific reusable logic in `web_src/src/hooks/`.

## Database Transaction Guidelines

We are moving away from `database.Conn()` inside `pkg/models` and from the `FindX` / `FindXInTransaction` dual API. CI tracks remaining legacy usage via `make check.models.tx.debt`; do not add new `*InTransaction` definitions or `database.Conn()` call sites in `pkg/models`.

**Why:**

- Calling `database.Conn()` inside model code breaks transaction isolation when the caller already holds a `tx`
- Conn wrappers plus `*InTransaction` methods duplicate API surface without adding behavior

**Preferred pattern:** pass an explicit `*gorm.DB` as the first parameter. Callers outside `pkg/models` obtain it with `database.DB(ctx)` (request-scoped, attaches OpenTelemetry trace context).

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

- **NEVER** call `database.Conn()` inside `pkg/models` — pass the `*gorm.DB` from the caller instead
- **NEVER** call a model function that uses `database.Conn()` internally while you already hold a `tx`
- **Always propagate** the `*gorm.DB` through the entire call chain — pass it as the first parameter to functions that need database access
- **Do not add** new `FindX` + `FindXInTransaction` pairs or conn wrappers; use a single function with an explicit `*gorm.DB` parameter
- **Context constructors** that perform database queries must accept `tx *gorm.DB` as their first parameter

When touching legacy `*InTransaction` or conn-wrapper code, migrate to the explicit-parameter pattern when practical and update the debt baseline with `make check.models.tx.debt.baseline.update`.

### Model file layout (`pkg/models`)

Order declarations in each model file as follows:

1. **Struct** — package constants used by the model, then the struct type
2. **Constructors** — `New…` functions that build values for the model (including name/ID helpers)
3. **Getters** — methods on the struct (e.g. `TableName()`, computed accessors)
4. **Database access** — functions whose first parameter is `tx *gorm.DB` (or `db *gorm.DB`)

Place private helpers after the public API in the file.

### Models API shape (`pkg/models`)

Choose one style per concern and stick to it. Prefer object style when you already have a model handle; do not invent free functions that re-take IDs you already hold.

| Situation | Prefer | Example |
| --- | --- | --- |
| Operation on a loaded model | Method on the struct | `node.HardDelete(tx)` |
| Multi-step / configurable DB work for a model | Package constructor + collaborator/builder | `NewNodeResourceCleaner(tx, node).ForUnreferenced().WithLimit(n).Run()` |
| Lookup / list when you do **not** have a handle | Package function with `tx` first | `ListDeletedCanvasNodes(tx, …)`, `FindCanvas(tx, …)` |

Rules:

- **Do not** add `models.HardDeleteCanvasNode(tx, orgID, nodeID)` (or similar) when the caller already has `*CanvasNode` — that forces an extra find and mixes procedural style with OO for the same concern.
- **Do not** hang multi-step cleanup/publish logic as a thick method chain on the aggregate when a dedicated collaborator is clearer (`NodeResourceCleaner`, canvas publisher patterns).
- Keep SQL / GORM deletes and queries in `pkg/models`. Workers and gRPC actions **orchestrate** (lock → clean → hard-delete); they do not own batched delete queries.
- Receivers on model methods should use a short name consistent with the type (`c` for `*CanvasNode`, etc.), matching nearby code in the file.

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

## Cursor Cloud specific instructions

### Environment overview

The dev environment is entirely Docker-based. The VM needs Docker installed (with fuse-overlayfs storage driver and iptables-legacy for nested container support). All build, lint, test, and run commands execute inside Docker containers via `docker compose exec`.

### Starting the development environment

1. `make dev.up` — builds the dev-base Docker image and starts containers (app, db/PostgreSQL, rabbitmq). First run builds the image (~3-5 min); subsequent runs reuse the cached image.
2. `make dev.setup` — installs npm deps, downloads Go modules, runs protobuf codegen, creates and migrates both `superplane_dev` and `superplane_test` databases.
3. `make dev.server` — starts `air` (Go hot-reload) and Vite dev server inside the app container. Health check at `http://localhost:8000/health`.

### Key gotchas

- The `app` container starts with `sleep infinity` by default. You must explicitly run `make dev.server` to start the API + UI stack.
- All `make` commands must be run from the `/agent/repos/superplane` directory.
- The Docker daemon must be started manually in cloud VMs: `sudo dockerd &>/tmp/dockerd.log &` — wait ~3-4 seconds before issuing Docker commands.
- After Docker daemon starts, ensure the socket is accessible: `sudo chmod 666 /var/run/docker.sock`.
- Owner setup is enabled by default (`OWNER_SETUP_ENABLED=yes`). On first UI load at `http://localhost:8000`, you'll be prompted to create an admin account with email/password.
- `BLOCK_SIGNUP=yes` is the default, so only the owner setup flow works for account creation (no open registration).
- If `make dev.setup` fails on `go mod download` with missing files, run `make dev.clean.go.cache` then `make dev.setup.go`.
- The `make dev.setup` Makefile target already creates and migrates both `superplane_dev` and `superplane_test` databases.
