---
name: superplane-expressions
description: Use when editing canvas node configuration that contains Expr expressions or {{ }} templating. Covers payload access, nil-safe patterns, functions, YAML scalar styles, and expression vs text field types.
---

# SuperPlane Expressions and Templating

Load this skill before editing any canvas node configuration field that involves expressions or dynamic text. For provider-specific formatting (Discord embeds, Slack mrkdwn, Telegram Markdown, etc.), also load the **superplane-messaging** skill.

---

## 1. Expr syntax and safe patterns

SuperPlane uses the [Expr](https://github.com/expr-lang/expr) language. Expressions appear in two contexts depending on the field type (see section 5).

### Payload access

| Accessor | Returns | Example |
|----------|---------|---------|
| `$['Node Name']` | Upstream node's payload (by display name) | `$['GitHub onPush'].ref` |
| `root()` | The trigger event that started the run | `root().data.body.event` |
| `previous()` | Immediate upstream node's payload | `previous().data.status` |
| `previous(n)` | Walk *n* levels upstream | `previous(2).data.version` |
| `config` | Parent blueprint node configuration (blueprints only) | `config.environment` |
| `memory.find(ns, match)` | List of matching memory records | `memory.find("machines", {"id": "abc"})` |
| `memory.findFirst(ns, match)` | First matching record or nil | `memory.findFirst("hosts", {"env": "prod"}).ip` |

Node names in `$['...']` must match the **display name** exactly (case-sensitive). When two nodes share a name, the one closest in the execution chain wins.

`previous()` is **not available** when a node has multiple inputs (e.g., after a `merge`). In that case, use `$['Node Name']` to reference specific upstream nodes by name instead.

### Nil-safe patterns

Guard against nil with the safe-navigation operator `?.`, the nil-coalescing operator `??`, or ternary expressions:

```
root().data.incident?.priority?.summary ?? "unknown"

$['Read status'].data.data.values[0].lastStatus ?? "none"

len($['Read status'].data.data.values) > 0
  ? $['Read status'].data.data.values[0].lastStatus
  : "unknown"
```

Use `let` bindings to avoid repeating long accessor chains:

```
let vals = $['Read status'].data.data.values;
len(vals) > 0 ? vals[0].lastHealthyAtUnix : 0
```

### Truncation and slicing

Enforce maximum lengths with slice syntax:

```
let t = $['Generate title'].data.text;
len(t) > 256 ? t[:253] + "..." : t
```

### String functions

`lower()`, `upper()`, `trim()`, `split()`, `replace()`, `indexOf()`, `hasPrefix()`, `hasSuffix()`, `join()`

```
split($['Webhook'].data.body.ref, "/")[2]

indexOf(lower($['Slack Message'].data.text), "p1") >= 0
```

### Array functions

`filter()`, `map()`, `first()`, `last()`, `len()`, `count()`, `any()`, `all()`, `join()`

```
count($['Alerts'].data.data.result, .value[1] == "2")

join(map($['List'].data.items, .name), ", ")
```

### Type conversion

`int()`, `float()`, `string()`, `toJSON()`, `fromJSON()`, `toBase64()`, `fromBase64()`

```
string(int((now().Unix() - int($['Status'].data.data.values[0].lastHealthyAtUnix)) / 60)) + " minutes"
```

### Date and time

`now()` returns the current UTC time. Methods: `.Unix()`, `.Year()`, `.Month()`, `.Day()`, `.Hour()`, `.Minute()`, `.Format()`, `.Add()`.

`date(value)` parses RFC3339, `YYYY-MM-DD`, or Unix timestamps (seconds, millis, micros, nanos auto-detected). Optional second argument for timezone: `date(value, "America/New_York")`.

`duration("1h30m")` returns a Go duration.

```
int(now().Unix())

date("2026-04-22T10:00:00Z").Format("Jan 2, 2006")

now().Add(duration("-24h")).Format("2006-01-02")
```

### Conditional logic

Ternary and boolean operators:

```
$['Check'].data.status != "clear" || $['Health'].data.body.healthy == false

$['Incident'].data.priority == "P1" ? "high" : "low"
```

---

## 2. Payload structures

Every node payload follows the same shape:

```json
{ "data": { ... }, "timestamp": "2026-...", "type": "..." }
```

When accessing data in expressions, you always go through `.data`:

### Webhook trigger

```
root().data.body       // parsed JSON request body
root().data.headers    // HTTP headers map
```

### Integration triggers (examples)

```
root().data.workflow_run.head_branch       // github.onWorkflowRun
root().data.head_commit.message            // github.onPush
root().data.incident.priority.summary      // pagerduty.onIncident
root().data.incident.title                 // firehydrant.onIncident
root().data.text                           // slack.onAppMention
root().data.blocks                         // slack.onAppMention (Block Kit)
```

### Downstream node access

```
$['Run Deployment'].data.workflow_run.html_url
$['Create PD incident'].data.incident.html_url
$['Create issue on GitHub'].data.url
$['Generate Summary'].data.text                  // openai.textPrompt output
$['Read previous status'].data.data.values       // readMemory output
```

Use `superplane index triggers --name <name>` or `superplane index components --name <name>` to inspect the exact payload schema for any trigger or component.

---

## 3. YAML scalar styles with `{{ }}` expressions

YAML treats `{` as the start of a flow mapping. **Always quote strings that contain `{{ }}`.**

### Double-quoted (most common)

Use when you need `\n` escape sequences. Escape inner double quotes with `\"`:

```yaml
title: "Incident: {{ $[\"Generate title\"].data.text }}"
body: "Status: {{ $[\"Health check\"].data.status }}\nChecked at: {{ $[\"Timer\"].data.calendar.hour }}:{{ $[\"Timer\"].data.calendar.minute }}"
```

### Single-quoted

Inner double quotes pass through without escaping. Cannot use `\n` (literal backslash-n):

```yaml
expression: '$["Read status"].data.data.values[0].lastStatus == "ok"'
title: '{{ $["Generate title"].data.text }}'
```

### Block scalar (`|-` or `|`)

Good for long multiline bodies. `{{ }}` works without any quoting:

```yaml
body: |-
  Deployment of {{ $['Release'].data.release.name }} failed.

  Link: {{ $['Deploy'].data.workflow_run.html_url }}

  Rollback started automatically.
```

### Pitfalls

- **Unquoted** `{{ }}` values will cause a YAML parse error. Always use quotes or block scalars.
- In **double-quoted** strings, `$["Name"]` requires escaping: `$[\"Name\"]`.
- In **single-quoted** strings, a literal single quote is doubled: `''`.
- **Block scalars** strip trailing newlines with `|-` and keep them with `|`.

---

## 4. Expression field types

How you write an expression depends on the field type:

### `expression` type (raw Expr, no `{{ }}`)

Used by `filter`, `if`, and `merge` (`stopIfExpression`). The entire value is one Expr expression evaluated to a boolean or value.

```yaml
expression: '$["Check"].data.status == "ok" && $["Timer"].data.calendar.hour >= 9'
```

### `text` and `string` types (`{{ }}` interpolation)

Used by most message/content fields. Wrap each expression in `{{ }}`. Multiple expressions and plain text can be mixed in one value.

```yaml
text: "Deploy {{ $['Release'].data.release.name }} completed. See: {{ $['CI'].data.workflow_run.html_url }}"
```

All `{{ }}` expression results are converted to strings via `fmt.Sprintf("%v", value)`. A boolean `true` becomes `"true"`, an int `42` becomes `"42"`, and a map becomes its Go `map[...]` representation. Use `toJSON()` if you need structured data as a string (e.g., `{{ toJSON(root().data) }}`).

To check which type a field uses, run `superplane index components --name <component>` and look at the `type` property for each field.

### Error behavior

If an expression fails to compile or evaluate, the node execution **fails** with an error status. The error message (e.g., `"expression evaluation failed: ..."` or `"error resolving field X: ..."`) is stored on the execution and visible in the canvas UI. Common causes: referencing a node name that doesn't exist in the execution chain, accessing a nil field without a guard, or a syntax error in the expression.

---

## 5. Examples

### Filter: compound condition with nil-safe access

```yaml
expression: |-
  indexOf(
    lower(
      toJSON(root().data.blocks ?? [])
      + " "
      + (root().data.issue?.body ?? "")
      + " "
      + (root().data.issue?.title ?? "")
      + " "
      + (root().data.comment?.body ?? "")
    ),
    "p1 issue"
  ) >= 0
```

### Multi-expression body with fallback

```yaml
body: "{{ $['Generate description'].data.text ?? 'No description available' }}\n\n- [ ] Resolve\n- [ ] Root cause investigation"
```

### Date formatting in an email subject

```yaml
subject: "Endpoint is down (status {{ $[\"Health check request\"].data.status }})"
```

### Time calculation with memory

```yaml
body: "Approximate time healthy: {{ len($[\"Read previous status\"].data.data.values) > 0 ? string(int((now().Unix() - int($[\"Read previous status\"].data.data.values[0].lastHealthyAtUnix)) / 60)) + \" minutes\" : \"unknown (no prior check in memory)\" }}"
```

### Memory lookup in a condition

```yaml
expression: 'memory.findFirst("healthCheckMonitor", {"monitorKey": "default"}) != nil && memory.findFirst("healthCheckMonitor", {"monitorKey": "default"}).lastStatus == "ok"'
```

### Storing current timestamp in memory

```yaml
valueList:
  - name: "lastHealthyAtUnix"
    value: "{{ int(now().Unix()) }}"
```
