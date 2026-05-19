# Dashboard table panels

Table panels list rows from **canvas memory** (recommended for ephemeral environment
dashboards) or from executions / runs. Configuration lives on each panel's
`content` object and can be edited via the panel form or YAML tab.

## Memory source

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

The editor scans live canvas memory and suggests namespaces and field names.

## Row actions

Row actions invoke a **trigger node** on the canvas (`InvokeNodeTriggerHook`).
Downstream steps (such as HTTP Request) run when that trigger fires.

- `node` — trigger node id or name (required)
- `hook` — defaults to `run`
- `template` — Start template name when applicable
- `payload` — map of dot-paths to literals or `{{ CEL }}` templates merged into the run payload
- `confirm` — optional confirmation dialog (supports `{{ }}` interpolation)
- `show` — CEL `{{ }}`, legacy `field == "value"`, or dashboard expressions

## CEL

Fields wrapped in `{{ ... }}` use [CEL](https://github.com/google/cel-spec).
Each row exposes its memory keys plus `now` (Unix seconds).

## Filters (`where`)

Structured filters are ANDed:

| `op` | Meaning |
|------|---------|
| `eq` / `neq` | String equality |
| `contains` / `not_contains` | Substring |
| `gt` / `lt` | Numeric compare |
| `exists` / `not_exists` | Non-empty / empty field |
