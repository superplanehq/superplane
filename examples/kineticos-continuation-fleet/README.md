# KineticOS

**CAD image of an assembly with a broken component in → modified CAD file out, so production keeps running.**

<p align="center">
  <a href="http://localhost:3001">
    <img alt="Open in SuperPlane" src="https://img.shields.io/badge/%E2%96%B6%20Open%20in%20SuperPlane-6E56CF?style=for-the-badge&labelColor=2D2A45" height="40">
  </a>
</p>

> ☝️ Opens the **SuperPlane app** — the canvas with the full agent fleet (every
> perception, design, printability, and validation agent + the gates between
> them), hosted locally on **http://localhost:3001** (KineticOS keeps :3000).
> To bring it up from scratch — no Docker Desktop needed:
>
> ```bash
> # 1. a headless container runtime
> brew install colima docker && colima start --vm-type vz --vz-rosetta
> # 2. host the SuperPlane app on :3001
> docker run -d --name superplane -p 3001:3000 \
>   -e BASE_URL=http://localhost:3001 -e ALLOWED_WS_ORIGINS=http://localhost:3001 \
>   -v spdata:/app/data ghcr.io/superplanehq/superplane-demo:stable
> # 3. point the fleet canvas at KineticOS and load it — host.docker.internal
> #    reaches the host from inside the SuperPlane container
> infra/superplane/apply.sh http://host.docker.internal:3000   # or import the YAML in the UI
> ```
>
> See [`infra/superplane/`](infra/superplane/) for the canvas + the full walkthrough.

KineticOS ingests a CAD image of a mechanical assembly that has a broken or
missing component, locates the break, **reconstructs the intended undamaged
geometry** (never the cracked artifact), picks a **continuation strategy** —
an off-the-shelf substitute, a 3D-printable insert, a small bridge adapter, or
a simplified-geometry version — re-parameterizes the design against locally
loaded stock, then spins up a **dedicated, ephemeral cloud validator per job**
and hands back a **v1 CAD file** the operator can save, slice, and use to keep
the line running until the OEM replacement arrives.

> The CAD output itself is **v1** — an OpenSCAD-style script that captures the
> mating interfaces and the chosen strategy. The surrounding system (gates,
> per-job ephemeral runtime, audit trail, multi-plane orchestration) is fit to
> the final vision so v1 can be swapped for a real CAD-kernel output later
> without disturbing anything else.

Two load-bearing constraints, kept strictly separate:

> **Render runs things. Superplane decides when and whether things run.**

## Render — four ways

1. **Web service** — `kineticos` hosts the Next.js app (UI + API + `/healthz`).
   See [`infra/render/render.yaml`](infra/render/render.yaml).
2. **Per-job ephemeral service** — `cad-validator-{jobId}` is born and killed
   by [`src/lib/render/client.ts`](src/lib/render/client.ts) for every job.
   Receives the v1 CAD output, runs the canary→bulk validator passes, streams
   progress frames back. No shared multi-tenant endpoint — one job, one service.
3. **Healthcheck probe** — Render polls [`/healthz`](src/app/healthz/route.ts)
   to drive its readiness gate.
4. **Managed Postgres** — `kineticos-pg` backs the durable Job + audit store
   when `DATABASE_URL` is set ([`src/lib/db/**`](src/lib/db/)).

## Superplane — four ways

1. **Workflow run record** — every Job starts a Superplane run; the canvas
   URL is stamped on `Job.audit.superplaneRunId`
   ([`src/lib/superplane/client.ts`](src/lib/superplane/client.ts) ·
   `startRun`).
2. **Step transitions** — every status change is emitted as a step event
   (`emit`) so the run record is a live timeline of the pipeline.
3. **Gates** — four distinct gates (`composite_confidence` 2.G,
   `continuation_strategy` 3.G, `printability` 4.3, `output_acceptance` 8.1)
   live as pure policy functions in [`src/lib/gates/`](src/lib/gates/) and
   recorded through `recordGate`. Human-scoped gates park the job in the
   **Gates** tab; the rest pass through.
4. **Audit fan-in** — gate decisions are mirrored into the run record, giving
   one immutable provenance trail across every plane — the liability record
   the AuditTab renders.

### The agent fleet (Canvas)

[`infra/superplane/kineticos-fleet.canvas.yaml`](infra/superplane/kineticos-fleet.canvas.yaml)
is the **whole process as one SuperPlane Canvas** — 34 nodes that fan the six
perception agents out in parallel, fuse them, branch the two design tracks, and
gate between every phase (`http evaluate-gate → if → approval`) before the line
resumes production. Each action node is an `http` executor calling one KineticOS
agent at [`/api/agents/*`](src/app/api/agents) (roster: `GET /api/agents`); it
runs at zero credentials. See [`infra/superplane/README.md`](infra/superplane/README.md)
to load it, and `infra/superplane/smoke.sh` to rehearse a full run offline.

## Runs at zero credentials

Every integration is optional. With **no** environment variables the full
pipeline runs end-to-end on deterministic fallbacks: Claude → rule-based
stages, Superplane → an in-process control-plane model, Render → a simulated
provision/teardown lifecycle, the validator → simulated telemetry, Postgres →
an in-process `Map`. Each key upgrades exactly one plane from simulated to
real.

## Quickstart

```bash
npm install
npm run dev          # http://localhost:3000
```

Open the app → **Intake** tab → drop a CAD image of the broken assembly → add
an optional assembly context / failure note → **Ingest CAD image**, then watch
the **Pipeline** tab drive the job through perception → continuation design →
printability → provisioning → CAD output → validation, with Superplane gates
in between. The **Continuation CAD output (v1)** panel shows the emitted
OpenSCAD file with a download link. Low-confidence jobs pause in the
**Gates** tab; the full provenance trail is in **Audit**.

Optional — turn planes real:

```bash
cp .env.example .env.local
# ANTHROPIC_API_KEY   → Claude for the LLM-bearing stages
# SUPERPLANE_API_KEY  → the real Superplane control plane
# RENDER_API_KEY      → real ephemeral per-job cad-validator provisioning
# DATABASE_URL        → durable jobs + audit trail (npm run db:migrate && npm run db:seed)
```

## Architecture

The single Next.js app maps the plan's four planes and every gate onto modules;
see **[ARCHITECTURE.md](ARCHITECTURE.md)** for the full Phase-0–9 → file map.
The canonical `Job` object in [`src/lib/types.ts`](src/lib/types.ts) is the
spine everything keys off — including the new `ContinuationStrategy` enum and
the `ContinuationCadOutput` v1 deliverable.
[`infra/render/render.yaml`](infra/render/render.yaml) shows the production
Render split.

## Status

Scaffold + architecture, aligned to the CAD-continuation vision. The pipeline,
gates, dual-layer store, Render/Superplane stub interfaces, live UI, and v1
CAD output emitter run end-to-end. The v1 CAD output is intentionally a
placeholder (OpenSCAD script) — the surrounding system is built to accept a
real CAD-kernel STEP/STL output without changing any other module.
