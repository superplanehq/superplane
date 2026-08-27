import type { FactoriesWorkOrderCheck } from "@/api-client";

import {
  CONFIDENCE_CHECK_NAME,
  CONFIDENCE_SCORE_MAX,
  confidenceBandForScore,
  confidenceCheckLevel,
  confidenceSuitabilityAnalysis,
  confidenceSuitabilitySummary,
} from "../lib/confidenceScore";
import { REVIEW_CANDIDATES } from "../pages/onboarding/first-run/reviewCandidateFixtures";
import {
  CLOSED_WORK_ORDER,
  OPEN_WORK_ORDER,
  OPEN_WORK_ORDER_SECONDARY,
  PR_CLOSURE_COMPLETED_WORK_ORDER,
  RUNNING_WORK_ORDER,
} from "./factoryPageResponses";

/**
 * Mock checks for Storybook — scores that dedicated automations attach to a
 * work order (risk review, coverage, confidence). Shaped like the
 * `ListWorkOrderChecks` API response entries. Timestamps are relative to
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

const CONFIDENCE_ANALYSIS = confidenceSuitabilityAnalysis({
  source: "GitHub",
  reasons: [
    "The GitHub issue names a testable retry policy for the refund dispatcher.",
    "The dispatcher and the provider client are mapped in the repository.",
    "The score is not 5 because no test simulates provider degradation.",
  ],
});

const CI_PASSED_ANALYSIS = `### Pipeline

[Semaphore run #4182](https://superplanehq.semaphoreci.com/) on \`fix/refund-retry-policy\`, all 6 blocks green.

The previous run failed on \`Test__RefundDispatcher__RetriesOnProviderTimeout\`; the fix landed in the follow-up commit and the full suite now passes.`;

const CI_FAILED_ANALYSIS = `### Pipeline

[Semaphore run #4190](https://superplanehq.semaphoreci.com/) on \`fix/reconciliation-worker\`, failed in the **Backend tests** block.

\`\`\`
--- FAIL: Test__ReconciliationWorker__SkipsSettledLedgerEntries (0.42s)
    reconciliation_worker_test.go:88: expected 0 dispatched refunds, got 2
\`\`\`

The CI loop is retrying after an automated fix attempt.`;

export const OPEN_WORK_ORDER_CHECKS: FactoriesWorkOrderCheck[] = [
  {
    id: "check-risk-review",
    key: "risk-review",
    name: "Risk score",
    score: 65,
    maxScore: 100,
    level: "LEVEL_CAUTION",
    previousScore: 82,
    summary: "Moderate risk: retry policy changes affect every refund path and remove the backoff cap.",
    analysis: RISK_REVIEW_ANALYSIS,
    automation: { appId: "app-pr-risk-review", appName: "PR Risk Review" },
    runId: "run-risk-review-101",
    updatedAt: minutesAgo(12),
  },
  {
    id: "check-code-coverage",
    key: "code-coverage",
    name: "Code quality",
    score: 82,
    maxScore: 100,
    format: "FORMAT_PERCENT",
    level: "LEVEL_POSITIVE",
    previousScore: 81,
    summary: "Overall coverage rose to 82% (+1.3%), with a small drop in the provider client package.",
    analysis: CODE_COVERAGE_ANALYSIS,
    automation: { appId: "app-coverage-report", appName: "Coverage Report" },
    runId: "run-coverage-101",
    updatedAt: minutesAgo(9),
  },
  {
    id: "check-test-coverage",
    key: "test-coverage",
    name: "Test coverage",
    score: 74,
    maxScore: 100,
    format: "FORMAT_PERCENT",
    level: "LEVEL_NEUTRAL",
    summary: "74% of the new code is covered; gaps concentrate in retry error handling.",
    analysis: TEST_COVERAGE_ANALYSIS,
    automation: { appId: "app-coverage-report", appName: "Coverage Report" },
    runId: "run-coverage-101",
    updatedAt: minutesAgo(9),
  },
  {
    id: "check-confidence",
    key: "confidence",
    name: "Confidence score",
    score: 4,
    maxScore: 5,
    level: "LEVEL_POSITIVE",
    previousScore: 3,
    summary: confidenceSuitabilitySummary("High"),
    analysis: CONFIDENCE_ANALYSIS,
    automation: { appId: "app-line-confidence", appName: "Line Confidence" },
    runId: "run-confidence-101",
    updatedAt: minutesAgo(4),
  },
  // Boolean check — pass/fail rather than a score. The run history reads
  // fail, fail, pass: two red segments, then the current green one.
  {
    id: "check-ci",
    key: "ci",
    name: "CI",
    score: 1,
    maxScore: 1,
    format: "FORMAT_BOOLEAN",
    level: "LEVEL_POSITIVE",
    previousScore: 0,
    recentScores: [0, 0, 1],
    summary: "The full Semaphore pipeline passed after two automated fix attempts.",
    analysis: CI_PASSED_ANALYSIS,
    automation: { appId: "app-ci-loop", appName: "CI Loop" },
    runId: "run-ci-101",
    updatedAt: minutesAgo(3),
  },
];

const VERIFY_STEP_CHECK_KEYS = new Set(["risk-review", "code-coverage"]);

/** Checks the Verify step reports on the line board. */
export const VERIFY_STEP_CHECKS: FactoriesWorkOrderCheck[] = OPEN_WORK_ORDER_CHECKS.filter((check) =>
  VERIFY_STEP_CHECK_KEYS.has(check.key ?? ""),
);

/** Two checks only — the risk review has landed, coverage is still running. */
export const RUNNING_WORK_ORDER_CHECKS: FactoriesWorkOrderCheck[] = [
  {
    id: "check-risk-review-running",
    key: "risk-review",
    name: "Risk score",
    score: 38,
    maxScore: 100,
    level: "LEVEL_NEUTRAL",
    summary: "Low-moderate risk: the change is additive and scoped to one worker.",
    analysis:
      "### Summary\n\nThe change adds a new reconciliation worker without touching existing dispatch paths. Blast radius is limited to the new queue.\n\n### Concerns\n\n- The worker shares a database pool with the dispatcher; a slow query could starve it under load.",
    automation: { appId: "app-pr-risk-review", appName: "PR Risk Review" },
    runId: "run-risk-review-102",
    updatedAt: minutesAgo(25),
  },
  {
    id: "check-confidence-running",
    key: "confidence",
    name: "Confidence score",
    score: 3,
    maxScore: 5,
    level: "LEVEL_NEUTRAL",
    previousScore: 4,
    summary: confidenceSuitabilitySummary("Medium"),
    analysis: confidenceSuitabilityAnalysis({
      source: "GitHub",
      reasons: [
        "The GitHub issue names the refund retry, but the target ledger table is still unclear.",
        "The refund post path is mapped. The agent needed one correction during planning.",
        "Verification has not run yet, so this score can still change.",
      ],
    }),
    automation: { appId: "app-line-confidence", appName: "Line Confidence" },
    runId: "run-confidence-102",
    updatedAt: minutesAgo(11),
  },
  // Failing boolean check — the CI loop is mid-retry on this running order,
  // with two failed runs in the history strip.
  {
    id: "check-ci-running",
    key: "ci",
    name: "CI",
    score: 0,
    maxScore: 1,
    format: "FORMAT_BOOLEAN",
    level: "LEVEL_CRITICAL",
    recentScores: [0, 0],
    summary: "Backend tests failed on the reconciliation worker; the CI loop is retrying.",
    analysis: CI_FAILED_ANALYSIS,
    automation: { appId: "app-ci-loop", appName: "CI Loop" },
    runId: "run-ci-102",
    updatedAt: minutesAgo(6),
  },
];

/** Fallback map for fixtures that do not override `checksByOrderId` —
 * the open order carries the full set, the running order a partial one,
 * and every other order (closed, draft, failed) has none. */
const LEVEL_FOR_CHECK: Record<ReturnType<typeof confidenceCheckLevel>, FactoriesWorkOrderCheck["level"]> = {
  positive: "LEVEL_POSITIVE",
  neutral: "LEVEL_NEUTRAL",
  caution: "LEVEL_CAUTION",
  critical: "LEVEL_CRITICAL",
};

const REVIEW_CANDIDATE_CHECKS_BY_ORDER_ID: Record<string, FactoriesWorkOrderCheck[]> = Object.fromEntries(
  REVIEW_CANDIDATES.map((candidate) => [
    candidate.workOrderId,
    [
      {
        id: `check-confidence-${candidate.workOrderId}`,
        key: "confidence",
        name: CONFIDENCE_CHECK_NAME,
        score: candidate.confidenceScore,
        maxScore: CONFIDENCE_SCORE_MAX,
        level: LEVEL_FOR_CHECK[confidenceCheckLevel(candidate.confidenceScore)],
        summary: confidenceSuitabilitySummary(confidenceBandForScore(candidate.confidenceScore)),
        analysis: confidenceSuitabilityAnalysis({ source: "GitHub", reasons: candidate.reasons }),
        automation: { appId: "app-line-confidence", appName: "Line Confidence" },
        runId: `run-confidence-${candidate.workOrderId}`,
        updatedAt: minutesAgo(4),
      },
    ],
  ]),
);

export const DEFAULT_CHECKS_BY_ORDER_ID: Record<string, FactoriesWorkOrderCheck[]> = {
  [OPEN_WORK_ORDER.id!]: OPEN_WORK_ORDER_CHECKS,
  [OPEN_WORK_ORDER_SECONDARY.id!]: VERIFY_STEP_CHECKS,
  [CLOSED_WORK_ORDER.id!]: OPEN_WORK_ORDER_CHECKS,
  [PR_CLOSURE_COMPLETED_WORK_ORDER.id!]: VERIFY_STEP_CHECKS,
  [RUNNING_WORK_ORDER.id!]: RUNNING_WORK_ORDER_CHECKS,
  "wo-failed-refunds": VERIFY_STEP_CHECKS,
  "wo-board-done-rejected": VERIFY_STEP_CHECKS,
  "wo-board-done-canceled": VERIFY_STEP_CHECKS,
  ...REVIEW_CANDIDATE_CHECKS_BY_ORDER_ID,
};

/** A single critical check — the smallest interesting state. */
export const CRITICAL_WORK_ORDER_CHECKS: FactoriesWorkOrderCheck[] = [
  {
    id: "check-risk-review-critical",
    key: "risk-review",
    name: "Risk score",
    score: 91,
    maxScore: 100,
    level: "LEVEL_CRITICAL",
    summary: "Critical risk: the change disables idempotency checks on refund submission.",
    analysis:
      "### Summary\n\nThe diff removes the idempotency key from `ProviderClient.refund`. A retry after a network timeout would submit the refund twice.\n\n### Concerns\n\n- Duplicate refunds are unrecoverable without manual reconciliation.\n- No test covers the timeout-then-retry sequence.",
    automation: { appName: "PR Risk Review" },
    updatedAt: minutesAgo(2),
  },
];
