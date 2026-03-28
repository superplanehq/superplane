# Professorianci — SuperPlane Hackathon Plan

**Team:** Dragan, Braca, Fedja
**Date:** March 28, 2026 — Novi Sad
**Project:** Incident Copilot + Workflow Quality Gates

---

## The Pitch (One Sentence)

We build an AI-powered Incident Copilot that auto-triages production alerts, AND a workflow quality gate that ensures the copilot (and every canvas) is safe before it goes live.

---

## What We Built

### Track A — Incident Copilot (Dragan)
An autonomous incident response workflow in SuperPlane Canvas:
- **Trigger:** PagerDuty `onIncident` (filtered to P1/P2 only)
- **Fan-out:** 3 parallel nodes fetch recent deploys (GitHub), metrics (Datadog), incident timeline (PagerDuty)
- **Merge:** Waits for all 3 data sources (2-minute timeout)
- **AI Triage:** Claude component receives all context, produces structured severity assessment with root cause hypotheses
- **Output:** Slack message with evidence pack to `#hackathon-demo` channel
- **Approval gate** before any remediation action
- **4 annotation widgets** explaining each stage

**File:** `templates/canvases/incident-copilot.yaml` (13 nodes, 10 edges)

### Track B — Workflow Quality Gate (Braca)
Static analysis engine that validates ANY canvas — implemented as both a Go backend package and TypeScript frontend linter with full parity:

**9 Lint Rules:**
| Rule | Severity | What it catches |
|------|----------|----------------|
| `duplicate-node-id` | error | Two nodes sharing the same ID |
| `duplicate-node-name` | warning | Ambiguous expression references |
| `invalid-edge` | error | Dangling refs, self-loops, duplicate edges, widget endpoints |
| `cycle-detected` | error | Circular dependencies in workflow graph (Kahn's algorithm) |
| `orphan-node` | warning | Nodes not reachable from any trigger (BFS) |
| `dead-end` | warning | Non-terminal nodes with no outgoing edges |
| `missing-approval-gate` | error | Destructive actions without upstream approval (reverse BFS) |
| `missing-required-config` | error/warn | Empty prompts, missing channels, single-input merges |
| `invalid-expression` | error/warn | Unbalanced `{{ }}`, references to non-existent nodes |
| `unreachable-branch` | info | Filter nodes with no default outgoing edge |

**Quality Scoring:**
- Score 0-100 with letter grades A-F
- Per-category caps: errors max -60pts, warnings max -30pts, info max -10pts
- All 3 existing templates score Grade A

**Integration Points:**
- **Pre-save quality gate** — logs quality issues on every canvas save (warn-only, never blocks)
- **REST API** — `POST /api/v1/canvases/{id}/lint` returns full lint result as JSON
- **Frontend badge** — green/red badge in canvas header with tooltip showing all issues and quality score
- **36 unit tests** including dogfood tests against all 3 existing templates

**Files:**
- `pkg/linter/linter.go` — Go linter engine (9 rules, quality scoring)
- `pkg/linter/linter_test.go` — 36 tests, all passing
- `pkg/grpc/actions/canvases/lint_canvas.go` — REST API handler
- `pkg/grpc/actions/canvases/update_canvas_version.go` — pre-save quality gate
- `pkg/public/server.go` — route registration
- `web_src/src/utils/canvasLinter.ts` — TypeScript linter (full parity with Go)
- `web_src/src/ui/CanvasPage/Header.tsx` — quality gate badge UI

### Track C — Demo & Glue (Fedja)
- 4 mock data files for realistic demo scenario
- Slack channel configured: `#hackathon-demo` (C0APV7H889F)
- Demo script with quality gates narrative

**Files:**
- `docs/mock-incident.json` — PagerDuty incident payload (API Gateway 5xx spike)
- `docs/mock-github-release.json` — GitHub release v2.14.3
- `docs/mock-datadog-metrics.json` — Error rate, latency, request count time series
- `docs/mock-pagerduty-logs.json` — Incident timeline log entries

---

## Why This Wins

1. **Incident Copilot** is directly on-theme (AI + automation + production systems)
2. **Quality Gate** makes SuperPlane enterprise-ready — real product value
3. Together they tell one story: "We built the feature AND the safety net"
4. Every DevOps person relates to 3am incident pages
5. Demo is visual and compelling — canvas workflow + Slack output + red/green quality badge
6. Quality scoring (A-F grades) gives an instant readability to canvas health
7. Full Go + TypeScript parity means the badge is always accurate

---

## Demo Script (5 minutes)

### Slide 1 — The Problem (30 seconds)
"It's 3am. PagerDuty fires. Your engineer wakes up, spends 20 minutes across 5 dashboards gathering context before they even understand what's happening."

### Live Demo — The Incident Copilot (90 seconds)
1. Show the canvas: "Here's our Incident Copilot — built entirely in SuperPlane's Canvas"
2. Walk through the flow: trigger, filter, parallel data collection, merge, AI triage, Slack output, approval gate
3. Point out the **green quality badge** in the header: "Score: 100/100, Grade A — this workflow is validated before it goes live"
4. Fire the webhook: `curl -X POST http://localhost:8000/api/v1/webhooks/<webhook-id> -H "Content-Type: application/json" -d @docs/mock-incident.json`
5. Watch nodes light up in real-time
6. Switch to Slack: show the evidence pack arriving with severity assessment
7. "47 seconds. From alert to actionable triage."

### Live Demo — The Quality Gate (90 seconds)
1. "But how do you know this workflow is safe before it goes live?"
2. Show the green badge: "Quality Gate: A (100/100)"
3. **Break something:** Delete an edge — badge turns red immediately: "1 error — orphan node detected"
4. **Fix it:** Reconnect the edge — badge turns green again
5. **Break differently:** Remove the approval gate — badge shows: "Destructive action has no upstream approval gate"
6. **Show the API:** `curl -X POST http://localhost:8000/api/v1/canvases/<id>/lint` — show JSON output with quality score
7. "Every canvas gets a quality score. Errors are caught before they reach production."

### Live Demo — Deep Validation (30 seconds)
1. Add a node with an expression referencing a non-existent node — badge catches it
2. Create a cycle — badge catches it
3. "9 rules, from graph cycles to expression validation. The linter catches what humans miss."

### Slide 2 — What We Built (30 seconds)
- Incident Copilot: AI triage in < 60 seconds vs 20 minutes manual
- Quality Gate: 9 lint rules, quality scoring (A-F), REST API, real-time badge
- Full Go + TypeScript parity — backend and frontend always agree
- 36 unit tests, all 3 existing templates pass with Grade A
- Zero backend changes needed for copilot, minimal changes for quality gate

### Slide 3 — What's Next (30 seconds)
- Quality gate as a pre-publish hook (block publish when grade < C)
- Linter as a built-in SuperPlane feature
- Copilot templates for common incident types (database, network, deployment)
- Self-healing: AI suggests workflow fixes when linter finds issues
- Integration contract tests using the same quality gate framework

---

## Pre-Hackathon Checklist

- [x] All three: clone repo, run `make dev.setup && make dev.start`
- [x] Dragan: review Canvas API and available integration components
- [x] Braca: review canvas data model (how nodes/edges are stored)
- [x] Fedja: find a real PagerDuty/Datadog alert payload format for mock data
- [x] All: agree on a Slack channel for demo output (`#hackathon-demo` — C0APV7H889F)

---

## Technical Stats

| Metric | Value |
|--------|-------|
| Go lines written | ~1,200 (linter + API + tests) |
| TypeScript lines written | ~400 (frontend linter + badge) |
| YAML template lines | ~280 (incident copilot) |
| Mock data files | 4 JSON files |
| Lint rules | 9 (full Go/TS parity) |
| Unit tests | 36 (all passing) |
| Template quality scores | 100/A, 100/A, 95/A |
| Devil's advocate reviews | 2 rounds, 26 issues found and fixed |

---

## Fallback Plan

If anything goes wrong with the full Incident Copilot:
- Simplify to just 2 nodes: trigger -> AI triage -> Slack (skip the parallel fan-out)
- The quality gate stands on its own as a valuable feature regardless
- Worst case: quality gate demo + copilot design walkthrough still tells the story
