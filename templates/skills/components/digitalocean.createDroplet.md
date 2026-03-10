# DigitalOcean Create Droplet Skill

Use this guidance when planning or configuring the `digitalocean.createDroplet` component.

## Purpose

`digitalocean.createDroplet` provisions a new DigitalOcean Droplet and waits until it reaches `active` status.

## Required Configuration

- `name` (required): droplet hostname.
- `region` (required): region slug (integration resource).
- `size` (required): size slug (integration resource).
- `image` (required): image slug (integration resource).

## Optional Configuration

- `sshKeys`: list of SSH key fingerprints or IDs.
- `tags`: list of tags.
- `userData`: cloud-init user data script.

## Output Fields

- `id`: droplet ID.
- `name`: hostname.
- `status`: droplet status (should be `active`).
- `region`: region information.
- `networks`: network details including IP addresses.

## Planning Rules

1. Use `region`, `size`, and `image` as integration resource selectors for dropdowns in the UI.
2. The component polls until the droplet is active; downstream nodes receive a fully ready droplet.
3. Use the output `id` to chain with `digitalocean.getDroplet`, `digitalocean.deleteDroplet`, or `digitalocean.manageDropletPower`.
