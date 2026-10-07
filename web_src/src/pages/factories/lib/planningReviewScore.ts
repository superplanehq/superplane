import { CLARITY_CHECK_KEY, CONFIDENCE_CHECK_NAME, clampConfidenceScore } from "./confidenceScore";
import type { WorkOrderCheckLevel, WorkOrderCheckPresentation } from "./workOrderChecks";

export const PLANNING_COMPLEXITY_CHECK_KEY = "complexity";
export const PLANNING_VERIFIABILITY_CHECK_KEY = "verifiability";

/** Review sub-parameters use a 1–3 scale: 3 is good, 2 is partial, 1 is bad. */
export const PLANNING_REVIEW_SCORE_MAX = 3;

export const PLANNING_REVIEW_METRIC_KEYS = [
  CLARITY_CHECK_KEY,
  PLANNING_COMPLEXITY_CHECK_KEY,
  PLANNING_VERIFIABILITY_CHECK_KEY,
] as const;

const LEVEL_SEVERITY: Record<WorkOrderCheckLevel, number> = {
  positive: 0,
  neutral: 1,
  caution: 2,
  critical: 3,
};

const METRIC_LABEL: Record<(typeof PLANNING_REVIEW_METRIC_KEYS)[number], string> = {
  [CLARITY_CHECK_KEY]: "Clarity",
  [PLANNING_COMPLEXITY_CHECK_KEY]: "Complexity",
  [PLANNING_VERIFIABILITY_CHECK_KEY]: "Verifiability",
};

export type PlanningReviewMetric = WorkOrderCheckPresentation & { key: string };

export function isPlanningReviewMetric(check: Pick<WorkOrderCheckPresentation, "key">): boolean {
  return check.key != null && (PLANNING_REVIEW_METRIC_KEYS as readonly string[]).includes(check.key);
}

/** Checks stored on the old 1–5 scale do not form a review until re-analysis. */
function onReviewScale(check: Pick<WorkOrderCheckPresentation, "maxScore">): boolean {
  return check.maxScore <= PLANNING_REVIEW_SCORE_MAX;
}

export function hasPlanningReviewScores(
  checks: Pick<WorkOrderCheckPresentation, "key" | "maxScore">[] | undefined,
): boolean {
  const keys = new Set(
    (checks ?? [])
      .filter(onReviewScale)
      .map((check) => check.key)
      .filter(Boolean),
  );
  return PLANNING_REVIEW_METRIC_KEYS.every((key) => keys.has(key));
}

export function planningReviewMetrics(checks: WorkOrderCheckPresentation[] | undefined): PlanningReviewMetric[] {
  return PLANNING_REVIEW_METRIC_KEYS.flatMap((key) => {
    const check = (checks ?? []).find((entry) => entry.key === key && onReviewScale(entry));
    if (!check) {
      return [];
    }
    return [{ ...check, key, name: METRIC_LABEL[key] }];
  });
}

export function planningReviewHeadline(metrics: PlanningReviewMetric[]): WorkOrderCheckPresentation | undefined {
  if (metrics.length === 0) {
    return undefined;
  }
  const weakest = metrics.reduce((lowest, check) => Math.min(lowest, check.score), PLANNING_REVIEW_SCORE_MAX);
  return {
    id: "planning-review-confidence",
    key: "confidence",
    name: CONFIDENCE_CHECK_NAME,
    score: clampConfidenceScore(weakest, PLANNING_REVIEW_SCORE_MAX),
    maxScore: PLANNING_REVIEW_SCORE_MAX,
    format: "fraction",
    level: weakestLevel(metrics),
    summary: weakestMetric(metrics)?.summary,
  };
}

/** Same bands as the review agent: 3 healthy, 2 caution, 1 critical. */
export function planningReviewLevel(score: number): WorkOrderCheckLevel {
  if (score >= PLANNING_REVIEW_SCORE_MAX) {
    return "positive";
  }
  if (score >= 2) {
    return "caution";
  }
  return "critical";
}

/** At the top score there is nothing to fix, so the chip shows the summary instead of the per-check drawer. */
export function planningReviewAtMax(metrics: Pick<WorkOrderCheckPresentation, "score">[]): boolean {
  return metrics.length > 0 && metrics.every((metric) => metric.score >= PLANNING_REVIEW_SCORE_MAX);
}

export function planningReviewFromChecks(checks: WorkOrderCheckPresentation[] | undefined): {
  headline: WorkOrderCheckPresentation;
  metrics: PlanningReviewMetric[];
} | null {
  if (!hasPlanningReviewScores(checks)) {
    return null;
  }
  const metrics = planningReviewMetrics(checks);
  const headline = planningReviewHeadline(metrics);
  if (!headline) {
    return null;
  }
  return { headline, metrics };
}

function weakestMetric(metrics: PlanningReviewMetric[]): PlanningReviewMetric | undefined {
  return metrics.reduce<PlanningReviewMetric | undefined>((weakest, check) => {
    if (!weakest || check.score < weakest.score) {
      return check;
    }
    return weakest;
  }, undefined);
}

function weakestLevel(metrics: PlanningReviewMetric[]): WorkOrderCheckLevel {
  return metrics.reduce<WorkOrderCheckLevel>((weakest, check) => {
    return LEVEL_SEVERITY[check.level] > LEVEL_SEVERITY[weakest] ? check.level : weakest;
  }, "positive");
}
