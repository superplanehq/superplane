# Hetzner Create Snapshot Skill

Use this guidance when planning or configuring the `hetzner.createSnapshot` component.

## Purpose

`hetzner.createSnapshot` creates a snapshot image from an existing Hetzner Cloud server.

It waits until the Hetzner action completes, then emits snapshot details that can be used by downstream nodes.

## Required Configuration

- `server` (required): the source server ID.

## Optional Configuration

- `description` (optional): snapshot name/description shown in Hetzner Cloud.

## Output Fields

- `imageId`: resulting snapshot image ID.
- `imageType`: expected to be `snapshot`.
- `snapshotName`: snapshot description when set.
- `serverId`: source server ID.
- `actionId`: Hetzner action ID for tracking.

## Common Mapping

Use the snapshot output in `hetzner.createServer`:

- Image: `{{ $["Create Snapshot"].data.imageId }}`

## Planning Rules

1. Prefer using a stable snapshot naming convention in `description` (for example include environment and timestamp).
2. When creating a server from this snapshot in the same flow, map `createServer.image` to `imageId` from this component's output.
3. Keep `server` as an integration resource selection instead of hardcoded literals when possible.
