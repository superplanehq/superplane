# DigitalOcean Create Load Balancer Skill

Use this guidance when planning or configuring the `digitalocean.createLoadBalancer` component.

## Purpose

`digitalocean.createLoadBalancer` creates a new DigitalOcean Load Balancer.

## Required Configuration

- `name` (required): load balancer name (supports expressions).
- `region` (required): region slug (integration resource).
- `entryProtocol` (required): one of http, https, tcp, udp.
- `entryPort` (required): port for incoming traffic.
- `targetProtocol` (required): one of http, https, tcp, udp.
- `targetPort` (required): port for backend traffic.

## Optional Configuration

- `algorithm` (optional): `round_robin` (default) or `least_connections`.

## Output Fields

- `id`: load balancer ID.
- `name`: load balancer name.
- `ip`: load balancer IP.
- `status`: current status.
- `region`: region information.

## Planning Rules

1. The load balancer starts with status `new` and transitions to `active` asynchronously.
2. Chain with droplet creation to add targets after both resources are ready.
