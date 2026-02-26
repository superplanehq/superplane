# GitHub On Pull Request Skill

Use this guidance when planning or configuring the `github.onPullRequest` trigger.

## Purpose

`github.onPullRequest` starts a workflow when pull request lifecycle events occur.

Use it for PR-open and PR-close automations such as environment provisioning and cleanup.

## Required Configuration

- `repository` (required): repository to monitor.
- `actions` (required): one or more PR actions to listen for.
  - Common values: `opened`, `closed`
- `continuationEnabled` (optional): resume shared run session for the same PR.

## Planning Rules

When generating workflow operations that include `github.onPullRequest`:

1. Always set `configuration.repository`.
2. Always set `configuration.actions` explicitly.
3. For "on PR open and close" requests, include both actions:
   - `actions: ["opened", "closed"]`
4. If open and close require clearly different downstream flows, prefer two separate `github.onPullRequest` trigger nodes:
   - one with `actions: ["opened"]`
   - one with `actions: ["closed"]`
5. Ensure `actions` matches the node intent/name:
   - names/intents with "closed", "close", "destroy", "cleanup", "teardown" -> use `actions: ["closed"]`
   - names/intents with "opened", "open", "create env", "provision" -> use `actions: ["opened"]`
   - never label a node as closed while configuring `opened` (or vice versa)
6. If only repository is missing, ask one short clarifying question for repository name only.
7. If user replies with a short value (for example `front`), treat it as repository and proceed without re-asking.
8. Reuse an existing repository value from current canvas config when already present, unless user asks for a different repository.

## Payload Field Mapping (Important)

For `github.pullRequest` events:

- Action: `root().data.action` (for example `opened`, `closed`)
- PR number: `root().data.number`
- PR state: `root().data.pull_request.state`
- Merged flag: `root().data.pull_request.merged`
- Repository name: `root().data.repository.name`
- Repository owner/org login: `root().data.repository.owner.login`

## Common Patterns

Single trigger for both actions:

1. Add `github.onPullRequest` with `actions: ["opened", "closed"]`.
2. Branch downstream by `root().data.action` if needed.

Separate triggers (preferred when flows are independent):

1. Add `On Pull Request Opened` with `actions: ["opened"]`.
2. Add `On Pull Request Closed` with `actions: ["closed"]`.
3. Connect each trigger to its own flow.

## Mistakes To Avoid

- Omitting `actions`.
- Leaving `actions` at default when the user asked for both open and closed behavior.
- Configuring `actions: ["opened"]` for a close/destroy/cleanup flow.
- Mismatch between node label/intent and configured action.
- Re-asking for repository after user already provided it.
- Mixing opened/closed logic in one path without clear action-based branching.
