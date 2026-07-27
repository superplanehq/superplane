# Canvas Memory Feature Blueprint

## Feature Summary

Workflow authors persist, query, update, and delete App-scoped JSON state across
runs. This feature implements the
[Canvas Memory](../../requirements/canvas-memory.md) requirements.

Evidence: [`pkg/models/canvas_memory.go`](../../../pkg/models/canvas_memory.go),
[`pkg/components/memorywrite/`](../../../pkg/components/memorywrite/),
[`pkg/components/readmemory/`](../../../pkg/components/readmemory/),
[`pkg/components/deletememory/`](../../../pkg/components/deletememory/), and
[`protos/canvases.proto`](../../../protos/canvases.proto).

## Component Blueprint Composition

- [Registry and Runtime](../components/registry-and-runtime.component.md):
  memory actions are registered component implementations whose schemas validate
  namespaces, matches, values, and emission modes.
- [Workflow Execution](../components/workflow-execution.component.md):
  execution contexts bind memory operations to the active Canvas and route
  `found`, `notFound`, `deleted`, and write outcomes downstream.
- [API Gateway and Realtime](../components/api-gateway-and-realtime.component.md):
  Canvas memory APIs expose authorized list and mutation operations used by
  operational surfaces such as Console widgets.
- [PostgreSQL](../containers/postgresql.container.md): stores
  `CanvasMemory` rows keyed by Canvas and namespace with JSON values and source
  metadata.

## Feature-Specific Flow

Add, upsert, and update components validate configuration and call the
Canvas-bound memory context, which writes JSON rows in PostgreSQL. Read Memory
queries one namespace using exact JSON-field matches and emits either all
matches or the latest match through `found`; an empty result uses `notFound`.
Delete Memory removes only matching rows and emits `deleted` or `notFound`.
API and Console reads reuse the same Canvas-scoped model.

## System Contracts

- Every query and mutation includes the active Canvas ID; namespace names do
  not cross that boundary.
- Values are JSON and matching uses containment of all configured field/value
  pairs.
- Empty match sets are rejected for targeted read, update, and delete
  operations.
- Memory writes survive the run and node execution that created them.
- One-by-one read emission is bounded by the configured maximum emit count.
- API access to memory inherits Canvas authorization; runtime components receive
  a context already scoped to their executing Canvas.

## Requirement Coverage

- **REQ-MEM-001:** Add, upsert, and update components persist JSON in
  `canvas_memories`; append and match-based replacement behavior is selected by
  each component's validated configuration.
- **REQ-MEM-002:** Read Memory queries a Canvas namespace and routes matching
  values through `found` or an empty result through `notFound`.
- **REQ-MEM-003:** Delete Memory removes only rows matching the configured
  Canvas, namespace, and fields, so subsequent reads cannot return them.
- **REQ-MEM-004:** Model queries require a Canvas ID and API operations apply
  Canvas authorization, preserving isolation even when namespace names match.

## Architecture Decision Records

### ADR-001: Scope durable memory to a Canvas

**Context:** Workflows need reusable state without allowing namespace collisions
to expose another App's data.

**Decision:** Store each memory row with a mandatory `CanvasID` and require that
scope in every model query and mutation.

**Consequences:** Workflows can reuse simple namespace names safely; sharing
state across Apps requires an explicit orchestration or integration path.

## Open Questions

- What quotas, retention, ordering, and pagination guarantees should apply?
- Should manually managed and node-produced namespaces remain mutually locked
  by source?
