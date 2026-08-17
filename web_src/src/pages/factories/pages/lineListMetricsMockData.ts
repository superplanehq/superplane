import {
  REFUND_LINE_FEATURE_ID,
  REFUND_LINE_HOTFIX_ID,
  REFUND_LINE_ONBOARDING_ID,
  REFUND_LINE_PLAN_ID,
} from "../__fixtures__/factoryPageResponses";

/**
 * Stand-in line-list metrics until an aggregation API exists.
 *
 * None of these fields exist on the factory-line API yet. This type is the
 * contract a later endpoint should fill.
 *
 * Window: last 30 days.
 */
export type LineListMetrics = {
  /**
   * Share of closed work orders on this line whose pull request merged to main.
   * Numerator is merged work orders. Denominator is closed work orders
   * (merged + not merged). Omit when the line has no closed work orders.
   */
  successRatePct: number;
  /** Merged-to-main work orders in the window (success-rate numerator). */
  mergedCount: number;
  /** Closed work orders in the window (success-rate denominator). */
  totalClosedCount: number;
  /**
   * Average human interventions per work order: steering comments, tweaks,
   * and re-dispatches. 1.0 means one intervention per work order.
   */
  reworkPerWorkOrder: number;
  /**
   * Tracked cost (model tokens + execution compute) divided by merged work
   * orders. Failed and reworked runs raise this number.
   */
  costPerSuccessUsd: number;
  /** Daily success-rate samples in the window, oldest first. 0–100. */
  successTrendPct: number[];
  /** Success-rate points vs the prior 30 days. Positive is better. */
  successDeltaPts: number;
  /** Rework change vs the prior 30 days. Negative is better. */
  reworkDelta: number;
  /** Cost change vs the prior 30 days, in USD. Negative is better. */
  costDeltaUsd: number;
  /**
   * Merged work orders per day in the window. Completions, not closed
   * work orders.
   */
  throughputPerDay: number;
  /** Daily merged-work-order counts in the window, oldest first. */
  throughputTrend: number[];
};

/** Per-line mock values keyed to the Refunds Factory fixture lines. */
export const LINE_LIST_METRICS_BY_ID: Record<string, LineListMetrics | null> = {
  [REFUND_LINE_PLAN_ID]: {
    successRatePct: 82,
    mergedCount: 41,
    totalClosedCount: 50,
    reworkPerWorkOrder: 1.4,
    costPerSuccessUsd: 3.2,
    successTrendPct: [68, 70, 69, 72, 74, 73, 76, 77, 78, 79, 80, 81, 80, 82],
    successDeltaPts: 6,
    reworkDelta: -0.2,
    costDeltaUsd: -0.4,
    throughputPerDay: 1.4,
    throughputTrend: [2, 3, 2, 4, 3, 2, 5, 3, 4, 3, 4, 2, 3, 4],
  },
  [REFUND_LINE_HOTFIX_ID]: {
    successRatePct: 54,
    mergedCount: 13,
    totalClosedCount: 24,
    reworkPerWorkOrder: 3.8,
    costPerSuccessUsd: 11.45,
    successTrendPct: [62, 65, 60, 58, 61, 55, 59, 57, 56, 54, 58, 52, 55, 54],
    successDeltaPts: -8,
    reworkDelta: 0.9,
    costDeltaUsd: 2.1,
    throughputPerDay: 0.4,
    throughputTrend: [1, 0, 2, 1, 0, 1, 0, 2, 1, 0, 1, 1, 0, 2],
  },
  [REFUND_LINE_ONBOARDING_ID]: null,
  [REFUND_LINE_FEATURE_ID]: {
    successRatePct: 71,
    mergedCount: 22,
    totalClosedCount: 31,
    reworkPerWorkOrder: 2.1,
    costPerSuccessUsd: 5.4,
    successTrendPct: [64, 65, 66, 68, 67, 69, 70, 68, 70, 71, 69, 72, 70, 71],
    successDeltaPts: 3,
    reworkDelta: 0.1,
    costDeltaUsd: 0.2,
    throughputPerDay: 0.7,
    throughputTrend: [1, 2, 1, 2, 1, 2, 2, 1, 2, 2, 1, 2, 1, 2],
  },
};

/**
 * Purpose copy for each line. The API has no description field. This copy
 * says when to use the line.
 */
export const LINE_LIST_DESCRIPTION_BY_ID: Record<string, string> = {
  [REFUND_LINE_PLAN_ID]: "Default path for planned product work that needs review before merge.",
  [REFUND_LINE_HOTFIX_ID]: "Production-incident path for urgent fixes.",
  [REFUND_LINE_ONBOARDING_ID]: "First-run path for a new workspace.",
  [REFUND_LINE_FEATURE_ID]: "Feature path that includes a pull request and a CI loop.",
};

export function metricsForLine(lineId: string | undefined): LineListMetrics | null {
  if (!lineId) {
    return null;
  }
  return LINE_LIST_METRICS_BY_ID[lineId] ?? null;
}

export function descriptionForLine(lineId: string | undefined): string | undefined {
  if (!lineId) {
    return undefined;
  }
  return LINE_LIST_DESCRIPTION_BY_ID[lineId];
}
