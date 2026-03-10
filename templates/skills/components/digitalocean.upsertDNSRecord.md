# DigitalOcean Upsert DNS Record Skill

Use this guidance when planning or configuring the `digitalocean.upsertDNSRecord` component.

## Purpose

`digitalocean.upsertDNSRecord` provides an idempotent create-or-update flow for DNS records. It looks up existing records by type and name, then creates or updates accordingly.

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
- `action`: whether the record was `created` or `updated`.

## Planning Rules

1. Prefer this over `digitalocean.createDNSRecord` when the record may already exist.
2. Use for blue/green deployments where the DNS record needs updating on each deploy.
