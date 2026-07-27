# Business Problem

## Verified Problem

Engineering processes span source control, CI/CD, infrastructure,
observability, incident management, communication, and AI systems. A single
script or CI job is a poor fit when a process must coordinate several of those
systems, retain state across delays or restarts, branch and fan in, require
human approval, and expose its current operational state. SuperPlane exists to
make those processes explicit and executable as versioned workflow graphs
rather than leaving the coordination in disconnected tools and manual
handoffs. ([README.md:3-7](../../README.md#L3-L7),
[README.md:21-35](../../README.md#L21-L35))

The repository verifies specific authoring and operating friction:

- Builders must select components, satisfy configuration schemas, and wire
  compatible channels and payloads; the AI authoring PRD identifies these as
  barriers to a first successful workflow.
  ([docs/prd/ai-canvas-builder-sidebar.md:12-22](../../docs/prd/ai-canvas-builder-sidebar.md#L12-L22))
- Runtime state may need to survive between paths and runs, such as mapping a
  pull request to an ephemeral environment. Without canvas memory, users must
  recompute or carry that data through one path.
  ([docs/prd/canvas-memory.md:12-20](../../docs/prd/canvas-memory.md#L12-L20))
- Automation tied to personal credentials creates lifecycle, attribution, and
  least-privilege problems. The service-account design documents those
  problems directly.
  ([docs/prd/service-accounts.md:3-20](../../docs/prd/service-accounts.md#L3-L20))

## Inferred Impact

The following outcomes are reasonable inferences from the product design, not
measured business facts in this repository:

- Manual coordination increases lead time and creates opportunities for missed
  gates, duplicated actions, and inconsistent recovery.
- Opaque automation makes it harder for builders, operators, approvers, and
  auditors to establish what ran, against which version, with what result.
- Bespoke glue code shifts effort from the engineering outcome to retry,
  persistence, permission, and status-display infrastructure.

The implementation supports these inferences through run and execution
records, cancellable work, approval steps, version history, and run inspection.
([protos/canvases.proto:292-324](../../protos/canvases.proto#L292-L324),
[test/e2e/approvals_test.go:41-60](../../test/e2e/approvals_test.go#L41-L60),
[test/e2e/runs_view_test.go:21-45](../../test/e2e/runs_view_test.go#L21-L45))

## Why Now

**Verified:** SuperPlane is in beta, supports both self-hosted and managed use,
and explicitly targets guardrails for workflows involving humans and AI.
Breaking changes remain possible while core primitives and integrations
mature. ([README.md:11-19](../../README.md#L11-L19))

**Inferred:** As AI agents gain the ability to act on engineering systems, the
cost of implicit permissions and nondeterministic handoffs rises. A durable,
reviewable execution layer can constrain those actions without requiring every
team to build equivalent controls independently.

## Open Questions

- Which coordination failures are most frequent for actual users, and what is
  their current time or incident cost?
- Which initial use cases—delivery, incident response, infrastructure
  operations, or agent orchestration—produce the highest retained usage?
- What evidence demonstrates that users replace existing glue code rather than
  adding another operational layer?
- What customer segments, willingness-to-pay signals, or adoption baselines
  have been validated? The repository does not supply these business facts.
