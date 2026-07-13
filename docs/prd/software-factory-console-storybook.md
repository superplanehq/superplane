# Software Factory Console (Storybook)

**Date:** 2026-07-13  
**Status:** Approved for planning  
**Branch intent:** `feat/software-factory-console-storybook`

## Goal

Add a full **AppPage Console** Storybook story — same pattern as `Pages/AppPage/Console` → SuperPlane SaaS — that illustrates SuperPlane as a **software factory**: engineering throughput with an **AI vs human** center of gravity (spend, authorship, PR quality), plus rich table formats (avatars, progress, trends) and scorecards already on `main`.

## Narrative

**Primary story (chosen):** Engineering throughput / AI vs human.

**Spotlight (chosen default):** Latest merged PR — who shipped it, whether it was AI-assisted or human, title + link, timing, and a check strip. That anchors “what just landed” while the rest of the board answers factory-level questions.

## Approach

**Hybrid (recommended and approved):**

1. Port and register **Spotlight** as a real console panel type so it can appear in `console.yaml` and render inside `AppPageHarness` fixtures.
2. Reuse production **Scorecard**, **Table** (avatars / progress / trend / value+trend), **Number**, and **Chart** panels already on `main`.
3. Express leaderboard-style views as **sorted rich tables** in v1 (do **not** register Ranking as a panel type yet).
4. Ship a hand-crafted **Software Factory** canvas fixture + Storybook story under `Pages/AppPage/Console`.

Do **not** revive the stale prototype scorecard/table implementations from [#6004](https://github.com/superplanehq/superplane/pull/6004); only take Spotlight (widget + content model + editor patterns as needed for product parity with other panels).

## Dashboard layout (v1)

Approximate grid (ids are illustrative; final YAML may adjust sizes):

| Region | Panel type | Purpose |
|--------|------------|---------|
| Top wide | `spotlight` | Latest merged PR (author, AI/human cue, title+href, duration, checks) |
| Top right | 2–3× `scorecard` | e.g. AI-authored PR %, AI spend (7d), active contributors |
| Mid left | `table` | Recent merges — avatar, AI/human badge, checks/progress, cycle-time trend |
| Mid right | `chart` and/or `number` | AI spend vs merged PRs (by person or over time) |
| Bottom | `table` or `chart` | Regular PR metrics — review latency, size, reopen rate, with trend columns |

Optional small markdown/html helper card only if needed for legend (“AI-assisted” definition); prefer self-explanatory panels.

## Spotlight product surface

Spotlight must be a first-class panel, not Storybook-only chrome:

- **FE:** `panelTypes` entry, `ConsolePanelCards` / view routing, content normalize + validate, form editor (slots mirroring prototype `spotlightContent`), widget renderer adapted from prototype `WidgetSpotlight`.
- **BE/YAML:** `pkg/yaml/console.go` validation aligned with FE (same pattern as scorecard).
- **Stories:** keep/refresh `Console/Spotlight` widget stories; editor story optional if form is wired like scorecard.
- **Data:** panel resolves against the top row of its data source (memory / executions / runs), mapping configured field paths / CEL into spotlight slots (actor, title, href, timestamp, duration, approver, checks).

Prototype editor/YAML-tab UX from #6004 may be simplified to match existing Scorecard/Table form patterns on `main` rather than copying the prototype editor harness verbatim.

## Fixture & Storybook

- New fixture under `web_src/src/pages/app/__fixtures__/console/` (JSON canvas snapshot shape matching SaaS: org/canvas ids, `consoleYaml`, memory/runs as needed).
- Register in `consoleFixtures.ts`.
- New story export in `AppPageConsole.stories.tsx`, e.g. **Software Factory**, `query=view=console`.
- Data is **hand-crafted and coherent** (fictional personas, consistent AI spend ↔ merge counts). Not a live capture from production.
- Scrubbing rules match existing console fixtures (deterministic fake emails/UUIDs if any PII-like fields appear).

## Out of scope (v1)

- Ranking as its own registered panel type.
- Porting Scorecard/Table prototypes from #6004 (superseded by main).
- Closing or rebasing the entire `prototype/new-console-panels` PR as-is.
- Backend workers or real integrations that compute AI authorship / spend — fixture memory only.
- Multiple Software Factory variants (one story is enough for v1).

## Success criteria

- Storybook `Pages/AppPage/Console` → **Software Factory** loads offline via `AppPageHarness` and shows spotlight + scorecards + rich tables without network.
- Spotlight panels validate in YAML (FE + Go) and render in the live console path used by AppPage.
- Layout clearly communicates AI vs human throughput without requiring a README essay.
- Existing SaaS / PR Risk / Docs / Release console stories remain unchanged.

## Implementation order

1. Port Spotlight renderer + content model onto current `main` patterns; register panel type end-to-end.
2. Widget + YAML tests for spotlight.
3. Build Software Factory fixture YAML + memory data + AppPage story.
4. Polish Storybook layout (sizes, empty states, dark mode sanity).
5. PRD note in `docs/prd/console-and-widgets.md` for the new panel type.

## Risks

- Prototype Spotlight (~54 commits behind main) will need a **surgical port**, not a merge of `prototype/new-console-panels`.
- AppPage fixture harness must supply memory namespaces the YAML expects — follow SaaS fixture patterns closely.
- AI/human labeling is a **presentation convention in fixture data**, not a product claim about detection accuracy.
