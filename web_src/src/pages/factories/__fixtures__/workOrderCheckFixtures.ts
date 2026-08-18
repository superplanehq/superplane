import type { WorkOrderCheckPresentation } from "../WorkOrderChecksSection";

import { OPEN_WORK_ORDER, RUNNING_WORK_ORDER } from "./factoryPageResponses";

/**
 * Mock checks for Storybook — scores that dedicated automations attach to a
 * work order (risk review, coverage, confidence). Timestamps are relative to
 * now so the cards always read as recent.
 */

function minutesAgo(minutes: number): string {
  return new Date(Date.now() - minutes * 60_000).toISOString();
}

const RISK_REVIEW_ANALYSIS = `### Summary

The change replaces the retry policy in the refund dispatcher and touches the payment provider client. The blast radius is moderate: every refund path goes through the modified dispatcher, but the provider client changes are additive.

### Concerns

- The new retry policy removes the exponential backoff cap. A provider outage could produce a burst of retries against a degraded dependency.
- \`RefundDispatcher.dispatch\` no longer wraps provider errors, so callers that matched on \`ProviderError\` may silently change behavior.
- No migration guard for in-flight refunds created before the schema change.

### Recommended reviewers

- **mina-k** — owns the refund dispatcher and reviewed the last three changes in this area.
- **jordanp** — wrote the provider client error taxonomy.`;

const CODE_COVERAGE_ANALYSIS = `### Coverage by package

| Package | Coverage | Change |
| --- | --- | --- |
| \`pkg/refunds\` | 91% | +2.4% |
| \`pkg/providers\` | 78% | −1.1% |
| \`pkg/workers\` | 74% | ±0.0% |

The drop in \`pkg/providers\` comes from the new \`RetryPolicy\` type: its failure branches are not exercised. Two focused tests on the backoff cap and the give-up path would recover the loss.`;

const TEST_COVERAGE_ANALYSIS = `### New code in this work order

- \`RefundDispatcher.dispatch\` — covered by 6 new cases, including both terminal failure paths.
- \`RetryPolicy.next\` — only the happy path is covered. The jitter bounds and the max-attempts cutoff have no direct tests.
- \`ProviderClient.refund\` — covered through integration tests only; no unit coverage for the timeout branch.

Uncovered lines are concentrated in error handling, which is also where the risk review raised concerns. Prioritize \`RetryPolicy\` tests before merging.`;

const CONFIDENCE_ANALYSIS = `The agent completed all planned steps without human correction. Static checks, unit tests, and the E2E refund suite passed on the first run.

Confidence is not higher because the change modifies retry behavior that only manifests under provider degradation, which no automated test simulates.`;

export const OPEN_WORK_ORDER_CHECKS: WorkOrderCheckPresentation[] = [
  {
    id: "check-risk-review",
    name: "Risk review",
    score: 65,
    maxScore: 100,
    level: "caution",
    previousScore: 82,
    summary: "Moderate risk: retry policy changes affect every refund path and remove the backoff cap.",
    analysis: RISK_REVIEW_ANALYSIS,
    sourceName: "PR Risk Review",
    appId: "app-pr-risk-review",
    runId: "run-risk-review-101",
    updatedAt: minutesAgo(12),
  },
  {
    id: "check-code-coverage",
    name: "Code coverage",
    score: 82,
    maxScore: 100,
    format: "percent",
    level: "positive",
    previousScore: 81,
    summary: "Overall coverage rose to 82% (+1.3%), with a small drop in the provider client package.",
    analysis: CODE_COVERAGE_ANALYSIS,
    sourceName: "Coverage Report",
    appId: "app-coverage-report",
    runId: "run-coverage-101",
    updatedAt: minutesAgo(9),
  },
  {
    id: "check-test-coverage",
    name: "Test coverage",
    score: 74,
    maxScore: 100,
    format: "percent",
    level: "neutral",
    summary: "74% of the new code is covered; gaps concentrate in retry error handling.",
    analysis: TEST_COVERAGE_ANALYSIS,
    sourceName: "Coverage Report",
    appId: "app-coverage-report",
    runId: "run-coverage-101",
    updatedAt: minutesAgo(9),
  },
  {
    id: "check-confidence",
    name: "Confidence score",
    score: 8,
    maxScore: 10,
    level: "positive",
    previousScore: 7,
    summary: "All planned steps completed without human correction; every automated suite passed first try.",
    analysis: CONFIDENCE_ANALYSIS,
    sourceName: "Line Confidence",
    appId: "app-line-confidence",
    runId: "run-confidence-101",
    updatedAt: minutesAgo(4),
  },
];

/** Two checks only — the risk review has landed, coverage is still running. */
export const RUNNING_WORK_ORDER_CHECKS: WorkOrderCheckPresentation[] = [
  {
    id: "check-risk-review-running",
    name: "Risk review",
    score: 38,
    maxScore: 100,
    level: "neutral",
    summary: "Low-moderate risk: the change is additive and scoped to one worker.",
    analysis:
      "### Summary\n\nThe change adds a new reconciliation worker without touching existing dispatch paths. Blast radius is limited to the new queue.\n\n### Concerns\n\n- The worker shares a database pool with the dispatcher; a slow query could starve it under load.",
    sourceName: "PR Risk Review",
    appId: "app-pr-risk-review",
    runId: "run-risk-review-102",
    updatedAt: minutesAgo(25),
  },
  {
    id: "check-confidence-running",
    name: "Confidence score",
    score: 6,
    maxScore: 10,
    level: "caution",
    previousScore: 8,
    summary: "The agent needed one human correction during planning; verification has not run yet.",
    analysis:
      "The plan step was corrected once: the agent initially targeted the wrong ledger table. Implementation followed the corrected plan without further intervention.\n\nVerification is still in progress, so this score may change.",
    sourceName: "Line Confidence",
    appId: "app-line-confidence",
    runId: "run-confidence-102",
    updatedAt: minutesAgo(11),
  },
];

/** Fallback map for fixtures that do not override `checksByOrderId` —
 * the open order carries the full set, the running order a partial one,
 * and every other order (closed, draft, failed) has none. */
export const DEFAULT_CHECKS_BY_ORDER_ID: Record<string, WorkOrderCheckPresentation[]> = {
  [OPEN_WORK_ORDER.id!]: OPEN_WORK_ORDER_CHECKS,
  [RUNNING_WORK_ORDER.id!]: RUNNING_WORK_ORDER_CHECKS,
};

/** A single critical check — the smallest interesting state. */
export const CRITICAL_WORK_ORDER_CHECKS: WorkOrderCheckPresentation[] = [
  {
    id: "check-risk-review-critical",
    name: "Risk review",
    score: 91,
    maxScore: 100,
    level: "critical",
    summary: "Critical risk: the change disables idempotency checks on refund submission.",
    analysis:
      "### Summary\n\nThe diff removes the idempotency key from `ProviderClient.refund`. A retry after a network timeout would submit the refund twice.\n\n### Concerns\n\n- Duplicate refunds are unrecoverable without manual reconciliation.\n- No test covers the timeout-then-retry sequence.",
    sourceName: "PR Risk Review",
    updatedAt: minutesAgo(2),
  },
];
