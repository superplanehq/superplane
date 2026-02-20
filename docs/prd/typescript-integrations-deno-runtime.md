# TypeScript Integrations + Deno Runtime

## Overview

This PRD defines the current implementation for TypeScript integrations executed by Deno in SuperPlane.

The implemented model includes:

1. TypeScript integration discovery and registration through the same `pkg/registry` mechanism as Go integrations.
2. TypeScript integration runtime operations (`sync`, `handleAction`, `listResources`, `handleRequest`) executed via Deno.
3. TypeScript integration components discovered from integration manifests and executed in Deno through existing `core.Component` paths.

## Goals

1. Allow integrations to be authored in TypeScript.
2. Keep integration, integration-component, and integration-trigger discoverability aligned with Go registration/lookup semantics.
3. Keep workers runtime-agnostic (`Setup()`/`Execute()` paths unchanged).
4. Support one real integration reference implementation (`github2`) with secure token configuration.

## Non-Goals

- Artifact storage/versioning and publishing workflows.
- Trigger runtime execution parity in this phase.
- OAuth/browser action parity in this phase.
- Replacing Go integrations.

## Discovery Model

Base directory: `TYPESCRIPT_INTEGRATIONS_DIR`

Each integration directory must contain:

- `index.ts`
- `manifest.json`

Example:

- `${TYPESCRIPT_INTEGRATIONS_DIR}/github2/index.ts`
- `${TYPESCRIPT_INTEGRATIONS_DIR}/github2/manifest.json`

Integration manifest schema (current):

- Integration metadata:
  - `name`, `label`, `icon`, `description`, `instructions`, `configuration`
- Component references:
  - `name`
  - `directory` (relative folder containing component `index.ts` + `manifest.json`)
- Trigger references:
  - `name`
  - `directory` (relative folder containing trigger `index.ts` + `manifest.json`)

Example nested component path:

- `${TYPESCRIPT_INTEGRATIONS_DIR}/github2/components/getIssue/index.ts`
- `${TYPESCRIPT_INTEGRATIONS_DIR}/github2/components/getIssue/manifest.json`

## Registry Behavior

`pkg/registry` now loads TypeScript integrations from `TYPESCRIPT_INTEGRATIONS_DIR` during registry initialization.

Behavior currently implemented:

1. `ListIntegrations` and `DescribeIntegration` include TypeScript integrations.
2. `integration.Components()` returns TypeScript integration components.
3. `integration.Triggers()` returns discovered trigger definitions.
4. `registry.GetComponent("<integration>.<component>")` resolves integration components.
5. `registry.GetTrigger("<integration>.<trigger>")` resolves integration triggers.

## Runtime Contract (Implemented)

Integration entrypoint operations implemented in `pkg/runtime/typescript/contract.go` and executed via `pkg/runtime/typescript/runner.go`:

1. `integration.sync`
2. `integration.handleAction`
3. `integration.listResources`
4. `integration.handleRequest`

Integration runtime response supports:

- `outcome` (`pass` / `fail` / `noop`)
- optional `errorReason` and `error`
- optional `metadata`
- optional integration `state` and `stateDescription`
- optional `resources`
- optional HTTP response payload (`http`)

Integration cleanup currently remains a no-op in the Go wrapper and does not invoke a TypeScript operation.

## Integration Component Runtime

Integration components execute with the same component contract used by standalone TypeScript components:

1. `component.setup`
2. `component.execute`

Additionally, integration component setup/execute requests include:

- `integrationConfiguration` resolved from integration config fields via `IntegrationContext.GetConfig(...)`

This enables integration components to consume integration-level secrets/configuration (for example `github2.getIssue` using `apiToken`).

## Execution Architecture

Workers remain generic.

Runtime-specific behavior lives in registry wrappers:

1. `typeScriptRuntimeIntegration` maps integration lifecycle calls to Deno runtime calls.
2. `typeScriptRuntimeIntegrationComponent` maps `Setup()` and `Execute()` to Deno component runtime calls.
3. `typeScriptRuntimeIntegrationTrigger` currently provides discoverability only; runtime methods are not implemented yet.

## Environment and Runtime Controls

- `TYPESCRIPT_INTEGRATIONS_DIR`: integration discovery root.
- `DENO_BINARY`: Deno binary path (default `deno`).
- `DENO_EXECUTION_TIMEOUT`: subprocess timeout (default `30s`).

Current Deno invocation includes:

- `deno run --quiet --no-prompt --allow-net <entrypoint>`

Docker images now install Deno via `scripts/docker/install-deno.sh`.

## Reference Implementation

`sdk/integrations/github2` is the initial TypeScript integration:

1. Integration config: sensitive `apiToken`.
2. Integration operations:
   - `integration.sync`: validates token against GitHub API and sets integration state.
   - `integration.listResources`: returns empty list.
   - `integration.handleAction`: no-op.
   - `integration.handleRequest`: returns HTTP 404.
3. Integration component:
   - `github2.getIssue` fetches a GitHub issue using integration `apiToken`.

## Current Status

Completed:

1. Integration discovery from filesystem and registry registration.
2. Integration runtime ops (`sync`, `handleAction`, `listResources`, `handleRequest`) via Deno.
3. Integration component setup/execute in Deno with injected integration configuration.
4. Initial TypeScript integration (`github2`) and component (`github2.getIssue`).

Pending:

1. TypeScript integration trigger runtime (`Setup`, `HandleAction`) implementation.
2. Integration cleanup runtime op in TypeScript (if required).
3. Runtime hardening (`--allow-*` minimization, sandbox policy, and parity tests).

## Acceptance Criteria

1. TypeScript integrations are discoverable via existing integration APIs.
2. Integration components are discoverable and executable via existing node execution flow.
3. Integration config fields appear in API/UI and are available to TS integration/component runtime.
4. At least one integration (`github2`) executes `sync` and one component (`github2.getIssue`) end-to-end through Deno.
5. Existing Go integrations/components continue to work without runtime branching in workers.
