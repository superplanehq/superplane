# DigitalOcean Delete Snapshot Skill

Use this guidance when planning or configuring the `digitalocean.deleteSnapshot` component.

## Purpose

`digitalocean.deleteSnapshot` deletes a snapshot image from DigitalOcean.

## Required Configuration

- `snapshotId` (required): the snapshot ID to delete (supports expressions).

## Output Fields

- `snapshotId`: the deleted snapshot ID.

## Planning Rules

1. Use this for cleanup after creating new snapshots, to manage snapshot limits.
2. Keep deletion flows explicit; avoid automatic deletion unless the user clearly requested cleanup behavior.
