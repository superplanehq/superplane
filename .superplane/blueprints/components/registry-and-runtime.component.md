# Registry and Runtime Component Blueprint

## Capability Summary

The registry is the executable catalog for actions, triggers, integrations,
webhook handlers, setup providers, and widgets. Runtime contexts expose bounded
capabilities—HTTP, events, execution state, requests, secrets, files, memory,
auth, metadata, and integrations—to implementations during setup and execution.

Evidence: [`pkg/registry/registry.go`](../../../pkg/registry/registry.go),
[`pkg/registry/`](../../../pkg/registry/),
[`pkg/core/`](../../../pkg/core/),
[`pkg/workers/contexts/`](../../../pkg/workers/contexts/), and
[`pkg/registryimports/registryimports.go`](../../../pkg/registryimports/registryimports.go).

## Core Components

```component
name: RegistryRuntime
container: API and Web / Workers
responsibilities:
  - Resolving names to `core.Action`, `core.Trigger`, `core.Integration`, or `core.Widget`
  - Wrapping executable registrations with panic containment
```

```component
name: ExecutionContextBuilder
container: Workers
responsibilities:
  - Constructing capability-scoped setup, execution, and hook contexts
  - Binding contexts to workflow, node, integration, execution, and transaction state
```

Registrations occur through package initialization loaded by
`registryimports`. Each process copies the global registrations into its
`Registry`. #GRPCGatewayServices use it for discovery and validation;
#GitStagingVersioning uses it for node `Setup`; the workflow execution
capability uses it to run actions and triggers. Integration-qualified names
resolve through the registered integration’s own action/trigger lists.

## System Contracts

### Key Contracts

- Registry names are the persisted implementation identity in node `ref`
  values; an unknown name is a configuration/runtime error.
- Core blocks have unqualified names; integration implementations use a
  qualified name containing `.`.
- Panicable wrappers prevent one implementation panic from crossing the
  registry boundary unchecked.
- Widgets carry configuration only and are not wrapped as executable logic.
- Runtime HTTP goes through `registry.HTTPContext`, which applies response-size
  limits and a network policy blocking configured hosts/private ranges.
- Transaction-aware contexts receive the caller’s `*gorm.DB`; long external
  calls should use non-transactional contexts.
- Secrets and integration credentials enter implementations through context
  abstractions, not static registration data.

### Integration Contracts

- `core.Configurable` metadata drives API descriptions and frontend mappers.
- `Setup`, `Execute`, trigger/event, and hook contracts are invoked at distinct
  lifecycle stages.
- New implementations must be imported by `registryimports` to exist in every
  process registry.

## Architecture Decision Records

### ADR-001: Use a statically registered in-process plugin catalog

**Context:** API validation and workers must share exact implementation code and
configuration metadata.

**Decision:** Compile implementations into the application and register them at
package initialization, then expose them through `Registry`.

**Consequences:** Resolution is fast and type-safe and deployments have one
versioned catalog. Adding or changing an implementation requires rebuilding and
rolling the application image.

## Open Questions

- How should persisted nodes behave during rolling deployments where replicas
  briefly expose different registry revisions?
- Which runtime context capabilities need explicit audit events or quotas?
