# Line verification steps

## Overview

This PRD describes a **verification layer** for factory lines: a new line step
type, `verify`, that inspects the output of a work order (branch, pull request,
artifacts) before the line advances. A verification step runs a **verification
suite** — a set of parallel **checks** driven by an org-scoped **rule set** —
and records **findings**. Blocking findings stop the line; advisory findings
pass but stay visible.

This document is a specification. It does not ship backend or application code.
The companion Storybook designs (`web_src/src/pages/factories/verification/`)
show the intended UI. Related PRDs:

- [verification-suggestions.md](verification-suggestions.md) — findings become
  actionable suggestions with a dispatch-fix flow.
- [code-quality-pack.md](code-quality-pack.md) — prebuilt quality templates
  that use this layer.
- [codebase-health.md](codebase-health.md) — aggregation of findings into a
  health surface.

## Problem Statement

A factory line advances a work order when a step's canvas run passes
(`docs/design/factory.md`). The run result is the only gate. Nothing inspects
the produced work itself — the branch, the pull request, or the artifacts —
against the organization's quality standards.

Teams that run AI agents inside lines feel this gap most. An agent can produce
a pull request that builds and still violates the rules the team cares about:
untyped code, missing tests, committed secrets, dead code, oversized files,
unjustified dependencies. Today the only options are a human review step or a
custom canvas that each team must build and maintain by hand.

We want verification to be a first-class line concept: declarative, reusable
across lines, visible on the work order, and safe to trust because
deterministic checks — not model output — make the final call.

## Goals

1. Add a `verify` step type to factory lines, next to `runApp`, that gates
   line progression on verification results.
2. Define **rule sets** as org-scoped resources: named rules grouped by
   domain, each with a severity and a blocking or advisory enforcement level.
3. Define **verification suites**: which checks run for a step, in parallel,
   and which rule set governs them.
4. Record **findings** as first-class data: rule, severity, location,
   description, remediation, and a reviewable status.
5. Keep deterministic checks authoritative: an AI review check alone never
   fails or passes a work order.

## Non-Goals

- No changes to the `runApp` step type or to canvas run semantics.
- No automatic fixing in this PRD; the fix loop is specified in
  [verification-suggestions.md](verification-suggestions.md).
- No cross-work-order aggregation; that is
  [codebase-health.md](codebase-health.md).
- No new agent runtime. Checks run on the existing component executions
  (`claude.runcodeagent`, `cursor.launchAgent`, `pkg/components/runner/`).
- No repository-hosting features. Verification reads the work order's
  artifacts (branch, pull request) through existing integrations.

## Concepts

### Rule set

An org-scoped resource. A rule set groups **rules** by **domain**. The initial
domains match the quality pack: type safety, tests, secrets, dead code, file
size, dependencies. A rule has:

| Field | Meaning |
| --- | --- |
| `id` | Stable identifier, unique in the rule set (e.g. `type-safety/no-untyped-values`). |
| `name` | Short human name shown in UI and findings. |
| `domain` | Grouping key; one of the supported domains. |
| `description` | What the rule requires, in one or two sentences. |
| `severity` | `high`, `medium`, or `low`. Drives finding display and health weighting. |
| `enforcement` | `blocking` or `advisory`. Blocking findings fail the verification step. |

Rule sets are editable in the UI and as YAML. An organization can hold several
rule sets (for example, one strict set for production repositories and one
lighter set for prototypes).

### Verification suite

A named selection of **checks** bound to one rule set. A check is either:

- **Agent check** — an AI review scoped to one domain. The check runs as an
  agent component execution with the rule set's rules for that domain injected
  as the prompt or skill. Output is a list of candidate findings.
- **Command check** — a deterministic command (build, type check, linter,
  test run, secret scanner) executed by a runner component. Output is
  pass/fail plus parsed findings when the tool reports locations.

Checks in a suite run **in parallel**. A suite declares, per check, whether
its findings can block (the check-level flag combines with the rule-level
`enforcement`; both must be blocking for a finding to block).

### Verification run and findings

One execution of a suite for one work order produces a **verification run**
with per-check outcomes and a list of **findings**:

| Field | Meaning |
| --- | --- |
| `rule` | Rule id the finding violates. |
| `severity` | Copied from the rule at detection time. |
| `location` | File path plus line range when available; empty for repo-level findings. |
| `description` | What is wrong, specific to this occurrence. |
| `remediation` | Concrete steps to fix this occurrence. |
| `status` | `open`, `fixed`, `dismissed`, or `accepted`. |

`fixed` is set when a later verification run no longer reports the finding.
`dismissed` marks a false positive. `accepted` records a conscious exception;
accepted findings never block again for the same location and rule.

### Gate semantics

- A verification step **fails** when the run finishes with one or more open
  blocking findings, or when any command check marked as required fails.
- A failed step stops the line; the work order stays `open`, matching the
  current dispatch flow.
- Advisory findings never fail the step. They are recorded and surfaced.
- Agent checks alone cannot pass a step that a command check failed.
  Deterministic results always win.

## Line definition (conceptual)

```yaml
name: bug
steps:
  - name: implement
    type: runApp
    app: { app: "<canvas-uuid>", entrypoint: "start-work" }
  - name: verify
    type: verify
    suite: default-quality
    ruleSet: production
  - name: review
    type: runApp
    app: { app: "<canvas-uuid>", entrypoint: "request-approval" }
```

## UX walkthrough

The Storybook designs under `web_src/src/pages/factories/verification/` define
the visual contract:

- **`LineStepTypeEditor`** — the line editor step card gains a step type
  choice (`Run app` / `Verify`). Selecting `Verify` shows a suite picker, the
  governing rule set, and a per-check blocking/advisory toggle.
- **`RuleSetEditor`** — rules grouped by domain with severity and enforcement
  controls, plus a YAML preview pane. Empty state guides the first rule set.
- **`WorkOrderVerificationPanel`** — on the work order detail: run status
  (running, passed, failed), per-check outcomes, and findings grouped by
  severity. Command check results are visually distinct from agent review
  results.
- **`VerificationStatusBadge`** and **`CheckOutcomeChip`** — status
  primitives consistent with the existing run status badge.

## Future backend surface (specification only)

Recorded for a later implementation phase; nothing here is built now.

- **Models**: `RuleSet`, `VerificationSuite`, `VerificationRun`, `Finding` in
  `pkg/models`, following the explicit `*gorm.DB` parameter pattern.
- **Protos**: rule set and suite CRUD plus verification run read APIs in a new
  `protos/verifications.proto`; enums mapped to `pkg/models` constants.
- **Step type**: extend the line step JSON (`factory_lines`) with
  `type: verify`, `suite`, and `ruleSet` references; the dispatch worker
  starts a verification run instead of a canvas run for these steps.
- **Execution**: reuse component executions for checks; a finalizer collects
  check outcomes, computes the gate result, and marks the execution
  passed/failed so existing line advancement applies unchanged.
- **Authorization**: new endpoints registered in
  `pkg/authorization/interceptor.go`; rule sets under a `rule_sets` resource,
  runs readable with `work_orders:read`.

## Acceptance Criteria (for the eventual implementation)

1. A line can declare a `verify` step; dispatching a work order through it
   runs the suite and gates advancement on the result.
2. Open blocking findings fail the step; advisory findings do not.
3. A failed required command check fails the step even when agent checks
   report no findings.
4. Findings persist with rule, severity, location, description, remediation,
   and status, and are visible on the work order.
5. Accepted findings do not block subsequent runs for the same rule and
   location.

## Open Questions

- Should a suite support per-repository overrides, or is one rule set per
  step enough for v1?
- Do we need a manual "re-run verification" action on the work order, or is
  re-dispatching the line sufficient?
- How long do verification runs and findings stay queryable before archival?
