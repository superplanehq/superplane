# Canvas Memory

## Overview

Canvas memory gives an App durable JSON state that can be written, read, and
cleared across paths and runs. It supports operational mappings and shared
workflow context without requiring users to maintain an external store.

## Terminology

- **Namespace:** A named collection of memory entries within one Canvas.
- **Entry:** One stored JSON value and its identifying metadata.
- **Upsert:** Add an entry or replace the entry matching a chosen field.

## Requirements

### REQ-MEM-001: Write durable App state

**User story:** As a workflow builder, I want a workflow to store JSON values
in a namespace, so that later steps and runs can reuse operational context.

**Acceptance criteria:**

- **AC-MEM-001.1:** When a memory-write component succeeds, a later run of the
  same App shall be able to read the stored entry.
- **AC-MEM-001.2:** When append or uniqueness behavior is selected, SuperPlane
  shall preserve or replace entries according to that configured behavior.

### REQ-MEM-002: Read and route memory results

**User story:** As a workflow builder, I want to read a namespace or match a
  specific entry, so that the workflow can continue with stored data.

**Acceptance criteria:**

- **AC-MEM-002.1:** When a matching entry exists, SuperPlane shall emit the
  configured value through the found outcome.
- **AC-MEM-002.2:** When no matching entry exists, SuperPlane shall emit the
  not-found outcome without substituting data from another App.

### REQ-MEM-003: Clear obsolete state

**User story:** As a workflow builder, I want to clear selected memory, so that
temporary mappings do not outlive the resources they represent.

**Acceptance criteria:**

- **AC-MEM-003.1:** When a clear operation identifies a namespace or matching
  entry, SuperPlane shall remove only the selected Canvas memory.
- **AC-MEM-003.2:** After a successful clear, subsequent reads shall no longer
  return the removed entry.

### REQ-MEM-004: Preserve Canvas isolation

**User story:** As an App owner, I want memory isolated to my App's Canvas, so
that workflows cannot accidentally read another App's runtime state.

**Acceptance criteria:**

- **AC-MEM-004.1:** When two Apps use the same namespace name, each App shall
  read only its own entries.
- **AC-MEM-004.2:** When a user lacks access to an App, SuperPlane shall not
  expose that App's memory through list or delete operations.

## Traceability

- **Product context:** [App-scoped memory](../../README.md#how-it-works)
- **Detailed behavior:** [Canvas memory PRD](../../docs/prd/canvas-memory.md)
- **API evidence:** [Canvas memory RPCs](../../protos/canvases.proto)
- **Runtime evidence:** [add memory component](../../pkg/components/addmemory/add_memory.go),
  [upsert memory component](../../pkg/components/upsertmemory/upsert_memory.go),
  and [update memory component](../../pkg/components/updatememory/update_memory.go)
- **Feature blueprint:** [Canvas Memory](../blueprints/features/canvas-memory.feature.md)

## Open Questions

- What quotas, retention, ordering, and pagination guarantees apply to memory?
- Is a user-facing memory browser part of the supported product direction?
