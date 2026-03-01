# GitHub On Issue Comment Skill

Use this guidance when planning or configuring the `github.onIssueComment` trigger.

## Purpose

`github.onIssueComment` starts a workflow when a new comment is posted on a GitHub issue.

It is intended for comment-driven automations like command handling, triage, and notifications.

## Required Configuration

- `repository` (required): repository to monitor.
- `contentFilter` (optional): regex pattern used to match comment body text.

## Planning Rules

When generating workflow operations that include `github.onIssueComment`:

1. Always set `configuration.repository`.
2. If the user did not specify a repository, ask one short clarifying question for the repository name only (not `owner/repo`) before proposing operations.
3. If the user then replies with a short repository value (for example `front`), treat it as the answer and proceed without asking again; never ask to convert it to `owner/repo`.
4. If a repository is already known in the current flow (from user input or an existing GitHub node), reuse that repository for related GitHub nodes unless the user asks for a different one.
5. Only set `configuration.contentFilter` when the user asks for filtering behavior.
6. Treat `contentFilter` as a regex, not a plain substring.
7. Prefer downstream branching with `if` if user intent requires complex logic beyond one regex.

## Event Semantics

- Trigger should respond to issue comment events (not PR review comments).
- Use payload fields from `comment`, `issue`, `repository`, and `sender` for routing and decisions.

## Common Pattern

1. Add `github.onIssueComment` trigger.
2. Optionally filter quick commands with `contentFilter` (for example `^/deploy\b`).
3. Route to actions like posting a reply, updating an issue, or triggering a workflow.

## Mistakes To Avoid

- Omitting `repository`.
- Asking for `owner/repo` format instead of just the repository name.
- Treating `contentFilter` as non-regex text.
- Assuming this trigger handles pull request review comments.
