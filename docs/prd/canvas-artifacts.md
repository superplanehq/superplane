# Canvas Artifacts

## Overview

This PRD defines artifact support in SuperPlane across two areas:

1. Runtime integration so components can create/read/list artifacts during execution.
2. Canvas-scoped API endpoints (via protobuf + gRPC-Gateway) to list and download artifacts for canvases, nodes, and executions.

The API is organization-scoped through existing auth context and canvas authorization rules.

## Problem Statement

Artifact primitives exist in the codebase, but they were not fully wired into execution contexts and were not exposed through the existing protobuf API surface. As a result:

- Components cannot reliably use a consistent artifact context across all execution entry points.
- Users and tools cannot query artifacts through stable, canvas-scoped API routes.
- Artifact download behavior is undefined for API consumers.

## Goals

1. Expose artifact storage in `core.ExecutionContext` for all execution flows.
2. Provide protobuf-defined list/get APIs for artifact scopes:
   - Canvas
   - Node
   - Execution
3. Standardize download responses to return artifact content directly (raw body).
4. Keep authorization and tenancy aligned with existing canvas APIs.

## Non-Goals

- Adding non-local artifact backends (S3, GCS, etc.) in this phase.
- Adding artifact upload endpoints for external clients.
- Adding artifact metadata APIs (size, checksum, timestamps, content type inference).
- Adding pagination/search/filtering for artifact lists.
- Adding unit/e2e tests in this phase.

## User Stories

1. As a component author, I can write execution artifacts using `ExecutionContext.Artifacts.Execution`.
2. As a user, I can list all artifacts produced for a canvas, node, or execution.
3. As a user, I can download an artifact as raw content for direct consumption by CLI/scripts.

## Functional Requirements

### Runtime Integration

- `core.ExecutionContext` must expose:
  - `Artifacts.Canvas`
  - `Artifacts.Node`
  - `Artifacts.Execution`
- Artifact storage interface supports:
  - `Create(name)`
  - `Get(name)`
  - `List()`
- Artifact objects must support `Write`, `Read`, and `Close`.
- Local storage implementation must:
  - Isolate by resource type + resource id.
  - Protect against path traversal and invalid path segments.
  - Create directories as needed.
  - Return sorted names from `List()`.
- Node-level artifact namespace must include canvas scope to avoid collisions across canvases.

### API Endpoints

The following endpoints are required:

- `GET /api/v1/canvases/{canvas_id}/artifacts`
- `GET /api/v1/canvases/{canvas_id}/artifacts/{name}`
- `GET /api/v1/canvases/{canvas_id}/nodes/{node_id}/artifacts`
- `GET /api/v1/canvases/{canvas_id}/nodes/{node_id}/artifacts/{name}`
- `GET /api/v1/canvases/{canvas_id}/executions/{execution_id}/artifacts`
- `GET /api/v1/canvases/{canvas_id}/executions/{execution_id}/artifacts/{name}`

Behavior:

- List endpoints return a `ListArtifactsResponse` with artifact names.
- Download endpoints return `google.api.HttpBody`.
- Download responses must contain only artifact content in the body.

### Validation and Authorization

- Requests require the same authenticated org context used by other canvas APIs.
- Canvas must belong to the caller organization.
- Node endpoints must validate node existence in the canvas.
- Execution endpoints must validate execution existence in the canvas.
- Invalid IDs return `InvalidArgument`.
- Missing resources return `NotFound`.
- Storage/read failures return `Internal`.

## API Contract (Protobuf)

Add RPCs to `Canvases` service:

- `ListCanvasArtifacts(ListCanvasArtifactsRequest) returns (ListArtifactsResponse)`
- `GetCanvasArtifact(GetCanvasArtifactRequest) returns (google.api.HttpBody)`
- `ListNodeArtifacts(ListNodeArtifactsRequest) returns (ListArtifactsResponse)`
- `GetNodeArtifact(GetNodeArtifactRequest) returns (google.api.HttpBody)`
- `ListExecutionArtifacts(ListExecutionArtifactsRequest) returns (ListArtifactsResponse)`
- `GetExecutionArtifact(GetExecutionArtifactRequest) returns (google.api.HttpBody)`

Messages:

- `ListCanvasArtifactsRequest { string canvas_id }`
- `ListNodeArtifactsRequest { string canvas_id, string node_id }`
- `ListExecutionArtifactsRequest { string canvas_id, string execution_id }`
- `GetCanvasArtifactRequest { string canvas_id, string name }`
- `GetNodeArtifactRequest { string canvas_id, string node_id, string name }`
- `GetExecutionArtifactRequest { string canvas_id, string execution_id, string name }`
- `ListArtifactsResponse { repeated string names }`

## Storage Design

### Local Filesystem Layout

Base directory:

- `ARTIFACTS_ROOT_DIRECTORY` env var
- fallback: `/tmp/superplane-artifacts`

Path model:

- `<root>/<resource_type>/<resource_id>/<artifact_name>`

Resource types:

- `canvas`
- `node`
- `execution`

Node resource id format:

- `<canvas_id>:<node_id>`

## Implementation Plan

### Phase 1: Runtime Wiring

1. Update core artifact interfaces and execution context contract.
2. Implement local artifact storage with create/get/list/close.
3. Inject artifact context in all execution context constructors and execution entry points.

### Phase 2: API Surface

1. Add artifact RPCs and messages in `protos/canvases.proto`.
2. Map RPCs to required canvas-scoped HTTP routes.
3. Use `google.api.HttpBody` for download RPC responses.

### Phase 3: Service/Action Layer

1. Implement canvas/node/execution artifact actions.
2. Enforce canvas/node/execution ownership validation.
3. Add authorization interceptor mappings for new RPC methods.

### Phase 4: Generated Artifacts

1. Regenerate protobuf/go gateway files (`make pb.gen`).
2. Regenerate OpenAPI spec (`make openapi.spec.gen`).
3. Regenerate API clients as needed.

## Acceptance Criteria

1. Components can create/read/list artifacts through execution context in all runtime paths.
2. All six artifact endpoints are available under `/api/v1/canvases/...`.
3. List endpoints return artifact names for the correct scope.
4. Download endpoints return raw artifact content in HTTP response body.
5. Access control for new RPC methods is covered by authorization interceptor rules.
6. Generated protobuf/gateway/OpenAPI artifacts are updated and build passes.

## Risks

- Large artifacts are currently read into memory for download responses.
- Default `application/octet-stream` content type may not reflect original file type.
- Local filesystem storage is host-local and may be ephemeral depending on deployment topology.

## Future Enhancements

1. Stream artifact downloads instead of loading full content in memory.
2. Support pluggable storage backends (object storage).
3. Add artifact metadata endpoints (size, hash, created_at, mime type).
4. Add external upload APIs with validation and size limits.
