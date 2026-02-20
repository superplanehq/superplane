# Runtime Runner HTTP + gRPC Contract

## Overview

This document defines a transport-neutral contract for the long-running runtime runner service.

The same operations are exposed through:

1. HTTP endpoints for immediate adoption.
2. gRPC methods for future migration.

Both use the same request/response envelope and semantics.

## Goals

1. Keep Go runtime-agnostic by calling a single internal runner API.
2. Support language-specific runners (starting with TypeScript/Deno) behind the same contract.
3. Preserve clear operation boundaries across triggers, components, and integrations.
4. Keep HTTP and gRPC shapes aligned so migration is incremental.

## Endpoint and Method Mapping

1. `POST /v1/triggers/{name}/setup` -> `SetupTrigger`
2. `POST /v1/components/{name}/setup` -> `SetupComponent`
3. `POST /v1/components/{name}/execute` -> `ExecuteComponent`
4. `POST /v1/integrations/{name}/sync` -> `SyncIntegration`
5. `POST /v1/integrations/{name}/cleanup` -> `CleanupIntegration`
6. `GET /v1/capabilities` -> `ListCapabilities`

Health endpoints remain HTTP-only:

1. `GET /healthz`
2. `GET /readyz`

## Shared Request Envelope

All operation requests include:

1. `request.request_id`: caller-generated traceable ID.
2. `request.version`: payload version string (`v1`).
3. `request.timeout_ms`: max allowed execution time.
4. `context`: runtime context (org/workspace/user/canvas/node and labels/metadata).
5. `input`: operation-specific payload.

## Shared Response Envelope

All operation responses include:

1. `ok`: operation success flag.
2. `output`: operation-specific output payload.
3. `logs`: structured runtime logs.
4. `error`: typed error object when `ok=false`.
5. `metrics`: optional numeric metrics map.

## Error Codes

Canonical error codes:

1. `INVALID_INPUT`
2. `NOT_FOUND`
3. `TIMEOUT`
4. `EXECUTION_ERROR`
5. `UNAVAILABLE`

HTTP status and gRPC status should map from the same canonical code.

## Capabilities

`ListCapabilities` returns discovered runtime modules with:

1. `kind` (`trigger`, `component`, `integration`)
2. `name`
3. `operations`
4. `schema_hash`

This enables startup-time validation and compatibility checks from the Go side.

## Operational Controls

Runtime runner behavior is controlled by environment variables:

- `TYPESCRIPT_RUNNER_ENABLE_HTTP`
- `TYPESCRIPT_RUNNER_HTTP_HOST`
- `TYPESCRIPT_RUNNER_HTTP_PORT`
- `TYPESCRIPT_RUNNER_ENABLE_GRPC`
- `TYPESCRIPT_RUNNER_GRPC_ADDRESS`
- `TYPESCRIPT_RUNNER_AUTH_TOKEN`
- `TYPESCRIPT_RUNNER_LOG_REQUESTS`

Authentication behavior:

- `GET /healthz` and `GET /readyz` are unauthenticated.
- Other HTTP and gRPC operations require `Authorization: Bearer <token>` when `TYPESCRIPT_RUNNER_AUTH_TOKEN` is set.

## Source of Truth

The contract source of truth is:

- `pkg/runtime/runner/proto/runtime_runner.proto`

Current TypeScript runner implementation:

- `sdk/typescript/runner/server.ts`
- `sdk/typescript/runner/registry.ts`

HTTP is currently the primary transport, but this proto keeps the gRPC contract fully aligned for future rollout.
