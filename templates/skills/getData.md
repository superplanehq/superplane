# getData Component Skill

Use this guidance when planning or configuring the `getData` component.

## Purpose

`getData` reads canvas-level stored data so downstream nodes can use values persisted by prior runs.

## Planning Rules

When generating workflow operations that include `getData`:

1. Use it before cleanup/destroy actions when the target identifier was stored in a previous run.
2. For list-backed storage, use lookup mode by stable identifier (for example `pull_request`).
3. If only one field is needed downstream, set lookup to return that specific field (for example `sandbox_id`).
4. Use expression-capable matching where appropriate (for example matching by trigger payload values).
5. Wire downstream deletion/cleanup components to the `getData` output, not to guessed/static IDs.

## Common Lifecycle Pattern

1. `setData` stores `{ pull_request, sandbox_id, ... }`.
2. Later trigger path runs `getData` with `matchBy: pull_request` and dynamic `matchValue`.
3. Cleanup node reads returned `sandbox_id` and deletes the correct resource.

## Mistakes To Avoid

- Attempting cleanup with hardcoded IDs when stored data exists.
- Looking up list entries without a deterministic key.
- Returning full objects when downstream node expects a single scalar field.
