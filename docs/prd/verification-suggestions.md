# Verification suggestions and dispatch fix

## Overview

This PRD turns verification **findings** into actionable **suggestions**. A
suggestion pairs a finding with its exact location, concrete remediation
steps, and a ready-to-run fix prompt. From a suggestion, one action —
**Dispatch fix** — creates a fix work order on a designated line or starts an
agent run directly.

This document is a specification. It does not ship backend or application
code. The companion Storybook designs
(`web_src/src/pages/factories/verification/`) show the intended UI. It builds
on [line-verification.md](line-verification.md), which defines findings, and
feeds [codebase-health.md](codebase-health.md), which aggregates them.

## Problem Statement

A finding that only says "this is wrong" creates review work instead of
removing it. The reader must locate the code, decide the fix, and route the
work by hand — create an issue, write an agent prompt, or fix it themselves.

Findings from the verification layer already carry location and remediation.
The missing piece is the loop: show open findings where people already work
(the work order, the factory), and let one click route the fix back through
the factory's own machinery — a work order on a line, or a direct agent run.

SuperPlane already has a narrower suggestions mechanism: template-defined
agent prompts stored per canvas (`agentSuggestions` in
`templates/manifest.json`, dismissals persisted on
`Canvas.DismissedAgentSuggestionIDs`). This PRD extends the same idea to
verification results, with server-side data instead of install-time strings.

## Goals

1. Present every open finding as a **suggestion**: summary, exact location,
   remediation steps, and a prepared fix prompt.
2. Provide **Dispatch fix** with two targets: create a fix work order on a
   designated line, or start an agent run with the remediation prompt.
3. Surface suggestions on the **work order detail** (findings from that
   order's verification runs) and at the **factory level** (open suggestions
   across work orders).
4. Aggregate repeated suggestions into **recurring patterns** for
   [codebase-health.md](codebase-health.md).
5. Keep suggestion status in sync with finding status: fixing, dismissing, or
   accepting a finding resolves its suggestion.

## Non-Goals

- No fully automatic fixing without a human action in v1. Dispatch fix is
  always user-initiated.
- No replacement of the template `agentSuggestions` mechanism; the two
  coexist.
- No suggestion authoring UI. Suggestions derive from findings only.
- No notification channels in v1 (email, Slack pushes); the run summary
  report in [codebase-health.md](codebase-health.md) covers reporting.

## Concepts

### Suggestion

A read model over an open finding:

| Field | Meaning |
| --- | --- |
| `finding` | The underlying finding (rule, severity, location, description, remediation, status). |
| `fixPrompt` | Prepared agent instruction: the remediation steps plus location context, ready to dispatch. |
| `sourceWorkOrder` | The work order whose verification run produced the finding. |
| `occurrences` | Count of matching findings across runs (same rule and location group). |

A suggestion is **open** while its finding is `open`. It resolves when the
finding becomes `fixed`, `dismissed`, or `accepted`.

### Dispatch fix

One primary action with a target choice:

- **Fix work order** — creates a `draft` work order titled from the
  suggestion, with the fix prompt as description, and offers dispatch to a
  designated fix line. This reuses the existing work order creation and
  dispatch flow unchanged.
- **Agent run** — starts an agent component run (for example
  `claude.runcodeagent` or `cursor.launchAgent`) with the fix prompt, without
  creating a work order. Suited to small, local fixes.

Both targets record the dispatch on the suggestion, so repeated dispatches
are visible and a suggestion under fix shows "Fix in progress".

### Recurring suggestions

The factory-level view groups suggestions by rule and location pattern and
tracks count, trend, and last-seen time. This grouping is the input for the
recurring patterns surface in [codebase-health.md](codebase-health.md).

## UX walkthrough

The Storybook designs under `web_src/src/pages/factories/verification/` define
the visual contract:

- **`SuggestionCard`** — finding summary, severity, exact location,
  remediation steps, and the primary **Dispatch fix** action with a target
  choice (fix work order or agent run). Secondary actions: **Dismiss** and
  **Accept**. Helper text under the primary action states the outcome, for
  example "This creates a draft work order. You choose when to dispatch it."
- **`WorkOrderSuggestionsList`** — open suggestions on the work order detail
  layout, with filters by severity and domain, and an empty state that
  confirms a clean verification run.
- **`RecurringSuggestionsTable`** — factory-level aggregation: pattern, rule,
  count, trend, last seen; each row links to the pattern card defined in
  [codebase-health.md](codebase-health.md).

## Future backend surface (specification only)

Recorded for a later implementation phase; nothing here is built now.

- **Read APIs**: list suggestions per work order and per factory, derived
  from findings; no separate suggestion table required in v1 beyond dispatch
  records.
- **Dispatch records**: a small model linking a finding to the fix work order
  or agent run it spawned, so status can render "Fix in progress" and later
  "Fixed by <work order>".
- **Actions**: dismiss and accept mutate the finding status defined in
  [line-verification.md](line-verification.md); dispatch-fix composes the
  existing `createWorkOrder`, dispatch, and agent-run paths.
- **Authorization**: suggestions readable with `work_orders:read`; dispatch
  fix requires `work_orders:create` (work order target) or the existing
  canvas-run permission (agent target).

## Acceptance Criteria (for the eventual implementation)

1. Every open finding appears as exactly one suggestion on its work order.
2. Dispatch fix to a work order creates a `draft` order with the fix prompt
   as description and links it back to the suggestion.
3. Dispatch fix to an agent run starts the run with the fix prompt and links
   it back to the suggestion.
4. Dismiss and accept update the finding status and resolve the suggestion.
5. A verification run that no longer reports a finding resolves its
   suggestion as fixed.
6. The factory-level view groups suggestions by rule and location pattern
   with correct counts.

## Open Questions

- Should dispatch fix support batching (one work order for many suggestions
  of the same rule), or is one-to-one enough for v1?
- Who may accept a finding — any member with `work_orders:update`, or a
  narrower reviewer role?
- Should the agent-run target be limited to organizations with AI features
  enabled, mirroring the AI Builder policy?
