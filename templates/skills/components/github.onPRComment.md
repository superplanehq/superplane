# GitHub On PR Comment Skill

Use this guidance when planning or configuring the `github.onPRComment` trigger.

## Purpose

`github.onPRComment` starts a workflow when comments are posted on the pull request conversation (GitHub `issue_comment` on PRs).

It is suitable for PR command handling, review automation, and comment-driven delivery flows.

## Required Configuration

- `repository` (required): repository to monitor.
- `contentFilter` (optional): regex pattern used to match comment body text.

## Planning Rules

When generating workflow operations that include `github.onPRComment`:

1. Always set `configuration.repository`.
2. If the user did not specify a repository, ask one short clarifying question for the repository name only (not `owner/repo`) before proposing operations.
3. If the user then replies with a short repository value (for example `front`), treat it as the answer and proceed without asking again; never ask to convert it to `owner/repo`.
4. If a repository is already known in the current flow (from user input or an existing GitHub node), reuse that repository for related GitHub nodes unless the user asks for a different one.
5. If the user specifies a command phrase or quoted trigger text (for example `"create env"`), always set `configuration.contentFilter` to a regex that matches that phrase.
6. Treat `contentFilter` as a regex, not a simple substring.
7. Use downstream branching for complex conditions instead of overloading a single regex.
8. Before adding any downstream `filter` after this trigger, inspect this trigger's existing `configuration.contentFilter` first.
9. Do not add a downstream `filter` that contradicts or duplicates this trigger's `contentFilter` in the same linear path.
10. If the user asks for a different command phrase than the existing trigger filter (for example existing `"create env"` but request is `"destroy"`), prefer one of:
   - update this trigger's `contentFilter` to the new phrase when replacing behavior is intended, or
   - create a separate trigger/path for the new phrase when both behaviors should coexist.
11. Never produce an unreachable path where a downstream filter can never match because the trigger already excluded those events.

When the user asks to react to a specific comment command, prefer a case-insensitive boundary-safe regex, for example:

- command phrase `create env` -> `(?i)\bcreate env\b`

## Event Semantics

- This trigger handles PR comment events, including:
  - review line comments
  - PR conversation comments
  - review submission comments
- Do not treat it as issue-only comment handling.
- Event payload shape for this trigger uses issue-comment style fields.

## Payload Field Mapping (Important)

For `github.prComment` events:

- PR number: `root().data.issue.number`
- Repository name: `root().data.repository.name`
- Repository owner/org login: `root().data.repository.owner.login`
- Full repo name: `root().data.repository.full_name`
- Comment text: `root().data.comment.body`
- Comment URL: `root().data.comment.html_url`
- PR URL: `root().data.issue.pull_request.html_url`
- For PR workflow context values, use the `issue` and `repository` paths above.
- This trigger handles only PR conversation comments (`issue_comment` where `issue.pull_request` exists).
- Do not use this trigger for line-level review comments or review submission comments.
- SuperPlane passes through the full GitHub webhook payload in `data`.

## Common Expression Paths

- PR number: `root().data.issue.number`
- PR title: `root().data.issue.title`
- PR URL: `root().data.issue.pull_request.html_url` (or `root().data.issue.pull_request.url`)
- Comment text: `root().data.comment.body`

## Common Pattern

1. Add `github.onPRComment` trigger.
2. If a specific comment command/phrase is given, set `contentFilter` (for example `(?i)\bcreate env\b` or `^/deploy\b`).
3. Route to actions like deployment workflows, status updates, or automated replies.

## Mistakes To Avoid

- Omitting `repository`.
- Guessing or inferring `repository` when the user has not provided it.
- Asking for `owner/repo` format instead of just the repository name.
- Assuming `contentFilter` is not regex.
- Leaving `contentFilter` empty when the request includes a specific command phrase.
- Using this trigger when the request is explicitly about non-PR issue comments only.
- Adding a downstream `filter` for `"destroy"` when this trigger is already filtered to `"create env"` (or vice versa).
