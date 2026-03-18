# DigitalOcean Create Alert Policy Skill

Use this guidance when planning or configuring the `digitalocean.createAlertPolicy` component.

## Purpose

`digitalocean.createAlertPolicy` creates a DigitalOcean monitoring alert policy that fires notifications when a droplet metric crosses a defined threshold.

It creates the policy synchronously and emits the full policy object, including its UUID, which can be used by downstream Get/Delete Alert Policy nodes.

## Required Configuration

- `description` (required): human-readable name for the policy, e.g. `"High CPU on web servers"`.
- `type` (required): the droplet metric to monitor. Common values:
  - `v1/insights/droplet/cpu` — CPU usage percentage
  - `v1/insights/droplet/memory_utilization_percent` — memory usage percentage
  - `v1/insights/droplet/public_outbound_bandwidth` — public outbound bandwidth (Mbps)
  - `v1/insights/droplet/public_inbound_bandwidth` — public inbound bandwidth (Mbps)
  - `v1/insights/droplet/load_1` / `load_5` / `load_15` — load averages
- `compare` (required): `"GreaterThan"` to alert above the threshold, `"LessThan"` to alert below it.
- `value` (required): numeric threshold that triggers the alert (e.g. `75` for 75% CPU).
- `window` (required): rolling evaluation window — `"5m"`, `"10m"`, `"30m"`, or `"1h"`.

## Optional Configuration

- `entities` (optional): list of specific droplet IDs to scope the policy. Leave empty to apply globally.
- `tags` (optional): list of tag names — the policy applies to all droplets carrying these tags.
- `enabled` (optional): defaults to `true`; set to `false` to create the policy in a disabled state.
- `email` (optional): list of email addresses to notify when the alert fires.
- `slackChannel` (optional): Slack channel to post alerts to, e.g. `"#alerts"`. Must be provided together with `slackUrl`.
- `slackUrl` (optional): Slack incoming webhook URL. Must be provided together with `slackChannel`.

## Output Fields

- `data.uuid`: the alert policy UUID — store this to use with Get/Delete Alert Policy.
- `data.description`: policy description.
- `data.type`: metric type being monitored.
- `data.compare`: comparison operator.
- `data.value`: configured threshold.
- `data.window`: evaluation window.
- `data.enabled`: whether the policy is active.
- `data.alerts.email`: configured email notification list.
- `data.alerts.slack`: configured Slack channels (array of `{url, channel}`).

## Common Mapping

- `description` ← static string or `{{ $.canvasInput.alertName }}`
- `type` ← static select value, e.g. `"v1/insights/droplet/cpu"`
- `compare` ← `"GreaterThan"` for most threshold alerts
- `value` ← static number or `{{ $.canvasInput.threshold }}`
- `entities` ← `["{{ $.steps.createDroplet.data.id }}"]` to scope to a newly created droplet
- Save the UUID for later: store `{{ $["Create Alert Policy"].data.uuid }}` in canvas memory

## Planning Rules

1. Always configure at least one notification channel — either `email`, or both `slackChannel` + `slackUrl`.
2. `slackChannel` and `slackUrl` must always be provided together — never one without the other.
3. Use `entities` to scope to specific droplets; use `tags` to scope by tag; omit both to apply globally. Both can be used simultaneously for an OR match.
4. Use a short `window` (`"5m"`) for rapid-response alerting; use `"30m"` or `"1h"` to avoid noise from transient spikes.
5. Store the output `uuid` in canvas memory when a matching Delete Alert Policy node exists downstream.
