# Discord Get Last Mention Skill

Use this guidance when planning or configuring the `discord.getLastMention` component.

## Purpose

`discord.getLastMention` fetches recent messages in a Discord channel and returns the most recent one that mentions the bot.

It is intended for pull-based mention handling when webhook or realtime trigger delivery is not available.

## Required Configuration

- `channel` (required): Discord channel to query.
- `since` (optional): Lower-bound date string; only mentions at/after this time are considered. Supports expressions.
  Accepted formats include ISO 8601 (recommended, e.g. `2026-03-10T16:00:00Z`) and Go timestamps (e.g. `2026-03-16 04:17:08.750328135 +0000 UTC`).

## Planning Rules

When generating workflow operations that include `discord.getLastMention`:

1. Always set `configuration.channel`.
2. Prefer reusing the same channel used by nearby Discord actions unless the user requests another channel.
3. Use output channel branching (`found` / `notFound`) to handle “no mention found” gracefully.
4. When mention text must match commands, add `if` or `filter` nodes after this component.

## Output Semantics

The component emits `discord.getLastMention.result` with:

- `channel_id`
- `mention` (message payload when found)

Output channels:

- `found`: Mention payload exists.
- `notFound`: No mention matched current filters.

## Common Pattern

1. Run `discord.getLastMention`.
2. Branch on output channel `found` / `notFound`.
3. Parse `mention.content` and run command-specific actions.

## Mistakes To Avoid

- Omitting `channel`.
- Assuming a mention always exists.
- Parsing commands before checking the `found` channel.
