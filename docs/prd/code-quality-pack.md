# Code quality pack

## Overview

This PRD specifies a **prebuilt pack of quality templates**: installable
canvases, one per quality domain, plus a **preset line** ("Verified delivery")
that composes them with the verification layer from
[line-verification.md](line-verification.md). The pack gives a new factory a
working quality workflow in one install, instead of every team building the
same checks by hand.

This document is a specification. It does not add templates to
`templates/manifest.json` or ship any canvas. The companion Storybook designs
(`web_src/src/pages/factories/quality-pack/`) show the install experience.

## Problem Statement

The verification layer defines *how* checks run, but every organization would
still start from an empty rule set and empty suites. The checks most teams
want are the same six: type safety, test coverage, secrets, dead code, file
size, dependencies. Building each one requires knowledge of agent components,
runner components, prompt design, and result plumbing — exactly the setup cost
templates exist to remove.

SuperPlane already distributes installable apps through
`templates/manifest.json` (Preview Envs, Auto Merge Bot, Semantic PRs, and
others), each with optional `agentSuggestions` and a `console.yaml`. The
quality pack follows this proven path.

## Goals

1. Ship six quality templates, one canvas per domain, each usable standalone
   or as a check inside a verification suite.
2. Ship a **preset line** definition, "Verified delivery": Build → Verify →
   Approval → Close.
3. Ship a default **rule set** the templates reference, so install produces a
   working configuration without manual rule authoring.
4. Package the review instructions as **skill files** (one `SKILL.md`-style
   asset per domain), following
   [ai-agent-component-skill-awareness.md](ai-agent-component-skill-awareness.md).
5. Give each template a `console.yaml` with quality scorecards so results are
   visible immediately after install.

## Non-Goals

- No new template distribution mechanism; the pack uses
  `templates/manifest.json` as is.
- No language-specific tooling beyond what runner components already execute;
  command steps call the repository's own build, test, and lint commands.
- No auto-installation into existing factories; installing the pack is always
  a user action.
- No marketplace or paid distribution concerns.

## The six templates

Each template is one canvas with the same shape: trigger → scan → parallel
agent checks per concern → deterministic verify → findings and report emit.

| Template | Domain | Agent checks (parallel) | Deterministic verify |
| --- | --- | --- | --- |
| Type Safety Review | type safety | untyped values; unsafe casts; missing narrowing | type check command (for example `tsc --noEmit`) |
| Test Coverage Review | tests | coverage gaps; weak assertions; test hygiene | test run command |
| Secret Scan | secrets | source scan; configuration files; history scan | secret scanner command |
| Dead Code Review | dead code | unused exports; orphaned files; stale annotations | build command |
| File Size Review | file size | oversized files; multi-responsibility modules | line count report command |
| Dependency Audit | dependencies | unused packages; single-use packages | vulnerability audit command |

Template shape in detail:

1. **Trigger** — an `onRun` trigger so the canvas works as a factory app and
   as a `verify`-step check; plus optional `schedule` and `webhook` triggers
   for standalone use.
2. **Scan** — a runner step that enumerates the changed files (from the work
   order's branch or pull request artifact) and groups them per concern.
3. **Agent checks** — parallel agent component steps (for example
   `claude.runcodeagent`), each scoped to one concern, with the domain skill
   file and the relevant rules injected. Output is a structured finding list.
4. **Deterministic verify** — a runner step that executes the domain's
   command. Its result is authoritative, per
   [line-verification.md](line-verification.md).
5. **Emit** — a step that records findings and a run summary (counts by
   severity) as the canvas output, so a verification step or a console can
   consume them.

## Skill assets

Each domain ships one skill file with the review instructions: what to look
for, how to classify severity, the required finding format (rule, location,
description, remediation), and explicit out-of-scope notes. Skill files use
plain, neutral language and follow the repository's Simplified Technical
English style. They live with the template assets and are injected into agent
checks at run time, per
[ai-agent-component-skill-awareness.md](ai-agent-component-skill-awareness.md).

## Preset line: Verified delivery

A factory preset installable together with the pack:

```yaml
name: verified-delivery
steps:
  - name: build
    type: runApp
    app: { app: "<build-canvas>", entrypoint: "on-run" }
  - name: verify
    type: verify
    suite: quality-pack-default
    ruleSet: quality-pack-default
  - name: approval
    type: runApp
    app: { app: "<approval-canvas>", entrypoint: "on-run" }
  - name: close
    type: runApp
    app: { app: "<close-canvas>", entrypoint: "on-run" }
```

The `verify` step references the pack's default suite, which contains the six
templates as checks. The approval step reuses the existing `approval`
component. The close step reuses `updateWorkOrderStatus`.

## Distribution

- One manifest entry per template in `templates/manifest.json`, following the
  existing entry shape, each with:
  - `agentSuggestions`: "Connect your repository." and "Adjust the rule set
    for this project."
  - `console.yaml`: a scorecard for findings by severity and a table of the
    latest runs.
- The pack's default rule set and suite install with the first template and
  are shared by the rest.
- Install order does not matter; each template is independently usable.

## UX walkthrough

The Storybook designs under `web_src/src/pages/factories/quality-pack/` define
the visual contract:

- **`QualityTemplateGallery`** — one card per template: name, domain, what it
  checks, and the number of checks per run. Cards follow the existing canvas
  card patterns. The primary action is "Install template".
- **`PresetLinePreview`** — the install-time preview of the Verified delivery
  line: its four steps in order, with the verification step expanded to show
  the six checks it runs. Helper text: "This will not start a run yet."

## Acceptance Criteria (for the eventual implementation)

1. Installing any single template yields a canvas that runs standalone from
   its triggers and emits findings in the standard format.
2. Installing the pack yields the default rule set, the default suite, and
   the Verified delivery line preset.
3. A work order dispatched through Verified delivery is gated by the six
   checks with the semantics of
   [line-verification.md](line-verification.md).
4. Each template's console shows findings by severity after the first run.
5. Skill files contain the finding format contract, and agent checks return
   findings in that format.

## Open Questions

- Do we pin repository commands (build, test, lint) per template
  configuration, or read them from a repository manifest file?
- Should the pack install one combined "quality" canvas as an alternative to
  six separate templates for small teams?
- Which additional domains follow the initial six (naming, observability,
  accessibility) and in what order?
