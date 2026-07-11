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
    HtmlPanelCard.tsx
    HtmlPanelEditor.tsx
    HtmlBody.tsx
    htmlSanitize.ts
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

## Panel Types

| Type | Purpose | Main content fields |
| --- | --- | --- |
| `markdown` | Notes, runbooks, links, status explanations | `title?`, `body?` |
| `html` | Custom HTML with inline styles, scoped `<style>` blocks, and Tailwind classes (scripts and external resources blocked) | `title?`, `body?`, `variables?` |
| `node` | Pin one canvas node with latest status and optional Run button | `title?`, `node`, `label?`, `showRun?`, `triggerName?`, `promptConfirmation?` |
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
| `format` | Display format: `text`, `number`, `percent`, `date`, `datetime`, `relative`, `duration`, `status`, `badge`, `code`, `link`, `avatar`, `progress`, or `trend`. `duration` always interprets its input as **milliseconds** — convert from seconds via CEL (`{{ seconds * 1000 }}`) before passing in. `date`, `datetime`, and `relative` render through the shared `Timestamp` component, so every timestamp cell has an identical hover card exposing Local, UTC, Relative, and raw ISO with a copy button (see [Timestamp hover](#timestamp-hover) below). `badge` is an alias for `status` and renders the value as a colored pill (green for `passed`/`ready`/`active`, red for `failed`, amber for `pending`, sky for `running`). `avatar` renders the resolved value as a circular avatar: direct image URLs are used as the image source, GitHub usernames (or author maps with a `username`) resolve to the GitHub avatar with the person's name in a tooltip, and values without a username fall back to an initials disc; blank values render as an em-dash. Use `avatarCommitterField` to supply a secondary person map for initials. `progress` renders a horizontal bar that fills to `field / progressTarget`; see the row below. `trend` compares the row's numeric value against **the row directly below it** (after filter + sort) and renders a diagonal arrow colored by whether the change is "better" or "worse"; see the trend section below. |
| `show` | Row expression controlling whether the cell is visible. |
| `href` | Link template for `link` columns. Accepts `{{ cel }}` expressions and templates (e.g. `{{ prUrl }}` or `https://github.com/{{ org }}/pull/{{ prNumber }}`), legacy single-brace `{field}` placeholders, and bare static URLs. The column editor shows a dedicated href input with a field picker (values inserted as `{{ field }}`) when the format is `link`. |
| `progressTarget` | Target (100%) for `format: progress`. Required for progress columns. Accepts a numeric literal (`"10"`, `"100.5"`), a row field path (`total`, `payload.goal`), or a full `{{ CEL }}` expression (`{{ base * 2 }}`). Resolved with the same helper as `field`. |
| `progressLabel` | Text rendered next to the progress bar: `none`, `number` (`5/10`), or `percent` (`50%`). Defaults to `percent`. The bar itself clamps at `[0, 100]%`, but the label and hover tooltip always show the real percentage, so overshoots (`120%`, `12/10`) stay visible. |
| `trendBetter` | For `format: trend` or numeric columns with `showTrend: true`. `up` (default) treats an increase as better (green), `down` treats a decrease as better. |
| `trendDisplay` | For `format: trend` or numeric columns with `showTrend: true`. `percent` (default) prints a signed percent change, `value` prints a signed absolute delta, `none` renders arrow only. |
| `showTrend` | When `true` on `number`, `percent`, or `duration`, renders the formatted value plus a trend chip in the same cell (same row-below semantics as `format: trend`). Ignored on other formats. |

Column fields can be direct paths such as `status` or expression templates where supported by the widget helpers.

#### Trend columns

`format: trend` turns a numeric column into a **trend-only** row-to-row comparison. The cell resolves the column's `field` (path or `{{ CEL }}`) on both the current row and the row directly below it, then renders one of the following:

| Case | Cell |
| --- | --- |
| Both values finite and different | Diagonal arrow (`↗` up / `↘` down) in green when the change is "better" and red when "worse", plus the signed magnitude according to `trendDisplay`. |
| Delta is exactly `0` | Gray `- 0` (no change). |
| Current or previous value isn't a finite number | Gray `-` (incomparable). |
| Percent mode with a previous value of `0` | Gray `-` (percent undefined). |
| Last visible row while the previous entry has not been fetched yet | Gray `...` (pending). Not used when the next row is already loaded but still hidden behind the progressive display window — that case compares against the hidden row. |
| Last row with no more data expected | Gray `- 0` (no baseline). |

The tooltip always shows both the percent change and the absolute delta when both are meaningful (e.g. `+12.5% · +4`).

Notes:

- **"Previous" is always the row directly below** in the current filter + sort order, so authors control what "previous" means by choosing the sort. For chronological trends, sort by `createdAt`.
- On progressive tables, when more rows are already loaded but still hidden behind the display window, the last visible trend cell compares against that first hidden row instead of showing pending `...`. Pending is reserved for baselines that have not been fetched yet.
- Percent math is `(current - previous) / |previous| * 100`, rounded to one decimal, capped at ±999% (values beyond the cap render as `>+999%` / `<-999%`).
- `trend` cells never carry a link/href — combine with a separate `link` column when both are needed.
- Prefer **`showTrend: true`** on `number` / `percent` / `duration` when you want the formatted value and the trend in one column (e.g. `4.5s  ↘ -10%`). Keep `format: trend` when you want a dedicated delta column.

Example — duration and pass rate with value + trend in one column:

```yaml
render:
  kind: table
  sort:
    field: createdAt
    order: desc
  columns:
    - field: name
      label: Node
    - field: durationMs
      label: Duration
      format: duration
      showTrend: true
      trendBetter: down
      trendDisplay: percent
    - field: passRate
      label: Pass rate
      format: percent
      showTrend: true
      trendBetter: up
```

Example — standalone trend column where a shorter run is a win:

```yaml
render:
  kind: table
  sort:
    field: createdAt
    order: desc
  columns:
    - field: name
      label: Node
    - field: durationMs
      label: Duration
      format: duration
    - field: durationMs
      label: Trend
      format: trend
      trendBetter: down
      trendDisplay: percent
```

### Timestamp hover

Every timestamp-formatted surface across the console renders through the shared `Timestamp` component (`web_src/src/components/Timestamp/`), the same component used by the runs sidebar. Hovering a timestamp reveals a card with four representations of the same instant:

- **Local** — locale-aware absolute time in the viewer's timezone, e.g. `10 Jul 2026, 15:46:53 CEST`
- **UTC** — the same instant rendered in UTC without a timezone suffix
- **Relative** — verbose live text, e.g. `5 minutes ago` / `in 3 hours`
- **Timestamp** — the raw full-precision ISO string with a copy button

The visible cell/panel label matches the column `format`:

- `format: date` — locale calendar day (no time-of-day)
- `format: datetime` — locale absolute timestamp with timezone
- `format: relative` — compact live-updating text without an "ago" suffix (e.g. `5m`, `2h`)

The presentational details block is exported as `TimestampDetails` (`@/components/Timestamp`) so tooltips that can't host a Radix hover card render an identical grid. Chart tooltips use it: whenever `xFormat` is `date`, `datetime`, or `relative`, hovering a chart point surfaces the same Local / UTC / Relative / ISO details block. Chart X-axis ticks stay short (`Jul 6` / `May 26 4:10 PM`) so densely-binned buckets keep readable.

Number panels follow the same rule: setting `render.format` to `date`, `datetime`, or `relative` wraps the aggregated value in `Timestamp`, so `max(updatedAt)`-style "last event" KPIs get the same hover details as tables.

Formatters and coercion live in `web_src/src/pages/app/console/widget/widgetFormat.ts` (`formatValue`, `coerceWidgetTimestamp`) and delegate to `@/lib/datetime` (`formatAbsolute`, `formatDate`, `formatRelative`, `formatISO`, `formatUTC`) plus `@/lib/date` (`formatTimeAgo`). Do not add ad-hoc `toLocale*` timestamp formatting in widget code — extend the shared helpers instead.


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

Only triggers that expose a user-invokable `run` hook (`core.HookTypeUser`) can be fired manually. In practice this is the built-in **`start`** and **`schedule`** triggers, so the console UI filters on `node.component` against a hardcoded allowlist (`web_src/src/pages/app/console/manualRunTriggers.ts`). `WidgetTable` **hides** row actions whose target node is an event-driven trigger (for example `github.onPullRequest`) — the backend would reject the invoke anyway, so a disabled button would just clutter the row. `TablePanelForm`'s trigger node dropdown filters to the same set so authors cannot configure a row action against an event trigger in the first place. Backend authorization in `InvokeNodeTriggerHook` remains the source of truth; adding a new manual-run trigger requires updating the frontend allowlist alongside the trigger implementation.

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
| `parseJson(s)` | Parse a JSON-encoded string into a structured value (list, map, scalar). Non-string inputs pass through unchanged; invalid JSON or `null` input returns `null`. Useful as a wholesale value (`{{ parseJson(blob) }}`), wrapped in another function (`size(parseJson(tags))`), or for equality checks (`parseJson(value) == null`). |
| `join(list, sep)` | Concatenate the elements of a list into a single string with an explicit separator. Non-array `list` returns `""`; non-string `sep` collapses to `""`; `null`/`undefined` elements render as `""`. Required to splice a mapped list into the output — a bare array value renders as JSON (`["a","b"]`) so it stays inspectable. Use `join(list, "")` for seamless fragment concatenation or any other separator for delimited output. Pairs with the `.map`/`.filter` macros on list-mode memory variables (`join(rows.map(r, "- " + r.name), "\n")`) — see [List mode](#list-mode). |
| `firstLine(s)` | Text before the first line break in `s`. Treats `\r\n` and bare `\r` the same as `\n`. Returns `""` for `null`/`undefined`. Use it to keep multi-line run outputs from blowing up table cells: `{{ firstLine(payload.message) }}`. |
| `substring(s, start, end?)` | Slice of `s` from `start` (inclusive) to `end` (exclusive). When `end` is omitted, returns everything from `start` onward. Negative `start` counts from the end (`-3` = last 3). Indices are clamped to the string length, and `end <= start` returns `""`. Non-string input is coerced via `String(value)`; `null`/`undefined` returns `""`. |
| `truncate(s, n, suffix?)` | First `n` characters of `s`, with `suffix` appended only when truncation actually happened. Inputs shorter than `n`, non-numeric `n`, and negative `n` return `s` unchanged. Use it for "show first 80 chars with `…`" cells: `{{ truncate(payload.message, 80, "…") }}`. |
| `splitIndex(s, sep, i)` | Nth segment of `split(s, sep)` returned as a scalar (cel-js can't index a function-call result inline). Negative `i` counts from the end (`-1` = last). Returns `""` for out-of-range / non-numeric `i`; an empty separator returns `s` unchanged. The separator is unescaped (`\n`, `\r`, `\t`, `\\`) because cel-js's lexer copies string literals verbatim, so `splitIndex(value, "\n", 0)` and `splitIndex(value, "|", 0)` both work as you'd expect. A `"\n"` separator also matches `\r\n` and bare `\r`, so it agrees with `firstLine` on Windows line endings. |
| `trim(s, chars?)` | Strip leading / trailing whitespace, or — when `chars` is supplied — strip leading / trailing characters that appear in `chars`. |
| `replace(s, old, new)` | Replace every occurrence of `old` in `s` with `new`. Empty `old` returns `s` unchanged. |
| `indexOf(s, sub)` | First index of `sub` in `s`, or `-1` when missing. Useful inside `where` filters / boolean expressions. |

`parseJson` has a real limitation: **cel-js does not support postfix `.field` / `[i]` / `.method(...)` after a function call result.** That means expressions like `parseJson(blob).items[0].id` or `parseJson(tags).map(t, t)` will fail to parse. To iterate or dot-access parsed data, either shape it as a real list/map upstream (canvas memory values are JSON-typed, so storing `{"tags": ["a","b"]}` lets you write `tags.map(t, t)` natively) or compose with macros that take the parsed value as an argument (`size`, `string`, etc.).

The string-trimming helpers (`firstLine`, `substring`, `truncate`, `splitIndex`) all return scalars **for the same reason** — authors can't write `split(s, "\n")[0]` or `s.substring(0, 80)` against cel-js. The single-call form (`firstLine(s)`, `substring(s, 0, 80)`) sidesteps the parser limitation. They are the recommended way to trim long values that come straight from the `runs` data source (raw run outputs, error messages, etc.); equivalent expressions like `value[:80]` work in expr-lang at node-config / write time but **not** in widget cells.

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

### Axis formatting

Cartesian charts (`bar`, `stacked-bar`, `line`, `area`) expose three optional axis fields:

- `xFormat` — display format applied to X-axis tick labels **and the hover tooltip header** (so the category in the tooltip matches the axis). Reuses the shared column-format vocabulary (`text`, `number`, `percent`, `date`, `datetime`, `relative`, `duration`). Useful when `xField` is a raw timestamp or numeric field — set `xFormat: date` instead of wrapping `xField` in a CEL expression like `{{ formatDate(createdAt, "MM/dd") }}`. Timestamp formats (`date`, `datetime`, `relative`) replace the tooltip header with the shared `TimestampDetails` block (Local / UTC / Relative / ISO with copy) so hovering a point exposes the exact instant — see [Timestamp hover](#timestamp-hover).
- `yLabel` — Y-axis title rendered alongside the ticks (for example `USD`, `Errors / day`). Omitted by default.
- `yFormat` — display format applied to Y-axis tick labels (`number`, `percent`, `duration`). Falls back to a locale-aware numeric default with thousands separators above 1k. The `format` declared on a series only affects tooltip values; configure `yFormat` to match it on the axis itself.

```yaml
render:
  kind: chart
  type: bar
  xField: createdAt
  xFormat: date
  yLabel: USD
  yFormat: number
  series:
    - field: cost
      label: Cost
      format: number
      prefix: "$"
```

Donut charts ignore `xFormat`, `yLabel`, and `yFormat` (no axes are rendered).

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

`render.format` accepts the same vocabulary as table columns. Numeric formats (`number`, `percent`, `duration`) render the raw aggregate; setting the format to `date`, `datetime`, or `relative` treats the aggregate as an epoch (ms or seconds) and wraps the visible label in the shared `Timestamp` component — this is the recommended way to build "last update" / "next run" KPIs. See [Timestamp hover](#timestamp-hover) for the shared hover behavior.

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

## Scorecard Panels

Scorecard panels are a single-KPI variant of the number panel with three extra affordances baked in:

- **Change chip** comparing the current aggregated value to the first finite point of the loaded series (`sparklineField`).
- **Target** (literal number or `{{ CEL }}`) that drives an optional progress bar and — when the change is incomputable — a fallback status color.
- **Status direction** (`better: "up" | "down"`) that colors the value change, the sparkline, and the vs-target polarity.

Multi-KPI and composite-memory shapes are intentionally not supported on scorecards. Use the standard `number` panel when you need those.

```yaml
type: scorecard
content:
  title: Open UX papercuts
  dataSource:
    kind: memory
    namespace: ux_papercuts
  render:
    kind: scorecard
    aggregation: last
    field: openCount
    format: number
    label: Open UX papercuts
    better: down
    target: "80"
    showProgress: true
    sparklineField: openCount
    showChange: both
    changeCaption: vs start of range
```

### Value pipeline

Value resolution is the single-source number pipeline: `dataSource` (`memory` | `executions` | `runs`) + `aggregation` + `field` (when needed) + `format` / `label` / `prefix` / `suffix`. Aggregations follow the standard number vocabulary (`count`, `sum`, `avg`, `min`, `max`, `first`, `last`). Aggregations other than `count` require a non-empty `field`.

To keep the change chip meaningful, most scorecards use `aggregation: last` on the same field as `sparklineField` so the primary value matches the end of the series.

### Change vs the start of the range

When `sparklineField` is set, the scorecard extracts the first finite value from the filtered rows and compares the current value against it using the same math as the table trend chip (`computeTrend`). The direction of "better" is controlled by `better`:

- `better: up` → an increase is good (green), a decrease is bad (red).
- `better: down` → a decrease is good (green), an increase is bad (red).

`render.showChange` controls the change chip's magnitude label:

- `percent` — `-22.8%`.
- `number` — `-29`.
- `both` (default) — `-29 (-22.8%)`.
- `none` — arrow only.

`render.changeCaption` (optional) prints a short caption next to the chip (e.g. `vs start of range`). When the series has fewer than two finite points, the chip is hidden entirely.

### Target and progress

`render.target` accepts a numeric literal or a full `{{ CEL }}` expression. The expression is evaluated once against the last filtered row plus the shared `now` global, so authors can bind to memory / execution fields (`{{ goal }}`) or compute a target (`{{ base * 1.1 }}`).

When `render.showProgress` is `true` and the target resolves to a finite positive number, the scorecard renders a thin direction-aware progress bar under the value:

- `better: up` — bar = `clamp(current / target, 0, 100%)`; the goal is met when `current >= target`.
- `better: down` — the goal is met when `current <= target` (bar fills to 100%); overshoot uses `target / current` so the bar shrinks as the value drifts further from the goal.

The percentage label under the bar always uses the raw ratio (so authors still see overshoot values above 100%).

### Status color priority

The colored value change, sparkline, and progress bar all read from the same status polarity, resolved in priority order:

1. If the change chip is computable, use its polarity (`better` → green, `worse` → red, `flat` → slate).
2. Otherwise, if the target resolves, use target-based status (`met` → green, otherwise red).
3. Otherwise, neutral slate.

This keeps the widget legible when only one signal is available (e.g. a `count` scorecard with no `sparklineField` still colors correctly via its target).

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

- **`memory`** — picks the first row from a memory namespace (default), or the full sorted array when `mode: list` is set. Use `matches` (property-equality) to filter and `orderBy` + `direction` to choose which row counts as "first". `createdAt` is the default order; the namespace `id` is also queryable. The exposed object spreads the memory row's `values` together with `id`, `namespace`, `createdAt`, and `updatedAt`.
- **`run`** — picks the most recent run with `select: latest`, `latest_passed`, or `latest_failed`. The exposed object spreads the run row and adds:
  - `status` — normalized to `passed | failed | cancelled | running | unknown`.
  - `nodeName`, `payload`, `durationMs` — convenience fields mirroring what the table widget exposes.
  - `$` — a map of node-execution outputs keyed by node display name. Use it as `{{ run.$["Node Name"].data.field }}` for run-level output references.

Variables resolve to `null` when no row matches (or `[]` in list mode); CEL access on `null` renders as an empty string, so a partial template never throws. The in-card editor surfaces a per-variable preview with one-click "insert" buttons, plus a live rendered preview that mirrors what the saved panel will display.

#### List mode

Add `mode: list` to a memory source to resolve the variable to every matching row instead of just the first. This unlocks CEL list macros (`map`, `filter`, `all`, `exists`, `size`) inside `{{ }}` so authors can render the rows as a Markdown / HTML list, count or filter them inline, etc. An optional `limit` (positive integer) caps the array; omit it to include every match.

```yaml
variables:
  - name: deploys
    source:
      kind: memory
      namespace: deployments
      orderBy: createdAt
      direction: desc
      mode: list
      limit: 20
```

cel-js doesn't allow `.method()` postfix after a function-call result, so chain macros directly off the bound variable (`deploys.filter(...).map(...)`).

When an interpolated expression resolves to an array, the renderer falls back to JSON (e.g. `["a","b"]`) so a stray `{{ deploys }}` or `{{ rows }}` reference stays inspectable instead of silently flattening. To splice a mapped list of HTML/Markdown fragments into the output, wrap it in the `join(list, sep)` builtin — use `""` for seamless concatenation:

```html
<div>{{ join(deploys.map(d, "<p>" + d.name + "</p>"), "") }}</div>
```

renders `<div><p>web</p><p>api</p>…</div>`. When you want a separator between elements (newlines for a Markdown bullet list, commas for an inline summary), pass it as the second argument instead:

```markdown
- Total deploys: {{ size(deploys) }}
- Passed: {{ size(deploys.filter(d, d.status == "passed")) }}

{{ join(deploys.map(d, "- " + d.name + " @ " + d.createdAt), "\n") }}
```

A bare `{{ deploys }}` renders the array as JSON for inspection; reach for `map` to shape each element and `join(..., "")` (or any separator) to splice the fragments into the output. Run-source variables always resolve to a single row today; pick the rows you need with `mode: list` on a memory namespace instead.

## HTML Panels

HTML panels render `content.body` directly as HTML. They share the markdown panel's variable system (same `{{ name.field }}` syntax and the same `MarkdownVariablesPanel` editor), so anything you can do with markdown variables works inside an HTML body too.

The editor is split code/preview: a monospace textarea for the HTML on the left, a live-rendered preview below it, and the shared variable manager on the right rail. Cmd/Ctrl+Enter saves, Escape cancels.

### Safety policy

The body is sanitized with DOMPurify at render time, before being injected via `dangerouslySetInnerHTML` into a scoped root element. The render pipeline is `interpolate variables -> DOMPurify allow-list -> scope <style> blocks -> render`, so variable values pulled from canvas memory or runs are sanitized the same way as hand-authored markup.

Concretely:

- **Allowed tags** — structural (`div`, `section`, `article`, `header`, `footer`, `main`, `aside`, `nav`, `p`, `h1`–`h6`, `blockquote`, `pre`, `hr`, `br`), inline (`span`, `strong`, `em`, `b`, `i`, `u`, `s`, `small`, `mark`, `sub`, `sup`, `code`, `kbd`, `samp`, `var`, `abbr`, `cite`, `dfn`, `q`, `time`, `data`), lists (`ul`, `ol`, `li`, `dl`, `dt`, `dd`), tables (`table`, `thead`, `tbody`, `tfoot`, `tr`, `th`, `td`, `caption`, `colgroup`, `col`), links and images (`a`, `img`, `figure`, `figcaption`), interactive (`details`, `summary`), and `style` (scoped — see below).
- **Allowed attributes** — `class`, `style`, `id`, `href`, `src`, `srcset`, `title`, `alt`, `width`, `height`, `colspan`, `rowspan`, `open`, `lang`, `dir`, `role`, `tabindex`, `name`, plus ARIA. Everything else is stripped, including every `on*` event handler.
- **Forbidden tags** — `script`, `noscript`, `iframe`, `frame`, `frameset`, `object`, `embed`, `applet`, `link`, `meta`, `base`, `head`, `html`, `body`, `title`, `template`, `form` and all form controls, `audio`, `video`, `source`, `track`, `canvas`, `math`, `svg`. These are removed even if they appear in the allow-list by accident.
- **`<img>` may load external images.** `src`/`srcset` go through the same URL allow-list as `href` below, so authors can write `<img src="https://cdn.example.com/logo.png">`. Authors should be aware that cross-origin image fetches do leak the canvas URL via the `Referer` header and can act as tracking pixels — only embed images from sources you trust. Other resource hooks (`poster` on `<video>`, deprecated `background`, `<object data>`, `ping`, `formaction`, `xlink:href`) are still always stripped, and inline `style` values that contain `url(...)`, `expression(...)`, `@import`, `behavior:`, `javascript:`, or `vbscript:` are dropped wholesale.
- **URL allow-list** — `href`, `src`, and `srcset` are restricted to `http(s)`, `mailto:`, `tel:`, fragments (`#…`), and relative paths. `javascript:` and `data:` URLs never survive on any attribute.
- **`<style>` blocks are kept and scoped.** Each rule is rewritten so its selectors are prefixed with `[data-console-html-root="<id>"]`, where `<id>` is unique to that panel instance, so user CSS cannot leak outside the widget root. Rules referencing `url(...)` are dropped, `@import` is dropped, and unknown at-rules (`@keyframes`, `@font-face`, etc.) are dropped. `@media` and `@supports` are recursed into so the scoping still applies.

### Tailwind classes

Tailwind v4 compiles only classes it finds while scanning source files. To make a stable, predictable set of utilities available to HTML widget authors, `web_src/src/App.css` includes a curated `@source inline(...)` safelist covering display, flex/grid, spacing, sizing, typography, color (`text-*`, `bg-*`, `border-*` for every default palette/shade), borders, rounding, shadow, opacity, overflow, positioning, z-index, cursor, and transition utilities. Authors can rely on those utility families without having to safelist anything per-panel; classes outside the safelist that aren't already in the bundle will simply have no effect.

Interactive variants (`hover:`, `focus:`) are safelisted for the families where they matter — color (`text-*`, `bg-*`, `border-*`), display, text decoration (`underline`, `line-through`, `no-underline`), `shadow*`, and `opacity-*`. So `hover:bg-blue-100`, `focus:shadow-lg`, `hover:underline`, and similar combinations all work. Hover/focus on sizing, spacing, and typography size families is **not** in the safelist (rare in practice, would multiply the bundle). The `dark:` variant is also not safelisted — dark mode in SuperPlane is opt-out via ancestors with `.dark-mode-disabled`, so widget authors should pick colors that work in both themes rather than relying on the variant.

### Authoring example

```yaml
- id: release-card
  type: html
  content:
    title: "{{ release.service }}"
    body: |
      <style>
        .badge {
          display: inline-block;
          padding: 2px 6px;
          border-radius: 4px;
          font-size: 11px;
          font-weight: 600;
        }
        .badge-passed { background: #dcfce7; color: #166534; }
        .badge-failed { background: #fee2e2; color: #991b1b; }
      </style>
      <div class="flex flex-col gap-2 text-sm">
        <div class="flex items-center gap-2">
          <strong>{{ release.service }}</strong>
          <span class="badge badge-{{ lastRun.status }}">{{ lastRun.status }}</span>
        </div>
        <p class="text-slate-600">Triggered by {{ lastRun.nodeName }}.</p>
      </div>
    variables:
      - name: release
        source:
          kind: memory
          namespace: releases
      - name: lastRun
        source:
          kind: run
          select: latest
```

The example uses Tailwind classes for layout and color, a scoped `<style>` block for the status badge, and CEL templates to interpolate live values. The backend stores the body as-is; sanitization is purely client-side at render time, which matches the markdown panel's trust model.

## Node Panels

Node panels display one or more pinned canvas nodes in a single card. The single and multi-node widgets share one implementation — `NodesPanelCard` — that renders the compact centered layout when the panel holds exactly one entry and a row list otherwise. Each entry resolves its `node` reference by id or name and can optionally show a manual Run button, with an optional `label` overriding the resolved canvas node name.

`showRun` only exposes the button, and only for **manual-run triggers** — the built-in `start` and `schedule` triggers whose component declares a user-invokable `run` hook. The UI filters on `node.component` against the hardcoded allowlist in `web_src/src/pages/app/console/manualRunTriggers.ts`; event triggers (for example `github.onPullRequest`) can still be pinned for status, but the Run button and the editor's Show-Run / trigger-template controls are hidden when a non-manual node is selected. The actual click still requires `canRunNodes`, and the backend authorization in `InvokeNodeTriggerHook` remains the source of truth.

`promptConfirmation` (default `false`) controls whether a Run click pops the confirmation dialog. Templates that declare input fields (`parameters`) always open the dialog so the operator can fill them in. Templates with no input fields fire immediately on click unless `promptConfirmation` is `true`, in which case a bare "Run X?" confirmation is shown first.

Every Run button — whether in the node panel or in a table row action — is disabled while its target trigger has an active canvas run in `STATE_STARTED`, so operators cannot enqueue duplicate runs while a pipeline is still executing. The shared `useConsoleTriggerLock` combines websocket-driven in-flight signals with a short submission-grace window; the tooltip switches to "A run for this trigger is already in progress." while the lock is engaged. Each Nodes panel holds a single lock instance shared by all of its entries, with submissions keyed by trigger node id — so two entries pointing at the same trigger lock together the moment either one fires, closing the window between the invoke call and the websocket-driven `STATE_STARTED` refresh.

> **Behavior change:** dashboards created before `promptConfirmation` existed used to prompt on every Run click. After upgrading, parameter-less triggers fire immediately on the first click; set `promptConfirmation: true` on the panel (or the individual Nodes entry) to restore the confirm-first behavior.

### Panel Types And Backward Compatibility

The Add Panel picker offers a single "Nodes" option (`type: nodes`) that covers both single- and multi-entry cases. The legacy `type: node` shape stays valid on the backend (`canvas_dashboard_yml.go`) and in YAML import so existing dashboards keep working; `NodesPanelCard` folds a `node` panel into a one-entry list at render time. When an author saves a legacy panel through the editor, the shape migrates to `type: nodes` on the fly (see `useConsolePanelState.migratedPanelType`), so once touched the panel adopts the modern list layout.

The plural `nodes` panel type renders several pinned canvas nodes in a single card, each with an optional purpose line. Use it for "Key Nodes" style summaries (for example, the entry/exit nodes of a preview-environment workflow):

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
| `showRun` | When true and the resolved node is a manual-run trigger (see [Node Panels](#node-panels)), surface a manual "Run" button. Still gated by `canRunNodes` and the shared in-flight lock. |
| `triggerName` | Optional start template name when the trigger exposes multiple templates. |
| `promptConfirmation` | When true, always confirm before running (default `false`). Templates with input fields always prompt regardless. |

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
