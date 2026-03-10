# DigitalOcean Create Snapshot Skill

Use this guidance when planning or configuring the `digitalocean.createSnapshot` component.

## Purpose

`digitalocean.createSnapshot` creates a snapshot image from a DigitalOcean Droplet and waits for completion.

## Required Configuration

- `dropletId` (required): the source droplet ID (supports expressions).

## Optional Configuration

- `name` (optional): snapshot name (supports expressions).

## Output Fields

- `actionId`: DigitalOcean action ID.
- `dropletId`: the source droplet ID.
- `status`: the final status (`completed`).

## Planning Rules

1. Use a stable naming convention for snapshots (e.g., include environment and timestamp).
2. When chaining from `digitalocean.createDroplet`, map `dropletId` from the upstream output.
3. The component polls until the snapshot action completes.
