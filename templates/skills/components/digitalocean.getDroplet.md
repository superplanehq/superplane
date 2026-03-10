# DigitalOcean Get Droplet Skill

Use this guidance when planning or configuring the `digitalocean.getDroplet` component.

## Purpose

`digitalocean.getDroplet` fetches the current details of a DigitalOcean Droplet by ID.

## Required Configuration

- `dropletId` (required): the droplet ID to fetch (supports expressions).

## Output Fields

- `id`: droplet ID.
- `name`: hostname.
- `status`: current status.
- `region`: region information.
- `networks`: network details including IP addresses.

## Planning Rules

1. Use this to check droplet status before performing power or delete operations.
2. Map `dropletId` from upstream outputs like `digitalocean.createDroplet` using `{{ $["Create Droplet"].data.id }}`.
