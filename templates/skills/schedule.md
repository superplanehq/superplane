# Schedule Trigger Skill

Use this guidance when planning or configuring the `schedule` trigger.

## Purpose

`schedule` starts a workflow on a recurring time-based cadence.

Use it for nightly jobs, periodic cleanup, recurring maintenance, and cron-style automation.

## Required Configuration

- `type` (required): one of `minutes`, `hours`, `days`, `weeks`, `months`, `cron`.
- Additional required fields depend on `type`:
  - `minutes`: `minutesInterval`
  - `hours`: `hoursInterval` (and usually `minute`)
  - `days`: `daysInterval` (and usually `hour`, `minute`)
  - `weeks`: `weeksInterval`, `weekDays` (and usually `hour`, `minute`)
  - `months`: `monthsInterval`, `dayOfMonth` (and usually `hour`, `minute`)
  - `cron`: `cronExpression`
- `timezone` is required for `days`, `weeks`, `months`, and `cron`.

## Planning Rules

When generating workflow operations that include `schedule`:

1. Prefer `type: "days"` for plain "every day/night at HH:MM" requests.
2. Set `daysInterval: 1` for daily/nightly intent.
3. Parse natural language times to 24-hour values:
   - `9:00 PM` -> `hour: 21`, `minute: 0`
   - `9:30 AM` -> `hour: 9`, `minute: 30`
4. For "every night" with a specific time, treat it as daily at that time.
5. Default schedule timezone to UTC for deterministic behavior unless user explicitly requests another timezone.
6. For UTC, set `timezone: "0"`.
7. Use `type: "cron"` only when the user requests cron syntax or advanced cadence not cleanly represented by built-in modes.
8. Never leave a schedule trigger partially configured. Always provide required fields in `configuration` in the same operation that creates/updates the node.
9. Do not ask follow-up timezone questions for schedule requests when the user already provided a specific time; default to UTC and proceed.

## Operation-Level Requirement

When creating or editing a `schedule` node, include a complete `configuration` object in the operation payload.

For `add_node` with `blockName: "schedule"`, always include `configuration`.

For an existing schedule node, use `update_node_config` and set the full schedule configuration explicitly.

Do not rely on defaults when user intent contains an explicit time.

## NL Intent Mapping (Important)

If user intent is equivalent to:

- "every night 9:00 destroy all my nodes"
- "run nightly at 9pm and clean up"

then configure the trigger as:

- `type: "days"`
- `daysInterval: 1`
- `hour: 21`
- `minute: 0`
- `timezone: "0"` (UTC)

Then route downstream to the requested cleanup/destroy steps.

## Canonical Operation Example

Use a shape equivalent to this when user asks for nightly 9:00 UTC:

```json
{
  "type": "add_node",
  "blockName": "schedule",
  "nodeName": "Nightly 9PM",
  "configuration": {
    "type": "days",
    "daysInterval": 1,
    "hour": 21,
    "minute": 0,
    "timezone": "0"
  }
}
```

If user explicitly asks for cron syntax instead, use:

```json
{
  "type": "update_node_config",
  "target": { "nodeName": "Nightly 9PM" },
  "configuration": {
    "type": "cron",
    "cronExpression": "0 21 * * *",
    "timezone": "0"
  }
}
```

## Mistakes To Avoid

- Using local timezone defaults when user intent expects globally stable schedule timing.
- Omitting `timezone` for `days`, `weeks`, `months`, or `cron`.
- Creating a `schedule` node without a complete `configuration` payload.
- Asking timezone clarifying questions for schedule requests that already include a concrete time.
- Choosing `cron` for simple daily schedules.
- Misreading 12-hour time (AM/PM) when converting to `hour`.
