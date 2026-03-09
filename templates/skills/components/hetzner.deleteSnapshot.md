# Hetzner Delete Snapshot Skill

Use this guidance when planning or configuring the `hetzner.deleteSnapshot` component.

## Purpose

`hetzner.deleteSnapshot` deletes a snapshot image from Hetzner Cloud.

## Required Configuration

- `snapshot` (required): snapshot image ID to delete.

## Output Fields

- `imageId`: the deleted snapshot image ID.

## Planning Rules

1. Use this component only for snapshot images, not system images.
2. When chaining from `hetzner.createSnapshot`, map `snapshot` from the upstream output image ID.
3. Keep deletion flows explicit and avoid automatic deletion unless the user clearly requested cleanup behavior.
