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
- Artifacts optionally carry a `key` (`VARCHAR(512)`, nullable, unique per factory via a partial index that excludes `NULL`) so a work order can be looked up from an external identifier — e.g. a pull request's URL — without already knowing the order id. `addWorkOrderArtifact`'s `artifactKey` field sets it; `findWorkOrder` (`by: artifactKey`) reads it. Setting it is currently only possible from the canvas component — the artifact-key field is not yet exposed on the REST API or CLI, so artifacts created that way can't be tagged with a key (known gap).

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
| `findWorkOrder` | `by` (`id`/`artifactKey`), conditional `orderId` or `artifactKey` | Resolves a work order by id or by an artifact's `key`, without needing a `factory_work_order_executions` row. Emits `workOrder.found` on the `found` channel on a match, or `workOrder.notFound` on the `notFound` channel otherwise — never fails the run just because nothing matched. |
| `updateWorkOrderStatus` | required `orderId` (defaults to `{{ order().id }}`), `status` (`draft`/`open`/`closed`), conditional `result` (`completed`/`rejected`/`failed`, required for `closed`; only `rejected` is valid when closing from `draft`) | Runs the FSM and records an enriched `order.status.updated`. |
| `addWorkOrderComment` | required `orderId` (defaults to `{{ order().id }}`), `body` | Appends an `order.comment.added` event. Authorship is derived from the executing canvas node (`kind = automation`, `automation = { nodeName, appName, lineName, stepName }`). |
| `addWorkOrderArtifact` | required `orderId` (defaults to `{{ order().id }}`), `artifactType` (`pr`/`markdown`/`branch`); for `pr`: required `url`, optional `number`; for `markdown`: required `body`; for `branch`: required `name`; optional `title` on `pr`/`markdown`; optional `artifactKey`; free-form `data` (`{name, value}` list, merged into the artifact's `data` map — typed fields win on name collisions) | Creates the artifact row + `order.artifact.added` event. |

`updateWorkOrderStatus` / `addWorkOrderComment` / `addWorkOrderArtifact` always target a work order explicitly via `orderId` — there is no implicit fallback. The field defaults to `{{ order().id }}`, which resolves the work order driving the current canvas run (via the `factory_work_order_executions` row created when the run was dispatched from a factory line) and only works in that context. Runs not dispatched from a line, e.g. a flow triggered by `github.onPullRequest`, must replace the default with an id resolved another way — typically `{{ previous().data.workOrder.id }}` after a `findWorkOrder` step. Canvas invocations attribute events to the caller line: the `automation` payload snapshots `{ nodeId, nodeName, appId, appName, lineId, lineName, stepIndex, stepName }` at write time (line/step are omitted when the run isn't attached to one), and status updates additionally carry the current `run` + `app` refs so the timeline can link straight back to the originating run. No acting user is attributed on canvas-driven events.

### Closing a work order from an external merge event

`findWorkOrder` plus the new `orderId` fields let a flow that starts on `github.onPullRequest` (not a factory line) close the work order a merged pull request belongs to:

```yaml
github.onPullRequest (merged)
  -> if ($.pull_request.merged == true)
  -> findWorkOrder (by: artifactKey, artifactKey: {{ $.pull_request.html_url }})
       [found]    -> updateWorkOrderStatus (orderId: {{ previous().data.workOrder.id }}, status: closed, result: completed)
       [notFound] -> (no-op; the merged PR isn't tied to a tracked order)
```

This assumes an earlier `addWorkOrderArtifact` step (e.g. when the PR was opened) tagged an artifact with `artifactKey: {{ $.pull_request.html_url }}` so `findWorkOrder` has something to match against.

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

For markdown, provide `--body` or `-f` / `--file` (file contents become `data.body`).

Work order commands (`orders`/`order`), for creating, dispatching, assigning,
listing, and inspecting work orders:

```bash
superplane factory orders list [flags]
superplane factory orders describe --order <uuid> [flags]
superplane factory orders create --title <title> [flags]
superplane factory orders dispatch --order <uuid> --line <line-name>
superplane factory orders assign --order <uuid> --assignee <id-or-email> [flags]
```

`orders list` flags:

- `--factory` — factory name or UUID (default: active factory).
- `--assignees` — filter by assignee, repeatable/comma-separated. Accepts
  user UUIDs or emails; emails are resolved against the organization's
  members.
- `--state` — filter by work order state, repeatable. Accepts the proto
  enum token (`STATE_DRAFT`, `STATE_OPEN`, `STATE_CLOSED`) or a short
  case-insensitive form (`draft`, `open`, `closed`). Defaults to `open`
  when omitted, so `orders list` only shows open work orders unless you
  ask otherwise. Pass `--state all` to remove the state filter entirely
  and see work orders in every state.
- `--result` — filter by work order result, repeatable. Accepts the proto
  enum token (`RESULT_COMPLETED`, `RESULT_REJECTED`, `RESULT_FAILED`) or a
  short case-insensitive form (`completed`, `rejected`, `failed`).
- `--unassigned` — only show work orders with no assignees.

```bash
superplane factory orders list --factory shipping --state open

superplane factory orders list \
  --assignees alice@example.com,bob@example.com \
  --result failed

superplane factory orders list --unassigned

# Show work orders in every state (draft, open, and closed)
superplane factory orders list --state all
```

`orders describe` shows a work order's title, assignees, description,
comments, and full event timeline (status changes, assignee changes,
comments, artifacts added, and line step executions), oldest event first:

- `--factory` — factory name or UUID (default: active factory).
- `--order` — work order UUID (`--order-id` is accepted as a deprecated
  alias, for consistency with `artifacts`' `--order-id`).

```bash
superplane factory orders describe --factory shipping --order "$OID"
```

`--output json`/`--output yaml` on `orders describe` returns
`{order, comments, events, eventsTruncated}` so scripts get full fidelity
in one call; the event timeline is capped at the API's max page size (200)
and `eventsTruncated` is `true` when there are more.

`orders create` creates a work order in `draft` state:

- `--factory` — factory name or UUID (default: active factory).
- `--title` — work order title (required).
- `--description` — description text (inline). Mutually exclusive with
  `--file`.
- `-f`/`--file` — read the description from a file, or `-` for stdin.
  Mutually exclusive with `--description`.
- `--assignee` — assignee user UUID or email, repeatable. When omitted
  entirely, the work order is assigned to the user running the command
  (the API itself does not default assignees; the CLI does).

```bash
superplane factory orders create --title "Ship the feature" --description "..."

superplane factory orders create \
  --title "Ship the feature" \
  -f ./description.md \
  --assignee alice@example.com \
  --assignee bob@example.com
```

`orders dispatch` dispatches a work order to a factory line, starting its
execution. A `draft` work order moves to `open` on its first dispatch;
dispatching fails if the line doesn't exist, has no steps, or the order
already has an active execution:

- `--factory` — factory name or UUID (default: active factory).
- `--order` — work order UUID (`--order-id` is accepted as a deprecated
  alias).
- `--line` — target factory line's name (required).

```bash
superplane factory orders dispatch --order "$OID" --line build
```

`orders assign` **sets** a work order's assignee list — it replaces the
existing assignees with exactly the ones given, rather than adding to them
(there is no additive assign/unassign endpoint):

- `--factory` — factory name or UUID (default: active factory).
- `--order` — work order UUID (`--order-id` is accepted as a deprecated
  alias).
- `--assignee` — assignee user UUID or email, repeatable; at least one is
  required (clearing all assignees is out of scope for this command).

```bash
superplane factory orders assign --order "$OID" --assignee alice@example.com --assignee bob@example.com
```

## Not implemented yet

- `superplane factory orders close`/`comment` and other work order status
  transitions (backend RPCs exist; CLI-side `create`/`dispatch`/`assign`
  are implemented, these are not yet).
- Work orders sourced from external systems or factory-app components.
- Full PRD approval flow (gated approvals on `open → closed`).
- Auto-close work order when a line finishes all steps.
- Editing or deleting comments and artifacts.
- UI attach-artifact composer (sidebar remains read-only; API/CLI/canvas create still refresh via websocket).
- Additional artifact types (screenshots, recordings). The schema is extensible via `type` + JSONB `data` when needed.
