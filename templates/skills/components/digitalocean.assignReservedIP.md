# DigitalOcean Assign Reserved IP Skill

Use this guidance when planning or configuring the `digitalocean.assignReservedIP` component.

## Purpose

`digitalocean.assignReservedIP` assigns or unassigns a DigitalOcean Reserved IP address to/from a droplet.

It polls until the DigitalOcean action completes and emits the action result.

## Required Configuration

- `reservedIP` (required): the reserved IP address to manage (integration resource selector or expression).
- `action` (required): `assign` or `unassign`.
- `dropletID` (required when action is `assign`): the target droplet ID to assign the reserved IP to.

## Optional Configuration

- `dropletID` is ignored when `action` is `unassign`.

## Output Fields

- `id`: DigitalOcean action ID.
- `status`: final action status (`completed`).
- `type`: `assign` or `unassign`.
- `started_at`: when the action started.
- `completed_at`: when the action completed.
- `resource_type`: will be `reserved_ip`.

## Common Mapping

Use the reserved IP in other nodes after assignment:

- Reserved IP value: `{{ $["Assign Reserved IP"].data.reserved_ip }}` (or use the same IP configured as input).

## Planning Rules

1. For blue/green deployments, sequence: create new droplet → assign reserved IP → delete old droplet.
2. For failover, use `assign` targeting the replacement droplet; the reserved IP will be automatically unassigned from the current droplet.
3. When `action` is `unassign`, the `dropletID` field is not required and should be omitted.
4. The component polls every 5 seconds; reserved IP actions typically complete in under 10 seconds.
