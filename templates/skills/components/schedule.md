# Schedule Trigger Skill

Use this guidance when planning or configuring the `schedule` trigger.

## Purpose

`schedule` starts workflows on recurring intervals (minutes/hours/days/weeks/months) or cron.

## Required Timezone Format (Critical)

For schedule types that use timezone (`days`, `weeks`, `months`, `cron`), `configuration.timezone` must be a numeric UTC offset string.

Valid examples:

- `"-5"`
- `"0"`
- `"5.5"`
- `"+8"`

Invalid examples:

- `"UTC"`
- `"GMT+8"`
- `"America/New_York"`
- `"current"`

## Planning Rules

When generating workflow operations that include `schedule`:

1. Always inspect trigger configuration schema first.
2. Only set `configuration.timezone` for `days`, `weeks`, `months`, or `cron`.
3. If the user specifies timezone by region/name (for example `UTC`, `PST`, `America/Los_Angeles`), convert it to numeric offset before writing configuration.
4. If you cannot safely determine the numeric offset for the user-intended schedule time, ask one short clarification and return `operations: []`.
5. Never emit schedule operations with non-numeric timezone values.

## Type-Safe Reminders

- `minute`, `hour`, interval fields, and `dayOfMonth` must be JSON numbers, not strings.
- `cronExpression` must be a string in valid 5-field or 6-field cron syntax.

## Mistakes To Avoid

- Setting `timezone: "UTC"` or any IANA timezone name.
- Setting timezone for `minutes` or `hours` schedule types.
- Using string numbers for numeric fields (for example `"21"` instead of `21`).
