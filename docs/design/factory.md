# Factory

A **factory** is an org-scoped production unit: it holds **work orders**, defines **lines** that route work through factory-owned **apps**, and tracks **executions** as canvas runs.

Behind the experimental feature flag `factories` (`pkg/features/features.go`). Enable per org via Installation Admin → Organization → Experimental features, or `POST /admin/api/organizations/{orgId}/experimental-features/factories`. All `/api/v1/factories/*` routes require the flag.

## Model

| Entity | Role |
| --- | --- |
| **Factory** | Named container (`name`, `description`). Has lines and work orders. |
| **Line** | Named sequence of steps. Each step runs one factory app entrypoint. |
| **Step** | Type `runApp` today. References a factory-owned canvas by ID and an `onRun` trigger node as entrypoint. |
| **Work order** | Unit of work: title, description, assignees, `created_by`. States: `draft` → `ready` → `open` → `closed`. Close result: `completed`, `rejected`, or `failed`. |
| **Work-order artifact** | Typed output attached to a work order (`pr` with a required URL, or `markdown` with an inline body). Optional structured `data` for richer PR metadata. |
| **Execution** | One line step run for a work order. Links to a canvas run; tracks pending / running / finished and pass / fail / cancel. |
| **Factory app** | Canvas with `factory_id` set. Listed under the factory; steps must point at these apps. |

Lines are stored on `factory_lines`; steps are JSON on the line row. Work orders and assignees live in `factory_work_orders` / `factory_work_order_assignees`. Executions in `factory_work_order_executions`. Artifacts in `factory_work_order_artifacts`. All history is appended to `factory_work_order_events`.

## Dispatch flow

1. User dispatches an **open** work order to a **line name** (`PATCH …/orders/{id}/dispatch`).
2. Backend starts **step 0**: creates a pending canvas run on the step’s app entrypoint with work-order payload in run input.
3. Run finalizer marks the execution finished from run result.
4. If the run **passed**, the next step starts automatically; otherwise the line stops (work order stays open).
5. Only one active execution per work order **per line** at a time.

Run input shape (for app triggers):

```json
{
  "work_order": {
    "id": "...",
    "title": "...",
    "description": "...",
    "factory_id": "..."
  }
}
```

## Work order lifecycle

States (persisted on `factory_work_orders.state`; `result` is set only on `closed`):

```
[*] → draft → ready → open → closed
              ↑        ↑       ↓
              ────────  ← reopen (open or ready)
```

| Transition | How | Notes |
| --- | --- | --- |
| `→ draft` | `POST …/orders` | Every new work order starts as `draft`. |
| `draft → ready` | `PATCH …/orders/{id}/status` (`STATE_READY`) | "Mark ready" in the UI. |
| `ready → open` | `PATCH …/orders/{id}/dispatch` **or** `PATCH …/status` | Dispatch auto-promotes `ready` → `open` and records a status event. |
| `open → closed` | `PATCH …/orders/{id}/close` (legacy) **or** `PATCH …/status` (`STATE_CLOSED` + result) | Close **requires** a result: `completed`, `rejected`, or `failed`. |
| `closed → open` / `closed → ready` | `PATCH …/status` | Reopens the order and clears its `result`. |
| `ready → draft` | `PATCH …/status` (`STATE_DRAFT`) | "Back to draft" affordance. |

Guardrails:

- `dispatch` only works from `ready` or `open`, and still requires no active execution.
- `close` only works from `open`; the API rejects invalid transitions (e.g. `draft → closed`).
- `UpdateStatus` is the single writer for the lifecycle; `Close(...)` is kept as a thin wrapper for the existing REST endpoint and canvas components.

Every transition writes an `order.status.updated` event (`fromState`, `toState`, `fromResult`, `toResult`). Two coarse legacy events fire alongside it so older timeline logic keeps working:

- `order.opened` fires only on the initial `ready → open` promotion. Reopens from `closed` do **not** re-emit `order.opened`; the `order.status.updated` event is authoritative and the timeline renders it as "reopened as Open / Ready".
- `order.closed` fires on any transition into `closed`.

**Display status** in the UI derives both from `state` and from executions:

| Persisted state | Derived UI status | Notes |
| --- | --- | --- |
| `draft` | Draft | Being scoped. |
| `ready` | Ready | Ready to dispatch — also filterable and dispatchable. |
| `open`, no active execution | Open | Idle between runs. |
| `open`, active execution | Running | Line step in flight. |
| `open`, last execution failed | Failed | Attention section. |
| `closed`, `result=completed` | Completed | |
| `closed`, `result=rejected` | Rejected | |
| `closed`, `result=failed` | Failed (closed) | Same red styling as an in-flight failure. |

## Comments and artifacts

- **Comments** are timeline-only. They persist as `order.comment.added` events with `{ body, author { kind, userId?, label? } }`. `kind` is one of `user`, `llm`, or `system`. The UI renders comments inline in the activity timeline; LLM comments show a small badge.
- **Artifacts** are first-class rows in `factory_work_order_artifacts`. Each artifact has a required `type` (`pr` or `markdown`) plus optional `title`, `url`, `body`, and JSONB `data`. `pr` requires `url`; `markdown` requires `body`. Any provided `url` must be an absolute `http(s)` URL with a host — the model rejects `javascript:`, `data:`, `file:`, `mailto:`, and protocol-relative URLs so a user with `factories:update` cannot smuggle a dangerous scheme into a link teammates will click. The client mirrors this check with `lib/safeExternalUrl` before rendering `href`s. Creation is transactional with an `order.artifact.added` event. The Work Order detail sidebar lists artifacts and offers an **Attach** dialog.

## API

REST gateway on `protos/factories.proto`:

- Factories: list, create, describe (includes lines).
- Lines: create, update.
- Apps: list factory-owned canvases.
- Work orders: list (filters: state, result, assignees, unassigned), create, describe, update assignees, dispatch, close, **update status**, **add comment**, **list/attach artifacts**, list events.

New RPCs (all under `/api/v1/factories/{factoryId}/orders/{orderId}/…`):

| RPC | HTTP | Action |
| --- | --- | --- |
| `UpdateWorkOrderStatus` | `PATCH …/status` | `factories:update` |
| `AddWorkOrderComment` | `POST …/comments` | `factories:update` |
| `ListWorkOrderArtifacts` | `GET …/artifacts` | `factories:read` |
| `AddWorkOrderArtifact` | `POST …/artifacts` | `factories:update` |

Permissions use the `factories` resource (`read`, `create`, `update`); all endpoints stay behind the `factories` experimental feature flag.

## Canvas components

Built-in factory components in `pkg/components/factory/`, registered on the standard palette (behind the `factories` flag on the FE via `FACTORY_BLOCK_NAMES`).

| Component | Config | Effect |
| --- | --- | --- |
| `createWorkOrder` | `title`, `description`, `assignees[]` | Creates a work order in `draft`. |
| `updateWorkOrderStatus` | `workOrderId`, `status` (`draft`/`ready`/`open`/`closed`), conditional `result` (`completed`/`rejected`/`failed`, required for `closed`) | Runs the FSM and records `order.status.updated`. |
| `addWorkOrderComment` | `workOrderId`, `body`, `authorKind` (`user`/`llm`/`system`, default `llm`), optional `authorLabel` | Appends an `order.comment.added` event. |
| `addWorkOrderArtifact` | `workOrderId`, `artifactType` (`pr`/`markdown`), conditional `url`/`title`/`body`, optional `data` map | Creates the artifact row + `order.artifact.added` event. |

Components resolve the target work order by explicit ID (config), so a single canvas run can operate on any order in its factory. Canvas invocations record the current `run` reference (and no acting user) on the emitted events.

## UI

When the flag is on:

- **Home** — Factories section alongside Apps; link to full list.
- **`/factories`** — list and create factories.
- **Factory detail** — work orders (owner pills: My Work / Unassigned / All; status pills: All / **Active (default: draft + ready + open + running + failed)** / Draft / Ready / Open / Running / Failed / Completed / Rejected; the Failed pill also matches orders closed as failed; the "Work Orders" badge counts orders matching the default `Active` filter). Dispatch popover, factory apps sidebar, lines sidebar. The "failed" display status is derived from the **latest finished execution**: a passing retry supersedes an earlier failure, and failures older than `order.stateUpdatedAt` (bumped only on lifecycle transitions — not on assignee / comment / artifact writes) are treated as belonging to a previous attempt so reopening a closed order clears the failed pill until a new dispatch actually fails.
- **Work order detail** — status-aware action bar (Mark Ready / Dispatch / Back to Draft / Complete / Reject / Reopen), inline **comment composer**, activity timeline (comments, status transitions, artifacts, dispatches), assignees panel, and an **Artifacts** sidebar with attach-PR / attach-markdown dialog.
- **Factory app canvas** — header link back to factory.

When the flag is off, factories are hidden. The legacy **Setup Factory** starter on `/apps/new` (template install) remains for orgs without the flag.

## Line definition (conceptual)

```yaml
name: bug
steps:
  - name: implement
    type: runApp
    app: { app: "<canvas-uuid>", entrypoint: "start-work" }
  - name: verify
    type: runApp
    app: { app: "<canvas-uuid>", entrypoint: "run-checks" }
```

Entrypoints must be `onRun` triggers on the factory app’s live version.

## Not implemented yet

- CLI (`superplane factory …`).
- Work orders sourced from external systems or factory-app components.
- Full PRD approval flow (`on work order ready` trigger, gated approvals).
- Auto-close work order when a line finishes all steps.
- Editing or deleting comments and artifacts.
- Additional artifact types (screenshots, recordings). The schema is extensible via `type` + JSONB `data` when needed.
