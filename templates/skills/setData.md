# setData Component Skill

Use this guidance when planning or configuring the `setData` component.

## Purpose

`setData` writes values to canvas-level data storage so values can be reused in later runs.

## Required Configuration Shape

`setData` must be configured with:

- `key` (required string)
- `valueList` (required list, at least one item)
- `operation` (required: `set` or `append`)
- `uniqueBy` (optional, only meaningful for `append`)

`valueList` items must be objects with:

- `name` (required string)
- `value` (required expression/string)

Do not use a legacy `value` field; use `valueList` only.

## Planning Rules

When generating workflow operations that include `setData`:

1. Use it immediately after creating external resources whose IDs are needed later (for example sandbox IDs).
2. Use a stable key name (for example `ephemeral_sandboxes`).
3. For multiple resources over time, use append/list behavior and include a stable identifier field in the value payload (for example `pull_request`).
4. If deduplication is needed, use append with `uniqueBy` set to that stable identifier.
5. Include all fields needed by future cleanup/reconciliation steps (for example `sandbox_id`, `creator`, `requester`, `pull_request`).
6. Always provide at least one `valueList` field item; empty `valueList` is invalid and leaves the node unconfigured.
7. For PR sandbox lifecycles, prefer:
   - key like `pr_sandboxes`
   - operation `append`
   - uniqueBy `pull_request`
   - fields including `pull_request` and `sandbox_id`

## Common Lifecycle Pattern

1. Create resource (for example `daytona.createSandbox`).
2. `setData` persists identity record for later runs.
3. Another trigger path can fetch and act on the stored record using `getData`.

## Canonical Operation Example

```json
{
  "type": "add_node",
  "blockName": "setData",
  "nodeName": "Store PR Sandbox Mapping",
  "configuration": {
    "key": "pr_sandboxes",
    "operation": "append",
    "uniqueBy": "pull_request",
    "valueList": [
      { "name": "pull_request", "value": "{{ root().data.issue.number }}" },
      { "name": "sandbox_id", "value": "{{ $[\"Create Sandbox\"].data.id }}" },
      { "name": "requester", "value": "{{ root().data.comment.user.login }}" }
    ]
  }
}
```

## Mistakes To Avoid

- Keeping identifiers only in transient execution context when later runs need them.
- Storing values without a stable lookup key for future retrieval.
- Overwriting full list state when append/upsert semantics are intended.
- Omitting `valueList` or leaving it empty.
- Using invalid field shape inside `valueList` (must be `{name, value}`).
