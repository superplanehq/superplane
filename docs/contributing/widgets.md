# Widgets

Widgets are markdown-embedded data blocks that render canvas memory entries or
execution events as a table, chart, or single-number stat. They are authored
inside Apps markdown panels using a fenced ` ```widget ` (or the deprecated
` ```query ` alias) code block whose body is YAML.

```widget
source: memory
namespace: environments
columns:
  - label: PR
    field: pr_number
  - label: Title
    field: pr_title
where:
  - field: status
    op: eq
    value: ready
actions:
  - label: Destroy
    kind: trigger
    trigger: destroy
    confirm: "Destroy PR #{{ pr_number }}?"
```

Widgets are implemented in
[`web_src/src/ui/Markdown/WidgetBlock.tsx`](../../web_src/src/ui/Markdown/WidgetBlock.tsx)
and CEL evaluation lives in
[`web_src/src/ui/Markdown/widgetExpr.ts`](../../web_src/src/ui/Markdown/widgetExpr.ts).

## Sources

A widget is bound to one of two data sources via the required `source` key.

### `source: memory`

Reads canvas memory entries (the same key/value bag that components write
through the runtime memory API).

| field | type | required | notes |
|-------|------|----------|-------|
| `namespace` | string | yes | only entries in this namespace are surfaced |

Each row exposes `id`, `namespace`, and the entry's `values.*` keys are
hoisted to the row root so `field: pr_number` resolves to `values.pr_number`.

### `source: executions`

Reads canvas events and their executions through the existing
`useInfiniteCanvasEvents` hook; live updates flow over the canvas WebSocket.

| field | type | required | notes |
|-------|------|----------|-------|
| `trigger` | string | no | filter to events whose root node id matches this slug |
| `status` | enum | no | one of `running` / `passed` / `failed` / `cancelled` |
| `limit`  | number | no | default 10, capped at 100 |

Each row exposes:

| key | description |
|-----|-------------|
| `id` | event id |
| `root` | the full `CanvasEventWithExecutions` payload |
| `status` | normalized aggregate status |
| `duration` | seconds between event start and the latest execution update |
| `executions` | `CanvasNodeExecutionRef[]` |
| `node` | per-node-id index of the latest execution |

## Columns

`columns` is an optional list. When omitted, memory widgets auto-derive
columns from the union of `values` keys; executions widgets fall back to a
default trio (run id, status, started time).

```yaml
columns:
  - label: Status
    field: status
    format: badge
```

`field` is either a dot-path (`root.data.issue.number`) or a single CEL
expression wrapped in `{{ ... }}`.

`format` controls cell rendering:

| value | renders as |
|-------|-----------|
| `plain` (default) | text |
| `link` | anchor with truncated URL display |
| `link:Label` | anchor with custom display text |
| `relative` | relative time with absolute hover title |
| `date` | absolute UTC timestamp |
| `badge` | rounded pill (status-aware palette for known run statuses) |
| `code` | inline `<code>` block |

## Filters (`where`)

A list of conditions ANDed together. Each condition has a comparator (`op`),
a `field` (dot-path or CEL), and a literal `value` (or CEL).

| op | semantics |
|----|-----------|
| `eq` / `neq` | string equality on the stringified value |
| `contains` / `not_contains` | substring on the stringified value |
| `gt` / `lt` | numeric comparison after `parseFloat` |
| `exists` / `not_exists` | non-empty / empty (`value` is ignored) |

```yaml
where:
  - field: status
    op: eq
    value: running
```

## Actions

A list of buttons rendered in an extra "Actions" column on tables. Every
action requires `label` and `kind` (`trigger`, `approve`, `cancel`, or
`push-through`). Common optional fields:

| field | type | notes |
|-------|------|-------|
| `variant` | enum | `default` / `primary` / `danger` |
| `icon` | enum | `trash` / `play` / `refresh` / `stop` / `external-link` |
| `confirm` | string | shows a confirm dialog whose body is the (CEL-templated) string |
| `show` | string | row predicate; CEL `{{ ... }}` or legacy `field == "value"` |

`kind: trigger` actions also accept:

| field | type | notes |
|-------|------|-------|
| `trigger` | string (required) | trigger slug on the canvas |
| `template` | string | optional event template |
| `fill` | object | nested map of dot-paths → CEL templates that build the trigger payload (always coerced to strings) |

`kind: approve` and `kind: push-through` require `node`. `kind: cancel`
needs no extra fields.

## Render kinds

The optional `render` block selects an output mode. When omitted, widgets
render a table.

```yaml
render:
  kind: chart
  chart:
    type: bar
    x: name
    y: duration
```

| `render.kind` | extra block | notes |
|---------------|-------------|-------|
| `table` (default) | — | row-per-entry table with optional Actions column |
| `chart` | `chart: { type, x, y?, group?, label?, aggregate?, color? }` | recharts-backed bar / line / area / stacked-bar / donut |
| `number` | `number: { field, aggregate, label, format, sparkline? }` | single big stat with optional sparkline |

`chart.x`, `chart.y`, `chart.group`, and `number.field` are `MaybeExpr` —
literal dot-paths today, optionally CEL expressions tomorrow.

## CEL expressions in `{{ ... }}`

Any expression-bearing widget field accepts a single
[CEL](https://github.com/google/cel-spec) expression wrapped in `{{ ... }}`.
Bare strings (no braces) keep the legacy dot-path semantics, so widgets
authored before CEL support continue to work unchanged.

Each expression is parsed once per panel render and evaluated against every
row, so adding CEL to a widget with hundreds of rows is cheap.

### Where you can use CEL

| field | meaning of a literal | meaning of `{{ ... }}` |
|-------|---------------------|------------------------|
| `columns[i].field` | dot-path into row | CEL evaluated against row |
| `where[i].field` | dot-path into row | CEL evaluated against row |
| `where[i].value` | literal scalar to compare against | CEL evaluated against row |
| `render.chart.x` / `render.chart.y` / `render.chart.group` | dot-path | CEL |
| `render.number.field` | dot-path | CEL |
| `actions[i].show` | legacy `field == "value"` comparator | CEL boolean |
| `actions[i].confirm` | string with no interpolation | string with one or more `{{ ... }}` segments |
| `actions[i].fill[path]` | string with no interpolation | string with one or more `{{ ... }}` segments |

### Variables available to expressions

In every expression, the row's keys are top-level identifiers, and these
globals are merged in:

| name | type | description |
|------|------|-------------|
| `now` | int | seconds since the Unix epoch, computed once per render |

For `source: memory` rows, the row's keys are `id`, `namespace`, and the
hoisted `values.*` entries. For `source: executions` rows, the keys are
`id`, `status`, `duration`, `node`, plus the full event under `root` and
the executions under `executions`.

### Standard CEL features

cel-js ships with arithmetic (`+ - * / %`), comparisons (`< <= > >= == !=`),
logical (`&& || !`), the ternary operator (`a ? b : c`), index access
(`a[0]`, `m["key"]`), the `in` operator, the `has(obj.field)` macro, and
collection macros (`.all()`, `.exists()`, `.exists_one()`, `.filter()`,
`.map()`).

### Custom functions

Type coercions and a few string/date helpers are registered as free
functions (cel-js does not currently allow custom string methods, so use
`contains(s, sub)` rather than `s.contains(sub)`).

| signature | example | result |
|-----------|---------|--------|
| `int(x)` | `int("42")` | `42` (fail-soft → `0`) |
| `float(x)` | `float("3.14")` | `3.14` |
| `string(x)` | `string(42)` | `"42"` |
| `lower(s)` | `lower("HI")` | `"hi"` |
| `upper(s)` | `upper("hi")` | `"HI"` |
| `contains(s, sub)` | `contains(repo, "core")` | bool |
| `startsWith(s, p)` | `startsWith(name, "feat-")` | bool |
| `endsWith(s, p)` | `endsWith(name, ".test")` | bool |
| `matches(s, re)` | `matches(version, "^v\\d+\\.\\d+$")` | bool |
| `duration(seconds)` | `duration(3725)` | `"1h 2m"` |
| `timestamp(seconds)` | `timestamp(0)` | ISO 8601 string |

### Examples

```yaml
# Show the age of each row as a human-readable duration.
columns:
  - label: PR
    field: pr_number
  - label: Age
    field: "{{ duration(int(now) - int(created_at)) }}"
```

```yaml
# Filter to executions started more than 2 hours ago.
where:
  - field: "{{ int(now) - int(created_at) }}"
    op: gt
    value: "{{ 7200 }}"
```

```yaml
# Chart Y is computed per row.
render:
  kind: chart
  chart:
    type: bar
    x: name
    y: "{{ (int(now) - int(created_at)) / 60 }}"
```

```yaml
# Success-rate stat: passed → 1.0, anything else → 0.0, then average.
render:
  kind: number
  number:
    field: '{{ status == "passed" ? 1.0 : 0.0 }}'
    aggregate: avg
    label: Success rate
    format: percent
```

```yaml
# Conditional button using CEL plus a multi-segment confirm template.
actions:
  - label: Stop
    kind: cancel
    show: '{{ status == "running" }}'
    confirm: 'Stop {{ upper(repo) }} run #{{ pr_number }}?'
```

### Error handling

CEL is fail-soft inside widgets:

- A parse error logs a `console.warn` and renders the field as empty.
- A runtime error (missing identifier, type mismatch, division by zero,
  ...) is caught; the affected column cell renders empty, the affected
  row is dropped from charts and number aggregates, the `where` predicate
  evaluates to false, the `show` predicate fails closed, and template
  segments render as empty (literal text around them is preserved).

### Out of scope

- CEL inside canvas component nodes (those still use `expr-lang`).
- Replacing the AutoComplete preview engine in
  [`web_src/src/lib/exprEvaluator.ts`](../../web_src/src/lib/exprEvaluator.ts).
- Async expressions, user-defined functions, expression editor / autocomplete.
