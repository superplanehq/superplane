---
name: pre-pipeline-review
description: >-
  Reviews SuperPlane code changes for correctness and reuse, then runs fast
  Make targets (lint, format, build) that Semaphore CI runs so easy checks do
  not fail in the pipeline. Use when the user asks to review a change, finish
  a PR, preflight CI, check that the pipeline will pass, or run easy/quick
  checks before opening a pull request.
---

# Pre-pipeline review

Review the current SuperPlane diff for correctness and reuse, then run only the
**fast** Make targets that match touched paths. Goal: catch pipeline failures
from lint, format, and build without running slow unit or E2E suites.

Do not duplicate [clean-code](../clean-code/SKILL.md),
[ui-copy](../ui-copy/SKILL.md),
[simplified-technical-english](../simplified-technical-english/SKILL.md), or
[commit-and-pr-messages](../commit-and-pr-messages/SKILL.md). Point at them when
relevant. Read [AGENTS.md](../../../AGENTS.md) in full when you need setup or
command facts. For UI or models work, also read
[web_src/AGENTS.md](../../../web_src/AGENTS.md) or
[pkg/models/AGENTS.md](../../../pkg/models/AGENTS.md).

Path-to-target matrix and skip list: [checks.md](checks.md).

## Workflow

Copy this checklist and track progress:

```
Pre-pipeline review:
- [ ] 1. Scope the diff
- [ ] 2. Review for correctness and reuse
- [ ] 3. Map paths to Make targets (checks.md)
- [ ] 4. Ensure app container is up
- [ ] 5. Format, then run matching checks
- [ ] 6. Fix failures and re-run until green or blocked
- [ ] 7. Report findings and check results
```

### 1. Scope the diff

Run `git status` and `git diff` (staged and unstaged). If the user named a
branch or PR, include that range. Note which areas changed: Go, JS/TS, protos,
models, gRPC actions, components, migrations, docs-only.

### 2. Review for correctness and reuse

Review the change against the bar below. Search the repo before flagging or
accepting new helpers, components, or patterns. Prefer existing code over new
abstractions.

### 3. Map paths to Make targets

Open [checks.md](checks.md). Select only the targets that match touched paths.
Do not run the skip list unless the user explicitly asks.

### 4. Ensure app container is up

```bash
make dev.test.is.running
```

If that fails, run `make dev.up` and wait. Do **not** run `make dev.setup`
unless deps or protos clearly need it. Never drop or recreate databases
(`db.delete`, `dev.setup.no.cache`, etc.).

### 5. Format, then run matching checks

Use `make` only. Do not invent host `go` / `npm` commands when a Make target
exists.

1. Run format for the languages you touched (`make format.go`, `make format.js`).
2. Run the matching `check.*` / `lint` targets from [checks.md](checks.md).
3. Prefer parallel independent checks when safe.

### 6. Fix and re-run

If a check fails:

- Fix the underlying issue (format drift, lint, type error, missing auth wire-up).
- Re-run format if you edited files, then re-run the failed check.
- Do **not** expand the ESLint budget, skip checks, or hand-edit generated
  artifacts to get green.

Stop and report if you hit a real blocker (missing setup, unrelated infra,
needs user decision).

### 7. Report

Use this structure:

```markdown
## Review

### Critical
- …

### Suggestions
- …

(If none: say so.)

## Checks

| Target | Result |
| --- | --- |
| `make lint` | pass / fail / skipped (reason) |

## Blockers
- … (or "None")
```

## Review bar

Must catch:

- Wrong or missing behavior, broken edge cases, inconsistent state across
  callers of changed APIs.
- Duplication or reinventing the wheel: before accepting new helpers or UI,
  search `web_src/src/lib`, `web_src/src/hooks`, `web_src/src/components/ui`,
  and nearby packages under `pkg/`.
- No new `web_src/src/utils/` folders or `utils.ts` files.
- Do not hand-edit generated files: `pkg/protos/`, `pkg/openapi_client/`,
  `web_src/src/api-client/`, `api/`. Edit `protos/` and run `make pb.gen`.
- Do not hand-write migrations; use `make db.migration.create NAME=<name>`.
- Proto request enums mapped via `pkg/grpc/actions/common.go` (`Proto*` /
  `*ToProto`). New workers registered in `cmd/server/main.go`. New gRPC
  endpoints covered in `pkg/authorization/interceptor.go`.
- User-facing product name is **SuperPlane** (capital P).
- When UI strings changed, follow ui-copy and STE skills.

Severity:

- **Critical**: Must fix before merge (bugs, CI-breaking issues, policy
  violations above).
- **Suggestion**: Improve if cheap; not a merge blocker.

## Related skills

- Clean-code standards: [clean-code](../clean-code/SKILL.md)
- UI strings: [ui-copy](../ui-copy/SKILL.md),
  [simplified-technical-english](../simplified-technical-english/SKILL.md)
- Commit / PR wording: [commit-and-pr-messages](../commit-and-pr-messages/SKILL.md)
