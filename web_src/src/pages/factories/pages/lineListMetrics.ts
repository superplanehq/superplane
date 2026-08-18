import type { FactoriesFactoryLineMetrics } from "@/api-client";

/**
 * Line-list performance metrics for the last 30 days.
 * Daily trend series cover the last 14 days, oldest first.
 */
export type LineListMetrics = {
  successRatePct: number;
  mergedCount: number;
  totalClosedCount: number;
  reworkPerWorkOrder: number;
  costPerSuccessUsd: number;
  successTrendPct: number[];
  successDeltaPts: number;
  reworkDelta: number;
  costDeltaUsd: number;
  throughputPerDay: number;
  throughputTrend: number[];
};

export function lineListMetricsFromApi(metrics: FactoriesFactoryLineMetrics | undefined): LineListMetrics | null {
  if (!metrics) {
    return null;
  }
  return {
    successRatePct: metrics.successRatePct ?? 0,
    mergedCount: metrics.mergedCount ?? 0,
    totalClosedCount: metrics.totalClosedCount ?? 0,
    reworkPerWorkOrder: metrics.reworkPerWorkOrder ?? 0,
    costPerSuccessUsd: metrics.costPerSuccessUsd ?? 0,
    successTrendPct: metrics.successTrendPct ?? [],
    successDeltaPts: metrics.successDeltaPts ?? 0,
    reworkDelta: metrics.reworkDelta ?? 0,
    costDeltaUsd: metrics.costDeltaUsd ?? 0,
    throughputPerDay: metrics.throughputPerDay ?? 0,
    throughputTrend: metrics.throughputTrend ?? [],
  };
}
