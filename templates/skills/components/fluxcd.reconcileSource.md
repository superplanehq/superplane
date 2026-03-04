# Flux CD Reconcile Source Skill

Use this guidance when planning or configuring the `fluxcd.reconcileSource` component.

## Purpose

`fluxcd.reconcileSource` forces reconciliation of a Flux CD resource by patching the `reconcile.fluxcd.io/requestedAt` annotation via the Kubernetes API.

## Required Configuration

- `kind` (required): the Flux resource kind to reconcile. Options: Kustomization, HelmRelease, GitRepository, HelmRepository, OCIRepository, Bucket.
- `name` (required): the name of the Flux resource.

## Optional Configuration

- `namespace` (optional): Kubernetes namespace of the resource. Defaults to the integration's configured namespace (typically flux-system).

## Output Fields

- `kind`: the resource kind
- `namespace`: the resource namespace
- `name`: the resource name
- `annotations`: updated annotations including the reconciliation timestamp
- `resourceVersion`: updated resource version
- `lastAppliedRevision`: last successfully applied revision (if available)
- `lastAttemptedRevision`: last attempted revision (if available)

## Common Mapping

Chain with the reconciliation trigger:

- Kind: `{{ $["On Reconciliation Completed"].data.involvedObject.kind }}`
- Name: `{{ $["On Reconciliation Completed"].data.involvedObject.name }}`
- Namespace: `{{ $["On Reconciliation Completed"].data.involvedObject.namespace }}`

## Planning Rules

1. Use this after an approval step to implement approval-gated deployments.
2. The component uses the Kubernetes API directly, so the integration's ServiceAccount must have patch permissions on the target resource.
3. All expression fields support dynamic values, so resource kind, namespace, and name can be determined at runtime from upstream events.
