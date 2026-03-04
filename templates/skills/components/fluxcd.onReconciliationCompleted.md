# Flux CD On Reconciliation Completed Skill

Use this guidance when planning or configuring the `fluxcd.onReconciliationCompleted` trigger.

## Purpose

`fluxcd.onReconciliationCompleted` fires when a Flux CD resource completes a successful reconciliation. It receives webhooks from FluxCD's notification controller.

## Setup

1. The canvas must be saved to generate a webhook URL.
2. In the cluster, create a FluxCD Notification Provider of type `generic` pointing to the webhook URL.
3. Create a FluxCD Alert referencing the provider and the resources to monitor.

## Configuration

- `sharedSecret` (optional): shared secret for the Authorization: Bearer header sent by FluxCD.
- `kinds` (optional): filter by resource kind (Kustomization, HelmRelease, GitRepository, etc.). Leave empty for all kinds.

## Event Data

- `involvedObject.kind`: the Flux resource kind (e.g. Kustomization)
- `involvedObject.name`: the resource name
- `involvedObject.namespace`: the resource namespace
- `severity`: event severity (info for success)
- `reason`: reconciliation reason (e.g. ReconciliationSucceeded)
- `message`: human-readable message
- `metadata.revision`: the source revision

## Common Mapping

Use the event data in downstream nodes:

- Resource name: `{{ $["On Reconciliation Completed"].data.involvedObject.name }}`
- Revision: `{{ $["On Reconciliation Completed"].data.metadata.revision }}`
- Namespace: `{{ $["On Reconciliation Completed"].data.involvedObject.namespace }}`

## Planning Rules

1. The trigger only fires for successful reconciliations (severity=info, reason contains "Succeeded").
2. Use kind filtering when the Alert covers multiple resource types but you only want specific kinds.
3. For security, always set a shared secret in production environments.
