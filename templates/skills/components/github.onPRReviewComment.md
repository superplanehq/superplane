# GitHub On PR Review Comment Skill

Use this guidance when planning or configuring the `github.onPRReviewComment` trigger.

## Purpose

`github.onPRReviewComment` starts a workflow when pull request review comments are posted.

It is intended for line-level review automation and review-submission workflows.

## Required Configuration

- `repository` (required): repository to monitor.
- `contentFilter` (optional): regex pattern used to match review content.

## Planning Rules

When generating workflow operations that include `github.onPRReviewComment`:

1. Always set `configuration.repository`.
2. Only set `configuration.contentFilter` when the user asks for filtering behavior.
3. Treat `contentFilter` as a regex, not a plain substring.
4. Use downstream branching for advanced logic beyond a single regex.

## Event Semantics

- This trigger handles:
  - `pull_request_review_comment` (line-level review comments)
  - `pull_request_review` (submitted review body)
- Do not use it for PR conversation comments from `issue_comment`.
- SuperPlane passes through the full GitHub webhook payload in `data`.

## Common Expression Paths

- PR number: `root().data.pull_request.number`
- Branch name: `root().data.pull_request.head.ref`
- Head commit SHA: `root().data.pull_request.head.sha`
- Review comment text: `root().data.comment.body` (`pull_request_review_comment`)
- Review submission text: `root().data.review.body` (`pull_request_review`)

## Common Pattern

1. Add `github.onPRReviewComment` trigger.
2. Optionally filter commands (for example `^/deploy\b`).
3. Route to actions like posting status, triggering validation, or deployment workflows.

## Mistakes To Avoid

- Omitting `repository`.
- Treating `contentFilter` as non-regex text.
- Using this trigger for PR conversation comments (`issue_comment`).
