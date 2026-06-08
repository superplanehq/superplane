---
name: superplane-dashboard-and-widgets
description: >-
  Implements and configures SuperPlane canvas dashboards (markdown, node, table,
  chart, number panels), widget data sources, CEL/templates, table row trigger
  actions, and dashboard YAML. Use when editing dashboard UI, panelTypes,
  useWidgetData, WidgetTable, canvas_dashboard_yml, Get/UpdateCanvasDashboard,
  or docs/prd/dashboard-and-widgets.md.
---

# SuperPlane dashboard and widgets

Use this skill when working on **per-canvas dashboards**: the workflow v2 overlay, typed panels, widget renderers, YAML import/export, or backend validation.

**Canonical reference:** [docs/prd/dashboard-and-widgets.md](../../../docs/prd/dashboard-and-widgets.md) — read it for full schemas, examples, and maintenance notes. This skill is the operational subset for agents.

---

## Product rules (do not break)

- One dashboard per **canvas** (not templates). Stored in `canvas_dashboards` as JSON `panels` + `layout`.
- Dashboard mode hides the graph; **12-column** `react-grid-layout` (`DashboardView`).
- **Edit** (panels, layout, YAML import): `canvases:update`, not template, canvas not deleted.
- **Run** (node panel Run, table row actions): same as edit — `InvokeNodeTriggerHook`; UI uses `canRunNodes`.
- YAML import is **replace-all** (max **50** panels, **1 MiB** payload).
- User-facing name: **SuperPlane** (capital P).
- Row actions are **`kind: trigger` only** — they fire trigger nodes; they do not call HTTP Request nodes directly.

---

## Layer map

| Layer | Key paths |
| --- | --- |
| Host | `web_src/src/pages/workflowv2/index.tsx` — mode, feature flag, header |
| Overlay gate | `dashboard/WorkflowDashboardOverlay.tsx` |
| Overlay | `dashboard/DashboardOverlay.tsx` — query/mutation, context |
| Context | `dashboard/DashboardContext.tsx`, `DashboardContextProvider.tsx` |
| Trigger hook | `dashboard/useDashboardTriggerNode.ts` |
| Grid | `dashboard/DashboardView.tsx` |
| Local state | `dashboard/useDashboardPanelState.ts` (500ms debounced save) |
| Schema | `dashboard/panelTypes.ts` — types, templates, validators, `normalizeTablePanelContent` |
| YAML (FE) | `dashboard/dashboardYaml.ts`, `DashboardYamlModal.tsx` |
| Widget data | `dashboard/widget/useWidgetData.ts` |
| Widget UI | `dashboard/widget/WidgetTable.tsx`, `WidgetChart.tsx`, `WidgetNumber.tsx` |
| Backend | `pkg/models/canvas_dashboard.go`, `canvas_dashboard_yml.go` |
| API | `pkg/grpc/actions/canvases/get_canvas_dashboard.go`, `update_canvas_dashboard.go` |
| Proto | `protos/canvases.proto` — `DashboardPanel`, `CanvasDashboard` |

**Invariant:** `panelTypes.ts` validators, `canvas_dashboard_yml.go` (or `console_yml.go` on the renamed console path), widget `types.ts`, and the generated JSON Schema at `api/schemas/console-panel-content.schema.json` must all agree. Frontend fast-fail; backend authoritative on import; the JSON Schema is the documentation mirror.

**Node references:** always accept **id or name** via `resolveDashboardNode` in `DashboardContext.tsx`.

### Generated type contracts (regeneration lockstep)

When a panel `content` shape changes you must regenerate the wire contracts so the proto enum, the OpenAPI spec, and the JSON Schema all stay in lockstep:

- `Console.Panel.type` is a **proto enum** (`pb.Console_Panel_Type`) — fail-closed in serialization (`pkg/grpc/actions/canvases/console_serialization.go::panelTypeToModel`). Adding a new panel kind means: (1) adding the value to `protos/canvases.proto`, (2) adding the lowercase constant to `pkg/models/console_yml.go`, (3) extending the `panelTypeFromModel`/`panelTypeToModel` switches, (4) adding the lowercase form to `PANEL_TYPES` in `web_src/src/pages/app/console/panelTypes.ts`, and (5) adding it to both `API_TO_PANEL_TYPE` and `PANEL_TYPE_TO_API`. Also extend the CLI tables in `pkg/cli/commands/apps/console/convert.go`.
- `Console.Panel.content` is documented via a generated JSON Schema. The source-of-truth TS union lives at `web_src/src/pages/app/console/schema/panelContent.ts` and reuses the existing per-type interfaces. `npm run generate:console-schema` writes `api/schemas/console-panel-content.schema.json`; `scripts/inject-console-panel-content-schema.mjs` then attaches it to the `ConsolePanel` definition in `api/swagger/superplane.swagger.json` as `x-content-schema` (stringified, plus an `x-content-schema-ref` pointing to the file). Both run automatically from `make openapi.spec.gen`.
- Run `make pb.gen` to regenerate everything end-to-end (proto + gateway + OpenAPI spec + schema injection + Go SDK + TS SDK).

---

## Panel types

| `type` | Runtime | Main `content` |
| --- | --- | --- |
| `markdown` | GFM body with `{{ name.field }}` interpolation | `title?`, `body?`, `variables?` |
| `node` | Status chip + optional Run | `node`, `showRun?`, `triggerName?` |
| `table` | `WidgetTable` | `dataSource`, `render.kind: "table"` |
| `chart` | `WidgetChart` (SVG) | `dataSource`, `render.kind: "chart"` |
| `number` | `WidgetNumber` | `dataSource`, `render.kind: "number"` |

New panels: `templateForPanelType` in `panelTypes.ts`. Draft states (e.g. empty memory namespace) should stay valid where possible.

---

## Data sources (`useWidgetData`)

```ts
{ kind: "memory", namespace: string, fieldPath?: string }
{ kind: "executions", node?: string, limit?: number }
{ kind: "runs", limit?: number }
```

| Kind | Query | Notes |
| --- | --- | --- |
| `memory` | `useCanvasMemoryEntries` | Filter by namespace; `fieldPath` flattens nested lists (`memoryRow.ts`) |
| `executions` | `useInfiniteCanvasEvents` | Flatten `executions[]`; optional node filter; eager pages until `limit` or cap (~500 events) |
| `runs` | `useInfiniteCanvasRuns` | `totalCount` for count KPIs |

Execution rows get `status`, `nodeName`, `durationMs`. Status vocabulary: `passed`, `failed`, `running`, `pending`, `cancelled`, `unknown`.

---

## Table panels (most complex)

### Columns

Non-empty `field`; optional `label`, `format` (`text`, `number`, `status`, `relative`, `link`, …), `show`, `href`.

### Filters

`render.where[]` — AND list; ops: `eq`, `neq`, `contains`, `not_contains`, `gt`, `lt`, `exists`, `not_exists`.

### Row actions (trigger)

Required: `kind: trigger`, `node` (id or name). Optional: `hook` (default `run`), `template`, `payload`, `confirm`, `show`, `variant`, `icon`.

Runtime flow: `WidgetTable` → `mergeTriggerPayload` → `onTriggerNode` → `useDashboardTriggerNode` → `InvokeNodeTriggerHook` → invalidate events/runs/memory queries.

Legacy fields normalized in FE: `target` → `node`, `triggerName` → `template`.

### Expressions

- **`{{ CEL }}`** — `cel-js` via `widget/celExpr.ts`; row env + `now` (Unix seconds).
- **Legacy** `show` — e.g. `status == "running"` (`showExpression.ts`, `rowVisibility.ts`).
- Prefer structured `where` for simple validated filters.

**Lint:** loose equality in legacy expressions is intentional (scalar normalization). Do not add `eslint-disable` for `==` in dashboard code; refactor instead.

Editor memory hints: `MemoryDiscoveryPanel.tsx`, `useMemoryCatalog.ts` (suggestions only; YAML still validated).

### Markdown variables

- `content.variables[]` carries named live data refs; body uses `{{ name.field }}` (or `{{ name.$["Node"].data.x }}` for runs).
- Sources: `{ kind: "memory", namespace, orderBy?, direction?, matches? }` (first row wins; default `orderBy: createdAt desc`) or `{ kind: "run", select: latest | latest_passed | latest_failed }`.
- Resolution lives in `useMarkdownVariables.ts`; interpolation in `markdownInterpolation.ts` (reuses `celExpr.compileTemplate`/`evalTemplate`). Validation: `markdownVariables.ts` (FE) + `validateMarkdownContent` in `canvas_dashboard_yml.go` (BE).
- Run vars expose `status`, `nodeName`, `payload`, `durationMs`, and a `$` map of node executions (same shape as the table widget).

---

## Chart and number

**Chart** `render.type`: `bar`, `stacked-bar`, `line`, `area`, `donut`. `xField` + `series[]`; omit `series[].field` to count rows per bucket.

**Number** aggregations: `count`, `sum`, `avg`, `min`, `max`, `first`, `last` — non-`count` requires `field`.

---

## YAML

```yaml
apiVersion: v1
kind: Dashboard
metadata:
  canvasId: <uuid>   # export only; ignored on import
  name: <display>
spec:
  panels: [{ id, type, content }]
  layout: [{ i, x, y, w, h, minW?, minH? }]
```

- FE: `dashboardYaml.ts` — parse/serialize + `validatePanelContent`
- BE: `DashboardFromYML` / `DashboardToYML` in `canvas_dashboard_yml.go`
- Unknown fields rejected; missing `panels`/`layout` → empty lists

---

## Agent workflows

### Fix a dashboard bug

1. Reproduce in dashboard mode (not template); note panel `type` and `dataSource.kind`.
2. Trace: panel card → `useWidgetData` → widget renderer → (if trigger) `useDashboardTriggerNode`.
3. Check permissions in `pkg/authorization/interceptor.go` if RPC-related.
4. Add/update test under `web_src/src/pages/workflowv2/dashboard/**/*.spec.ts`.

### Add or change panel `content` fields

1. `widget/types.ts` (if widget-facing)
2. `panelTypes.ts` — interface, `templateForPanelType`, `validatePanelContent`, normalization
3. `canvas_dashboard_yml.go` — mirror validation
4. Panel card + form component
5. YAML tests: `dashboardYaml.spec.ts`, `canvas_dashboard_yml_test.go`

### Add a new panel type

1. `PANEL_TYPES`, `PANEL_TYPE_META`, validator, template
2. `AllowedDashboardPanelTypes` in Go
3. `*PanelCard.tsx` + `DashboardView` `PanelCardRouter`
4. Update [docs/prd/dashboard-and-widgets.md](../../../docs/prd/dashboard-and-widgets.md)

### Add a data source kind

1. Extend types in `widget/types.ts` + `panelTypes.ts`
2. `DataSourceForm.tsx` editor
3. Branch in `useWidgetData.ts`
4. Backend YAML validator + tests

### Configure memory table (user/agent task)

Use PRD example; namespace must match canvas memory keys. Row actions target **trigger nodes** only.

---

## Verification

```bash
# Frontend unit tests (dashboard package)
cd web_src && npm run test:run -- src/pages/workflowv2/dashboard

# After UI edits (Docker dev env)
make format.js
make check.lint.ui
make check.build.ui

# After Go validation/API edits
make format.go
make lint
make check.build.app
go test ./pkg/models -run 'TestDashboard|TestValidateDashboardContent'
go test ./pkg/grpc/actions/canvases -run CanvasDashboard
```

---

## Repo conventions

- No `web_src/src/utils/*` — use `lib/` or `hooks/`.
- Dashboard has **strict ESLint budget** — refactor touched code; do not raise the budget.
- Split large components for Fast Refresh where the codebase already does.
- **Never** hand-write DB migrations; `make db.migration.create NAME=<dash-name>` if persistence changes.
- AGENTS.md: protobuf enum mapping, authorization on new RPCs.

---

## Quick file index

| Task | Start here |
| --- | --- |
| Grid / add panel | `DashboardView.tsx`, `useDashboardPanelState.ts` |
| Table CEL / filters / actions | `WidgetTable.tsx`, `celExpr.ts`, `evalTableWhere.ts`, `mergeTriggerPayload.ts` |
| Table editor | `TablePanelForm.tsx`, `TablePanelFormRows.tsx` |
| Trigger from dashboard | `useDashboardTriggerNode.ts`, `dashboardTriggerParameters.ts` |
| Node status chip | `NodePanelCard.tsx`, `deriveNodeStatuses.ts` |
| Header dashboard actions | `dashboardHeaderActions.ts`, `useDashboardModeActions.ts` |
| API hooks | `web_src/src/hooks/useCanvasData.ts` — `useCanvasDashboard`, `useUpdateCanvasDashboard` |
