# Wait Component Skill

Use this guidance when planning or configuring the `wait` component.

## Purpose

The `wait` component pauses workflow execution, then emits to its default output channel when the wait completes.

It supports two modes:

- `interval`: wait for a fixed amount of time
- `countdown`: wait until a specific date/time

## Required Configuration

- `mode` (required): `interval` or `countdown`.
- For `interval` mode:
  - `waitFor` (required): string value representing an integer or expression result (for example `"20"` or `"{{ $.wait_time }}"`).
  - `unit` (required): `seconds`, `minutes`, or `hours`.
- For `countdown` mode:
  - `waitUntil` (required): target date/time value (supports expressions).

## Planning Rules

When generating workflow operations that include `wait`:

1. Always set `configuration.mode`.
2. For `interval`, always set both `configuration.waitFor` and `configuration.unit`.
3. Always provide `configuration.waitFor` as a string (never a raw JSON number).
4. For `countdown`, always set `configuration.waitUntil`.
5. Do not mix interval and countdown fields in the same configuration.
6. Keep `waitFor` positive for practical execution; zero or negative values can fail.
7. Do not invent extra output channels for this component.

## Expression Context

Wait fields can use expressions with workflow data, for example:

- `$["Node Name"].data.wait_seconds`
- `root().data.policy.delay_minutes`
- `previous().data.retry_after_seconds`
- `now().Add(duration("24h")).Format("2006-01-02T15:04:05Z")`

## Good Configuration Examples

Interval mode:

- `mode: "interval"`
- `waitFor: "10"`
- `unit: "minutes"`

Countdown mode:

- `mode: "countdown"`
- `waitUntil: "2026-12-31T23:59:59Z"`

## Mistakes To Avoid

- Omitting `mode`.
- Using `interval` without both `waitFor` and `unit`.
- Setting `waitFor` as a raw number instead of a string.
- Using `countdown` without `waitUntil`.
- Setting an invalid `unit` value.
- Using a past `waitUntil` timestamp.
