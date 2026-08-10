# Factory

A **factory** is an org-scoped production unit: it holds **work orders**, defines **lines** that route work through factory-owned **apps**, and tracks **executions** as canvas runs.

Behind the experimental feature flag `factories` (`pkg/features/features.go`). Enable per org via Installation Admin → Organization → Experimental features, or `POST /admin/api/organizations/{orgId}/experimental-features/factories`. All `/api/v1/factories/*` routes require the flag.

## Model

| Entity | Role |
| --- | --- |
| **Factory** | Named container (`name`, `description`). Has lines and work orders. |
| **Line** | Named sequence of steps. Each step runs one factory app entrypoint. |
| **Step** | Type `runApp` today. References a factory-owned canvas by ID and an `onRun` trigger node as entrypoint. |
| **Work order** | Unit of work: title, description, assignees, `created_by`. States: `draft` → `open` → `closed` (with `open ↔ draft` back-to-draft, `closed → open` reopen, and `draft → closed` abandon). Close result: `completed`, `rejected`, or `failed` (restricted per source state — see lifecycle table). |
| **Work-order artifact** | Typed output attached to a work order (`pr` with a required URL, or `markdown` note). All extra content lives in a free-form JSONB `data` map — markdown notes typically use `data.body` for the inline content. |
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
    "factory_id": "...",
    "source": {
      "issue": { "number": 42, "title": "..." }
    }
  }
}
```

`source` is present when the work order was created by a factory-app component
run. Manual work orders omit it. `source` is the root trigger event from the
linked source run (`source_run_id` on the work order record).

Expressions on a dispatched run should prefer `order()` over
`root().data.work_order`. `order()` resolves the live work order for the
current run (`id`, `title`, `description`, `factory_id`, `state`, `result`,
`source`) and returns `nil` when the run is not attached to a work order.
`order().artifacts` is a list field loaded lazily only when the expression
references it (e.g. `none(order().artifacts, {#.type == "pr"})`).
`root().data.work_order` remains the onRun snapshot and does not include
artifacts.

## Work order lifecycle

States (persisted on `factory_work_orders.state`; `result` is set only on `closed`):

```
[*] → draft ─────→ open ─────→ closed
        │  ↑        ↓             ↓
        │  └── back ┘        ← reopen
        │      to
        │     draft
        └────────── abandon (rejected) ──────────→ closed
```

| Transition | How | Notes |
| --- | --- | --- |
| `→ draft` | `POST …/orders` | Every new work order starts as `draft`. |
| `draft → open` | `PATCH …/orders/{id}/dispatch` **or** `PATCH …/status` (`STATE_OPEN`) | Dispatch auto-promotes `draft` → `open` and records a status event. Explicit `PATCH …/status` also works ("open it without a dispatch yet"). |
| `open → closed` | `PATCH …/orders/{id}/close` (legacy) **or** `PATCH …/status` (`STATE_CLOSED` + result) | Close **requires** a result: `completed`, `rejected`, or `failed`. |
| `draft → closed` | `PATCH …/status` (`STATE_CLOSED`, `RESULT_REJECTED`) | Abandon-before-dispatch. Only `rejected` is valid — `completed` / `failed` imply the order actually ran. |
| `closed → open` | `PATCH …/status` (`STATE_OPEN`) | Reopens the order and clears its `result`. |
| `open → draft` | `PATCH …/status` (`STATE_DRAFT`) | "Back to draft" affordance when a run needs re-scoping. |

Guardrails:

- `dispatch` only works from `draft` or `open`, and still requires no active execution.
- Valid close results depend on the source state (per `factoryWorkOrderCloseResultsByFromState`): `open → closed` accepts `completed`, `rejected`, `failed`; `draft → closed` accepts `rejected` only.
- `UpdateStatus` is the single writer for the lifecycle; `Close(...)` is kept as a thin wrapper for the existing REST endpoint and canvas components.

Every transition writes exactly one `order.status.updated` event (`fromState`, `toState`, `fromResult`, `toResult`) — the sole authoritative lifecycle event. When the transition is caused by a canvas run, the event also carries `automation` (line + step + node) and `run` + `app` refs so the timeline can attribute it back to the caller. On the first `draft → open`, the originating run/app snapshot from `SourceRunID` is included even when no automation is present.

**Display status** in the UI derives both from `state` and from executions:

| Persisted state | Derived UI status | Notes |
| --- | --- | --- |
| `draft` | Draft | Being scoped. Dispatchable — the first dispatch promotes it to `open`. |
| `open`, no active execution | Open | Idle between runs. |
| `open`, active execution | Running | Line step in flight. |
| `open`, last execution failed | Failed | Attention section. |
| `closed`, `result=completed` | Completed | |
| `closed`, `result=rejected` | Rejected | |
| `closed`, `result=failed` | Failed (closed) | Same red styling as an in-flight failure. |

## Comments and artifacts

- **Comments** are timeline-only. They persist as `order.comment.added` events with `{ body, author { kind, userId?, automation? } }`. `kind` is `user` or `automation`. `user` comments carry the authenticated caller's id; `automation` comments carry an `automation` ref (`{ nodeId, nodeName, appId, appName }`) captured from the executing canvas node so the timeline can render "commented via `<node>` in `<app>`" without any free-form author label. The UI renders comments inline in the activity timeline; automation comments show a small badge.
- **Artifacts** are first-class rows in `factory_work_order_artifacts`. They can be created by the `addWorkOrderArtifact` canvas component (automation authorship), by `POST …/artifacts` (interactive user authorship), or by the CLI (`superplane factory artifacts add`). Each artifact has a required `type` (`pr`, `markdown`, or `branch`) plus a JSONB `data` map that carries everything else — the on-wire shape is `{ id, type, data, createdBy, createdAt }`, and `url`, `title`, `number`, `body`, `name`, and any free-form extras all live inside `data`. `pr` requires `data.url`; `markdown` requires `data.body`; `branch` requires `data.name`. `data.url` is optional on any other type, but whenever it is present — regardless of type — the model enforces that it is an absolute `http(s)` URL with a host and rejects `javascript:`, `data:`, `file:`, `mailto:`, and protocol-relative URLs, so no caller can smuggle a dangerous scheme into a link teammates will click. The client mirrors this check with `lib/safeExternalUrl` before rendering `href`s. Creation is transactional with an `order.artifact.added` event that includes the artifact `data` so the timeline can render markdown inline without a second fetch. The Work Order detail sidebar lists artifacts read-only (no attach composer in the UI yet).

## API

REST gateway on `protos/factories.proto`:

- Factories: list, create, describe (includes lines).
- Lines: create, update.
- Apps: list factory-owned canvases.
- Work orders: list (filters: state, result, assignees, unassigned), create, describe, update assignees, dispatch, close, **update status**, **add comment**, **list artifacts**, **create artifact**, list events.

New RPCs (all under `/api/v1/factories/{factoryId}/orders/{orderId}/…`):

| RPC | HTTP | Action |
| --- | --- | --- |
| `UpdateWorkOrderStatus` | `PATCH …/status` | `work_orders:update` |
| `AddWorkOrderComment` | `POST …/comments` | `work_orders:update` |
| `ListWorkOrderArtifacts` | `GET …/artifacts` | `work_orders:read` |
| `CreateWorkOrderArtifact` | `POST …/artifacts` | `work_orders:update` |

Factory structure (create/update/delete factory + lines) uses the `factories` resource.
Work-order lifecycle (create/list/describe orders, status, assignees, dispatch, close, comments, artifacts, events) uses the separate `work_orders` resource (`read`, `create`, `update`). That lets limited tokens (runners/agents) mutate work orders without `factories:update`. All endpoints stay behind the `factories` experimental feature flag.

## Canvas components

Built-in factory components in `pkg/components/factory/`, registered on the standard palette (behind the `factories` flag on the FE via `FACTORY_BLOCK_NAMES`).

| Component | Config | Effect |
| --- | --- | --- |
| `createWorkOrder` | `title`, `description`, `assignees[]` | Creates a work order in `draft`. |
| `updateWorkOrderStatus` | `status` (`draft`/`open`/`closed`), conditional `result` (`completed`/`rejected`/`failed`, required for `closed`; only `rejected` is valid when closing from `draft`) | Runs the FSM and records an enriched `order.status.updated`. |
| `addWorkOrderComment` | `body` | Appends an `order.comment.added` event. Authorship is derived from the executing canvas node (`kind = automation`, `automation = { nodeName, appName, lineName, stepName }`). |
| `addWorkOrderArtifact` | `artifactType` (`pr`/`markdown`); for `pr`: required `url`, optional `number`; for `markdown`: required `body`; optional `title` on both; free-form `data` (`{name, value}` list, merged into the artifact's `data` map — typed fields win on name collisions) | Creates the artifact row + `order.artifact.added` event. |

Components target the work order that owns the current canvas run — the link is the `factory_work_order_executions` row created when the run was dispatched, so component authors don't have to (and can't) supply the work order ID by hand. Canvas invocations attribute events to the caller line: the `automation` payload snapshots `{ nodeId, nodeName, appId, appName, lineId, lineName, stepIndex, stepName }` at write time, and status updates additionally carry the current `run` + `app` refs so the timeline can link straight back to the originating run. No acting user is attributed on canvas-driven events.

## UI

When the flag is on:

- **Home** — Factories section alongside Apps; link to full list.
- **`/factories`** — list and create factories.
- **Factory detail** — work orders (owner pills: My Work / Unassigned / All; status pills: All / **Active (default: draft + open + running + failed)** / Draft / Open / Running / Failed / Completed / Rejected; the Failed pill also matches orders closed as failed; the "Work Orders" badge counts orders matching the default `Active` filter). Dispatch popover, factory apps sidebar, lines sidebar. The "failed" display status is derived from the **latest finished execution**: a passing retry supersedes an earlier failure, and failures older than `order.updatedAt` are treated as belonging to a previous attempt so reopening a closed order clears the failed pill until a new dispatch actually fails.
- **Work order detail** — status-aware action bar (`Draft`: Dispatch / Reject; `Open`: Dispatch / Back to Draft / Complete / Reject; `Closed`: Reopen; the `Back to Draft` button hides while a step execution is in flight, per the FSM guard). Inline **comment composer**, activity timeline (comments, status transitions, artifacts, dispatches) with line-centric automation attribution, assignees panel, and a read-only **Artifacts** sidebar.
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

## CLI

Minimal factory CLI for artifacts (flag-based; optional active factory):

```bash
superplane factory active [factory]   # set/show active factory (name or UUID)
superplane factory artifacts list --factory <name-or-id> --order-id <uuid>
superplane factory artifacts add --order-id <uuid> --type <pr|markdown|branch> [flags]
```

`--factory` accepts a factory name or UUID. When omitted, the active factory
from `superplane factory active` is used. `--order-id` is the work-order UUID.
`--type` is `pr`, `markdown`, or `branch`. Typed flags build the API `data` map:

```bash
superplane factory artifacts list \
  --factory shipping \
  --order-id "$OID"

# Uses the active factory
superplane factory artifacts add \
  --order-id "$OID" \
  --type markdown \
  --title "PLAN.md" \
  -f ./PLAN.md

superplane factory artifacts add \
  --factory shipping \
  --order-id "$OID" \
  --type markdown \
  --title "PLAN.md" \
  -f ./PLAN.md

superplane factory artifacts add \
  --order-id "$OID" \
  --type pr \
  --url https://github.com/org/repo/pull/7 \
  --number 7

superplane factory artifacts add \
  --order-id "$OID" \
  --type branch \
  --name feature/login
```

For markdown, provide `--body` or `-f` / `--file` (file contents become `data.body`). Full factory CLI surface beyond artifacts is still pending.

### Runner env auth (factory-dispatched runs)

When a Runner node (`runBash` / `runCommands` / `runPython` / `runJavaScript`) executes on **Host** as part of a factory work-order dispatch, SuperPlane injects:

| Variable | Purpose |
|----------|---------|
| `SUPERPLANE_URL` | Public API base URL |
| `SUPERPLANE_TOKEN` | Short-lived scoped JWT bound to this node execution + org (`work_orders:read|update:<order_id>` only; no `factories:update`). Dies when the execution finishes; always freshly minted (not sticky across tasks). |
| `SUPERPLANE_FACTORY_ID` | Factory UUID |
| `SUPERPLANE_ORDER_ID` | Work-order UUID |

CLI already accepts non-interactive auth via `SUPERPLANE_URL` + `SUPERPLANE_TOKEN` (no `connect`). Env auth cannot persist an active factory, so pass `--factory` / `--order-id` explicitly:

```bash
superplane factory artifacts add \
  --factory "$SUPERPLANE_FACTORY_ID" \
  --order-id "$SUPERPLANE_ORDER_ID" \
  --type markdown \
  --title PLAN.md \
  -f ./PLAN.md
```

The `superplane` binary must be on `PATH`. **Host** mode fleets get it from fleet-manager user-data at EC2 boot (GitHub latest release). Docker mode does not get auto token inject yet — install the CLI via **Setup commands** if needed. Non-factory runs do not get an auto token; set env/secrets manually if needed. On factory-linked Host runs these keys are always overwritten so a prior task or node env cannot sticky-reuse credentials.

## Not implemented yet

- Broader CLI (`superplane factory` list/create/dispatch/…).
- Work orders sourced from external systems or factory-app components.
- Full PRD approval flow (gated approvals on `open → closed`).
- Auto-close work order when a line finishes all steps.
- Editing or deleting comments and artifacts.
- UI attach-artifact composer (sidebar remains read-only; API/CLI/canvas create still refresh via websocket).
- Additional artifact types (screenshots, recordings). The schema is extensible via `type` + JSONB `data` when needed.
