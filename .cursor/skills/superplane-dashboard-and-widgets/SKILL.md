---
name: superplane-dashboard-and-widgets
description: >-
  Implements and configures SuperPlane canvas consoles (markdown, node, table,
  board, chart, number, scorecard panels), widget data sources, CEL/templates,
  table row trigger actions, and console YAML. Use when editing console UI,
  panelTypes, useWidgetData, WidgetTable, WidgetBoard, canvas_console_yml,
  Get/UpdateCanvasConsole, or docs/prd/console-and-widgets.md.
---

# SuperPlane console and widgets

Use this skill when working on **per-canvas consoles**: the console mode overlay, typed panels, widget renderers, YAML import/export, or backend validation.

**Canonical reference:** [docs/prd/console-and-widgets.md](../../../docs/prd/console-and-widgets.md) — read it for full schemas, examples, and maintenance notes. This skill is the operational subset for agents.

---

## Product rules (do not break)

- One console per **canvas** (not templates). Stored as versioned JSON `panels` + `layout` on the canvas version.
- Console mode hides the graph; **12-column** `react-grid-layout` (`ConsoleView`).
- **Edit** (panels, layout, YAML import): `canvases:update`, not template, canvas not deleted.
- **Run** (node panel Run, table / board row actions): same as edit — `InvokeNodeTriggerHook`; UI uses `canRunNodes`.
- YAML import is **replace-all** (max **50** panels, **1 MiB** payload).
- User-facing name: **SuperPlane** (capital P).
- Row actions are **`kind: trigger` only** — they fire trigger nodes; they do not call HTTP Request nodes directly.

---

## Layer map

| Layer | Key paths |
| --- | --- |
| Console page | `web_src/src/pages/app/console/ConsoleView.tsx` — grid, Add Panel picker, YAML modal wiring |
| Context | `console/ConsoleContext.tsx`, `ConsoleContextProvider.tsx` |
| Trigger hook | `console/useConsoleRunTrigger.ts`, `useConsoleTriggerLock.ts` |
| Panel router | `console/ConsolePanelCards.tsx` |
| Schema | `console/panelTypes.ts` — types, templates, validators, `normalizeTablePanelContent`, `normalizeBoardPanelContent` |
| YAML (FE) | `console/consoleYaml.ts`, `ConsoleYamlModal.tsx` |
| Widget data | `console/widget/useWidgetData.ts` |
| Widget UI | `console/widget/WidgetTable.tsx`, `WidgetBoard.tsx`, `WidgetChart.tsx`, `WidgetNumber.tsx`, `WidgetScorecard.tsx` |
| Backend | `pkg/yaml/console.go` — YAML import/export + validators |
| Proto | `protos/canvases.proto` — console panels live on the canvas version |

**Invariant:** `panelTypes.ts` validators (plus satellite modules like `boardPanelContent.ts` and `nodesPanelContent.ts`), `pkg/yaml/console.go`, and widget `types.ts` must agree. Frontend fast-fails; backend is authoritative on import.

**Node references:** always accept **id or name** via `resolveConsoleNode` in `ConsoleContext.tsx`.

---

## Panel types

| `type` | Runtime | Main `content` |
| --- | --- | --- |
| `markdown` | GFM body with `{{ name.field }}` interpolation | `title?`, `body?`, `variables?` |
| `html` | Sanitized HTML body with `{{ name.field }}` interpolation, scoped `<style>`, Tailwind via safelist | `title?`, `body?`, `variables?` |
| `nodes` | Adaptive card: one entry uses the compact single-node layout; multiple entries render as a row list. Optional per-entry Run button (manual-run triggers only). Optional `formMode: "inline"` renders the trigger parameter form directly in the widget body (prompt-submission style) for manual-run start triggers that have parameters. Inline entries can suppress the redundant node/field labels and customize submit copy. | `title?`, `nodes[]` with `node`, `label?`, `description?`, `showRun?`, `triggerName?`, `promptConfirmation?`, `formMode?`, `showNodeLabel?`, `showFieldLabels?`, `submitLabel?` |
| `node` *(legacy)* | Same renderer as `nodes` — the merged card folds legacy single-node content into a one-entry list. Kept for import compatibility; migrates to `nodes` on first save. | `node`, `showRun?`, `triggerName?` |
| `table` | `WidgetTable` | `dataSource`, `render.kind: "table"` |
| `board` | `WidgetBoard` — kanban lanes grouped by a scalar `groupBy` field; same data sources / filters / row actions as the table panel | `dataSource`, `render.kind: "board"` with `groupBy`, `lanes[]`, `card`, optional `otherLane`, `where`, `sort`, `rowActions` |
| `chart` | `WidgetChart` (SVG) | `dataSource`, `render.kind: "chart"` |
| `number` | `WidgetNumber` | `dataSource`, `render.kind: "number"` |
| `scorecard` | `WidgetScorecard` — single KPI only (no multi-KPI or composite memory); adds change vs the immediately previous value in the series, direction-aware target/progress, and a status-colored sparkline via the shared `Sparkline` | `dataSource`, `render.kind: "scorecard"` with `aggregation`, optional `field`, `better`, `target`, `showProgress`, `sparklineField`, `showChange`, `changeCaption` |

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

Non-empty `field`; optional `label`, `format` (`text`, `number`, `status`, `relative`, `link`, `trend`, …), `show`, `href`. `format: trend` also accepts `trendBetter` (`up`/`down`, default `up`) and `trendDisplay` (`percent`/`value`/`none`, default `percent`); the cell compares against the row directly below in the filtered/sorted table (or the first already-loaded row still hidden by the progressive display window).

### Filters

`render.where[]` — AND list; ops: `eq`, `neq`, `contains`, `not_contains`, `gt`, `lt`, `exists`, `not_exists`.

### Row actions (trigger)

Required: `kind: trigger`, `node` (id or name). Optional: `hook` (default `run`), `template`, `payload`, `confirm`, `show`, `variant`, `icon`.

Runtime flow: `WidgetTable` / `WidgetBoard` → `WidgetRowActionButton` → `mergeTriggerPayload` → `onTriggerNode` → `useConsoleRunTrigger` → `InvokeNodeTriggerHook` → invalidate events/runs/memory queries.

Legacy fields normalized in FE: `target` → `node`, `triggerName` → `template`.

Manual-run gate: only the built-in `start` and `schedule` triggers expose a user-invokable `run` hook. The UI filters on `node.component` against the hardcoded allowlist in `web_src/src/pages/app/console/manualRunTriggers.ts` — `TablePanelForm` and `BoardPanelForm` hide non-manual triggers from the dropdown, `WidgetTable` / `WidgetBoard` hide their row actions, and `NodesPanelCard`/`NodesPanelForm` hide the Run affordance. Backend authorization stays in `InvokeNodeTriggerHook`; adding a new manual-run trigger requires a matching entry in the frontend allowlist.

### Expressions

- **`{{ CEL }}`** — `@marcbachmann/cel-js` via `widget/celExpr.ts`; row env + `now` (Unix seconds). The adapter upfront-coerces safe-integer JS numbers (and, on retry, numeric strings) to `BigInt` for int arithmetic, and normalizes safe-integer BigInt results back to plain `number` on the way out.
- **Legacy** `show` — e.g. `status == "running"` (`showExpression.ts`, `rowVisibility.ts`).
- Prefer structured `where` for simple validated filters.

**Lint:** loose equality in legacy expressions is intentional (scalar normalization). Do not add `eslint-disable` for `==` in dashboard code; refactor instead.

Editor memory hints: `MemoryDiscoveryPanel.tsx`, `useMemoryCatalog.ts` (suggestions only; YAML still validated).

### Markdown variables

- `content.variables[]` carries named live data refs; body uses `{{ name.field }}` (or `{{ name.$["Node"].data.x }}` for runs).
- Sources: `{ kind: "memory", namespace, orderBy?, direction?, matches?, mode?, limit? }` (default `mode: single` first-row wins, `orderBy: createdAt desc`) or `{ kind: "run", select: latest | latest_passed | latest_failed }`.
- `mode: list` resolves the memory variable to the full sorted array of matching rows (optionally capped by `limit`), unlocking CEL list macros (`rows.map(r, ...).filter(...)`) inside `{{ }}`; pair with the `join(list, sep)` builtin in `celExpr.ts` to flatten into Markdown / HTML.
- Resolution lives in `useMarkdownVariables.ts` (`pickMemoryRows` is the exported helper that branches on mode); interpolation in `markdownInterpolation.ts` (reuses `celExpr.compileTemplate`/`evalTemplate`). Validation: `markdownVariables.ts` (FE, including `validateMarkdownContent`) + `validateMarkdownContent` / `validateHTMLContent` in `pkg/models/console_yml.go` (BE).
- Run vars expose `status`, `nodeName`, `payload`, `durationMs`, and a `$` map of node executions (same shape as the table widget).

### HTML widget safety

- Render pipeline (`HtmlBody.tsx`): interpolate variables → DOMPurify allow-list → scope `<style>` blocks → `dangerouslySetInnerHTML` into `div[data-console-html-root="<id>"]`.
- Sanitizer (`htmlSanitize.ts`) blocks `<script>` and all `on*` handlers, removes head-like and resource-fetching elements (`link`, `meta`, `base`, `iframe`, `object`, `embed`, `audio`, `video`, `form`, `svg`, `math`, …), allows `<img src>`/`<img srcset>` for `http(s)`/relative URLs (cross-origin image fetches are permitted by policy), strips `poster`/`background`/`data`/`xlink:href`, restricts `href`/`src`/`srcset` to `http(s)`/`mailto:`/`tel:`/fragments, and rewrites every `<style>` rule to scope selectors under the widget root while dropping `@import`, `url(...)`, and unknown at-rules.
- Tailwind v4 classes must be in the curated `@source inline(...)` safelist in `web_src/src/App.css` to apply at runtime — extend it conservatively, never bypass it.

---

## Chart and number

**Chart** `render.type`: `bar`, `stacked-bar`, `line`, `area`, `donut`. `xField` + `series[]`; omit `series[].field` to count rows per bucket.

**Number** aggregations: `count`, `sum`, `avg`, `min`, `max`, `first`, `last` — non-`count` requires `field`.

**Scorecard** shares the number aggregation vocabulary but is single-KPI only (no multi-KPI / composite memory). Comparison model:

- **Change** = current value vs the immediately previous value in the series. The series is derived from `sparklineField` when set, or the primary `field` as a fallback. Only `first` / `last` aggregations expose a natural "previous" (adjacent anchor via `pickChangeAnchors`); combining aggregations (`sum` / `avg` / `min` / `max` / `count`) hide the chip. Reuses `computeTrend` (`widgetTrend.ts`) for percent/absolute math.
- **Target** = literal number or `{{ CEL }}` (evaluated against the newest filtered row + `now`), used for optional `showProgress` and fallback status color.
- `better: "up" | "down"` controls the polarity for the value change, the sparkline, and the vs-target status.
- The form relabels the two directional aggregations as `Latest` / `Earliest` because all data sources are newest-first (`first` → Latest, `last` → Earliest). Persisted YAML still uses `first` / `last`.

Helpers live in `widget/scorecardMath.ts` (`extractScorecardSeries`, `pickChangeAnchors`, `resolveScorecardTarget`, `computeScorecardProgress`, `computeScorecardChange`, `resolveScorecardStatus`, `formatScorecardChangeLabel`). Rendering is in `widget/WidgetScorecard.tsx`; the sparkline itself comes from the shared `widget/Sparkline.tsx` (shared with `WidgetNumber`) with a `className` prop for status coloring.

---

## YAML

```yaml
apiVersion: v1
kind: Console
metadata:
  canvasId: <uuid>   # export only; ignored on import
  name: <display>
spec:
  panels: [{ id, type, content }]
  layout: [{ i, x, y, w, h, minW?, minH? }]
```

- FE: `consoleYaml.ts` — parse/serialize + `validatePanelContent`
- BE: `ConsoleFromYML` / `VersionToConsoleYML` in `pkg/yaml/console.go`
- Unknown fields rejected; missing `panels`/`layout` → empty lists

---

## Agent workflows

### Fix a console bug

1. Reproduce in console mode (not template); note panel `type` and `dataSource.kind`.
2. Trace: panel card → `useWidgetData` → widget renderer → (if trigger) `useConsoleRunTrigger`.
3. Check permissions in `pkg/authorization/interceptor.go` if RPC-related.
4. Add/update test under `web_src/src/pages/app/console/**/*.spec.ts`.

### Add or change panel `content` fields

1. `widget/types.ts` (if widget-facing)
2. `panelTypes.ts` — interface, `templateForPanelType`, `validatePanelContent`, normalization (satellite modules like `boardPanelContent.ts` / `nodesPanelContent.ts` follow the same pattern)
3. `pkg/yaml/console.go` — mirror validation + tests
4. Panel card + form component
5. YAML tests: `consoleYaml.spec.ts` / `consoleYaml.validation.spec.ts`, `pkg/yaml/console_test.go`

### Add a new panel type

1. `PANEL_TYPES`, `PANEL_TYPE_META`, validator, template in `panelTypes.ts`
2. `ConsolePanelType*` constant + `AllowedConsolePanelTypes` in `pkg/yaml/console.go` and a per-type validator
3. `*PanelCard.tsx` + case in `ConsolePanelCards.tsx`
4. Icon in `ConsoleView.tsx` `PANEL_TYPE_ICONS`
5. Update [docs/prd/console-and-widgets.md](../../../docs/prd/console-and-widgets.md)

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
# Frontend unit tests (console package)
cd web_src && npm run test:run -- src/pages/app/console

# After UI edits (Docker dev env)
make format.js
make check.lint.ui
make check.build.ui

# After Go validation/API edits
make format.go
make lint
make check.build.app
go test ./pkg/yaml -count=1
go test ./pkg/grpc/actions/canvases -count=1
```

---

## Repo conventions

- No `web_src/src/utils/*` — use `lib/` or `hooks/`.
- Console has **strict ESLint budget** — refactor touched code; do not raise the budget.
- Split large components for Fast Refresh where the codebase already does.
- **Never** hand-write DB migrations; `make db.migration.create NAME=<dash-name>` if persistence changes.
- AGENTS.md: protobuf enum mapping, authorization on new RPCs.

---

## Quick file index

| Task | Start here |
| --- | --- |
| Grid / add panel | `ConsoleView.tsx` |
| Table CEL / filters / actions | `WidgetTable.tsx`, `WidgetRowActionButton.tsx`, `celExpr.ts`, `evalTableWhere.ts`, `mergeTriggerPayload.ts` |
| Table editor | `TablePanelForm.tsx`, `TablePanelFormRows.tsx` |
| Board renderer / editor | `WidgetBoard.tsx`, `BoardPanelCard.tsx`, `BoardPanelForm.tsx`, `boardPanelContent.ts` |
| Trigger from console | `useConsoleRunTrigger.ts`, `consoleTriggerParameters.ts` |
| Node status chip / Run button | `NodesPanelCard.tsx`, `NodesPanelInlineRunForm.tsx`, `useConsoleRunTrigger.ts`, `useConsoleTriggerLock.ts`, `deriveNodeStatuses.ts` |
| API hooks | `web_src/src/hooks/useCanvasData.ts` — `useCanvasVersion`, `useUpdateCanvasVersion` |
