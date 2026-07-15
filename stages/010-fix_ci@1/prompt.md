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

## Context
- failure_class: deterministic
- failure_signature: wait_checks|deterministic|script failed with exit code: <n> ## output /bin/bash: line <n>: @scripts/wait-checks.sh: no such file or directory
- human.gate.approve.answer: A
- human.gate.approve.label: [A] Approve
- human.gate.approve.question: Approve Plan
- human.gate.label: [A] Approve
- human.gate.selected: A
- verify.summary: format.js passed; canvasAppPreferencePresentation.spec.ts (1 test) and index.spec.tsx (23 tests) all passed with the generatedApiClientStub plugin in vitest.config.ts resolving the previously-failing sdk.gen import


Fix GitHub PR check failures for the workflow goal: Remove "Pin App" from homepage - keep only star

## Change

Remove the "Pin App" action from the homepage. Keep only the star functionality.

## Priority

P3

Requirements:

- Re-read the previous `wait_checks` / `gh pr checks` output in the stage preamble
- Fix only what is needed for the failing checks (prefer targeted changes)
- Stay within `plan.md` scope unless a check failure forces a small adjacent fix
- Do not open a new PR — Fabro will push the run branch and re-wait checks