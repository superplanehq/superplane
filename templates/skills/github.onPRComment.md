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
2. If the user did not specify a repository, ask one short clarifying question for the repository name only (not `owner/repo`) before proposing operations.
3. If the user then replies with a short repository value (for example `front`), treat it as the answer and proceed without asking again.
4. If the previous assistant turn asked for repository and the user responds with a short text value, never ask for repository again in the next turn.
5. If a repository is already known in the current flow (from user input or an existing GitHub node), reuse that repository for related GitHub nodes unless the user asks for a different one.
6. Only set `configuration.contentFilter` when the user asks for filtering behavior.
7. Treat `contentFilter` as a regex, not a simple substring.
8. For simple comment-command matching, prefer `configuration.contentFilter` on the trigger (for example `^create env\b`, `^destroy\b`) instead of adding an `if` node.
9. Use downstream `if` branching only for logic that cannot be expressed cleanly as a single trigger filter (multi-field checks, combined predicates, non-comment payload logic).
10. If the user wants to listen to separate commands (for example "give me envs" and "destroy"), create separate `github.onPRComment` trigger nodes with separate `contentFilter` values.
11. Do not merge distinct command listeners into a single trigger regex just to reduce node count.

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

## Common Pattern

1. Add `github.onPRComment` trigger.
2. Optionally filter command comments (for example `^/deploy\b`).
3. Route to actions like deployment workflows, status updates, or automated replies.

For multiple independent PR comment commands:

1. Add one `github.onPRComment` trigger per command intent.
2. Set each trigger's `contentFilter` specifically for that command.
3. Route each trigger to its own path.

## Mistakes To Avoid

- Omitting `repository`.
- Guessing or inferring `repository` when the user has not provided it.
- Asking for `owner/repo` format instead of just the repository name.
- Re-asking for repository after user already provided a short repository answer (for example `front`).
- Assuming `contentFilter` is not regex.
- Adding an unnecessary `if` node when a trigger `contentFilter` is sufficient.
- Combining separate command listeners into one overloaded trigger/filter.
- Using this trigger when the request is explicitly about non-PR issue comments only.
