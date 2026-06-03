# Canvas Change Requests

This guide describes the git-native canvas versioning and change-request flow in SuperPlane.

## Lifecycle

When change management is enabled for an app:

1. Create or continue a **draft git branch** (`drafts/<user-id>` by default).
2. **Commit** `canvas.yaml`, `console.yaml`, and other files to that branch (CLI, UI, or git push).
3. Create a change request referencing the **draft tip commit SHA**.
4. Review the change request and collect approvals.
5. **Publish** the change request to merge the draft branch to `main` and materialize live.

Version identifiers in change requests are **40-character git commit SHAs**, not UUIDs. `based_on_version_id` references the `main` tip SHA at creation time.

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
  - Approval records an approval event, but does not publish.
- Unapprove:
  - Allowed only for users with an active approval on an open change request.
  - Removes that user’s active approval.
- Publish:
  - Allowed only for open, non-conflicted change requests.
  - Requires all configured approver requirements to be actively approved.
  - Merges the draft branch to `main` and runs live materialization.
- Reject:
  - Allowed for open change requests (including conflicted ones).
  - Rejected change requests move to the rejected pile (`STATUS_REJECTED`).
  - Active approval records are invalidated.
- Reopen:
  - Allowed only for rejected change requests.
  - Reopening recalculates diff/conflicts and returns status to `STATUS_OPEN`.
  - Active approval records are invalidated, allowing a fresh approval cycle.
- Resolve conflicts:
  - Commits resolved YAML to the draft branch and re-materializes the draft tip.
  - After resolving, the change request may become non-conflicted and can then be approved/published.

## Approver Configuration

Canvas settings in `canvas.yaml` define who can approve/reject change requests:

- `Any user`
- `Specific user`
- `Role`

These rules are evaluated before publish is allowed.

## Conflict Detection

Nodes are marked conflicted when overlapping changes are structurally different between the change-request version (draft tip SHA) and the live canvas (`main` tip).

If both sides changed the same node but the resulting structure is identical, it is not marked as conflicted.

## CLI Commands

Draft branches:

`superplane apps drafts create [name-or-id]`

`superplane apps drafts list [name-or-id]`

`superplane apps drafts delete <branch-name> [name-or-id]`

Commit and read:

`superplane apps canvas update --draft -f <canvas.yaml>`

`superplane apps console set --draft -f <console.yaml>`

`superplane apps canvas get --draft -o yaml`

Change requests:

`superplane apps change-requests list [name-or-id] [--status <filter>] [--mine] [--query <text>] [--limit <n>] [--before <rfc3339>]`

`superplane apps change-requests get <change-request-id> [name-or-id]`

`superplane apps change-requests create [name-or-id] [--version-id <sha>] [--title <text>] [--description <text>]`

`superplane apps change-requests approve <change-request-id> [name-or-id]`

`superplane apps change-requests unapprove <change-request-id> [name-or-id]`

`superplane apps change-requests publish <change-request-id> [name-or-id]`

`superplane apps change-requests reject <change-request-id> [name-or-id]`

`superplane apps change-requests reopen <change-request-id> [name-or-id]`

`superplane apps change-requests resolve <change-request-id> [name-or-id] --file <canvas.yaml> [--auto-layout horizontal] [--auto-layout-scope <scope>] [--auto-layout-node <id>]`

Notes:

- `[name-or-id]` can be omitted if an active app is set with `superplane apps active`.
- `--status` supports `all`, `open`, `conflicted`, `rejected`, `published`.
- `--version-id` on create defaults to the current user's draft branch tip SHA when omitted.

See also [Git-Native Apps](git-native-apps.md).
