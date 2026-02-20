# TypeScript Contract + SDK + Deno Runtime Components

## Overview

This PRD defines the current first phase of TypeScript support in SuperPlane:

1. A stable TypeScript execution contract.
2. A TypeScript SDK for implementation authors.
3. Deno execution of TypeScript components through the same `core.Component` interface used by Go components.

This phase explicitly excludes implementation artifact storage, version publishing workflows, and alias management.

## Goals

1. Allow authors to implement components in TypeScript.
2. Keep TypeScript components discoverable through the same registry/API flow as Go components.
3. Execute TypeScript component `setup` and `execute` in Deno.
4. Preserve existing worker behavior (worker remains runtime-agnostic).
5. Provide a clear local authoring model with `manifest.json` + `index.ts`.

## Non-Goals

- Storing/publishing implementation artifacts.
- Version alias workflows (`latest`, `stable`, `prod`).
- Trigger and action implementations in TypeScript (in this phase).
- Replacing existing Go runtime.

## Scope

In scope:

- TypeScript runtime contract for component operations.
- SDK types/runtime helper.
- Filesystem discovery of TypeScript components.
- Registry registration and API discoverability.
- Deno execution for `component.setup` and `component.execute`.
- Timeout control for Deno subprocesses.

Out of scope:

- Artifact/version management APIs.
- Hot reload of manifests/source without restart.
- Full Deno permission hardening policy (follow-up).

## Discovery Model

TypeScript components are discovered from `TYPESCRIPT_COMPONENTS_DIR`.

Each component is a directory with:

- `index.ts`
- `manifest.json`

Example:

- `${TYPESCRIPT_COMPONENTS_DIR}/noop2/index.ts`
- `${TYPESCRIPT_COMPONENTS_DIR}/noop2/manifest.json`

Discovery occurs at server startup.

## Registry and API Integration

Discovered TypeScript components are registered as `core.Component` implementations in `pkg/registry`.

Manifest metadata feeds existing APIs:

- name / label / description / documentation
- icon / color
- configuration
- output channels
- example output

As a result, `ListComponents` and `DescribeComponent` include TypeScript components through the same path as Go components.

## Runtime Contract

Current operations:

- `component.setup`
- `component.execute`

Request/response structures are defined in:

- `pkg/runtime/typescript/contract.go`
- `sdk/typescript/types.ts`

## Execution Architecture

Worker flow is unchanged:

1. Worker obtains component from registry.
2. Worker calls `component.Setup(...)` and `component.Execute(...)` using existing code paths.

For TypeScript components:

- `Setup()` builds a `component.setup` runtime request and invokes Deno.
- `Execute()` builds a `component.execute` runtime request and invokes Deno.
- Runtime responses are mapped back into existing execution state operations (`Pass`, `Emit`, `Fail`, metadata/KV updates).

There is no special runtime branch in the node execution worker.

## Runtime Controls

- `DENO_BINARY` (default: `deno`)
- `DENO_EXECUTION_TIMEOUT` (default: `30s`)

Current gap:

- Restrictive `--allow-*` permission profile and sandbox tests are not fully implemented yet.

## Functional Requirements

1. TypeScript component definitions must be discoverable via standard component APIs.
2. `setup` and `execute` must be independently invokable (no implicit setup on execute).
3. Manifest configuration must be returned by APIs and rendered by UI.
4. Runtime failures must map to existing execution error behavior.
5. Existing Go components must remain unaffected.

## Implementation Status

Completed:

1. Contract + SDK baseline.
2. Filesystem discovery and registry registration.
3. Deno runtime invocation for setup/execute in component implementation.
4. Initial TypeScript component example (`noop2`).

Pending:

1. Trigger and action support.
2. Sandbox permission hardening.
3. Runtime parity tests and stress/security test coverage.

## Acceptance Criteria

1. At least one TypeScript component is discoverable in `ListComponents` and `DescribeComponent`.
2. TypeScript component `setup` and `execute` both run in Deno.
3. Manifest-defined configuration appears in API payloads and UI forms.
4. Worker code path remains generic (no TypeScript-specific routing logic in node worker).
5. Existing Go component behavior remains unchanged.

## Risks and Mitigations

1. Startup-time manifest failures block registration:
   - Mitigation: strict, actionable startup errors.
2. Runtime behavior drift from Go components:
   - Mitigation: parity fixture coverage and CI checks.
3. Sandbox/security gaps:
   - Mitigation: add restrictive Deno permission profile and dedicated tests.
4. Operational complexity with external runtime:
   - Mitigation: clear env controls and staged rollout.

## Open Questions

1. Should TypeScript component manifests/source support hot reload in development?
2. Should Deno runtime be per-execution process or pooled model?
3. What default permission policy should be enforced for network/filesystem/env?
4. When should trigger/action TypeScript support be included in scope?
