import type { LineListMetrics } from "./lineListMetrics";

export const LINE_LIST_METRICS_EMPTY = "—";

export function formatSuccessRate(metrics: LineListMetrics | null): string {
  if (!metrics) {
    return LINE_LIST_METRICS_EMPTY;
  }
  return `${Math.round(metrics.successRatePct)}%`;
}

export function formatReworkRate(metrics: LineListMetrics | null): string {
  if (!metrics) {
    return LINE_LIST_METRICS_EMPTY;
  }
  const value = metrics.reworkPerWorkOrder;
  const rounded = value % 1 === 0 ? value.toFixed(0) : value.toFixed(1);
  return `${rounded}x`;
}

export function formatCostPerSuccess(metrics: LineListMetrics | null): string {
  if (!metrics) {
    return LINE_LIST_METRICS_EMPTY;
  }
  return `$${metrics.costPerSuccessUsd.toFixed(2)}`;
}

function signedNumber(value: number, digits: number): string {
  const abs = Math.abs(value).toFixed(digits);
  if (value > 0) {
    return `+${abs}`;
  }
  if (value < 0) {
    return `−${abs}`;
  }
  return digits === 0 ? "0" : (0).toFixed(digits);
}

/** Success-rate change vs the prior window. Example: `+6 pts`. */
export function formatSuccessDelta(metrics: LineListMetrics | null): string {
  if (!metrics) {
    return LINE_LIST_METRICS_EMPTY;
  }
  return `${signedNumber(metrics.successDeltaPts, 0)} pts`;
}

/** Rework change vs the prior window. Example: `−0.2x`. */
export function formatReworkDelta(metrics: LineListMetrics | null): string {
  if (!metrics) {
    return LINE_LIST_METRICS_EMPTY;
  }
  return `${signedNumber(metrics.reworkDelta, 1)}x`;
}

/** Cost change vs the prior window. Example: `−$0.40`. */
export function formatCostDelta(metrics: LineListMetrics | null): string {
  if (!metrics) {
    return LINE_LIST_METRICS_EMPTY;
  }
  const abs = `$${Math.abs(metrics.costDeltaUsd).toFixed(2)}`;
  if (metrics.costDeltaUsd > 0) {
    return `+${abs}`;
  }
  if (metrics.costDeltaUsd < 0) {
    return `−${abs}`;
  }
  return abs;
}

/** Completions per day. Example: `1.4 per day`. */
export function formatThroughput(metrics: LineListMetrics | null): string {
  if (!metrics) {
    return LINE_LIST_METRICS_EMPTY;
  }
  const value = metrics.throughputPerDay;
  const rounded = value % 1 === 0 ? value.toFixed(0) : value.toFixed(1);
  return `${rounded} per day`;
}
