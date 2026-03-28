# Professorianci — SuperPlane Hackathon Plan

**Team:** Dragan, Braca, Fedja
**Date:** March 28, 2026 — Novi Sad
**Project:** Incident Copilot + Workflow Quality Gates

---

## The Pitch (One Sentence)

We build an AI-powered Incident Copilot that auto-triages production alerts, AND a workflow linter that ensures the copilot (and every canvas) is safe before it goes live.

---

## What We're Building

### Track A — Incident Copilot (Dragan)
An autonomous incident response workflow in SuperPlane Canvas:
- **Trigger:** PagerDuty `onIncident` or Datadog `onAlert`
- **Fan-out:** Parallel nodes fetch recent deploys (GitHub), metrics (Datadog), logs, pod status
- **AI Triage:** Claude component receives all context, produces structured severity assessment
- **Output:** Slack message with evidence pack, severity, recommended actions
- **Approval gate** before any remediation action

No backend changes needed — this is canvas design + AI prompt engineering.

### Track B — Workflow Linter / Quality Gate (Braca)
Static analysis that validates ANY canvas before publish:
- Detect orphan nodes (not connected to anything)
- Detect missing required configuration on nodes
- Verify approval gates exist before destructive actions
- Check for cycles in non-loop paths
- Validate expressions syntax
- Output: pass/fail with list of issues

Pure logic — TypeScript or Go, no deep UI changes needed.

### Track C — Demo & Glue (Fedja)
- Build the demo scenario: mock incident data, realistic alert payload
- Slack output formatting (the "evidence pack" that arrives in channel)
- Prepare before/after narrative for presentation
- Help wherever a bottleneck appears
- If time allows: green/red badge on canvas showing linter status

---

## Why This Wins

1. **Incident Copilot** is directly on-theme (AI + automation + production systems)
2. **Workflow Linter** makes SuperPlane enterprise-ready — real product value
3. Together they tell one story: "We built the feature AND the safety net"
4. Every DevOps person relates to 3am incident pages
5. Demo is visual and compelling — canvas workflow + Slack output + red/green linting

---

## Timeline

| Time | Dragan | Braca | Fedja |
|------|--------|-------|-------|
| 0:00-0:15 | Verify dev setup works | Verify dev setup works | Verify dev setup works |
| 0:15-0:30 | Design copilot canvas flow | Study canvas data model / graph structure | Prepare mock incident payload |
| 0:30-1:30 | Build canvas: trigger -> fan-out -> AI node -> Slack | Build linter core: orphan detection, missing config, cycle detection | Build Slack output template, mock data for all fan-out nodes |
| 1:30-2:15 | Wire AI prompt, tune triage output | Add approval gate check, expression validation | Integrate linter output into demo flow, help where needed |
| 2:15-2:30 | End-to-end test of full copilot flow | Run linter against copilot canvas (eat our own dogfood) | Capture screenshots, prepare demo script |
| 2:30-2:45 | Polish | Polish | Build 3-slide pitch |
| 2:45-3:00 | **Present** | **Present** | **Present** |

---

## Demo Script (3 minutes)

**Slide 1 — The Problem**
"It's 3am. PagerDuty fires. Your engineer wakes up, spends 20 minutes across 5 dashboards gathering context before they even understand what's happening."

**Live Demo — The Solution**
Show the Incident Copilot canvas -> trigger it with mock alert -> watch nodes execute -> show Slack evidence pack arriving with severity + context + recommendations + approval button.

**Live Demo — The Safety Net**
Run the linter against the copilot canvas -> show it passing. Then break something (remove a connection) -> show it catching the error. "Before this goes live, we know it's safe."

**Slide 2 — What We Built**
- Incident Copilot: AI triage in < 60 seconds vs 20 minutes manual
- Workflow Linter: catches errors before they hit production
- Both use SuperPlane's existing integrations, no backend changes

**Slide 3 — What's Next**
- Linter as a built-in SuperPlane feature (pre-publish hook)
- Copilot templates for common incident types
- Self-healing: AI suggests workflow fixes when linter finds issues

---

## Pre-Hackathon Checklist

- [ ] All three: clone repo, run `make dev.setup && make dev.start` BEFORE the event
- [ ] Dragan: review Canvas API and available integration components
- [ ] Braca: review canvas data model (how nodes/edges are stored)
- [ ] Fedja: find a real PagerDuty/Datadog alert payload format for mock data
- [ ] All: agree on a Slack channel for demo output

---

## Fallback Plan

If anything goes wrong with the full Incident Copilot:
- Simplify to just 2 nodes: trigger -> AI triage -> Slack (skip the parallel fan-out)
- The linter stands on its own as a valuable feature regardless
- Worst case: linter demo + copilot design walkthrough still tells the story
