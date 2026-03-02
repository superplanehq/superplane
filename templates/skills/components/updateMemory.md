# Update Memory Component Skill

Use this guidance when planning or configuring the `updateMemory` component.

## Purpose

The `updateMemory` component updates existing canvas-scoped memory rows by namespace and exact key/value matches.

Use it when memory rows already exist and you need to patch fields on those rows.

## Required Configuration

- `namespace` (required): memory namespace string.
- `matchList` (required): list of key/value pairs used as exact matches.
- `valueList` (required): list of key/value pairs to set on each matching row.

`updateMemory` requires at least one match entry and at least one value entry.

## Planning Rules

When generating workflow operations that include `updateMemory`:

1. Reuse the same namespace and key names used by `addMemory` and `readMemory`.
2. Use stable identifiers in `matchList` (`id`, `sandbox_id`, `pull_request`) to avoid updating unrelated rows.
3. Keep `valueList` focused on fields that should be overwritten.

## Output Patterns

The component emits `memory.updated` with:

- channel `found` when `data.count > 0`
- channel `notFound` when `data.count == 0`
- `data.namespace`
- `data.matches`
- `data.values` (the update patch that was applied)
- `data.updated` (array of updated memory objects)
- `data.count`

## Example Configuration

- `namespace: "machines"`
- `matchList:`
  - `{ "name": "pull_request", "value": "{{ root().data.issue.number }}" }`
  - `{ "name": "creator", "value": "{{ root().data.sender.login }}" }`
- `valueList:`
  - `{ "name": "status", "value": "running" }`
  - `{ "name": "updated_by", "value": "workflow" }`

## Mistakes To Avoid

- Empty namespace.
- Empty `matchList`.
- Empty `valueList`.
- Broad match criteria that can update unrelated entries.
