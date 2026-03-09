# Upsert Memory Component Skill

Use this guidance when planning or configuring the `upsertMemory` component.

## Purpose

The `upsertMemory` component updates existing canvas-scoped memory rows by namespace and exact key/value matches, and creates a new row when no match exists.

Use it when you want update-or-create behavior without splitting logic across `updateMemory` and `addMemory`.

## Required Configuration

- `namespace` (required): memory namespace string.
- `matchList` (optional): list of key/value pairs used as exact matches.
- `valueList` (required): list of key/value pairs to write.

`upsertMemory` requires at least one value entry. If `matchList` is empty, it performs namespace-level upsert.

## Planning Rules

When generating workflow operations that include `upsertMemory`:

1. Reuse stable namespace and key names used by other memory components.
2. Use deterministic keys in `matchList` (`environment`, `id`, `pull_request`) when you need multiple records in the same namespace.
3. Leave `matchList` empty when you need exactly one row per namespace (for example storing only a current `value` field).
4. Include all fields that should exist on both create and update in `valueList`.
5. Avoid broad matching criteria that could update multiple unrelated rows.

## Output Patterns

The component emits `memory.upserted` with:

- default channel always
- `data.namespace`
- `data.matches`
- `data.values` (the values written)
- `data.operation` (`updated` or `created`)
- `data.records` (affected records)
- `data.count`

## Example Configuration

- `namespace: "deployments"`
- `matchList:`
  - `{ "name": "environment", "value": "{{ root().environment }}" }`
  - `{ "name": "latest_deployment_source", "value": "manual_run" }`
- `valueList:`
  - `{ "name": "latest_deployment", "value": "{{ root().latest_deployment }}" }`
  - `{ "name": "environment", "value": "{{ root().environment }}" }`
  - `{ "name": "latest_deployment_source", "value": "manual_run" }`

## Namespace Singleton Example

- `namespace: "deployments"`
- `valueList:`
  - `{ "name": "value", "value": "{{ root().sha }}" }`

## Mistakes To Avoid

- Empty namespace.
- Empty `valueList`.
- Using changing fields (for example timestamps) as match keys.
