# KineticOS — Architecture (review map)

This document maps the execution plan (Phases 0–9) onto the actual files, so a
reviewer can navigate the scaffold quickly. The build is a **single Next.js 15
app** (the proven shape for these hackathons) rather than the plan's literal
multi-service monorepo — but every plane and gate from the plan is present as a
module, and [`infra/render/render.yaml`](infra/render/render.yaml) shows how the
single app fans out into the plan's separate Render services for production.

## Vision (v2)

> CAD image of an assembly with a broken component → modified CAD file out, so
> production keeps running temporarily.

Input is a CAD image (a render/screenshot of the assembly with a broken or
missing component visible in situ). Output is a `ContinuationCadOutput` — a v1
OpenSCAD-style CAD file, a small BOM, and a textual operator runbook that
together let the line keep moving until the proper OEM replacement arrives.

The system chooses between four continuation strategies:

| Strategy | When |
|---|---|
| `substitute_component` | An off-the-shelf or community-cataloged part fits the surrounding assembly (Track A hit). |
| `printed_insert` | The default for unique geometry: a 3D-printable substitute that respects the assembly's mating interfaces. |
| `bridge_adapter` | A small printed adapter so an available in-stock standard fastener can stand in. |
| `simplified_geometry` | Strip non-load features so in-house tooling can produce the part today. |

## The load-bearing split

> **Render runs things. Superplane decides when and whether things run.**

| Plane | Module | Responsibility |
|---|---|---|
| Perception | `src/agents/perception/**` | CAD image → broken-component class → undamaged intent → mating interfaces |
| Continuation Design | `src/agents/design/**` | strategy choice → sourced substitute OR generated insert/adapter |
| Printability | `src/agents/material-adapter.ts` | re-parameterize against locally-loaded stock; printability margin |
| Orchestration | `src/lib/superplane/client.ts` + `src/lib/gates/` + `src/worker/jobs.ts` | workflow state, gates, fan-in, audit |
| Per-Job Runtime | `src/lib/render/client.ts` | births/kills the dedicated ephemeral `cad-validator-{jobId}` service |
| CAD Output | `src/agents/fabrication.ts` | emits the v1 `ContinuationCadOutput` + runs canary/bulk validator |
| Edge / Machine | `src/lib/edge/client.ts` | simulated on-prem agent: inventory + validator telemetry |

Both sponsor integrations are **clean stubs that run at zero credentials** and
upgrade to real APIs behind a flag (`SUPERPLANE_API_KEY`, `RENDER_API_KEY`).

## How Render is used (4 ways)

1. **Web service `kineticos`** — declared in [`render.yaml`](infra/render/render.yaml).
2. **Per-job ephemeral `cad-validator-{jobId}`** — created at runtime by
   [`src/lib/render/client.ts → provisionRuntime`](src/lib/render/client.ts);
   deleted on terminal status by `teardownRuntime`. The orphan-reaper cron
   (commented in `render.yaml`) reconciles via `listEphemeralServices`.
3. **`/healthz`** — Render's healthcheck probe target
   ([`src/app/healthz/route.ts`](src/app/healthz/route.ts)).
4. **Managed Postgres `kineticos-pg`** — `DATABASE_URL` injected from the
   managed DB into the web service.

## How Superplane is used (4 ways)

1. **`startRun`** — a Superplane run is opened on job ingest; the run id and
   canvas URL are stamped on `Job.audit.superplaneRunId`.
2. **`emit`** — every JobStatus transition is emitted as a step event, giving
   the run record a live timeline of the pipeline.
3. **`recordGate`** — every gate decision (proceed / human_review / block) is
   recorded into the run record alongside the local audit trail.
4. **Workflow gating** — the four gates (`composite_confidence`,
   `continuation_strategy`, `printability`, `output_acceptance`) live as pure
   policy functions in [`src/lib/gates/`](src/lib/gates/) and the worker
   pauses the job (`status: needs_human`, `pendingGate` set) when one routes
   to a human.
5. **The agent fleet (Canvas)** — [`infra/superplane/kineticos-fleet.canvas.yaml`](infra/superplane/kineticos-fleet.canvas.yaml)
   is the entire pipeline as a 34-node SuperPlane Canvas: the six perception
   agents fan out in parallel from the ingest webhook, a `merge` fuses them, the
   two design tracks branch on a sourcing hit, and each phase ends in the
   `http evaluate-gate → if → approval` gate pattern before the line resumes.
   Every action node is an `http` executor hitting one agent endpoint —
   granular [`/api/agents/[agent]`](src/app/api/agents) for the perception
   sensors + design tracks, coarse [`/api/stages/*`](src/app/api/stages) for
   phases 4 and 5–7. The in-process worker and the Canvas are two drivers of the
   same agents + gates (`LOCAL_ORCHESTRATOR=1` selects the worker).

## The spine

[`src/lib/types.ts`](src/lib/types.ts) — the canonical `Job` object every stage
keys off (Phase 0.3), plus the Phase 2/3/4/8 output contracts, the gate types,
the new `ContinuationStrategy` enum, the v1 `ContinuationCadOutput` type, and
the **locked stage signatures** the worker imports and each stage module
implements.

## The pipeline (worker drives it, upserting the Job after every step)

`src/worker/jobs.ts` runs stages in order and evaluates a Superplane gate
between phases; a human-scoped gate pauses the job (`needs_human`) and resumes
on `resolveGate()`.

| Phase | Stage | File |
|---|---|---|
| 2A | CAD image conditioning / admissibility | `src/agents/perception/conditioning.ts` |
| 2B | Broken-component localization *(exemplar)* | `src/agents/perception/classification.ts` |
| 2C | Reconstruct the **undamaged intent** | `src/agents/perception/reconstruction.ts` |
| 2D | Interface extraction (terminal scale chain) | `src/agents/perception/dimensioning.ts` |
| 2E | Material & surface inference | `src/agents/perception/material.ts` |
| 2F | Telemetry track + sensor fusion | `src/agents/perception/telemetry.ts` |
| **2.G** | **Composite confidence gate** | `src/lib/gates/index.ts` → `compositeConfidenceGate` |
| 3A | Substitute sourcing (Track A) | `src/agents/design/sourcing.ts` |
| 3B | Generative continuation CAD B1–B8 (Track B) | `src/agents/design/generative-cad.ts` |
| **3.G** | **Continuation strategy gate** | `src/lib/gates/index.ts` → `designAcceptanceGate` |
| 4 | Printability adaptation + toolpath | `src/agents/material-adapter.ts` |
| **4.3** | **Printability feasibility gate** | `src/lib/gates/index.ts` → `structuralGate` |
| 5 | Provision dedicated cad-validator service | `src/lib/render/client.ts` → `provisionRuntime` |
| 6/7 | Emit v1 ContinuationCadOutput + canary/bulk validator | `src/agents/fabrication.ts` |
| 8 | Output validation | `src/agents/fabrication.ts` |
| **8.1** | **Output acceptance gate** → COMPLETE | `src/lib/gates/index.ts` → `acceptanceGate` |
| 8.2 | Teardown dedicated runtime | `src/lib/render/client.ts` → `teardownRuntime` |
| 8.3 | Audit seal | `src/lib/audit.ts` + `Job.audit` |

## Data & API

- `src/lib/store.ts` — dual-layer job store: globalThis `Map` always-on, Postgres
  write-through when `DATABASE_URL` is set. `src/lib/db/**` is the pg layer.
- `src/app/api/jobs/route.ts` — `POST` ingest (422 if neither `cadImageUris`
  nor `telemetryUri`), `GET` list.
- `src/app/api/jobs/[id]/route.ts` — `GET` one (polling target).
- `src/app/api/jobs/[id]/gate/route.ts` — `POST` resolve a scoped human gate.
- `src/app/healthz/route.ts` — Render health check target.

## UI (live polling, dark theme)

`src/app/page.tsx` tab shell → `src/components/tabs/{IntakeTab,PipelineTab,GatesTab,AuditTab}.tsx`,
all built on `src/components/ui.tsx` primitives and `src/lib/usePolling.ts`.
The PipelineTab carries the new **Continuation CAD output (v1)** panel that
renders the OpenSCAD payload + BOM + runbook with a `data:` download link.

## V1 CAD output — what's intentionally rough

The v1 `ContinuationCadOutput` payload is a plain-text OpenSCAD script. It
captures the resolved mating interfaces (bore, plate footprint, tooth count)
but the non-load features are simplified for fast 3D printing — gear teeth are
square notches, not involute curves; brackets are flat plates with corner
holes; everything else is a bored block. A v2 swap to a real CAD-kernel
B-rep through the `cad-validator-{jobId}` service is a drop-in replacement of
the `buildOpenscadScript` helper in `src/agents/fabrication.ts` — the
`ContinuationCadOutput` contract (`format`, `filename`, `contents`, `bom`,
`runbook`) is the same, and every other plane already keys off that shape.

## Not yet built (Phase 9 hardening — deliberately deferred for review)

- Orphan-reaper cron logic (the primitive `render.listEphemeralServices()`
  exists; the reconcile job/endpoint is a TODO in `render.yaml`).
- 2A→3A bounded reconstruction refinement loop (single-pass today).
- Real per-job mTLS / signing-key verification on the edge channel (key is
  generated and passed; verification is simulated).
- Real CAD kernel for the v2 continuation output (drop-in behind
  `buildOpenscadScript`).
