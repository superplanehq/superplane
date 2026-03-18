# DigitalOcean Get Alert Policy Skill

Use this guidance when planning or configuring the `digitalocean.getAlertPolicy` component.

## Purpose

`digitalocean.getAlertPolicy` retrieves the full configuration of a DigitalOcean monitoring alert policy by its UUID.

It emits the policy object, which can be inspected or used to conditionally drive downstream actions.

## Required Configuration

- `alertPolicy` (required): the UUID of the alert policy to retrieve. Supports expressions for dynamic lookup, e.g. `{{ $.steps.createAlertPolicy.data.uuid }}`.

## Output Fields

- `data.uuid`: the alert policy UUID.
- `data.description`: human-readable description.
- `data.type`: the metric type being monitored (e.g. `v1/insights/droplet/cpu`).
- `data.compare`: comparison operator — `"GreaterThan"` or `"LessThan"`.
- `data.value`: the configured threshold value.
- `data.window`: the evaluation window (`"5m"`, `"10m"`, `"30m"`, `"1h"`).
- `data.entities`: scoped droplet IDs (empty array if not scoped).
- `data.tags`: scoped tag names (empty array if not scoped).
- `data.enabled`: whether the policy is active.
- `data.alerts.email`: list of notification email addresses.
- `data.alerts.slack`: list of Slack channels (each has `url` and `channel`).

## Common Mapping

- `alertPolicy` ← `"{{ $.steps.createAlertPolicy.data.uuid }}"` when reading back a just-created policy
- `alertPolicy` ← canvas memory value when the UUID was stored from a previous run

## Planning Rules

1. Use this component when you need to verify the current state of a policy before modifying it.
2. The output `data.enabled` field can be used in a downstream `if` node to branch on whether the policy is active.
3. Prefer selecting the alert policy from the integration resource picker rather than hardcoding a UUID when building canvases manually.
