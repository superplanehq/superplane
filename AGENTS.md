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
- Targeted backend tests: `make test TEST_PACKAGES=./pkg/workers`
- After updating UI code, always run `make check.build.ui` to verify everything is correct
- After updating GoLang code, always check it with `make lint && make check.build.app`
- To generate DB migrations, use `make db.migration.create NAME=<name>`. Always use dashes instead of underscores in the name. We do not write migrations to rollback, so leave the `*.down.sql` files empty. After adding a migration, run `make db.migrate DB_NAME=<DB_NAME>`, where DB_NAME can be `superplane_dev` or `superplane_test`
- When validating enum fields in protobuf requests, ensure that the enums are properly mapped to constants in the `pkg/models` package. Check the `Proto*` and `*ToProto` functions in pkg/grpc/actions/common.go.
- When adding a new worker in pkg/workers, always add its startup to `cmd/server/main.go`, and update the docker compose files with the new environment variables that are needed.
- After adding new API endpoints, ensure the new endpoints have their authorization covered in `pkg/authorization/interceptor.go`
- For UI component workflow, see [web_src/AGENTS.md](web_src/AGENTS.md)
- After updating the proto definitions in protos/, always regenerate them, the OpenAPI spec for the API, and SDKs for the CLI and the UI:
  - `make pb.gen` to regenerate protobuf files
  - `make openapi.spec.gen` to generate OpenAPI spec for the API
  - `make openapi.client.gen` to generate GoLang SDK for the API
  - `make openapi.web.client.gen` to generate TypeScript SDK for the UI

## Coding Style & Naming Conventions

- Tests end with _test.go
- Always prefer early returns over else blocks when possible
- GoLang: prefer `any` over `interface{}` types
- When naming variables, avoid names like `*Str` or `*UUID`; Go is a typed language, we don't need types in the variables names
- When writing tests that require specific timestamps to be used, always use timestamps based off of `time.Now()`, instead of absolute times created with `time.Date`
