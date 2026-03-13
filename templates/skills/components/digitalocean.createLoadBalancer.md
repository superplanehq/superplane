# DigitalOcean Create Load Balancer Skill

Use this guidance when planning or configuring the `digitalocean.createLoadBalancer` component.

## Purpose

`digitalocean.createLoadBalancer` creates a new DigitalOcean load balancer with forwarding rules and optional droplet targets.

It polls until the load balancer status becomes **active**, then emits the full load balancer object.

## Required Configuration

- `name` (required): human-readable name for the load balancer.
- `region` (required): the region where the load balancer will be created (integration resource selector).
- `forwardingRules` (required): at least one forwarding rule object with `entryProtocol`, `entryPort`, `targetProtocol`, and `targetPort`.

## Optional Configuration

- `algorithm` (optional): `round_robin` (default) or `least_connections`.
- `dropletIds` (optional): one or more droplet IDs to add as targets (mutually exclusive with `tag`).
- `tag` (optional): a tag string to dynamically target droplets (mutually exclusive with `dropletIds`).

## Output Fields

- `id`: load balancer UUID.
- `name`: load balancer name.
- `ip`: assigned public IP address.
- `status`: will be `active` when emitted.
- `algorithm`: the balancing algorithm in use.
- `region`: region slug and name.
- `forwarding_rules`: list of active forwarding rules.
- `droplet_ids`: list of targeted droplet IDs (when using `dropletIds`).

## Common Mapping

Use the load balancer IP in downstream nodes:

- Load balancer IP: `{{ $["Create Load Balancer"].data.ip }}`
- Load balancer ID: `{{ $["Create Load Balancer"].data.id }}`

## Planning Rules

1. Always specify at least one forwarding rule. A typical HTTP setup uses `entry_protocol: http`, `entry_port: 80`, `target_protocol: http`, `target_port: 80`.
2. Use `dropletIds` to target specific known droplets, or `tag` for dynamic sets — never both in the same node.
3. Reference the `id` output when chaining into `digitalocean.deleteLoadBalancer` for cleanup workflows.
4. The component polls every 10 seconds; load balancer provisioning typically takes under 2 minutes.
