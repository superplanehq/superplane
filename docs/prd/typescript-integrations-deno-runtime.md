# TypeScript Integrations + Runtime Runner

## Overview

This PRD defines the current implementation for TypeScript integrations executed through the Runtime Runner in SuperPlane.

The implemented model includes:

1. TypeScript integration discovery and registration through the same `pkg/registry` mechanism as Go integrations.
2. TypeScript integration runtime operations (`sync`, `cleanup`) executed via the runtime runner.
3. TypeScript integration components discovered from integration manifests and executed through existing `core.Component` paths.

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

Integration operations currently delegated to the runtime runner:

1. `integration.sync`
2. `integration.cleanup`

Integration runtime response supports:

- `outcome` (`pass` / `fail` / `noop`)
- optional `errorReason` and `error`
- optional `metadata`
- optional integration `state` and `stateDescription`
- optional `resources`
- optional HTTP response payload (`http`)

Current Go wrapper behavior for operations not yet delegated:

1. `HandleAction`: returns `nil` (no-op).
2. `ListResources`: returns an empty list.
3. `HandleRequest`: returns HTTP 404.

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
3. `typeScriptRuntimeIntegrationTrigger` supports runtime `Setup()` through `SetupTrigger`; trigger actions remain unimplemented.

## Environment and Runtime Controls

- `TYPESCRIPT_INTEGRATIONS_DIR`: integration discovery root.
- `TYPESCRIPT_RUNNER_TRANSPORT`: runner transport (`http` or `grpc`, default `http`).
- `TYPESCRIPT_RUNNER_HTTP_BASE_URL`: runner HTTP base URL.
- `TYPESCRIPT_RUNNER_GRPC_ADDRESS`: runner gRPC address.
- `TYPESCRIPT_RUNNER_TIMEOUT`: request timeout.
- `TYPESCRIPT_RUNNER_AUTH_TOKEN`: optional shared bearer token.
- `TYPESCRIPT_RUNNER_VERSION`: request version.

The runtime runner service hosts Deno execution. Docker images install Deno via `scripts/docker/install-deno.sh`.

## Reference Implementation

`sdk/integrations/github2` is the initial TypeScript integration:

1. Integration config: sensitive `apiToken`.
2. Integration operations:
   - `integration.sync`: validates token against GitHub API and sets integration state.
   - `integration.cleanup`: supported through runner contract.
   - `integration.listResources`: currently returns empty list from Go wrapper.
   - `integration.handleAction`: currently no-op in Go wrapper.
   - `integration.handleRequest`: currently returns HTTP 404 from Go wrapper.
3. Integration component:
   - `github2.getIssue` fetches a GitHub issue using integration `apiToken`.

## Current Status

Completed:

1. Integration discovery from filesystem and registry registration.
2. Integration runtime ops (`sync`, `cleanup`) via runtime runner.
3. Integration component setup/execute in Deno with injected integration configuration.
4. Initial TypeScript integration (`github2`) and component (`github2.getIssue`).

Pending:

1. TypeScript integration trigger `HandleAction` implementation.
2. TypeScript integration runtime support for `handleAction`, `listResources`, and `handleRequest` where needed.
3. Runtime hardening (`--allow-*` minimization, sandbox policy, and parity tests).

## Acceptance Criteria

1. TypeScript integrations are discoverable via existing integration APIs.
2. Integration components are discoverable and executable via existing node execution flow.
3. Integration config fields appear in API/UI and are available to TS integration/component runtime.
4. At least one integration (`github2`) executes `sync` and one component (`github2.getIssue`) end-to-end through Deno.
5. Existing Go integrations/components continue to work without runtime branching in workers.
