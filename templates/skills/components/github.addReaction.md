# GitHub Add Reaction Skill

Use this guidance when planning or configuring the `github.addReaction` component.

## Purpose

`github.addReaction` adds a reaction to an existing GitHub comment.

It supports:

- pull request conversation comments (`issues/comments/{id}`)
- pull request review line comments (`pulls/comments/{id}`)

## Required Configuration

- `repository` (required): repository containing the target comment.
- `target` (required): `issueComment` or `reviewComment`.
- `commentId` (required): numeric GitHub comment ID.
- `content` (required): reaction content.

## Supported Reaction Values

- `+1`
- `-1`
- `laugh`
- `confused`
- `heart`
- `hooray`
- `rocket`
- `eyes`

## Common Mapping

When using `github.onPRComment` upstream:

- Repository: `root().data.repository.name`
- Comment ID: `root().data.comment.id`

For `github.onPRReviewComment` upstream:

- Repository: `root().data.repository.name`
- Comment ID: `root().data.comment.id`
- Target: `reviewComment`

## Planning Rules

1. Always set `target` explicitly when the source trigger can be either PR conversation or review comments.
2. If the source is `github.onPRComment`, default `target` to `issueComment`.
3. If the source is `github.onPRReviewComment`, default `target` to `reviewComment`.
4. Keep `commentId` as a direct expression from trigger data when possible.

## Mistakes To Avoid

- Using `issueComment` target for review line comment IDs.
- Omitting `repository` when using dynamic comment IDs.
- Passing unsupported reaction content values.
