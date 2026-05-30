# KineticOS — the SuperPlane agent fleet

> **A CAD image of a broken industrial machine comes in → a new, 3D-printable
> CAD file goes out, so the line keeps running.**

<p align="center">
  <a href="http://localhost:3001">
    <img alt="Open in SuperPlane" src="https://img.shields.io/badge/%E2%96%B6%20Open%20in%20SuperPlane-6E56CF?style=for-the-badge&labelColor=2D2A45" height="40">
  </a>
</p>

> SuperPlane is hosted locally on **:3001** (KineticOS keeps :3000). Bring it up
> with Colima — no Docker Desktop needed — then load this fleet:
> ```bash
> brew install colima docker && colima start --vm-type vz --vz-rosetta
> docker run -d --name superplane -p 3001:3000 \
>   -e BASE_URL=http://localhost:3001 -e ALLOWED_WS_ORIGINS=http://localhost:3001 \
>   -v spdata:/app/data ghcr.io/superplanehq/superplane-demo:stable
> infra/superplane/apply.sh http://host.docker.internal:3000   # SuperPlane-in-container → KineticOS-on-host
> ```

This directory holds the **agent fleet as a SuperPlane Canvas** —
[`kineticos-fleet.canvas.yaml`](kineticos-fleet.canvas.yaml) — that orchestrates
the *exact* KineticOS process end to end:

```
look at the machine  →  identify what's wrong  →  reconstruct the intended part
   →  generate a new 3D-printable CAD file  →  prove it holds  →  resume production
```

SuperPlane is the **brain**: it decides *when and whether* each agent runs, fans
the perception sensors out in parallel, branches the two design tracks, and parks
the job at a **human gate** the moment confidence, the design trail, the
printability margin, or the output validation falls short. The compute itself
runs in the KineticOS Next.js app (and Render's per-job validator); SuperPlane
never executes domain logic — it routes.

Every `TYPE_ACTION` node is an **`http` executor** that POSTs `{ "job_id": … }`
to one KineticOS endpoint. The whole fleet runs **at zero credentials**: each
agent has a deterministic fallback, so you can demo the full canvas offline.

---

## The fleet at a glance

`34 nodes` · `1 trigger` · `28 agent/gate/control nodes` · `5 doc widgets` · `40 edges`

| # | Node | Component | Calls | What the agent does |
|---|------|-----------|-------|---------------------|
| — | Ingest: broken-machine CAD image | `webhook` | *(event source)* | Fires when `POST /api/jobs` posts to `SP_INGEST_WEBHOOK` |
| 2A | Conditioning | `http` | `/api/agents/conditioning` | Is the CAD image admissible (sharp, exposed, on-part)? |
| 2B | Classification | `http` | `/api/agents/classification` | **Which** component is broken (drives everything downstream) |
| 2C | Reconstruction | `http` | `/api/agents/reconstruction` | Rebuild the *intended undamaged* geometry — needs 2B |
| 2D | Dimensioning | `http` | `/api/agents/dimensioning` | Mating interfaces + a terminating scale chain |
| 2E | Material inference | `http` | `/api/agents/material-infer` | Material class + surface finish |
| 2F | Telemetry fusion | `http` | `/api/agents/telemetry` | Failure mode from telemetry + sensor-fusion agreement |
| — | Fuse the perception sensors | `merge` | — | Barrier: wait for all sensors |
| 2.x | Assemble PerceptionResult | `http` | `/api/agents/perception-assemble` | Compose the composite-confidence score |
| **2.G** | **Gate · composite confidence** | `http` + `if` + `approval` | `/api/evaluate-gate` | Proceed, or scope a human to the weak field |
| 3A | Sourcing | `http` | `/api/agents/sourcing` | Hunt an off-the-shelf / community substitute |
| 3B | Generative CAD | `http` | `/api/agents/generative-cad` | Synthesise the continuation insert via the **B1–B8** trail |
| **3.G** | **Gate · continuation strategy** | `http` + `if` + `approval` | `/api/evaluate-gate` | Generated + load-bearing → human review |
| 4 | Printability adaptation | `http` | `/api/stages/material` | Re-parameterize to locally-loaded stock |
| **4.3** | **Gate · printability feasibility** | `http` + `if` + `approval` | `/api/evaluate-gate` | Margin below duty load → block / relax |
| 5–7 | Provision + emit + validate | `http` | `/api/stages/fabricate-report` | Birth the ephemeral Render `cad-validator-{jobId}`, emit the **v1 CAD output**, run canary→bulk |
| **8.1** | **Gate · output acceptance** | `http` + `if` + `approval` | `/api/evaluate-gate` | Fan-in of dimensional + structural + anomaly checks |
| 8.3 | Seal run + resume production | `http` | `/api/agents/finalize` | Seal the audit trail; mark the line resumed |
| — | ✅ Line resumed / ⛔ Halted | `noop` | — | Terminal sinks |

The fleet roster is also live at **`GET /api/agents`**.

### The gate pattern (used ×4)

Every phase ends in the same three-node shape:

```
 http evaluate-gate ──► if (decision == "proceed") ──true──► next phase
                                     │
                                    false
                                     ▼
                              approval (human gate) ──approved──► next phase
                                     │
                                  rejected
                                     ▼
                                ⛔ Halt
```

The gate policy itself lives once, in
[`src/lib/gates/index.ts`](../../src/lib/gates/index.ts); `/api/evaluate-gate`
returns `{ decision, reason, scoped_field }`. When a gate routes to a human it
names the **exact** weak field (the component class, the reconstruction overlay,
one caliper reading) — never a generic "approve?". The operator resolves it
through the app's existing `POST /api/jobs/{id}/gate` endpoint / **Gates** tab.

---

## Run it

### 0. Start KineticOS (the executors)

```bash
npm install && npm run dev        # http://localhost:3000 — runs at zero credentials
```

Smoke-test the fleet endpoints directly (what the canvas nodes do, in order):

```bash
infra/superplane/smoke.sh http://localhost:3000
```

### 1. Point the canvas at your KineticOS base URL

The YAML ships with `https://kineticos.onrender.com`. To retarget (e.g. local):

```bash
# writes a substituted copy to /tmp and (optionally) applies it.
# SuperPlane runs in a container, so it reaches KineticOS (host :3000) via
# host.docker.internal — use plain localhost:3000 only if SP runs on the host.
infra/superplane/apply.sh http://host.docker.internal:3000
```

### 2. Load the canvas into SuperPlane

**CLI** (preferred — needs the `superplane` CLI authenticated to your org):

```bash
superplane canvas create -f infra/superplane/kineticos-fleet.canvas.yaml
# or let apply.sh do the host-substitution + create in one step:
APPLY=1 infra/superplane/apply.sh https://your-kineticos.example.com
```

**UI:** open SuperPlane → **New canvas → Import YAML** → paste the (substituted)
file. The graph, gates, and annotations render exactly as laid out here.

### 3. Wire the ingest event source

1. In SuperPlane, create a **Webhook event source** and bind it to the
   **Ingest: broken-machine CAD image** trigger node. Copy its URL + signing token.
2. In KineticOS, set:

   ```bash
   SP_INGEST_WEBHOOK=<the event-source URL>
   SP_WEBHOOK_TOKEN=<the signing token>
   # leave LOCAL_ORCHESTRATOR unset so intake fires the webhook instead of the
   # in-process worker (see src/app/api/jobs/route.ts)
   ```

Now every `POST /api/jobs` (an operator dropping a CAD image in the **Intake**
tab) fires the canvas, and SuperPlane drives the whole fleet, calling back into
the agent endpoints and gating between phases.

> **Note on the trigger node.** SuperPlane models inbound webhooks as *event
> sources* that a trigger node binds to in the UI. The node here uses
> `component: "webhook"` as a placeholder — if your SuperPlane build names the
> generic inbound trigger differently, set it on that one node; everything
> downstream is unchanged.

---

## How this maps to the rest of KineticOS

| Concern | Where |
|---|---|
| The agents (Claude + deterministic fallback) | [`src/agents/**`](../../src/agents) |
| The four gate policies | [`src/lib/gates/index.ts`](../../src/lib/gates/index.ts) |
| Granular per-agent executors | [`src/app/api/agents/[agent]/route.ts`](../../src/app/api/agents) |
| Coarse phase executors (4, 5–7) | [`src/app/api/stages/**`](../../src/app/api/stages) |
| Gate evaluator | [`src/app/api/evaluate-gate/route.ts`](../../src/app/api/evaluate-gate/route.ts) |
| Wire contract (snake_case) | [`src/lib/contract.ts`](../../src/lib/contract.ts) |
| In-process equivalent (offline / no canvas) | [`src/worker/jobs.ts`](../../src/worker/jobs.ts) |

The in-process worker and this canvas are **two drivers of the same agents**:
set `LOCAL_ORCHESTRATOR=1` to drive the pipeline in-process (no SuperPlane);
leave it unset to let this fleet drive it. Same agents, same gates, same audit
trail either way.
