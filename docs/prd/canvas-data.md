# PRD: Canvas Data

## Overview

Introduce a new canvas-scoped concept called **Canvas Data**: a key-value store that allows workflow components to set and lookup values that are useful across the canvas (e.g., last deployed version, ephemeral machine IDs). Keys support full history so workflows can read the current value or previous values of any field.

Interaction with this data plane is exposed through **new core components**, keeping the capability consistent and easy to use from any workflow on the canvas.

## Problem Statement

Today, workflows on a canvas have no shared, persistent place to store or retrieve simple facts that matter across runs or across nodes. As a result:

- **Deployment workflows** cannot easily record “last deployed version” and have downstream nodes or later runs use it (e.g., for rollback or comparison).
- **Ephemeral environments** cannot maintain a list of created machines or resources for teardown or reuse.
- **State between runs** (e.g., last run timestamp, counters, feature flags) is hard to manage without external systems or brittle workarounds.

Users need a first-class, canvas-scoped store that is:

- **Simple**: key-value semantics, no schema required.
- **Historical**: every key supports history so “what was the previous value?” is a natural operation.
- **Workflow-native**: usable only from workflow components (core components), not a general-purpose DB.

## Goals

1. **Canvas-scoped key-value store (“Canvas Data”)**  
   - Keys and values are stored per canvas.  
   - No cross-canvas access; same isolation as the rest of the canvas model.

2. **History for every key**  
   - Each write to a key creates a new version.  
   - Workflows can read “current” or “previous by N” (or equivalent) for any key.

3. **Core components as the only interface**  
   - No direct API for arbitrary services; only **new core components** can set and lookup canvas data.  
   - Keeps the data plane consistent, auditable, and easy to reason about.

4. **Useful for common scenarios**  
   - Last deployed version of an application.  
   - Ephemeral machines/resources created (e.g., list or set of IDs).  
   - Other small, canvas-relevant state (timestamps, counts, flags) as needed.

## Non-Goals (Out of Scope for Initial Version)

- Cross-canvas or org-level “data” store.
- Direct REST/gRPC API for external systems to read/write canvas data (only via components).
- Large payloads or binary blobs; focus on small, string/JSON-friendly values.
- Fine-grained RBAC within canvas data (inherits canvas permissions).

## User Stories

1. **As a workflow author**, I can save the last deployed version (e.g., git SHA or tag) after a deploy step so that a later run or another node can use it (e.g., for rollback or “deploy only if changed”).
2. **As a workflow author**, I can record which ephemeral machines or resources were created (e.g., IDs or names) so that a teardown or cleanup step can remove them.
3. **As a workflow author**, I can look up the previous value of a key (e.g., “what was the last version before this deploy?”) to support rollback or diff logic.
4. **As a workflow author**, I can set and get arbitrary keys (e.g., `last_run_at`, `feature_x_enabled`) to keep small state on the canvas without external systems.

## Key Concepts

### Canvas Data

- **Scope**: One logical store per canvas. All keys live under that canvas; no sharing across canvases.
- **Keys**: String keys (e.g., `app/my-service/last_deployed_version`, `ephemeral/machines`). Namespacing is by convention (e.g., `app/…`, `ephemeral/…`).
- **Values**: Values are stored in a format that is workflow-friendly (e.g., string or JSON). Size limits TBD (e.g., 64 KB per value) to avoid misuse as a general blob store.
- **History**: Every write creates a new version. Versions are ordered (e.g., by timestamp or version id). “Current” = latest write; “previous” = one step back, etc.

### Data Plane

- The **data plane** is the set of capabilities (storage + history + access rules) that back Canvas Data.
- **Interaction with the data plane** is only through **core components**. No ad-hoc API for external services; components are the abstraction.

### Core Components (New)

New core components will provide:

1. **Set Canvas Data** (or similar name)  
   - Inputs: key (required), value (required).  
   - Effect: Writes the value to the key under the current canvas, creating a new history entry.  
   - Output: e.g., key, value, version or timestamp, for use in expressions or downstream nodes.

2. **Get Canvas Data** (or similar name)  
   - Inputs: key (required), optional “version” (e.g., current vs previous, or “N steps back”).  
   - Effect: Reads the value (and optionally metadata) for that key at the requested version.  
   - Output: value (and optionally version/timestamp) in the event payload for downstream use.

3. **Optional: List / History**  
   - List keys under a prefix and/or return history (list of versions) for a key.  
   - Can be a separate component or optional behavior of Get (e.g., “get history for key”).  
   - Enables “what were the last N values?” for rollback or auditing.

Naming and exact UX (e.g., “Set Data” / “Get Data” vs “Canvas Data: Set” / “Canvas Data: Get”) can be decided in design.

## Example Flows

### Last deployed version

1. Deploy component runs and outputs a version (e.g., `v1.2.3` or git SHA).
2. **Set Canvas Data** component: key = `app/backend/last_version`, value = `{{ $["Deploy"].version }}`.
3. In a later run (e.g., rollback workflow): **Get Canvas Data** with key `app/backend/last_version`, version = “previous”, then call rollback API with that version.

### Ephemeral machines

1. After creating VMs/machines, a step collects their IDs.
2. **Set Canvas Data** component: key = `ephemeral/machines`, value = JSON array of IDs (or append via read-modify-write in a future iteration).
3. Teardown workflow: **Get Canvas Data** for `ephemeral/machines`, then iterate and delete each machine.

### History lookup

1. **Get Canvas Data** with key `app/backend/last_version`, version = “current” → use for display or comparison.
2. **Get Canvas Data** with same key, version = “previous” (or “2 steps back”) → use for rollback target or diff.

## Design Considerations

### Storage and history

- **Backing store**: Persistent store (e.g., PostgreSQL) with a table(s) for canvas_id, key, value, version/timestamp, and any metadata (e.g., run_id or node_id that wrote it).
- **History retention**: Policy for how long to keep history (e.g., last N versions per key or last T time). To be defined; can start conservative (e.g., last 100 versions or 90 days).

### Key naming and conventions

- Recommend namespacing (e.g., `app/<service>/...`, `ephemeral/...`) to avoid collisions. Documentation and UI can suggest patterns; enforcement can be optional (e.g., no global reserved prefixes initially).

### Security and permissions

- Access to canvas data follows canvas-level permissions: only users (and service contexts) that can run or edit workflows on the canvas can cause data to be read or written via components.
- No separate “canvas data” permission in v1; same as “can run workflow on this canvas”.

### Expression and component context

- Components need access to **canvas id** (and optionally run id / node id) in the execution context so that Set/Get are scoped to the correct canvas. This may require small extensions to the execution context if not already present.

### Limits

- **Key length**: e.g., max 256 bytes.  
- **Value size**: e.g., 64 KB per value.  
- **Keys per canvas**: Optional cap (e.g., 10,000) to avoid abuse.  
- **History depth**: Retain last N versions per key (or by time); drop older ones.

## Success Metrics

- Workflow authors can implement “last deployed version” and “ephemeral machine list” without external state stores.
- Get/Set and “previous value” are used in real workflows (qualitative).
- No increase in critical incidents related to data loss or cross-canvas leakage.

## Open Questions

1. **Append vs overwrite**: Should a component support “append to list” for keys (e.g., ephemeral machines) or only full overwrite? (Append could be a later enhancement.)
2. **Version parameter**: “Previous” vs “N steps back” vs “at timestamp” — which to support in v1?
3. **UI**: Read-only view of canvas data (keys + current value + history) in the canvas or org settings?
4. **Naming**: “Canvas Data” vs “Canvas State” vs “Workflow Data” — final product name.

## Implementation Phases (Suggested)

1. **Phase 1 – Data plane**  
   - Model and storage for canvas-scoped key-value with versioned history.  
   - No public API; internal API for use by components only.

2. **Phase 2 – Core components**  
   - “Set Canvas Data” and “Get Canvas Data” core components.  
   - Support “current” and “previous” (or “N back”) in Get.

3. **Phase 3 – Polish**  
   - Optional List/History component or Get options.  
   - Documentation, examples (last deployed version, ephemeral machines).  
   - Optional read-only UI for canvas data.

4. **Phase 4 – Enhancements**  
   - Append/list semantics if needed.  
   - History retention and cleanup.  
   - Any additional version semantics (e.g., “at time”).

---

*This PRD describes the Canvas Data concept and its interaction via new core components. Implementation details (schemas, APIs, component specs) will be refined in technical design docs and implementation tickets.*
