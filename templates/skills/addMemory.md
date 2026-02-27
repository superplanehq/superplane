# Add Memory Component Skill

Use this guidance when planning or configuring the `addMemory` component.

## Purpose

The `addMemory` component appends a JSON object into canvas-scoped memory under a namespace.

Use it to persist values that later nodes can read in expressions with:

- `memory.find("<namespace>")`
- `memory.find("<namespace>", { ...matches })`
- `memory.findFirst("<namespace>")`
- `memory.findFirst("<namespace>", { ...matches })`

## Required Configuration

- `namespace` (required): memory namespace string.
- `valueList` (required): list of key/value entries to store in `values`.

`values` can still be accepted for compatibility, but prefer `valueList` for structured configuration.

## Planning Rules

When generating workflow operations that include `addMemory`:

1. Always set a stable `configuration.namespace` (for example `machines`, `tickets`, `deployments`).
2. Always include identifying fields in `valueList` that will be used later for lookup (`id`, `creator`, `pull_request`, etc.).
3. Prefer expressions in `valueList[*].value` when writing data from prior steps.
4. Keep namespace conventions consistent across the canvas; do not create near-duplicate namespace names.
5. Treat this as append-only storage; if replacement behavior is needed, pair with lookup-based branching and cleanup logic.

## Expression Read Patterns

Common follow-up patterns in later components:

- Existence check:
  - `memory.findFirst("machines", {"sandbox_id": root().data.sandbox_id}) != nil`
- Count matches:
  - `len(memory.find("machines", {"creator": "igor"})) > 0`
- Use first matching record:
  - `memory.findFirst("machines", {"pull_request": root().data.issue.number}).sandbox_id`

## Example Configuration

- `namespace: "machines"`
- `valueList:`
  - `{ "name": "sandbox_id", "value": "{{ $[\"Create Sandbox\"].id }}" }`
  - `{ "name": "creator", "value": "{{ root().data.sender.login }}" }`
  - `{ "name": "pull_request", "value": "{{ root().data.issue.number }}" }`

## Mistakes To Avoid

- Empty namespace or unstable namespace naming.
- Storing only opaque blobs without lookup fields.
- Assuming `memory.findFirst(...)` returns a value when no row matches (it returns `nil`).
