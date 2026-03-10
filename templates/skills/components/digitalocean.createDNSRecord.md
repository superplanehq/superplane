# DigitalOcean Create DNS Record Skill

Use this guidance when planning or configuring the `digitalocean.createDNSRecord` component.

## Purpose

`digitalocean.createDNSRecord` creates a new DNS record for a DigitalOcean domain.

## Required Configuration

- `domain` (required): domain name (supports expressions).
- `recordType` (required): one of A, AAAA, CNAME, MX, TXT, NS, SRV, CAA.
- `name` (required): hostname for the record; use `@` for apex domain (supports expressions).
- `data` (required): record data/value (supports expressions).

## Optional Configuration

- `ttl` (optional): time-to-live in seconds, defaults to 1800.

## Output Fields

- `id`: record ID.
- `type`: record type.
- `name`: record name.
- `data`: record data.
- `ttl`: record TTL.

## Planning Rules

1. For idempotent DNS updates, prefer `digitalocean.upsertDNSRecord` instead.
2. Use expressions to dynamically set record data from upstream droplet IPs.
