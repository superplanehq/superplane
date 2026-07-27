# Success Metrics

## North Star Outcome

**Verified product intent:** Users can turn complex, multi-tool engineering
processes into durable, guarded SuperPlane apps that humans and AI can operate
safely. ([README.md:3-7](../../README.md#L3-L7),
[README.md:21-28](../../README.md#L21-L28))

**Inferred north-star metric:** Share of target processes that reach a first
successful live run and remain actively operated through SuperPlane rather than
ad hoc scripts. The repository does not define a single official north-star
KPI, baseline, or target.

## Product Metrics

The metrics below combine explicitly proposed PRD metrics with metrics that can
be computed from existing product data. Unless noted, baselines and targets are
unknown in this repository.

### Time to first valid workflow

- **Definition:** Median elapsed time from new canvas creation to first
  committed workflow that produces a successful run.
- **Rationale:** Directly measures whether builders overcome configuration and
  wiring friction.
- **Baseline:** Unknown.
- **Target:** Unresolved.
- **Data source:** Canvas creation timestamps, version commits, and run results
  in PostgreSQL / run APIs.
- **Evidence:** Proposed in
  [docs/prd/ai-canvas-builder-sidebar.md:163-169](../../docs/prd/ai-canvas-builder-sidebar.md#L163-L169);
  run and version surfaces exist in
  [protos/canvases.proto:144-174](../../protos/canvases.proto#L144-L174) and
  [protos/canvases.proto:292-311](../../protos/canvases.proto#L292-L311).

### Successful run rate

- **Definition:** Percentage of finished runs with result `passed`, segmented by
  app, trigger, and version.
- **Rationale:** Indicates whether published workflows reliably complete.
- **Baseline:** Unknown.
- **Target:** Unresolved.
- **Data source:** `ListRuns` / `CanvasRun.Result`
  ([protos/canvases.proto:872-901](../../protos/canvases.proto#L872-L901)).

### Human-gated completion latency

- **Definition:** Time from approval or wait execution start to downstream
  continuation or cancellation.
- **Rationale:** Measures whether guarded processes move forward without
  becoming stuck.
- **Baseline:** Unknown.
- **Target:** Unresolved.
- **Data source:** Execution timestamps and approval metadata covered by
  [test/e2e/approvals_test.go:41-48](../../test/e2e/approvals_test.go#L41-L48)
  and [test/e2e/wait_test.go:46-53](../../test/e2e/wait_test.go#L46-L53).

### Agent-assisted edit adoption

- **Definition:** Percentage of canvases edited with agent chat or field
  suggest; proposal/apply success where applicable.
- **Rationale:** Indicates whether AI assistance reduces authoring friction
  without bypassing review.
- **Baseline:** Unknown.
- **Target:** Unresolved.
- **Data source:** Agent chat APIs and org AI feature gates
  ([protos/agents.proto:27-71](../../protos/agents.proto#L27-L71),
  [docs/prd/ai-canvas-builder-sidebar.md:163-169](../../docs/prd/ai-canvas-builder-sidebar.md#L163-L169)).

### Active operated apps

- **Definition:** Count of apps with at least one run, Console view, or
  authorized action in a rolling period.
- **Rationale:** Distinguishes authored prototypes from apps that become
  operational surfaces.
- **Baseline:** Unknown.
- **Target:** Unresolved.
- **Data source:** Runs, Console updates, and UI usage signals; Console
  behavior is documented in
  [docs/prd/console-and-widgets.md:15-23](../../docs/prd/console-and-widgets.md#L15-L23).

## Guardrail Metrics

| Guardrail | Why it matters | Measurement approach |
| --- | --- | --- |
| Failed or cancelled run rate | Prevents “more automation” from hiding broken flows | Run results by app/version |
| Permission denial / unauthorized action rate | Ensures RBAC and Console action gates remain effective | Auth interceptor and UI permission tests |
| Secret or sensitive-value exposure incidents | Protects credentials in configs, prompts, and logs | Security review and redaction policy adherence |
| Agent suggestion apply failure / discard rate | Detects low-quality AI assistance | Agent proposal outcomes where instrumented |
| Staging stale/discard rate | Indicates collaboration friction in stage/commit | Staging APIs and E2E staging flows |
| Event retention / storage growth | Prevents unbounded history and memory growth | Retention workers and memory cleanup patterns |

Evidence:
[test/e2e/canvas_permission_guards_test.go:17-48](../../test/e2e/canvas_permission_guards_test.go#L17-L48),
[docs/prd/inline-config-assistant.md:177-181](../../docs/prd/inline-config-assistant.md#L177-L181),
[docs/prd/canvas-memory.md:137-146](../../docs/prd/canvas-memory.md#L137-L146),
[pkg/workers/event_retention_worker_test.go](../../pkg/workers/event_retention_worker_test.go).

## Open Questions

- What official north-star metric and time horizon should product leadership
  adopt?
- What are current baselines for time-to-first-run, successful run rate, and
  active apps in self-hosted versus cloud deployments?
- Which metrics are already instrumented in production telemetry versus only
  inferable from database records?
- How should success differ for builders, operators, and installation admins?
- What customer retention, expansion, or revenue metrics matter beyond product
  usage? The repository does not contain them.
