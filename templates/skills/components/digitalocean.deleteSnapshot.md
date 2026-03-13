# DigitalOcean Delete Snapshot Skill

Use this guidance when planning or configuring the `digitalocean.deleteSnapshot` component.

## Purpose

`digitalocean.deleteSnapshot` deletes a snapshot image from DigitalOcean.

## Required Configuration

- `snapshotId` (required): the ID of the snapshot to delete.

## Output Fields

- `snapshotId`: the deleted snapshot ID.
- `deleted`: confirmation boolean (always `true` on success).

## Common Mapping

When chaining from `digitalocean.createSnapshot`, map the snapshot ID:

- Snapshot ID: `{{ $["Create Snapshot"].data.id }}`

## Planning Rules

1. Use this component to clean up snapshots that are no longer needed.
2. When chaining from `digitalocean.createSnapshot`, map `snapshotId` from the upstream output snapshot ID.
3. Keep deletion flows explicit and avoid automatic deletion unless the user clearly requested cleanup behavior.
4. Deletion is permanent and cannot be undone.
