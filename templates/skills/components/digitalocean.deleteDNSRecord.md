# DigitalOcean Delete DNS Record Skill

Use this guidance when planning or configuring the `digitalocean.deleteDNSRecord` component.

## Purpose

`digitalocean.deleteDNSRecord` deletes a DNS record from a DigitalOcean domain.

## Required Configuration

- `domain` (required): domain name (supports expressions).
- `recordId` (required): record ID to delete (supports expressions).

## Output Fields

- `domain`: the domain name.
- `recordId`: the deleted record ID.

## Planning Rules

1. Chain from `digitalocean.createDNSRecord` using the output `id` as `recordId`.
2. Keep deletion flows explicit.
