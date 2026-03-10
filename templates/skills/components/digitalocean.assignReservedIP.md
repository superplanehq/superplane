# DigitalOcean Assign Reserved IP Skill

Use this guidance when planning or configuring the `digitalocean.assignReservedIP` component.

## Purpose

`digitalocean.assignReservedIP` assigns or unassigns a Reserved IP to/from a DigitalOcean Droplet and waits for completion.

## Required Configuration

- `reservedIp` (required): the Reserved IP address (supports expressions).
- `action` (required): `assign` or `unassign`.

## Optional Configuration

- `dropletId` (required for assign): droplet ID to assign the IP to (supports expressions).

## Output Fields

- `actionId`: DigitalOcean action ID.
- `reservedIp`: the Reserved IP address.
- `action`: the action performed.
- `dropletId`: the droplet ID (when assigning).
- `status`: final status (`completed`).

## Planning Rules

1. For failover scenarios, unassign from the old droplet first, then assign to the new one.
2. The component polls until completion; downstream nodes can safely rely on the IP being reassigned.
