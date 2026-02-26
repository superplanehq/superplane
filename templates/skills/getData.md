# getData Component Skill

Use this guidance when planning or configuring the `getData` component.

## Purpose

`getData` reads canvas-level stored data so downstream nodes can use values persisted by prior runs.
It emits on two output channels:

- `found`: value/lookup match exists
- `notFound`: key or lookup match is missing

## Required Configuration Shape

`getData` must include:

- `key` (required string)
- `mode` (required: `value` or `listLookup`)

For `mode: "listLookup"`, also include:

- `matchBy` (required string)
- `matchValue` (required expression/string/number)
- `returnField` (optional string; use when downstream needs one field like `sandbox_id`)

Optional fan-out fields:

- `emitEachItem` (optional bool): when true and selected value is a list, emits one `found` event per list item
- `itemField` (optional string): when `emitEachItem` is true, return this field from each object item

## Planning Rules

When generating workflow operations that include `getData`:

1. Use it before cleanup/destroy actions when the target identifier was stored in a previous run.
2. For list-backed storage, use lookup mode by stable identifier (for example `pull_request`).
3. If only one field is needed downstream, set lookup to return that specific field (for example `sandbox_id`).
4. Use expression-capable matching where appropriate (for example matching by trigger payload values).
5. Wire downstream deletion/cleanup components to the `getData` output, not to guessed/static IDs.
6. Route normal delete/continue logic from `found`, and fallback/reply logic from `notFound`.
7. Always set `mode` explicitly in generated operations (do not rely on UI default).
8. For PR sandbox destroy flows, prefer `mode: "listLookup"` with:
   - `matchBy: "pull_request"`
   - `matchValue` from PR event expression
   - `returnField: "sandbox_id"`
9. For "delete every sandbox" list cleanup flows, prefer:
   - `mode: "value"`
   - `emitEachItem: true`
   - `itemField: "sandbox_id"`
   so downstream `daytona.deleteSandbox` receives one sandbox ID per event.

## Common Lifecycle Pattern

1. `setData` stores `{ pull_request, sandbox_id, ... }`.
2. Later trigger path runs `getData` with `matchBy: pull_request` and dynamic `matchValue`.
3. Cleanup node reads returned `sandbox_id` and deletes the correct resource.

## Canonical Operation Example

```json
{
  "type": "add_node",
  "blockName": "getData",
  "nodeName": "Get PR Sandbox Mapping",
  "configuration": {
    "key": "pr_sandboxes",
    "mode": "listLookup",
    "matchBy": "pull_request",
    "matchValue": "{{ root().data.issue.number }}",
    "returnField": "sandbox_id"
  }
}
```

## Mistakes To Avoid

- Attempting cleanup with hardcoded IDs when stored data exists.
- Looking up list entries without a deterministic key.
- Returning full objects when downstream node expects a single scalar field.
- Omitting `mode` (can leave node invalid in generated proposals).
- Using `mode: "value"` when the key stores a list and lookup by identifier is required.
- Forgetting `emitEachItem` for list fan-out flows where downstream actions expect a single value per execution.
