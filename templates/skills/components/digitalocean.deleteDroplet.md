# DigitalOcean Delete Droplet Skill

Use this guidance when planning or configuring the `digitalocean.deleteDroplet` component.

## Purpose

`digitalocean.deleteDroplet` deletes an existing DigitalOcean Droplet.

## Required Configuration

- `dropletId` (required): the droplet ID to delete (supports expressions).

## Output Fields

- `dropletId`: the ID of the deleted droplet.

## Planning Rules

1. Keep deletion flows explicit; avoid automatic deletion unless the user clearly requested cleanup behavior.
2. When chaining from `digitalocean.createDroplet`, map `dropletId` from the upstream output ID.
