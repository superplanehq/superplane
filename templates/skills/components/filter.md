# Filter Component Skill

Use this guidance when planning or configuring the `filter` component.

## Purpose

The `filter` component evaluates a boolean expression and only forwards events when the expression is true.

## Required Configuration

- `expression` (required): expression that evaluates to a boolean.

## Output Channels

- `filter` emits only on `default` when the expression evaluates to true.
- When the expression evaluates to false, no output event is emitted.

## Planning Rules

When generating workflow operations that include `filter`:

1. Always set `configuration.expression`.
2. For downstream edges from `filter`, use `source.handleId: "default"` (or omit handle and allow default resolution).
3. Never use `source.handleId: "true"` or `source.handleId: "false"` for `filter`.
4. If a true/false branch is required, use an `if` component instead of `filter`.

## Good Expression Examples

- `indexOf(lower(root().data.comment.body), "create env") != -1`
- `root().data.repository.name == "front"`
- `previous().data.exitCode == 0`

## Mistakes To Avoid

- Leaving `expression` empty.
- Using non-boolean expressions.
- Routing `filter` with `true`/`false` channels.
