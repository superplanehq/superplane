# DigitalOcean Delete Load Balancer Skill

Use this guidance when planning or configuring the `digitalocean.deleteLoadBalancer` component.

## Purpose

`digitalocean.deleteLoadBalancer` permanently deletes a DigitalOcean load balancer by its ID.

The operation is idempotent: if the load balancer does not exist (404), the component still emits a success event.

## Required Configuration

- `loadBalancerID` (required): the UUID of the load balancer to delete (integration resource selector or expression).

## Output Fields

- `loadBalancerID`: the UUID of the load balancer that was deleted.

## Common Mapping

Chain after `digitalocean.createLoadBalancer` to delete it in a cleanup workflow:

- Load Balancer ID: `{{ $["Create Load Balancer"].data.id }}`

## Planning Rules

1. Use the integration resource selector to pick an existing load balancer from the account.
2. To delete a load balancer created earlier in the same workflow, use the expression `{{ $["Create Load Balancer"].data.id }}` for `loadBalancerID`.
3. This action does **not** delete the droplets behind the load balancer.
4. The component is idempotent — safe to call even if the load balancer was already deleted.
