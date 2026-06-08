# Console and Widgets

> Naming note: the product surfaces this feature as **Console**. In the
> codebase and persisted data it is still called "Dashboard" (file names,
> Go types, DB table `canvas_dashboards`, YAML helper modules, etc.). The
> YAML `kind` is `Console`; legacy `kind: Dashboard` files exported before
> the rename are no longer accepted on import.

## Overview

Canvas consoles are per-canvas views for turning workflow state into an operational surface: notes, pinned nodes, tables, charts, and numbers. They are useful for release consoles, preview environments, incident views, and any workflow where the canvas itself is not the best day-to-day status display.

This guide is written for both humans configuring consoles and agents changing the implementation. It explains what users see, how console data is stored, how widgets fetch and render rows, and which files must stay in sync.

## What Users See

- A console belongs to one canvas. Templates do not have editable consoles.
- Users enter console mode from the workflow v2 header (URL `?view=console`).
- The canvas graph is hidden and replaced by a 12-column draggable grid.
- Users with `canvases:update` can add, move, resize, edit, delete, import, and export panels.
- Users without edit permission can still view consoles and export YAML.
- Runtime actions, such as node-panel runs and table row actions, require `canvases:update`; disabled controls explain why they cannot run.
- Importing console YAML is replace-all: all panels and layout items are replaced in one update.

## Implementation Map

> Internal code still uses the legacy "dashboard" naming. The folder name,
> file names, and Go symbols below are the actual identifiers in the
> codebase; only user-facing text was rebranded to Console.

Primary frontend paths:

```text
web_src/src/pages/workflowv2/
  index.tsx
  dashboard/
    WorkflowDashboardOverlay.tsx
    DashboardOverlay.tsx
    DashboardView.tsx
    DashboardContext.tsx
    DashboardContextProvider.tsx
    useDashboardPanelState.ts
    panelTypes.ts
    dashboardYaml.ts
    DashboardYamlModal.tsx
    useDashboardModeActions.ts
    useDashboardTriggerNode.ts
    dashboardHeaderActions.ts
    dashboardTriggerParameters.ts
    deriveNodeStatuses.ts
    TypedPanelShell.tsx
    PanelEditorDialog.tsx
    MarkdownPanelCard.tsx
    NodePanelCard.tsx
    TablePanelCard.tsx
    ChartPanelCard.tsx
    NumberPanelCard.tsx
    TablePanelForm.tsx
    TablePanelFormRows.tsx
    DataSourceForm.tsx
    MemoryDiscoveryPanel.tsx
    widget/
      types.ts
      useWidgetData.ts
      WidgetTable.tsx
      WidgetChart.tsx
      WidgetNumber.tsx
      celExpr.ts
      evalTableWhere.ts
      mergeTriggerPayload.ts
      rowVisibility.ts
      showExpression.ts
```

Primary backend paths:

```text
pkg/models/
  canvas_dashboard.go
  canvas_dashboard_yml.go
  canvas_dashboard_yml_test.go

pkg/grpc/actions/canvases/
  get_canvas_dashboard.go
  update_canvas_dashboard.go
  dashboard_serialization.go
  canvas_dashboard_test.go

protos/canvases.proto
```


## Architecture

| Layer | Responsibility |
| --- | --- |
| Workflow host | `index.tsx` owns workflow mode and header wiring. It delegates console-specific work to small dashboard modules. |
| Overlay | `WorkflowDashboardOverlay` decides whether to render. `DashboardOverlay` maps API data, wires query/mutation state, and provides context. |
| Context | `DashboardContext` exposes canvas nodes, node statuses, run permission, and trigger callbacks. `resolveDashboardNode` accepts node id or node name. |
| Grid | `DashboardView` renders the 12-column `react-grid-layout` surface and routes each panel to its card. |
| Panel cards | One card per panel type. Cards use `TypedPanelShell`, optional runtime body, and `PanelEditorDialog`. |
| Panel schema | `panelTypes.ts` defines content types, default templates, normalization, and fast frontend validation. |
| YAML | `dashboardYaml.ts` and `pkg/models/canvas_dashboard_yml.go` parse, export, and validate the canonical YAML shape. |
| Widgets | `widget/useWidgetData.ts` fetches rows. `WidgetTable`, `WidgetChart`, and `WidgetNumber` render rows. |
| Persistence | `canvas_dashboards` stores JSON panels and layout for one canvas. |

When changing panel shapes, keep frontend validation, backend validation, YAML tests, and widget types aligned. Do not let the frontend accept a shape the backend later rejects.

## Stored Data Model

Console data is stored as arbitrary JSON content plus a layout:

- `DashboardPanel`: `id`, `type`, `content`
- `DashboardLayoutItem`: `i`, `x`, `y`, `w`, `h`, optional `minW`, `minH`
- `CanvasDashboard`: `canvasId`, `panels[]`, `layout[]`, `updatedAt`

The panel `content` object is intentionally flexible, but every known panel type has a typed shape in `panelTypes.ts` and matching backend validation in `canvas_dashboard_yml.go`.

### Generated type contracts

The console API contract is documented in three regenerated artifacts that must stay in lockstep:

1. **Proto enum for `Console.Panel.type`** — `protos/canvases.proto` defines `Console.Panel.Type` (`MARKDOWN`, `NODE`, `NODES`, `TABLE`, `CHART`, `NUMBER`). The wire boundary is fail-closed: `panelTypeToModel` in `pkg/grpc/actions/canvases/console_serialization.go` rejects `TYPE_UNSPECIFIED` and unknown values with `InvalidArgument`. The Go model and the persisted JSONB still use the lowercase string form, so the YAML import path and the storage layer are unchanged.
2. **JSON Schema for `Console.Panel.content`** — `web_src/src/pages/app/console/schema/panelContent.ts` declares a TypeScript discriminated union over the existing per-type content interfaces. `npm run generate:console-schema` emits `api/schemas/console-panel-content.schema.json`; `scripts/inject-console-panel-content-schema.mjs` then attaches it to the `ConsolePanel` definition in `api/swagger/superplane.swagger.json` as a stringified `x-content-schema` vendor extension (the format must be a vendor extension because Swagger 2.0 lacks `anyOf`/`oneOf` support).
3. **Adapters at the SDK boundary** — the TS SDK emits the enum as SCREAMING_CASE (`"MARKDOWN"`) while the FE schema is lowercase. `apiPanelTypeToPanelType` / `panelTypeToApi` in `web_src/src/pages/app/console/panelTypes.ts` are the only allowed translation points; the Go SDK gets equivalent adapters in `pkg/cli/commands/apps/console/convert.go`.

`make pb.gen` runs all of the above end-to-end. Adding a new panel kind requires extending **every** mirror in the same change: the proto enum, the Go constants, the wire-boundary switch, the FE union (`PANEL_TYPES`), the API↔FE adapters, the CLI tables, and the TS schema source so the generated JSON Schema documents the new content shape.

## Panel Types

| Type | Purpose | Main content fields |
| --- | --- | --- |
| `markdown` | Notes, runbooks, links, status explanations | `title?`, `body?` |
| `node` | Pin one canvas node with latest status and optional Run button | `title?`, `node`, `showRun?`, `triggerName?` |
| `nodes` | Pin multiple canvas nodes in one card with live status and optional purpose lines | `title?`, `nodes[]` |
| `table` | Render rows from memory, executions, or runs | `title?`, `dataSource`, `render.kind: "table"` |
| `chart` | Render grouped data as bar, stacked bar, line, area, or donut | `title?`, `dataSource`, `render.kind: "chart"` |
| `number` | Render one aggregate KPI, or several KPIs side-by-side via `metrics[]` | `title?`, (`dataSource` + `render.kind: "number"`) or `metrics[]` |

New panels start as valid drafts where possible. For example, a new table panel may have an empty memory namespace and no columns while the user is still configuring it.

## Data Sources

Table, chart, and number panels share these data sources:

```ts
{ kind: "memory", namespace: string, fieldPath?: string }
{ kind: "executions", node?: string, limit?: number }
{ kind: "runs", limit?: number }
```

| Source | Query | Result rows |
| --- | --- | --- |
| `memory` | `useCanvasMemoryEntries` | Canvas memory entries filtered by namespace. Optional `fieldPath` flattens a nested value or list. |
| `executions` | `useInfiniteCanvasEvents` | Flattened `event.executions[]`, optionally filtered to one resolved node. Rows include `status`, `nodeName`, `durationMs`, and `payload` (the data the node received from its root event). |
| `runs` | `useInfiniteCanvasRuns` (+ `useEventExecutionsBatch` for `$`) | Raw run objects plus derived `status`, `nodeName` (the node that initiated the run, resolved from `rootEvent.nodeId`), `payload` (alias for `rootEvent.data`, the initial payload that triggered the run), `durationMs` (created-to-finished elapsed time in milliseconds — pair with `format: duration`), and `$` (a map keyed by node display name with each node's full execution including `outputs` — see [Addressing per-node outputs](#addressing-per-node-outputs)). Number widgets can use API `totalCount` for count KPIs. |

Execution widgets eager-load more event pages until they have enough execution rows for the configured `limit`, or until a bounded page cap is reached. This avoids count widgets flashing an intermediate value when the first event page has few executions.

## Table Panels

Table panels are the most configurable widget type. They can display canvas memory, executions, or runs. Memory-backed tables are the recommended pattern for ephemeral environment consoles.

### Memory Table Example

```yaml
type: table
content:
  dataSource:
    kind: memory
    namespace: environments
  render:
    kind: table
    columns:
      - field: pr_number
        label: PR
      - field: status
        label: Health
        format: status
      - field: created_at
        label: Uptime
        format: relative
    where:
      - field: status
        op: neq
        value: destroyed
    rowActions:
      - kind: trigger
        label: Destroy
        node: start
        template: destroy
        variant: danger
        confirm: "Destroy PR #{{ pr_number }}?"
        show: 'status == "running"'
        payload:
          issue.number: "{{ pr_number }}"
```

The editor scans live canvas memory and suggests namespaces and field names. Suggestions are only editor assistance; YAML import still uses the validators.

### Field Suggestions

The table editor populates the column dropdown, the filter / sort field datalists, and the row-action payload quick-insert chips with a field catalog derived from the data source:

| Data source | Field catalog |
| --- | --- |
| `memory` | Discovered live from canvas memory entries in the chosen namespace. |
| `executions` | Static catalog mirroring the execution row shape (`status`, `nodeName`, `durationMs`, `payload`, `state`, `result`, `resultReason`, `resultMessage`, `id`, `nodeId`, `canvasId`, `parentExecutionId`, `previousExecutionId`, `createdAt`, `updatedAt`). |
| `runs` | Static catalog mirroring `CanvasesCanvasRun` plus the derived fields appended by `collectRunRows` (`state`, `result`, `status`, `nodeName`, `payload`, `durationMs`, `$`, `id`, `canvasId`, `versionId`, `createdAt`, `updatedAt`, `finishedAt`, `rootEvent.nodeId`, `rootEvent.customName`). |

When suggestions are available, the column header bar also exposes an **Add all fields** button and quick-add chips that insert a column for each catalog field with `suggestColumnFormat`-derived formatting (e.g. `status` → `status`, `createdAt` → `relative`).

The column, filter, row-style, and sort field inputs are free-text — authors can type any **nested dot path** (e.g. `payload.user_id`, `rootEvent.customName`) or `{{ CEL }}` template. Catalog entries surface as `<datalist>` autocomplete suggestions. When the typed value matches a known catalog field, the column's `label` and `format` are auto-filled (only when the author hasn't already set them) so picking from the dropdown stays one-click.

#### Addressing per-node outputs

Run rows expose a `$` map keyed by **node display name** so authors can address the outputs of any node within that run. The shape mirrors the canvas-side expression syntax (`$['Node Name'].data`):

| Path | Resolves to |
| --- | --- |
| `$["deploy-prod"].outputs` | The raw outputs map (`{ <channel>: [<event>, ...] }`) for the `deploy-prod` execution. |
| `$["deploy-prod"].outputs.default[0]` | The first event emitted on the `default` channel. |
| `$["deploy-prod"].data` | Convenience shortcut for the **last** event of the `default` channel (or the first available channel) with one `data` envelope unwrapped — matches what canvas-side `$['Node'].data` resolves to. |
| `$["deploy-prod"].state`, `$["deploy-prod"].result` | Per-node state and result fields, useful for row styling. |

The same syntax works in both literal field paths (e.g. a column `field: $["deploy-prod"].data.url`) and in `{{ }}` CEL templates (e.g. `format: link, label: "{{ $['deploy-prod'].data.url }}"`). Under the hood the CEL compiler rewrites `$` to a safe identifier (cel-js doesn't accept `$` as an identifier), but authors don't need to be aware of the rewrite — string literals are preserved verbatim, so a `$` inside `"..."` or `'...'` is left alone.

**Missing nodes resolve to `undefined`** — when a given node didn't run for that run (e.g. the workflow forked and only one branch executed, or the execution hasn't reached that node yet), `$["other-node"].outputs` returns `undefined` and the widget cell renders as `-`. This is intentional and matches how missing dot paths behave elsewhere.

**Performance:** Run-level outputs are not part of the `ListRuns` payload (executions are returned as lightweight refs). The widget side-loads them via `ListEventExecutions(rootEventId)` for each visible run, capped by the panel's `limit`. React Query caches the response keyed by `(canvasId, rootEventId)`, so opening the run detail modal and the widget share results. Authors with very large `limit` values should expect one extra request per run on first load.

### Columns

Each column needs a non-empty `field`. Optional fields:

| Field | Meaning |
| --- | --- |
| `label` | Header text. Falls back to `field`. |
| `format` | Display format: `text`, `number`, `percent`, `date`, `datetime`, `relative`, `duration`, `status`, `badge`, `code`, or `link`. `duration` always interprets its input as **milliseconds** — convert from seconds via CEL (`{{ seconds * 1000 }}`) before passing in. `badge` is an alias for `status` and renders the value as a colored pill (green for `passed`/`ready`/`active`, red for `failed`, amber for `pending`, sky for `running`). |
| `show` | Row expression controlling whether the cell is visible. |
| `href` | Link template for `link` columns. |

Column fields can be direct paths such as `status` or expression templates where supported by the widget helpers.

### Structured Filters

`render.where` is an AND list. Each filter has `field`, `op`, and sometimes `value`.

| Operator | Meaning |
| --- | --- |
| `eq` / `neq` | String equality / inequality |
| `contains` / `not_contains` | Substring match |
| `gt` / `lt` | Numeric comparison |
| `exists` / `not_exists` | Non-empty / empty field |

Filters are validated in both `panelTypes.ts` and `canvas_dashboard_yml.go`.

### Row Actions

Row actions invoke a trigger node on the canvas through `InvokeNodeTriggerHook`. They do not call HTTP Request nodes directly. Downstream steps run because the trigger fires, just like a normal manual run.

Supported action fields:

| Field | Meaning |
| --- | --- |
| `kind` | Must be `trigger`. |
| `node` | Trigger node id or name. Required. Legacy `target` can still be normalized in the frontend. |
| `hook` | Trigger hook name. Defaults to `run`. |
| `template` | Start template name when the trigger supports templates. Legacy `triggerName` can still be normalized. |
| `label` | Button label. Defaults to `Run`. |
| `payload` | Map of dot paths to literals or `{{ CEL }}` templates, merged into trigger parameters. |
| `confirm` | Optional confirmation dialog text. Supports interpolation. |
| `show` | Optional expression controlling button visibility for each row. |
| `variant` | `default`, `primary`, or `danger`. |
| `icon` | `play`, `stop`, `trash`, `refresh`, or `external-link`. |

Runtime flow:

1. `WidgetTable` filters visible actions for the row.
2. Optional `confirm` text is interpolated.
3. `mergeTriggerParameters` builds hook parameters from the trigger template, row data, and action payload.
4. `DashboardContext.onTriggerNode` calls `useDashboardTriggerNode`.
5. `InvokeNodeTriggerHook` fires and console-related queries are invalidated.

## Expressions And Templates

Console widgets support two related expression styles:

- `{{ CEL }}` templates use CEL through `cel-js`. The row environment contains row fields and `now` as Unix seconds.
- Legacy expressions such as `status == "running"` are supported in `show` and some filters.

Use CEL templates for new payloads, confirmation text, and rich interpolation. Use structured `where` filters when the condition is simple and should be validated.

### CEL builtins

In addition to the standard CEL functions cel-js ships with, the dashboard exposes these helpers in every expression's environment:

| Builtin | Description |
| --- | --- |
| `int(v)`, `float(v)`, `string(v)` | Type coercions, with sensible fallbacks for nullish input. |
| `contains(s, sub)`, `startsWith(s, p)`, `endsWith(s, p)`, `matches(s, regex)` | String / regex predicates. |
| `lower(s)`, `upper(s)` | Case conversions. |
| `duration(seconds)` | Format a number of seconds as `5m 30s` / `1h 5m`. |
| `timestamp(seconds)` | Format epoch seconds as an ISO-8601 string. |
| `formatDate(value, pattern)` | Render any date-like value (ISO string, Date, epoch number) with tokens `yyyy yy MM M dd d HH H mm m ss s`. Renders in the viewer's local time. |
| `epochMs(value)` | Convert any date-like value (ISO-8601 string, Date instance, epoch seconds, epoch ms) to **milliseconds since epoch**. Returns `0` for unparseable input so arithmetic stays defined. Pairs with `duration()` for human-friendly elapsed-time output. |

Note that `field: finishedAt - createdAt` does **not** work directly because both values are ISO-8601 strings and CEL doesn't subtract strings. Use one of:

- `field: durationMs, format: duration` — easiest for the standard "how long did this take" cell on `runs` or `executions` rows (the derived field is already there).
- `field: '{{ duration((epochMs(finishedAt) - epochMs(createdAt)) / 1000) }}'` — the explicit CEL form, useful when comparing arbitrary timestamp fields or subtracting the trigger time from the finish time.
- `field: '{{ epochMs(finishedAt) - epochMs(createdAt) }}', format: duration` — gives you a numeric ms value the column formatter can render and downstream filters can compare against.

Loose equality in legacy expressions is implemented explicitly by normalizing scalar strings, numbers, booleans, and `null`; do not add `eslint-disable-next-line` directives for equality checks.

## Chart Panels

Chart render shape:

```yaml
render:
  kind: chart
  type: bar
  xField: service
  series:
    - field: cost
      label: Cost
      format: number
      prefix: "$"
  legend: auto
```

Supported `type` values:

- `bar` — one bar per `xField` bucket, with each configured series rendered side-by-side.
- `stacked-bar` — multiple series stacked on top of each other per `xField` bucket. Visually identical to `bar` with a single series; the editor surfaces a hint until you add a second series (or set `seriesField`).
- `line`
- `area`
- `donut` — one slice per distinct `xField` value, valued by the first series.

Rows that share the same resolved `xField` value are merged into a single chart point. If a series omits `field`, the chart counts rows per `xField` bucket. If `field` is present, the chart sums numeric values from that field across each bucket. Non-numeric values are ignored.

### Pivoting long-format rows with `seriesField`

Some data sources emit one row per (X, series) combination — e.g. one row per `(date, service)` cost line. Configure `seriesField` to pivot those rows into one series per distinct value:

```yaml
render:
  kind: chart
  type: stacked-bar
  xField: date
  seriesField: service
  series:
    - field: cost_usd
      label: Cost
      prefix: "$"
```

When `seriesField` is set, the chart uses the numeric `field` of the **first** configured series for values (summed per `(xField, seriesField)` bucket) and ignores additional series entries for shaping — colors and order come from the data, not the configured series list.

### Series formatting

Each series supports optional display fields used by the hover tooltip (and the donut value rows):

- `format` — one of `text`, `number`, `percent`, `duration`. Defaults to `number` when omitted. `duration` always interprets its input as **milliseconds** (so an average of `4527` renders as `4.5s`, not `1h 15m`); convert other units in CEL before aggregating.
- `prefix` — literal string rendered before the formatted value (for example `"$"`).
- `suffix` — literal string rendered after the formatted value (for example `" MWh"`).

Tooltips show the category in the header and one row per series with `label — prefix{value}suffix`. Donut tooltips append the slice's share of the total (for example `ec2 — $1,200 (52%)`).

### Legend

`render.legend` controls legend visibility:

- `auto` (default) — visible for donut charts or when 2+ series are configured; hidden otherwise.
- `show` — always visible (useful when you want consistent labels even for a single-series chart).
- `hide` — never rendered.

`WidgetChart` is implemented with Recharts via the shared shadcn `ChartContainer`. Category labels live in the tooltip rather than on the chart surface so densely-packed x-axes don't overlap.

## Number Panels

Number panels aggregate rows into a single KPI. Supported aggregations are:

- `count`
- `sum`
- `avg`
- `min`
- `max`
- `first`
- `last`

Aggregations other than `count` require a non-empty `field`.

### Display Symbols

`render.prefix` and `render.suffix` wrap the formatted value with a literal string. Use them for currency or unit hints — e.g. `prefix: "R$"`, `suffix: " MWh"`. They apply after `render.format`, so locale-aware formatting (`number`, `percent`, `duration`) is preserved. When the aggregate is null/empty the widget still renders the em-dash placeholder without symbols.

```yaml
render:
  kind: number
  aggregation: sum
  field: cost
  format: number
  prefix: "R$"
```

### Composite Memory Sources

Number panels backed by memory accept a composite data source that aggregates each namespace with its own configuration and then merges the partials. This is useful when the contributing namespaces have different schemas (for example "sum of cost" in one namespace and "count of tests" in another).

```yaml
type: number
content:
  dataSource:
    kind: memory
    combine: sum
    sources:
      - namespace: expenses
        aggregation: sum
        field: cost
      - namespace: tests
        aggregation: count
  render:
    kind: number
    format: number
    prefix: "R$"
```

Rules:

- `sources` is a non-empty array. Each entry needs a non-empty `namespace`, an `aggregation` from the standard set, and a `field` when the aggregation is anything other than `count`. An optional `fieldPath` flattens the entry the same way the single-namespace memory source does.
- `combine` is one of `sum`, `min`, `max`, or `avg`. `sum` is the default the form seeds when switching to the composite mode.
- When `sources` is set, `render.aggregation` and `render.field` must be absent — the per-source configuration is the source of truth.
- Partial values that are `null` (for example a namespace with no numeric rows under `sum`) are skipped during combine; the panel only renders the em-dash placeholder when every partial is null.
- `avg` is an unweighted mean of the available partials, not a row-weighted average across namespaces. Pick `sum` when row-level math is required.
- Sparklines are only available for the single-source mode in this iteration.

The editor exposes a Single / Multiple toggle in the Number panel form. Switching to Multiple seeds one source from the current single-source configuration so existing panels do not lose context.

### Multi-Number Mode

A number panel can also render **multiple independently-configured numbers** in a single card. Each metric has its own data source, aggregation, field, label, format, prefix/suffix, and optional sparkline; the metrics lay out in a flex row that wraps to new lines when the panel is narrow.

```yaml
type: number
content:
  title: Pipeline KPIs
  metrics:
    - dataSource:
        kind: runs
      render:
        kind: number
        aggregation: count
        label: Total runs
    - dataSource:
        kind: memory
        namespace: costs
      render:
        kind: number
        aggregation: sum
        field: cost
        label: Total cost
        format: number
        prefix: "R$"
```

Rules:

- `metrics` is a non-empty array. When `metrics` is present, top-level `dataSource` and `render` are not used and are not required.
- Each metric's `dataSource` must be a single-source `memory` / `executions` / `runs` shape — the composite (`sources` + `combine`) shape is not allowed inside a metric.
- Each metric's `render.kind` is `number`. `render.aggregation` is required (one of `count`/`sum`/`avg`/`min`/`max`/`first`/`last`), and `render.field` is required for aggregations other than `count`.
- Use `render.label` as the metric's name (it renders above the value).
- Multi-number mode is disjoint from the composite-combine mode: a panel is either single-value, composite-combined, or multi-number. The Number panel form exposes a Single / Multiple memory sources / Multiple numbers toggle that seeds the new mode from whatever is currently configured.

## Markdown Panels

Markdown panels render `content.body` using `react-markdown` with `remark-gfm`, `remark-breaks`, `rehype-raw`, and `rehype-sanitize`.

Supported authoring features:

- **GitHub-flavored markdown**, including hand-written pipe tables:

  ```markdown
  | Service | Status |
  | --- | --- |
  | api | passed |
  | web | failed |
  ```

- **Collapsible sections** using raw `<details>` / `<summary>` HTML:

  ```markdown
  <details>
  <summary>Troubleshooting</summary>

  - Flush the cache.
  - Roll back via the **Rollback** node panel.

  </details>
  ```

  Use `<details open>` to pre-expand a section. The body of a `<details>` is still parsed as markdown, so links, lists, and other formatting work inside accordions.

- **Safe-by-default raw HTML.** `rehype-sanitize` strips `<script>`, inline event handlers (`onclick`, `onerror`, …), and any tag outside the allowlist. The only raw HTML tags explicitly added on top of the default allowlist are `<details>` and `<summary>` (plus the `open` attribute on `<details>`). If you need a new tag, extend `MARKDOWN_SANITIZE_SCHEMA` in `MarkdownPanelCard.tsx` rather than disabling sanitization.

### Markdown variables

Markdown panels can reference live data through named variables. Variables are declared on `content.variables` and consumed from the body (or title) with the same `{{ }}` CEL templates that table widgets use:

```yaml
- id: deploy-summary
  type: markdown
  content:
    title: "Latest deploy of {{ release.service }}"
    body: |
      ## {{ release.service }}

      - Status: **{{ lastRun.status }}**
      - Triggered by: {{ lastRun.nodeName }}
      - Output URL: {{ lastRun.$["Deploy"].data.url }}
    variables:
      - name: release
        source:
          kind: memory
          namespace: releases
          orderBy: createdAt
          direction: desc
          matches:
            - field: env
              value: production
      - name: lastRun
        source:
          kind: run
          select: latest_passed
```

Two source kinds are supported:

- **`memory`** — picks the first row from a memory namespace. Use `matches` (property-equality) to filter and `orderBy` + `direction` to choose which row counts as "first". `createdAt` is the default order; the namespace `id` is also queryable. The exposed object spreads the memory row's `values` together with `id`, `namespace`, `createdAt`, and `updatedAt`.
- **`run`** — picks the most recent run with `select: latest`, `latest_passed`, or `latest_failed`. The exposed object spreads the run row and adds:
  - `status` — normalized to `passed | failed | cancelled | running | unknown`.
  - `nodeName`, `payload`, `durationMs` — convenience fields mirroring what the table widget exposes.
  - `$` — a map of node-execution outputs keyed by node display name. Use it as `{{ run.$["Node Name"].data.field }}` for run-level output references.

Variables resolve to `null` when no row matches; CEL access on `null` renders as an empty string, so a partial template never throws. The in-card editor surfaces a per-variable preview with one-click "insert" buttons, plus a live rendered preview that mirrors what the saved panel will display.

## Node Panels

Node panels resolve the configured `node` by id or name. They display the node's name and can optionally show a manual Run button.

`showRun` only exposes the button. The actual click still requires `canRunNodes`, and the backend authorization remains the source of truth.

## Multi-Node Panels

The plural `nodes` panel type renders several pinned canvas nodes in a single card, each with an optional purpose line. Use it for "Key Nodes" style summaries (for example, the entry/exit nodes of a preview-environment workflow) instead of stamping out one `node` panel per row.

```yaml
type: nodes
content:
  title: Key Nodes
  nodes:
    - node: pr-opened
      description: GitHub PR trigger that boots a new environment.
    - node: create-droplet
      description: Provisions the DigitalOcean droplet.
    - node: health-check
      description: Confirms the preview is responsive.
    - node: delete-droplet
      label: Tear Down
      description: Releases the droplet when the PR closes.
      showRun: true
```

Per-entry fields:

| Field | Meaning |
| --- | --- |
| `node` | Required. Canvas node id or name. |
| `label` | Optional override for the displayed row name. Falls back to the resolved canvas node name. |
| `description` | Optional short purpose line shown under the row name. |
| `showRun` | When true, surface a manual "Run" button. Still gated by `canRunNodes`. |
| `triggerName` | Optional start template name when the trigger exposes multiple templates. |

`content.nodes` may be an empty array on a freshly added panel; the card renders a "configure me" hint until the author adds at least one entry through the form.

## YAML Import And Export

Canonical Console YAML:

```yaml
apiVersion: v1
kind: Console
metadata:
  canvasId: 00000000-0000-0000-0000-000000000000
  name: Example canvas
spec:
  panels:
    - id: envs
      type: table
      content:
        dataSource:
          kind: memory
          namespace: environments
        render:
          kind: table
          columns:
            - field: service
              label: Service
  layout:
    - i: envs
      x: 0
      y: 0
      w: 12
      h: 6
```

Rules:

- `apiVersion` must be `v1`.
- `kind` must be `Console`. Legacy `kind: Dashboard` files exported before the rename are not accepted; re-export from the current UI to upgrade them.
- Unknown top-level, metadata, panel, and layout fields are rejected.
- `metadata.canvasId` and `metadata.name` are informational on export.
- Missing `spec.panels` or `spec.layout` means an empty list.
- Maximum panel count is 50.
- Maximum panel payload size is 1 MiB.
- Import replaces the whole console.

Frontend YAML parsing lives in `dashboardYaml.ts`. Backend YAML parsing and validation lives in `canvas_dashboard_yml.go`. Keep error behavior and accepted shapes aligned.

## Bundling A Console With An Installable App

Apps installed from a public GitHub repository (`POST /apps/install`) can ship an optional `console.yaml` alongside `canvas.yaml`. When present, the install flow loads it from the same ref and writes it as the new canvas's console.

- File path: `console.yaml` at the repo root, same branch (`main` or `master`) that `canvas.yaml` is read from.
- Schema: same `apiVersion: v1`, `kind: Console` shape documented above; parsed and validated by `models.DashboardFromYML`. A malformed file aborts the install with HTTP 400 before any canvas is created.
- Optional: a missing file is fine — the new canvas just starts with an empty console.
- Replace-all on install: `metadata.canvasId` is ignored; the panels and layout become the canvas's full console.

Implementation lives in `pkg/installation/fetch.go` (`FetchConsole`) and `pkg/installation/install.go` (`persistInstalledConsole`).

## Authorization

| Operation | Expected permission / state |
| --- | --- |
| View console | Canvas read access. |
| Edit panels or layout | `canvases:update`, not a template, and canvas not deleted remotely. |
| Import YAML | Same as edit. |
| Export YAML | Available in console mode, including read-only viewers. |
| Run node panel or table row action | `canvases:update`, not a template, and canvas not deleted remotely. |

The UI disables unavailable controls, but server-side authorization is authoritative. Check `pkg/authorization/interceptor.go` when adding new RPCs.

## Extension Checklist For Agents

When adding a panel type:

1. Add the type to `PANEL_TYPES` and `PANEL_TYPE_META` in `panelTypes.ts`.
2. Add a content interface, default template, normalization if needed, and a `validatePanelContent` branch.
3. Add backend validation in `canvas_dashboard_yml.go`.
4. Add a panel card and route it from `DashboardView`.
5. Add YAML round-trip tests and backend validator tests.
6. Update this document.

When adding a data source:

1. Extend widget data-source types.
2. Add frontend and backend validation.
3. Add editor support in `DataSourceForm`.
4. Implement fetching in `useWidgetData`.
5. Add tests for normalization, validation, and rendering.

When adding row action kinds:

1. Extend `WidgetRowAction` and `normalizeRowAction`.
2. Validate the new action in `panelTypes.ts` and `canvas_dashboard_yml.go`.
3. Add runtime UI and permission behavior in `WidgetTable`.
4. Prefer routing runtime behavior through `DashboardContext` so panel code stays testable.

## Testing And Verification

Focused checks for console work:

```bash
cd web_src
npm run test:run -- src/pages/workflowv2/dashboard
```

Repository checks after frontend changes:

```bash
make format.js
make check.lint.ui
make check.build.ui
```

Backend checks after changing Go validation or API behavior:

```bash
make format.go
make lint
make check.build.app
go test ./pkg/models -run 'TestDashboard|TestValidateDashboardContent'
```

Add or update tests in these areas:

- `web_src/src/pages/workflowv2/dashboard/**/*.spec.ts`
- `pkg/models/canvas_dashboard_yml_test.go`
- `pkg/grpc/actions/canvases/canvas_dashboard_test.go`

## Maintenance Notes

- The console code has strict lint-budget pressure. Prefer small helpers and component-only files where Fast Refresh applies.
- Do not update the ESLint budget to hide console regressions. Refactor the touched code instead.
- Keep draft panel states valid when possible so users can add a panel before fully configuring it.
- Keep user-facing text using `SuperPlane` capitalization.
- Do not manually create migrations. If console persistence changes, use `make db.migration.create NAME=<name>` and leave rollback files empty.
