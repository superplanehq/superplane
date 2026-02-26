# Daytona Delete Sandbox Skill

Use this guidance when planning or configuring `daytona.deleteSandbox`.

## Purpose

`daytona.deleteSandbox` deletes an existing Daytona sandbox.

## Required Configuration

- `sandbox` (required): sandbox ID or name to delete.
- `force` (optional): boolean flag for force deletion.

## Planning Rules

When generating workflow operations that include `daytona.deleteSandbox`:

1. Always set `configuration.sandbox`.
2. Prefer sandbox **ID** over sandbox name.
3. For same-run cleanup, map `sandbox` from the create step output.
4. For cross-run cleanup (for example destroy PR command), retrieve sandbox ID via `getData` first, then map `sandbox` from `getData` output.
5. Only set `force` when user explicitly asks for force deletion or workflow requires best-effort cleanup of running sandboxes.
6. If you maintain PR->sandbox mappings in canvas data, follow delete with `clearData` to remove the matched mapping entry.

## Canonical Config Examples

Same run:

- `sandbox: "{{ $[\"Create Sandbox\"].data.id }}"`

Cross run with stored PR mapping:

- `sandbox: "{{ $[\"Get PR Sandbox Mapping\"].data.value }}"`

## Mistakes To Avoid

- Missing `configuration.sandbox` (node remains invalid).
- Deleting by guessed/static value when a mapped value is available.
- Skipping `getData` in cross-run flows where sandbox ID is persisted.
