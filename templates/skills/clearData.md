# clearData Component Skill

Use this guidance when planning or configuring the `clearData` component.

## Purpose

`clearData` removes matching entries from list-based canvas data storage.

Use it to clear stale mappings (for example PR -> sandbox records) after cleanup flows.

## Required Configuration

- `key` (required): canvas data key storing a list
- `matchBy` (required): field in each list object to match
- `matchValue` (required): value used to identify items to remove

## Output Channels

- `cleared`: one or more list entries were removed
- `notFound`: key missing or no matching list entry found

## Planning Rules

When generating workflow operations that include `clearData`:

1. Use it after successful delete/cleanup operations to remove corresponding stored mapping records.
2. For PR sandbox mapping cleanup, prefer:
   - `key: "pr_sandboxes"`
   - `matchBy: "pull_request"`
   - `matchValue` from PR event expression (for example `{{ root().data.issue.number }}`)
3. Route normal confirmation/follow-up from `cleared`.
4. Route fallback/no-op messaging from `notFound` when needed.

## Mistakes To Avoid

- Using `setData` to mimic removal behavior.
- Omitting `matchBy` or `matchValue`.
- Connecting both `cleared` and `notFound` to destructive follow-up actions.
