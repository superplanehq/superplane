# DigitalOcean Create Snapshot Skill

Use this guidance when planning or configuring the `digitalocean.createSnapshot` component.

## Purpose

`digitalocean.createSnapshot` creates a point-in-time snapshot of an existing DigitalOcean Droplet.

It waits until the DigitalOcean action completes, then emits snapshot details that can be used by downstream nodes.

## Required Configuration

- `dropletId` (required): the ID of the droplet to snapshot.
- `name` (required): a human-readable name for the snapshot.

## Output Fields

- `id`: resulting snapshot ID.
- `name`: snapshot name.
- `created_at`: when the snapshot was created.
- `resource_id`: the ID of the droplet that was snapshotted.
- `resource_type`: expected to be `droplet`.
- `regions`: regions where the snapshot is available.
- `min_disk_size`: minimum disk size required to use this snapshot.
- `size_gigabytes`: size of the snapshot in GB.

## Common Mapping

Use the snapshot ID in downstream nodes:

- Snapshot ID: `{{ $["Create Snapshot"].data.id }}`
- Snapshot Name: `{{ $["Create Snapshot"].data.name }}`

## Planning Rules

1. Prefer using a stable snapshot naming convention in `name` (for example include environment and timestamp).
2. When using this snapshot in downstream nodes, map from the `id` output field.
3. The droplet will be briefly paused during snapshot creation. Plan accordingly for production workloads.
4. Keep `dropletId` as a dynamic expression when possible to allow flexible workflow execution.
