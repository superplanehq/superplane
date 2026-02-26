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

## Mistakes To Avoid

- Using local timezone defaults when user intent expects globally stable schedule timing.
- Omitting `timezone` for `days`, `weeks`, `months`, or `cron`.
- Choosing `cron` for simple daily schedules.
- Misreading 12-hour time (AM/PM) when converting to `hour`.
