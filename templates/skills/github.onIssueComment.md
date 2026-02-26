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
2. Only set `configuration.contentFilter` when the user asks for filtering behavior.
3. Treat `contentFilter` as a regex, not a plain substring.
4. Prefer downstream branching with `if` if user intent requires complex logic beyond one regex.

## Event Semantics

- Trigger should respond to issue comment events (not PR review comments).
- Use payload fields from `comment`, `issue`, `repository`, and `sender` for routing and decisions.

## Common Pattern

1. Add `github.onIssueComment` trigger.
2. Optionally filter quick commands with `contentFilter` (for example `^/deploy\b`).
3. Route to actions like posting a reply, updating an issue, or triggering a workflow.

## Mistakes To Avoid

- Omitting `repository`.
- Treating `contentFilter` as non-regex text.
- Assuming this trigger handles pull request review comments.
