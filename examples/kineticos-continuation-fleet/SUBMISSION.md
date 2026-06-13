# KineticOS — a SuperPlane agent-fleet example

A self-contained snapshot of **KineticOS**, a project built **on top of
SuperPlane**: a CAD image of a broken industrial machine comes in → a fleet of
agents looks at it, identifies the fault, reconstructs the intended part,
generates a new **3D-printable CAD file**, validates it, and **resumes
production** — with SuperPlane gating every phase.

> **Heads-up for maintainers:** this is a *downstream application* example, not a
> change to the SuperPlane platform. It's contributed as an `examples/` folder so
> it adds nothing to your build/CI and touches none of your tree. If you'd prefer
> only the workflow template, the single file worth upstreaming is the Canvas
> below — it can be dropped into `templates/canvases/` with `metadata.isTemplate:
> true` per `docs/contributing/templates.md`. Happy to reshape this PR to just
> that if you'd rather.

## The primary artifact — the Canvas

[`infra/superplane/kineticos-fleet.canvas.yaml`](infra/superplane/kineticos-fleet.canvas.yaml)
is a 34-node Canvas exercising a lot of SuperPlane in one real workflow:

- **Parallel fan-out + `merge`** — six perception agents run concurrently from
  the ingest trigger, then a `merge` barrier fuses them.
- **The gate pattern (×4)** — each phase ends in `http` (evaluate-gate) → `if`
  (`decision == "proceed"`) → `approval` (the human gate), with rejections
  routed to a halt sink.
- **Conditional branching** — an `if` picks an off-the-shelf substitute vs. a
  generated continuation part.
- **`http` executors** call the app's agent endpoints; **annotations** document
  each section on-canvas.

`infra/superplane/kineticos-fleet.local.canvas.yaml` is the same Canvas with
executor URLs pointed at `host.docker.internal:3000` for a locally-hosted
SuperPlane (container) calling the app (host).

## The rest of the folder (the downstream app)

- `src/app/api/agents/**` + `src/lib/contract.ts` — the Next.js HTTP executors
  each Canvas node calls (one agent per endpoint). **Illustrative** — this is
  the KineticOS app, not SuperPlane platform code; it isn't built by your CI.
- `infra/superplane/{README.md,apply.sh,smoke.sh}` — node→agent→endpoint map,
  a canvas loader (host substitution + `superplane canvas create`), and an
  offline end-to-end fleet rehearsal.
- `README.md` / `ARCHITECTURE.md` — the KineticOS project docs for context.

## Run it

```bash
# the app (executors) — runs at zero credentials
npm install && npm run dev                               # KineticOS on :3000

# SuperPlane locally (no Docker Desktop needed)
brew install colima docker && colima start --vm-type vz --vz-rosetta
docker run -d --name superplane -p 3001:3000 \
  -e BASE_URL=http://localhost:3001 -e ALLOWED_WS_ORIGINS=http://localhost:3001 \
  -v spdata:/app/data ghcr.io/superplanehq/superplane-demo:stable

# load the fleet (host.docker.internal reaches the host from the container)
infra/superplane/apply.sh http://host.docker.internal:3000   # then import in the UI
```

Full project: https://github.com/sahielbose/KineticOS-Superplane-Hackathon
