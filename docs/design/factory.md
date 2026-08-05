# Factory

A **factory** is an org-scoped production unit: it holds **work orders**, defines **lines** that route work through factory-owned **apps**, and tracks **executions** as canvas runs.

Behind the experimental feature flag `factories` (`pkg/features/features.go`). Enable per org via Installation Admin → Organization → Experimental features, or `POST /admin/api/organizations/{orgId}/experimental-features/factories`. All `/api/v1/factories/*` routes require the flag.

## Model

| Entity | Role |
| --- | --- |
| **Factory** | Named container (`name`, `description`). Has lines and work orders. |
| **Line** | Named sequence of steps. Each step runs one factory app entrypoint. |
| **Step** | Type `runApp` today. References a factory-owned canvas by ID and an `onRun` trigger node as entrypoint. |
| **Work order** | Unit of work: title, description, assignees, `created_by`. States: `open` / `closed`. Close result: `completed` or `rejected`. |
| **Execution** | One line step run for a work order. Links to a canvas run; tracks pending / running / finished and pass / fail / cancel. |
| **Factory app** | Canvas with `factory_id` set. Listed under the factory; steps must point at these apps. |

Lines are stored on `factory_lines`; steps are JSON on the line row. Work orders and assignees live in `factory_work_orders` / `factory_work_order_assignees`. Executions in `factory_work_order_executions`.

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

## Work order lifecycle

- **Create** — manual in UI/API today (`POST …/orders`).
- **Assign** — optional assignees (`PATCH …/assignees`).
- **Dispatch** — to a line (starts execution).
- **Close** — `completed` or `rejected` (`PATCH …/close`). Open orders only.

Display status in the UI (open orders): unassigned is an assignee filter only; status derives from executions — open, running, failed. Closed orders show completed / rejected.

## API

REST gateway on `protos/factories.proto`:

- Factories: list, create, describe (includes lines).
- Lines: create, update.
- Apps: list factory-owned canvases.
- Work orders: list (filters: state, result, assignees, unassigned), create, describe, update assignees, dispatch, close.

Permissions use the `factories` resource (`read`, `create`, `update`).

## UI

When the flag is on:

- **Home** — Factories section alongside Apps; link to full list.
- **`/factories`** — list and create factories.
- **Factory detail** — work orders (filters: My Work / Unassigned / All; status pills), dispatch popover, factory apps sidebar, lines sidebar.
- **Work order detail** — activity timeline, assignees, dispatch / complete / reject.
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
- Dedicated factory components on canvases (beyond passing work order in run input).
- Auto-close work order when a line finishes all steps.
