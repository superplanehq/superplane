# GitHub On PR Comment Skill

Use this guidance when planning or configuring the `github.onPRComment` trigger.

## Purpose

`github.onPRComment` starts a workflow when comments are posted on pull requests.

It is suitable for PR command handling, review automation, and comment-driven delivery flows.

## Required Configuration

- `repository` (required): repository to monitor.
- `contentFilter` (optional): regex pattern used to match comment body text.

## Planning Rules

When generating workflow operations that include `github.onPRComment`:

1. Always set `configuration.repository`.
2. Only set `configuration.contentFilter` when the user asks for filtering behavior.
3. Treat `contentFilter` as a regex, not a simple substring.
4. Use downstream branching for complex conditions instead of overloading a single regex.

## Event Semantics

- This trigger handles PR comment events, including:
  - review line comments
  - PR conversation comments
  - review submission comments
- Do not treat it as issue-only comment handling.

## Common Pattern

1. Add `github.onPRComment` trigger.
2. Optionally filter command comments (for example `^/deploy\b`).
3. Route to actions like deployment workflows, status updates, or automated replies.

## Mistakes To Avoid

- Omitting `repository`.
- Assuming `contentFilter` is not regex.
- Using this trigger when the request is explicitly about non-PR issue comments only.
