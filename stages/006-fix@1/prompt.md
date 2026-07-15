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
- **verify**: failed
  - Model: claude-sonnet-4-6, 17.8k tokens in / 2.8k out

## Context
- failure_class: deterministic
- failure_signature: verify|deterministic|npx vitest run src/pages/home/index.spec.tsx exited non-zero: 'failed to resolve import @/api-client/sdk.gen' — generated api client not present in this environment (pre-existing failure,identical on head~<n> before the implementation)
- human.gate.approve.answer: A
- human.gate.approve.label: [A] Approve
- human.gate.approve.question: Approve Plan
- human.gate.label: [A] Approve
- human.gate.selected: A
- verify.summary: format.js passed, canvasAppPreferencePresentation.spec.ts passed (1 test), index.spec.tsx failed with pre-existing environment error (missing @/api-client/sdk.gen generated file, reproducible on HEAD~1)


Fix the verification failures from the previous stage for the workflow goal: Remove "Pin App" from homepage - keep only star

## Change

Remove the "Pin App" action from the homepage. Keep only the star functionality.

## Priority

P3

Requirements:

- Re-read the previous stage preamble, especially `failure_reason` / verify output
- Correct the code (or tests) so the **same checks** can pass
- Stay within the scope of `plan.md`; do not start unrelated refactors or fix unrelated base-branch breakage unless it blocks the chosen checks
- Do not re-run the full verify matrix yourself beyond what you need to confirm a fix — the next stage will verify again