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
- After updating UI code, always run `npm run build` to verify everything is correct
- After updating GoLang code, always lint it with `make lint`
- For UI component workflow, see [web_src/AGENTS.md](web_src/AGENTS.md)
- After updating the proto definitions in protos/, always regenerate them, the OpenAPI spec for the API, and SDKs for the CLI and the UI:
  - `make pb.gen` to regenerate protobuf files
  - `make openapi.spec.gen` to generate OpenAPI spec for the API
  - `make openapi.client.gen` to generate GoLang SDK for the API
  - `make openapi.web.client.gen` to generate TypeScript SDK for the UI

## Coding Style & Naming Conventions

- Tests end with _test.go
- Always prefer early returns over else blocks when possible
