# Canvas Change Requests

This guide describes the current canvas versioning and change-request flow in SuperPlane.

## Lifecycle

When effective canvas versioning is enabled:

1. Update a draft version.
2. Create a change request from that draft.
3. Review the change request.
4. Approve, reject, or reopen according to status and conflict state.

## Status Model

Change requests use these statuses:

- `STATUS_OPEN`
- `STATUS_REJECTED`
- `STATUS_PUBLISHED`

Conflict is represented separately by metadata flag `is_conflicted`.

- A change request can be `STATUS_OPEN` and conflicted.
- `STATUS_CONFLICTED` is not used anymore.

## Action Rules

- Approve:
  - Allowed only for open, non-conflicted change requests.
  - Approval publishes the change request.
- Reject:
  - Allowed for open change requests (including conflicted ones).
  - Rejected change requests move to the rejected pile (`STATUS_REJECTED`).
- Reopen:
  - Allowed only for rejected change requests.
  - Reopening recalculates diff/conflicts and returns status to `STATUS_OPEN`.
- Resolve conflicts:
  - Updates the change-request version with a resolved canvas payload.
  - After resolving, the change request may become non-conflicted and can then be approved.

## Conflict Detection

Nodes are marked conflicted only when overlapping changes are structurally different between change-request version and live canvas.

If both sides changed the same node but resulting structure is identical (same JSON structure, position, config, etc.), it is not marked as conflicted.

## CLI Commands

`superplane canvases publish [name-or-id]`

- Creates a change request from your current draft version.
- Does not auto-approve or auto-publish.

`superplane canvases change-requests list [name-or-id] [--status <filter>] [--mine] [--query <text>] [--limit <n>] [--before <rfc3339>]`

`superplane canvases change-requests get <change-request-id> [name-or-id]`

`superplane canvases change-requests create [name-or-id] [--version-id <id>] [--title <text>] [--description <text>]`

`superplane canvases change-requests approve <change-request-id> [name-or-id]`

`superplane canvases change-requests reject <change-request-id> [name-or-id]`

`superplane canvases change-requests reopen <change-request-id> [name-or-id]`

`superplane canvases change-requests resolve <change-request-id> [name-or-id] --file <canvas.yaml> [--auto-layout horizontal] [--auto-layout-scope <scope>] [--auto-layout-node <id>]`

Notes:

- `[name-or-id]` can be omitted if an active canvas is set with `superplane canvases active`.
- `--status` supports `all`, `open`, `conflicted`, `rejected`, `published`.
