# Delete Memory Component Skill

Use this guidance when planning or configuring the `deleteMemory` component.

## Purpose

The `deleteMemory` component removes canvas-scoped memory rows by namespace and exact key/value matches.

Use it to clean up stale memory entries after resources are deleted or workflows finish.

## Required Configuration

- `namespace` (required): memory namespace string.
- `matchList` (required): list of key/value pairs used as exact matches.

`deleteMemory` requires at least one match entry and will not run with an empty match set.

## Planning Rules

When generating workflow operations that include `deleteMemory`:

1. Reuse the same namespace and key names used by `addMemory` and `readMemory`.
2. Prefer specific match keys (`id`, `sandbox_id`, `pull_request`) to avoid accidental bulk deletes.
3. Build match criteria that are specific enough to only delete the intended records.

## Output Patterns

The component emits `memory.deleted` with:

- channel `deleted` when `data.count > 0`
- channel `notFound` when `data.count == 0`
- `data.namespace`
- `data.matches`
- `data.deleted` (array of deleted memory objects)
- `data.count`

## Example Configuration

- `namespace: "machines"`
- `matchList:`
  - `{ "name": "pull_request", "value": "{{ root().data.issue.number }}" }`
  - `{ "name": "creator", "value": "{{ root().data.sender.login }}" }`

## Mistakes To Avoid

- Empty namespace.
- Empty `matchList`.
- Broad match criteria that can delete unrelated entries.
