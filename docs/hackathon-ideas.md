# Superplane Hackathon Ideas — Novi Sad 2026

Six Thinking Hats analysis of hackathon project ideas for the Superplane platform.

## Platform Summary

Superplane is an open-source DevOps control plane for event-based workflows:
- **Backend:** Go 1.25, gRPC + REST, PostgreSQL, RabbitMQ
- **Frontend:** React 19, TypeScript, React Flow (Canvas UI), Tailwind, shadcn
- **AI Agent:** Python 3.13, PydanticAI, Claude API (alpha)
- **45+ integrations:** GitHub, AWS, Slack, PagerDuty, Datadog, and more
- **Key gaps:** No native K8s integration, no workflow testing, no auto-error recovery, limited observability

---

## Top 10 Ideas (Ranked by Feasibility x Impact x Demo-ability)

### 1. Incident Copilot — AI-Powered First-5-Minutes Triage Agent

**Build an autonomous incident response workflow that triggers on PagerDuty/Datadog alerts and uses AI to gather context, correlate signals, and propose actions.**

- **Trigger:** PagerDuty `onIncident` or Datadog `onAlert`
- **Canvas flow:** Parallel fan-out to fetch recent deploys (GitHub), check dashboards (Datadog/Grafana), pull logs, check pod status (HTTP to K8s API)
- **AI node:** Claude component receives all context, produces structured triage summary
- **Output:** Posts evidence pack to Slack with severity assessment and recommended actions
- **Approval gate** before any remediation action

| Effort | Demo Impact | Skills Needed |
|--------|-------------|---------------|
| Low-Medium | Very High | Canvas design, AI prompting |

**Why it wins:** Directly on-theme. Uses existing integrations. Mostly canvas template + AI prompt engineering. No backend changes needed.

---

### 2. NL2Workflow — Natural Language to Complete Canvas Generation

**Type a sentence describing your workflow and get a fully wired canvas.**

Example input: *"When a PR is merged to main, run tests, if they pass deploy to staging with a 10-minute canary, then promote to production with approval"*

- Enhance the existing AI agent to produce complete canvas operations
- Leverage the pattern library + component catalog as context
- Generate canvas YAML, import into UI
- Interactive refinement: "Add a Slack notification if canary fails"

| Effort | Demo Impact | Skills Needed |
|--------|-------------|---------------|
| Medium | Very High | Python, PydanticAI, prompt engineering |

**Why it wins:** AI sidebar already exists but only does Q&A. Full generation is the natural next step. Jaw-dropping demo.

---

### 3. Canvas Replay — Workflow Execution Debugger & Time-Travel UI

**Visual execution replay: step through a workflow run node-by-node, seeing inputs/outputs/timing at each step.**

- New UI panel showing execution timeline
- Click any node to see input payload, output, duration, errors
- "Play" button animates execution flow through the canvas
- Highlight bottlenecks (slow nodes in red)
- Compare two runs side-by-side (success vs failure)

| Effort | Demo Impact | Skills Needed |
|--------|-------------|---------------|
| Medium | High | React, TypeScript, React Flow |

**Why it wins:** Pure frontend work. Execution data already exists in the backend. Visually stunning demo.

---

### 4. Workflow Test Runner — Test Framework for Canvases

**Testing mode: define expected inputs/outputs for a canvas and run assertions without hitting real integrations.**

- Mock mode for components (return predefined responses)
- Test definition: trigger event -> expected node execution order -> expected outputs
- "Test" button in Canvas UI runs the workflow in simulation
- Green/red indicators on each node (passed/failed)
- Coverage report: which paths were exercised

| Effort | Demo Impact | Skills Needed |
|--------|-------------|---------------|
| Medium-High | High | Go (backend), React (frontend) |

**Why it wins:** Fills a critical gap. Makes Superplane enterprise-ready. Shows deep product understanding.

---

### 5. Kubernetes Operator Integration — Native K8s Triggers & Components

**Add Kubernetes as a first-class integration.**

- New integration in `pkg/integrations/kubernetes/`
- **Triggers:** `onPodCrashLoop`, `onDeploymentRollout`, `onHPAScale`, `onNodeNotReady`
- **Components:** `applyManifest`, `scaleDeployment`, `rollbackDeployment`, `getPodLogs`
- Uses K8s API via `client-go`

| Effort | Demo Impact | Skills Needed |
|--------|-------------|---------------|
| Medium-High | High | Go, Kubernetes |

**Why it wins:** K8s is THE missing integration. Every platform engineer wants this.

---

### 6. Self-Healing Workflows — AI Error Recovery Agent

**When a workflow node fails, an AI agent analyzes the error, suggests a fix, and can auto-retry with corrected parameters.**

- Intercept node execution failures in the worker
- Pass error context + node config to Claude component
- AI proposes: retry with different params, skip node, alert human, or rollback
- Configurable autonomy level per canvas: "suggest only" / "auto-fix with approval" / "full auto"
- Audit log of all AI decisions

| Effort | Demo Impact | Skills Needed |
|--------|-------------|---------------|
| Medium-High | Very High | Go (workers), AI prompting |

**Why it wins:** Makes workflows resilient without manual intervention. Novel feature no competitor has.

---

### 7. Integration Marketplace — Community Component Store

**UI where users can browse, install, and publish custom components/integrations.**

- Browse page with categories, search, popularity
- One-click install (downloads integration config)
- Publish: package a custom HTTP-based integration as a template
- Rating/review system
- Featured workflows section

| Effort | Demo Impact | Skills Needed |
|--------|-------------|---------------|
| Medium | Medium-High | React, API design |

**Why it wins:** Creates ecosystem/community value. Pure frontend + API work.

---

### 8. GitOps for Workflows — Canvas-as-Code with Git Sync

**Store canvas definitions in Git, sync bidirectionally, enable PR-based workflow changes with diff visualization.**

- Export canvas to YAML in a Git repo (GitHub integration exists)
- Watch for YAML changes, auto-update canvas
- PR workflow: propose canvas change as YAML diff, visual diff in Superplane UI
- Branch-based canvas environments (staging vs production)

| Effort | Demo Impact | Skills Needed |
|--------|-------------|---------------|
| Medium | High | Go, Git APIs, React |

**Why it wins:** GitOps is how infrastructure teams already work. Bridges visual editing and code review.

---

### 9. Workflow Analytics Dashboard — Execution Intelligence

**Real-time dashboard: success rates, execution times, failure patterns, cost estimates, anomaly detection.**

- Aggregate execution data from existing tables
- Charts: success/fail ratio, p50/p95 duration, failure heatmap by node
- Anomaly detection: flag unusually slow or failing runs
- Cost estimation: track API calls, compute time per workflow
- Weekly digest email

| Effort | Demo Impact | Skills Needed |
|--------|-------------|---------------|
| Medium | Medium-High | React, SQL, charting |

**Why it wins:** Addresses observability gap. Data already exists. Visual and data-rich demo.

---

### 10. Workflow Linter — Static Analysis for Canvases

**Analyze workflows for common mistakes before execution.**

- Graph analysis: detect cycles, orphan nodes, missing connections
- Config validation: required fields empty, invalid expressions, deprecated components
- Security checks: secrets in plaintext, missing approval gates
- Performance hints: suggest parallelization, flag long chains
- Inline warnings on Canvas nodes (yellow/red badges)

| Effort | Demo Impact | Skills Needed |
|--------|-------------|---------------|
| Low-Medium | Medium-High | Go or TypeScript, graph algorithms |

**Why it wins:** Low complexity, high value. Quick to build and demo. Makes Superplane feel enterprise-grade.

---

## Six Thinking Hats Summary

### White Hat (Facts)
- 45+ integrations, gRPC+REST API, React Flow canvas, PydanticAI agent (alpha)
- Key gaps: no K8s, no workflow testing, no auto-recovery, no anomaly detection

### Red Hat (Gut Feelings)
- AI + Canvas combo will have the biggest wow-factor
- Incident response is emotionally compelling (everyone hates 3am pages)
- Projects that demo well in 5 minutes will win

### Black Hat (Risks)
- One-day scope: overambitious projects won't finish
- Go backend requires Go expertise for deep changes
- Local dev setup may eat hours (Docker, Postgres, RabbitMQ)
- Canvas UI is a 228KB monolith, risky to modify deeply

### Yellow Hat (Strengths)
- Integration registry is pluggable (clear pattern to follow)
- AI agent framework exists, extending it is incremental
- Expression engine enables powerful data transformation
- Strong API with auto-generated TypeScript client

### Green Hat (Creative Ideas)
- See the 10 ideas above

### Blue Hat (Action Plan)
- **Best overall pick:** Incident Copilot (#1) — low risk, high demo impact
- **Best AI pick:** NL2Workflow (#4) — extends existing agent
- **Best frontend pick:** Canvas Replay (#3) — pure React, data exists
- **Impress the core team:** Workflow Test Runner (#4) or Workflow Linter (#10)
- **Strong Go skills:** Kubernetes Integration (#5) or Self-Healing Workflows (#6)

---

## Suggested Day Plan

| Time | Activity |
|------|----------|
| 0:00-0:30 | Environment setup (`make dev.setup && make dev.start`) |
| 0:30-1:00 | Familiarize with Canvas UI, create a test workflow |
| 1:00-5:00 | Build your project (pick ONE from above) |
| 5:00-5:30 | Polish demo, write 3-slide pitch |
| 5:30-6:00 | Present |
