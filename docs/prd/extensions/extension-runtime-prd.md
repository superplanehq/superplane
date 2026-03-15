# Extension Runtime PRD

## Status

Draft

## Summary

Build a multi-tenant extension runtime for a SaaS application that allows customers to publish and install custom extensions. Extensions provide components and logic for interacting with third-party DevOps services such as GitHub, AWS, Google Cloud, DigitalOcean, and Datadog.

The runtime must prioritize isolation over general-purpose Node compatibility:

- extensions must not be able to interact with the SaaS application's cluster or internal services
- one customer's extension must not be able to interact with another customer's extension
- the system must support pluggable execution backends
- the first implementation should work well on Google Cloud, but the design must remain portable to self-hosted and other cloud environments

## Problem

Running arbitrary TypeScript in `node:worker_threads` is not a sufficient security boundary for hostile or semi-trusted customer code. The platform needs a stronger execution boundary and a narrower capability model.

At the same time, extension authors still need a practical developer experience, including the ability to use compatible npm packages and interact with common DevOps APIs.

## Goals

- Support customer-authored extensions written in TypeScript.
- Keep the engine as a control plane, not the trust boundary for untrusted code execution.
- Use a capability-based runtime model instead of giving extensions general Node host access.
- Support warm extension sessions with idle shutdown where the backend allows it.
- Allow multiple execution backends behind a common engine contract.
- Start with a backend model that fits Google Cloud well.
- Preserve a path to a future custom backend for cost and control reasons.

## Non-Goals

- Full Node.js compatibility for arbitrary code.
- Arbitrary filesystem, process, or raw network access from extensions.
- Per-extension custom container images as the default packaging model.
- Locking the platform to a single managed provider.

## Threat Model

The platform must assume extension code may be malicious or buggy.

Security invariants:

- extensions do not receive cluster credentials
- extensions do not get direct access to the Kubernetes API or internal control-plane services
- extension runtimes are scoped to a single tenant
- extension runtime state is never shared across tenants
- outbound access is explicitly granted and auditable
- secrets are brokered through platform APIs rather than ambient environment access

## Product Model

Extensions are packaged bundles with:

- manifest metadata
- a runtime profile

Extensions are installed per tenant and executed in tenant-scoped runtime sessions.

## Runtime Model

### Chosen Direction

Use a narrow capability model.

Extensions are written in TypeScript but do not run as arbitrary Node programs. They run against a restricted API surface supplied by the platform.

### Allowed

- bundled TypeScript or JavaScript
- pure JS npm packages that work within the supported runtime profile
- web-standard APIs required by the runtime profile
- platform-provided context objects for storage, logging, secrets, integrations, and outbound HTTP

### Not Allowed

- `fs`
- `child_process`
- raw TCP or UDP sockets
- native addons
- unrestricted environment variables
- arbitrary outbound network access
- runtime package installation

### Important Clarification

npm packages are allowed, but not with ambient Node authority. Packages must be compatible with the platform runtime profile and permission model.

## Runtime Profile

Baseline runtime profile: `portable-web-v1`

Expected characteristics:

- ES modules
- `fetch`, `Request`, `Response`, `Headers`, `URL`
- `crypto.subtle`
- timers
- `TextEncoder` and `TextDecoder`
- no filesystem or subprocess primitives

This profile is intentionally narrower than Node so that it can map to multiple backends.

## Extension SDK Shape

The SDK should revolve around:

- `metadata`
- `integrations`
- `components`
- `triggers`
- engine-provided runtime context

Conceptually:

```ts
defineExtension({
  metadata: { ... },
  integrations: [githubIntegration],
  components: [createIssueComponent],
  triggers: [issueOpenedTrigger],
});
```

## Engine / Backend Architecture

### Control Plane

The engine runs in the SaaS application environment and is responsible for:

- validating bundles
- storing extension artifacts
- managing installation records
- managing tenant metadata
- brokering commands to running extension sessions
- auditing execution and access

### Execution Plane

Untrusted code runs outside the engine in a pluggable backend.

### Communication Model

Primary model:

- the sandboxed extension runner opens an outbound control channel to the engine broker
- the engine sends commands over that channel
- results, logs, heartbeats, and lifecycle events flow back on the same channel

Reasons:

- avoids inbound access to sandbox instances
- works well for GKE custom and Cloud Run
- keeps the engine contract stable across backends
- reduces direct dependency on Pod or container network topology

Backends that cannot hold a long-lived channel may emulate the same logical protocol using request-response transport.

## Artifact Model

Current preferred artifact model:

1. use a shared prebuilt runner image
2. store extension bundles in object storage
3. start a generic runner with a bundle reference and metadata
4. runner downloads the bundle, verifies its digest, and starts it

This is preferred over building one image per extension version.

Implications:

- artifact storage is an engine concern
- the base backend contract does not require a mandatory `publish()` step
- some providers may still need an optional preparation step later

## Sandbox Provider Interface

Current direction for the base provider contract:

```ts
interface SandboxProvider {
  startSession(input: StartSessionInput): Promise<StartedSession>;
  invoke(input: InvokeInput): Promise<InvokeResult>;
  stopSession(input: StopSessionInput): Promise<void>;
  getSession(input: GetSessionInput): Promise<SessionState | null>;
  streamLogs(input: StreamLogsInput): AsyncIterable<LogEvent>;
}
```

Notes:

- the engine owns bundle validation and artifact storage
- `startSession()` receives an immutable extension reference plus bundle location and runtime metadata
- warm or cold lifecycle is backend-specific
- a provider-specific prepare step may be added later for upload-oriented backends

## Session Model

Preferred model:

- one warm extension session per tenant-scoped runtime instance
- session reused for multiple invocations
- session shut down after idle timeout

The engine contract must not assume a provider can keep a session alive forever.

## Backend Strategy

### Initial Direction

Start with a Cloud Run-compatible model while keeping the contract portable.

### Custom Backend Direction

Keep a future custom backend on Kubernetes in scope for cost and control.

## Provider Options Considered

### Google Cloud Run

Status:

- preferred first managed backend

Why:

- operationally close to the current Google Cloud deployment model
- works well with the shared runner image plus object storage artifact model
- supports warm instances and idle shutdown patterns
- provides a stronger isolation boundary than in-process Node workers

Tradeoffs:

- does not itself provide the capability model
- control channels are still subject to request timeout behavior
- may become expensive at scale

### Custom GKE Backend

Status:

- explicitly in scope as a future backend

Proposed model:

- one warm extension session per Pod
- shared runner image
- object storage artifact download at startup
- Pod opens an outbound control channel to the engine broker
- Pod deleted on idle timeout

Security direction:

- tenant-scoped sessions
- GKE Sandbox / hardened Pod runtime where possible
- strict network policy
- no direct ingress into extension Pods

Tradeoffs:

- more operational responsibility
- isolation quality depends on correct cluster hardening

### AWS EKS / Custom AWS Backend

Status:

- keep on the provider list

Likely model:

- similar to custom GKE
- one warm extension session per Pod
- outbound control channel back to the engine

Potential deployment variants:

- EKS on Fargate for stronger managed Pod isolation
- EKS on EC2 with dedicated nodes for more control

Tradeoffs:

- more operational complexity than a fully managed backend
- stronger isolation may require higher cost or more dedicated infrastructure

### Cloudflare Workers for Platforms

Status:

- keep on the provider list

Why it is attractive:

- very strong fit for narrow capability runtimes
- strong managed isolation story for untrusted code

Tradeoffs:

- less natural fit for the shared runner image plus external artifact fetch model
- persistent outbound control channels are not the natural execution model
- portability requires staying close to a web-standard runtime profile

Implication:

- likely needs a provider-specific adapter rather than the exact same lifecycle as Cloud Run or Kubernetes-based backends

### Deno Sandbox

Status:

- keep on the provider list

Why it is attractive:

- explicit microVM-oriented sandboxing model for untrusted code
- potentially good fit for a capability runtime

Tradeoffs:

- still requires the platform to define its own SDK and permission model
- may need provider-specific integration work

### Deno Deploy

Status:

- considered, but currently not a preferred backend

Reason:

- does not appear to align as well with the desired capability-restricted model
- not currently the preferred managed option for untrusted customer extensions

### AWS Lambda

Status:

- keep on the long-term consideration list, but not preferred

Reason:

- not a natural fit for warm outbound control channels
- better suited to request-response execution than resident worker sessions

## Decisions Made So Far

- Do not rely on `node:worker_threads` as the security boundary.
- Move to a capability-based runtime model.
- Allow npm packages only within the restricted runtime profile.
- Prefer a shared runner image plus object storage artifact model.
- Keep the engine as control plane only.
- Use an outbound control channel from the sandbox runner to the engine broker.
- Make the execution backend pluggable.
- Start from a Cloud Run-compatible design.
- Keep custom Kubernetes-based execution in scope for future cost and control.
- Pass a shared engine-provided runtime context to integrations, components, and triggers.

## Open Questions

- How the engine should populate the runtime context for real executions.
- Whether some provider interactions should use brokered platform integrations instead of raw HTTP.
- How much compatibility with official cloud SDKs should be targeted in `portable-web-v1`.
- Whether the first production backend should be Cloud Run only or Cloud Run plus custom GKE in parallel.

## Near-Term Next Steps

1. Implement real engine-side population of the shared runtime context.
2. Add engine-level dispatch tests for the GitHub reference extension.
3. Improve the user-facing CLI/API shape so invocations are addressed by block type, block name, and operation.
4. Start the sandbox-provider abstraction, with Cloud Run as the first target backend.
