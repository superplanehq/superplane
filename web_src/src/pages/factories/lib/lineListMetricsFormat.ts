import type { FactoriesLineMetrics } from "@/api-client";

/** Placeholder shown wherever a line has no metrics data (loading, error, or absent from the response). */
export const LINE_LIST_METRICS_EMPTY = "—";

function parseInt64(value: string | undefined): number {
  if (!value) {
    return 0;
  }
  const parsed = Number(value);
  return Number.isFinite(parsed) ? parsed : 0;
}

export function mergedCount(metrics: FactoriesLineMetrics): number {
  return parseInt64(metrics.mergedCount);
}

export function totalClosedCount(metrics: FactoriesLineMetrics): number {
  return parseInt64(metrics.totalClosedCount);
}

export function costPerSuccessUsd(metrics: FactoriesLineMetrics): number {
  return parseInt64(metrics.costPerSuccessCents) / 100;
}

export function costDeltaUsd(metrics: FactoriesLineMetrics): number {
  return parseInt64(metrics.costDeltaCents) / 100;
}

export function formatSuccessRate(metrics: FactoriesLineMetrics | null | undefined): string {
  if (!metrics) {
    return LINE_LIST_METRICS_EMPTY;
  }
  return `${Math.round(metrics.successRatePct ?? 0)}%`;
}

export function formatThroughput(metrics: FactoriesLineMetrics | null | undefined): string {
  if (!metrics) {
    return LINE_LIST_METRICS_EMPTY;
  }
  return `${(metrics.throughputPerDay ?? 0).toFixed(1)} / day`;
}

export function formatReworkRate(metrics: FactoriesLineMetrics | null | undefined): string {
  if (!metrics) {
    return LINE_LIST_METRICS_EMPTY;
  }
  return `${(metrics.reworkPerWorkOrder ?? 0).toFixed(1)} / order`;
}

export function formatCostPerSuccess(metrics: FactoriesLineMetrics | null | undefined): string {
  if (!metrics) {
    return LINE_LIST_METRICS_EMPTY;
  }
  return `$${costPerSuccessUsd(metrics).toFixed(2)}`;
}

// formatSignedDelta prefixes the *sign* rather than the formatted magnitude,
// so a `$` (or any other leading symbol) formatter reads "-$1.50" instead of
// the more awkward "$-1.50".
function formatSignedDelta(value: number, format: (magnitude: number) => string): string {
  if (value === 0) {
    return LINE_LIST_METRICS_EMPTY;
  }
  const sign = value > 0 ? "+" : "-";
  return `${sign}${format(Math.abs(value))}`;
}

export function formatSuccessDelta(metrics: FactoriesLineMetrics | null | undefined): string {
  if (!metrics) {
    return LINE_LIST_METRICS_EMPTY;
  }
  return formatSignedDelta(metrics.successDeltaPts ?? 0, (value) => `${Math.round(value)} pts`);
}

export function formatReworkDelta(metrics: FactoriesLineMetrics | null | undefined): string {
  if (!metrics) {
    return LINE_LIST_METRICS_EMPTY;
  }
  return formatSignedDelta(metrics.reworkDelta ?? 0, (value) => value.toFixed(1));
}

export function formatCostDelta(metrics: FactoriesLineMetrics | null | undefined): string {
  if (!metrics) {
    return LINE_LIST_METRICS_EMPTY;
  }
  return formatSignedDelta(costDeltaUsd(metrics), (value) => `$${value.toFixed(2)}`);
}
