# DigitalOcean Delete Alert Policy Skill

Use this guidance when planning or configuring the `digitalocean.deleteAlertPolicy` component.

## Purpose

`digitalocean.deleteAlertPolicy` permanently removes a DigitalOcean monitoring alert policy.

The operation is idempotent — if the policy has already been deleted (404), the component still emits a successful result.

## Required Configuration

- `alertPolicy` (required): the UUID of the alert policy to delete. Supports expressions, e.g. `{{ $.steps.createAlertPolicy.data.uuid }}`.

## Output Fields

- `data.alertPolicyUuid`: the UUID of the deleted alert policy.

## Common Mapping

- `alertPolicy` ← `"{{ $.steps.createAlertPolicy.data.uuid }}"` to delete a policy created earlier in the same workflow
- `alertPolicy` ← canvas memory value when the UUID was stored from a previous run

## Planning Rules

1. This operation is permanent and cannot be undone — confirm the UUID is correct before connecting this node.
2. Since the operation is idempotent, it is safe to re-run workflows that include this component without side effects.
3. Use canvas memory to pass the policy UUID between the Create and Delete nodes when they appear in the same canvas.
4. When the intent is to replace a policy (delete then recreate), chain this node before `digitalocean.createAlertPolicy` in the same workflow.
