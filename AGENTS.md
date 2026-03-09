# Repository Guidelines

## Project Structure & Module Organization

- Backend (GoLang): cmd/ with pkg/ (GoLang code), and test/.
- Frontend (TypeScript/React): web_src/ built with Vite.
- Tooling: Makefile (common tasks), protos/ (protobuf definitions for the API), scripts/ (protobuf generation), db/ (database structure and migrations).
- Documentation: Markdown files in docs/.
- gRPC API implementation in in pkg/grpc/actions
- Database models in pkg/models

## Build, Test, and Development Commands

- Setup dev environment: `make dev.setup`
- Run server: `make dev.start` - UI at http://localhost:8000
- One-shot backend tests: `make test` (Golang).
- Targeted backend tests: `make test PKG_TEST_PACKAGES=./pkg/workers`
- Targeted E2E tests: `make e2e E2E_TEST_PACKAGES=./test/e2e/workflows`
- For E2E test authoring, see [docs/development/e2e_tests.md](docs/development/e2e_tests.md)
- After updating UI code, always run `make check.build.ui` to verify everything is correct
- After editing JS code, always run `make format.js` to make sure that the files are consistently formatted
- After editing Golang code, always run `make format.go` to make sure that files are consistently formatted
- After updating GoLang code, always check it with `make lint && make check.build.app`
- **NEVER MANUALLY CREATE MIGRATION FILES**. ALWAYS use `make db.migration.create NAME=<name>` to generate DB migrations. Always use dashes instead of underscores in the name. We do not write migrations to rollback, so leave the `*.down.sql` files empty. After adding a migration, run `make db.migrate DB_NAME=<DB_NAME>`, where DB_NAME can be `superplane_dev` or `superplane_test`
- When validating enum fields in protobuf requests, ensure that the enums are properly mapped to constants in the `pkg/models` package. Check the `Proto*` and `*ToProto` functions in pkg/grpc/actions/common.go.
- When adding a new worker in pkg/workers, always add its startup to `cmd/server/main.go`, and update the docker compose files with the new environment variables that are needed.
- After adding new API endpoints, ensure the new endpoints have their authorization covered in `pkg/authorization/interceptor.go`
- For UI component workflow, see [web_src/AGENTS.md](web_src/AGENTS.md)
- For new components or triggers, see [docs/contributing/component-implementations.md](docs/contributing/component-implementations.md)
- For component design guidelines and quality standards, see [docs/contributing/component-design.md](docs/contributing/component-design.md)
- Whenever adding or updating components/triggers that should be AI-assisted, always create or update the matching skill file in `templates/skills/` (for example `templates/skills/<component-or-trigger>.md`).
- After updating the proto definitions in protos/, always regenerate them, the OpenAPI spec for the API, and SDKs for the CLI and the UI:
  - `make pb.gen` to regenerate protobuf files
  - `make openapi.spec.gen` to generate OpenAPI spec for the API
  - `make openapi.client.gen` to generate GoLang SDK for the API
  - `make openapi.web.client.gen` to generate TypeScript SDK for the UI

## Coding Style & Naming Conventions

- Tests end with \_test.go
- Always prefer early returns over else blocks when possible
- GoLang: prefer `any` over `interface{}` types
- GoLang: when checking for the existence of an item on a list, use `slice.Contains` or `slice.ContainsFunc`
- When naming variables, avoid names like `*Str` or `*UUID`; Go is a typed language, we don't need types in the variables names
- When writing tests that require specific timestamps to be used, always use timestamps based off of `time.Now()`, instead of absolute times created with `time.Date`
- The name of the application is "SuperPlane", not "Superplane" in all user-facing text (user interfaces, emails, notifications, documentation, etc.).

## Database Transaction Guidelines

When working with database transactions, follow these rules to ensure data consistency:

- **NEVER** call `database.Conn()` inside a function that receives a `tx *gorm.DB` parameter

  - ❌ Bad: `func process(tx *gorm.DB) { user, _ := models.FindUser(id) }` where FindUser calls `database.Conn()`
  - ✅ Good: `func process(tx *gorm.DB) { user, _ := models.FindUserInTransaction(tx, id) }`

- **Always propagate** the transaction context through the entire call chain

  - Pass `tx` as the first parameter to all functions that need database access
  - If a model method is used within a transaction, create an `*InTransaction()` variant that accepts `tx`

- **Context constructors** must accept `tx *gorm.DB` if they perform database queries

  - ❌ Bad: `NewAuthContext(orgID, service)` that internally calls `database.Conn()`
  - ✅ Good: `NewAuthContext(tx, orgID, service)` that uses the passed transaction

- **When creating new model methods**:
  - Create both variants: `FindUser()` and `FindUserInTransaction(tx *gorm.DB)`
  - The non-transaction variant should call the transaction variant: `return FindUserInTransaction(database.Conn(), ...)`

**Why this matters**: Using `database.Conn()` inside transaction contexts breaks isolation, causes data inconsistency on rollback, and can lead to race conditions.
