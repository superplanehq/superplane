# Read Memory Component Skill

Use this guidance when planning or configuring the `readMemory` component.

## Purpose

The `readMemory` component queries canvas-scoped memory by namespace and exact key/value matches.

Use it to retrieve previously stored values when later steps need deterministic lookup.

## Required Configuration

- `namespace` (required): memory namespace string.
- `resultMode` (required): `all` (all matches) or `latest` (only the newest match).
- `matchList` (required): list of key/value pairs used as exact matches.
- `emitMode` (optional, only when `resultMode = "all"`): `allAtOnce` (default) or `oneByOne`.

`readMemory` requires at least one match entry and will not run with an empty match set.

## Planning Rules

When generating workflow operations that include `readMemory`:

1. Always set `configuration.namespace` to the same namespace used by the corresponding `addMemory` writes.
2. Always include at least one stable identifying key in `matchList` (for example `sandbox_id`, `pull_request`, `creator`).
3. Prefer expressions in `matchList[*].value` to bind lookups to current execution context.
4. Reuse consistent field names between `addMemory.valueList[*].name` and `readMemory.matchList[*].name`.
5. Keep lookups specific; avoid broad keys that can match many rows unexpectedly.

## Output Patterns

The component emits `memory.read` with:

- `data.namespace`
- `data.matches`
- `data.resultMode`
- `data.emitMode`
- `data.values` (array of matched memory objects)
- `data.count`

Channel behavior:

- `found`: emitted when `data.count > 0`
- `notFound`: emitted when `data.count == 0`

Common follow-up patterns:

- Check if anything matched:
  - `{{ $["Read Memory"].data.count > 0 }}`
- Use the first matched value in later expressions:
  - `{{ $["Read Memory"].data.values[0].sandbox_id }}`

## Example Configuration

- `namespace: "machines"`
- `resultMode: "latest"`
- `emitMode: "allAtOnce"`
- `matchList:`
  - `{ "name": "pull_request", "value": "{{ root().data.issue.number }}" }`
  - `{ "name": "creator", "value": "{{ root().data.sender.login }}" }`

## Mistakes To Avoid

- Empty namespace.
- Invalid `resultMode` (must be `all` or `latest`).
- Empty `matchList`.
- Mismatched key names between write and read components.
- Assuming all reads return exactly one row; always account for zero or multiple matches.
