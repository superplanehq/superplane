# Canvas Data Memory via `setData` / `getData`

## Overview

This PRD defines persistent, canvas-scoped memory for workflow logic using the `setData`,
`getData`, and `clearData` components.

This is **not** AI chat/session storage.  
This is runtime data storage that components use to persist and retrieve values across
different paths and executions of the same canvas.

## Problem Statement

Many workflows need durable state between runs, for example:

- mapping `pull_request -> sandbox_id`
- storing IDs created in one path and consumed in another
- retaining per-canvas operational context

Without canvas memory, users must recompute values or pass them inline through a single path only.

## Goals

1. Provide durable data memory scoped to a canvas.
2. Keep the model compatible with `setData` / `getData` usage patterns.
3. Support namespaces with multiple entries.
4. Keep this fully backend/runtime; no dedicated UI is required.

## Non-Goals

- AI conversation persistence.
- New visual memory manager in canvas UI.
- Org-global memory shared by all canvases.

## Scope and Semantics

Memory is stored as rows shaped like:

- `canvas_id`
- `namespace`
- `values` (JSON object)

Example rows:

- `canvas_id=1, namespace=machines, values={id: 12121, pull_request: "dadas"}`
- `canvas_id=1, namespace=stuff, values={id: 8989, pull_request: "dadas"}`

Duplicates are allowed unless explicitly prevented by `setData` `uniqueBy` behavior.

## Component Behavior Contract

### `setData`

- Writes into a namespace for the current canvas.
- Maps `setData.key` to `namespace`.
- Maps `setData.value` / `setData.valueList` to `values`.
- Supports:
  - `operation: set` (replace semantics for target key/namespace scope)
  - `operation: append` (add additional records under same namespace)
  - optional `uniqueBy` for upsert-like replacement within namespace when matching
    field value exists

### `getData`

- Reads from namespace (`key`) in current canvas.
- Supports:
  - `mode: value` (read value(s) for namespace)
  - `mode: listLookup` with `matchBy` and `matchValue`
  - optional `returnField`
  - `emitEachItem` for per-item fan-out
- Emits `found` / `notFound` channels.

### `clearData`

- Removes records by namespace and optional match filters.
- Used for cleanup (for example removing PR mapping after resource deletion).

## Data Model

`canvas_memories` (name TBD):

- `canvas_id` (indexed)
- `namespace` (indexed)
- `values` (json/jsonb)

Recommended indexes:

- `(canvas_id, namespace)`
- optional expression indexes later for high-traffic lookup keys
  (for example `values->>'pull_request'`)

## API/Runtime Integration

- No new user-facing API required in v1 if components can read/write through existing
  execution contexts.
- Worker execution context must expose canvas data read/write/list operations.
- All memory operations must run within existing canvas execution transaction boundaries.

## Authorization

- Reuse existing canvas execution permissions.
- Memory is readable/writable only in context of the active canvas.

## No UI Requirement

- Do not add any dedicated memory UI.
- Do not show this data as canvas nodes/widgets.
- Existing component configuration panels for `setData` / `getData` remain the only
  interaction surface.

## Example Configuration

```yaml
- type: addMemory
  namespace: <...>
  values:
    id: "1"
    pull_request: "123"
    creator: "alex"
```

Equivalent `setData` intent:

- `key = namespace`
- `operation = append`
- `valueList/value = values object`
- optional `uniqueBy` (for example `pull_request`)

## Acceptance Criteria

1. `setData` can persist data for a canvas namespace.
2. Multiple rows can exist for same `canvas_id + namespace`.
3. `getData` can read by namespace and resolve list lookups (`matchBy` / `matchValue`).
4. `clearData` can remove namespace items deterministically.
5. No new memory-specific UI is introduced.

## Risks and Mitigations

- **Risk:** Namespace values grow unbounded.  
  **Mitigation:** document cleanup patterns with `clearData`; add retention strategy later.

- **Risk:** Inconsistent lookup fields across flows.  
  **Mitigation:** document conventions (`id`, `pull_request`, etc.) per namespace.

- **Risk:** JSON lookup performance degrades at scale.  
  **Mitigation:** add targeted indexes for commonly queried fields.

## Open Questions

1. Should `operation: set` collapse all rows in a namespace into a single row, or
   replace by `uniqueBy` when provided?
2. Do we want hard uniqueness constraints for common keys
   (for example one row per `pull_request`) or keep it purely logical via `uniqueBy`?
3. Should we add optional timestamps/metadata columns now or defer until auditing is required?
