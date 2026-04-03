# Canvas Run View

## Overview

This PRD defines **Run View**: a dedicated mode on the Canvas page for inspecting **one execution at a time**. Users pick a run from a **Runs Sidebar**; the **Run Canvas** shows only nodes that participated in that run, with per-node data scoped to **that run** (not live “latest only”). **Double-clicking** a node opens a detail surface (layout TBD) with parity to today’s **chain / run item** (metadata, configuration at run time, output/payload).

**Terms**

| Term | Meaning |
|------|---------|
| **Live Canvas** | Live mode graph (current definition + live signaling). |
| **Run View** | Mode on the Canvas page for run-scoped inspection. |
| **Runs Sidebar** | List of runs for this canvas. |
| **Run Canvas** | Graph area in Run View for the **selected** run. |

This is **not** a redesign of the global runs index outside the canvas. It is **not** a replacement for all console workflows until explicitly specified.

## Problem Statement

- **Live Canvas** fits a **control-panel** mental model (one definition, many concurrent signals). It is weaker for users who think in **discrete runs**: *which run am I looking at?*, *what did this node do in that run?*, *is the UI mixing runs?*
- **Observability** is hard: correlating **node state** with **one run’s data** and trusting what is shown.
- **Console Runs** improved discovery but remains a **stopgap** relative to canvas-native, graph-aligned inspection.
- Live emphasizes **now**; node semantics tied to **latest** hurt **historical** and **parallel** run debugging.

## Goals

1. Add **Run View** on the Canvas page alongside **Live Canvas**.
2. Provide a **Runs Sidebar**: all runs for this canvas, **newest first**, **latest run selected** by default.
3. **Run Canvas** shows **only nodes that ran** in the selected run; node chrome reflects **that run’s items** only.
4. **Double-click** a node → detail UI with **chain / run item**-level depth (details, config at run time, output/payload).
5. **Snapshot fidelity:** Run Canvas reflects the **workflow as it existed for that execution**; nodes later removed or changed on live **still render** for old runs with correct config/outputs for that run.

## Non-Goals

- Final **visual design** for the double-click shell (beyond content parity with chain item); mockups tracked separately.
- **Global** runs browser redesign unrelated to this canvas.
- **Large** queue/executor architecture changes beyond what is needed to list runs and bind snapshot data to the Run Canvas.
- **v1:** compare two runs, diff snapshot vs live, export/share run summary (see **Follow-ups**).

## Primary Users

- **Workflow builders / operators** who debug **one execution** end to end.
- Users familiar with **CI run pages**, **workflow runs**, or **trace UIs** who expect **one run = one trace**.

## User Stories

1. As a user, I can open **Run View** from the Canvas and see runs for **this** canvas in the **Runs Sidebar**.
2. As a user, I can select a run and see on the **Run Canvas** only the nodes that **participated** in that run.
3. As a user, I can see per-node run data for the **selected run**, not mixed with “latest on live.”
4. As a user, I can **double-click** a node to inspect details, **configuration at that point in time**, and **output/payload**.
5. As a user, I can inspect an **old** run even if nodes were **deleted or changed** on the Live Canvas afterward; the graph still makes sense for that run.

## Functional Requirements

### Live Canvas vs Run View

| | **Live** | **Run View** |
|---|----------|--------------|
| Purpose | Current definition + live signaling | Inspect **one** execution |
| **Live Canvas** | Full workflow as defined now | — |
| **Run Canvas** | — | Subset + snapshot for **selected run** |
| Node items | What live/latest implies today | **Only** data for **selected run** |

### Entry

- User enters **Run View** from the Canvas page. Exact control aligns with existing mode patterns (for example draft/live).

### Runs Sidebar

- Lists **all runs** for this canvas. Ordering and completeness should align with **console Runs** unless the spec documents an intentional difference.
- **Newest at top**; **most recent run** selected by default.
- New runs appear at the top when created.

### Run Canvas

- Renders **only nodes that ran** in the selected run. Exact definition of **ran** (for example entered, scheduled, completed) is an engineering/product spec detail.
- Run items / badges on nodes are scoped to the **selected run** only (not live latest).

### Node detail (double-click)

- Opens a **modal or panel** (visual design in progress).
- Content matches **chain item / run item** where applicable: status/details, **configuration at time of run**, **output/payload**, and other fields already exposed for deep inspection.

### Snapshot semantics

- **Run View** is a **point-in-time** view of the execution.
- Graph identity and layout (as applicable) come from the **run’s execution context**, not the current **Live Canvas**.
- Nodes **deleted or replaced** after the run **still appear** on the **Run Canvas** for that run; config and outputs **must** match that run, not current live.

### Relationship to console

- **Run View** is the **primary** path to inspect runs **on this canvas** from the Canvas UI.
- Console may remain for **cross-workflow** or legacy cases; the spec should state how duplication is avoided long term.

## Acceptance Criteria

1. User can switch to **Run View** and see the **Runs Sidebar** populated for the current canvas.
2. Selecting a run updates the **Run Canvas** to show **only** nodes that ran in that run (per agreed definition of “ran”).
3. Node-level run UI shows data for the **selected run** only, not live latest-only semantics.
4. Double-click opens a detail surface with **chain / run item** parity for the listed content types.
5. For a historical run, nodes removed or changed on **Live Canvas** after the run still appear on **Run Canvas** with consistent snapshot config/outputs.
6. Internal dogfood: **Run View** is usable as the default way to debug a single run on the canvas without relying on console as the first step.

## Risks and Mitigations

| Risk | Mitigation |
|------|------------|
| Two modes (live vs Run View) confuse users | Clear labels, entry points, and onboarding copy; align with existing mode switching. |
| Snapshot storage/reconstruction is costly or incomplete | Spec persistence/replay model early; treat snapshot correctness as a **hard** requirement for trust. |
| Duplicated UX vs console Runs | Document **source of truth** and deprecation/consolidation plan in the technical spec. |

## Open Questions

1. When a **new** run arrives while the user has another run selected, do we **keep selection**, **notify** (toast/badge), or **auto-switch** to latest?
2. Precise definition of **“node ran”** for filtering the Run Canvas (edge cases: skipped branches, failures, retries).
3. Long-term role of **console Runs** vs **Runs Sidebar** when both exist.

## Follow-ups (post-v1)

- Compare two runs side by side.
- Highlight diff between run snapshot and current **Live Canvas**.
- Export or share a run summary.

## Reference

- Console **Runs** section (baseline to match or supersede).
- **Chain item / run item** UI (content parity for the node detail surface).
- Design mocks (link when ready).
