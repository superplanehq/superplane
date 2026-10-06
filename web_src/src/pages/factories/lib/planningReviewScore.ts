import {
  CLARITY_CHECK_KEY,
  CONFIDENCE_CHECK_NAME,
  CONFIDENCE_SCORE_MAX,
  clampConfidenceScore,
} from "./confidenceScore";
import type { WorkOrderCheckLevel, WorkOrderCheckPresentation } from "./workOrderChecks";

export const PLANNING_COMPLEXITY_CHECK_KEY = "complexity";
export const PLANNING_VERIFIABILITY_CHECK_KEY = "verifiability";

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

export function hasPlanningReviewScores(checks: Pick<WorkOrderCheckPresentation, "key">[] | undefined): boolean {
  const keys = new Set((checks ?? []).map((check) => check.key).filter(Boolean));
  return PLANNING_REVIEW_METRIC_KEYS.every((key) => keys.has(key));
}

export function planningReviewMetrics(checks: WorkOrderCheckPresentation[] | undefined): PlanningReviewMetric[] {
  return PLANNING_REVIEW_METRIC_KEYS.flatMap((key) => {
    const check = (checks ?? []).find((entry) => entry.key === key);
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
  const weakest = metrics.reduce((lowest, check) => Math.min(lowest, check.score), CONFIDENCE_SCORE_MAX);
  return {
    id: "planning-review-confidence",
    key: "confidence",
    name: CONFIDENCE_CHECK_NAME,
    score: clampConfidenceScore(weakest),
    maxScore: CONFIDENCE_SCORE_MAX,
    format: "fraction",
    level: weakestLevel(metrics),
    summary: weakestMetric(metrics)?.summary,
  };
}

/** Same bands as the review agent: 4–5 healthy, 3 caution, 1–2 critical. */
export function planningReviewLevel(score: number): WorkOrderCheckLevel {
  if (score >= 4) {
    return "positive";
  }
  if (score >= 3) {
    return "caution";
  }
  return "critical";
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
