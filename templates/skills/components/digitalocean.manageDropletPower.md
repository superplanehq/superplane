# DigitalOcean Manage Droplet Power Skill

Use this guidance when planning or configuring the `digitalocean.manageDropletPower` component.

## Purpose

`digitalocean.manageDropletPower` performs power operations on a DigitalOcean Droplet and waits for completion.

## Required Configuration

- `dropletId` (required): the droplet ID (supports expressions).
- `action` (required): one of `power_on`, `shutdown`, `reboot`, `power_cycle`, `power_off`.

## Output Fields

- `actionId`: DigitalOcean action ID.
- `dropletId`: the droplet ID.
- `action`: the action performed.
- `status`: the final status (`completed`).

## Planning Rules

1. Prefer `shutdown` over `power_off` for graceful shutdowns.
2. Use `reboot` for quick restarts; use `power_cycle` for hard restarts.
3. The component polls until the action completes; downstream nodes can safely continue.
