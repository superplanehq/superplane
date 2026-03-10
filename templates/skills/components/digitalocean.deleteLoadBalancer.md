# DigitalOcean Delete Load Balancer Skill

Use this guidance when planning or configuring the `digitalocean.deleteLoadBalancer` component.

## Purpose

`digitalocean.deleteLoadBalancer` deletes a DigitalOcean Load Balancer.

## Required Configuration

- `loadBalancerId` (required): the load balancer ID to delete (supports expressions).

## Output Fields

- `loadBalancerId`: the deleted load balancer ID.

## Planning Rules

1. Keep deletion flows explicit.
2. Use expressions to map load balancer IDs from upstream components.
