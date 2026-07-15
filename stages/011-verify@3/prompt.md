Goal: Remove "Pin App" from homepage - keep only star

## Change

Remove the "Pin App" action from the homepage. Keep only the star functionality.

## Priority

P3

## Completed stages
- **plan**: succeeded
  - Model: claude-sonnet-4-6, 52.6k tokens in / 4.9k out
  - Files: /workspace/superplane/plan.md
- **approve**: succeeded
- **implement**: succeeded
  - Model: claude-sonnet-4-6, 48.0k tokens in / 10.8k out
  - Files: /workspace/superplane/web_src/src/pages/home/CanvasCardsGrid.tsx, /workspace/superplane/web_src/src/pages/home/CanvasFolderSection.tsx, /workspace/superplane/web_src/src/pages/home/canvasAppPreferencePresentation.spec.ts, /workspace/superplane/web_src/src/pages/home/canvasAppPreferencePresentation.ts, /workspace/superplane/web_src/src/pages/home/index.spec.tsx, /workspace/superplane/web_src/src/pages/home/index.tsx, /workspace/superplane/web_src/src/pages/home/types.ts, /workspace/superplane/web_src/src/pages/home/useHomePageCanvasList.ts
- **verify**: succeeded
  - Model: claude-sonnet-4-6, 12.0k tokens in / 1.0k out
- **fix**: succeeded
  - Model: claude-sonnet-4-6, 81.0k tokens in / 38.3k out
  - Files: /workspace/superplane/web_src/src/pages/home/index.spec.tsx, /workspace/superplane/web_src/vitest.config.ts
- **verify**: succeeded
  - Model: claude-sonnet-4-6, 12.0k tokens in / 1.0k out
- **ensure_pr**: failed
  - Script: `@scripts/ensure-pr.sh`
  - Output:
    ```
    /bin/bash: line 2: @scripts/ensure-pr.sh: No such file or directory
    ```
- **wait_checks**: failed
  - Script: `@scripts/wait-checks.sh`
  - Output:
    ```
    /bin/bash: line 2: @scripts/wait-checks.sh: No such file or directory
    ```
- **fix_ci**: succeeded
  - Model: claude-sonnet-4-6, 17.0k tokens in / 4.6k out

## Context
- human.gate.approve.answer: A
- human.gate.approve.label: [A] Approve
- human.gate.approve.question: Approve Plan
- human.gate.label: [A] Approve
- human.gate.selected: A
- verify.summary: format.js passed; canvasAppPreferencePresentation.spec.ts (1 test) and index.spec.tsx (23 tests) all passed with the generatedApiClientStub plugin in vitest.config.ts resolving the previously-failing sdk.gen import


Inspect the current git diff and `plan.md`. Run the **minimal** SuperPlane verification appropriate for the changed files.

Guidance for choosing commands:

- Prefer **targeted** checks on files/packages you changed (e.g. specific vitest files, `make test PKG_TEST_PACKAGES=./pkg/...`)
- JS/TS/UI: `make format.js`, then targeted vitest for touched specs. Only run `make check.build.ui` when the change is broad enough that a full UI typecheck is justified
- Go: `make format.go`, then targeted `make test PKG_TEST_PACKAGES=...` / package lint as needed. Use full `make lint` / `make check.build.app` only when justified by the diff
- Do **not** expand into unrelated full-suite failures you did not need to run

Rules:

- **Do not fix failures in this stage** — only run checks and report
- If **any** command you ran exits non-zero, the stage **must fail** — including when you believe the failure is pre-existing
- If all chosen commands pass, the stage succeeds
- End your response with a routing JSON object (Fabro reads the last one). Examples:

Success:
```json
{"outcome": "succeeded", "context_updates": {"verify.summary": "all chosen checks passed"}}
```

Failure:
```json
{"outcome": "failed", "failure_reason": "<command> failed: <short reason>", "context_updates": {"verify.summary": "<what failed>"}}
```
