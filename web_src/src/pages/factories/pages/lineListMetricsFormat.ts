import type { LineListMetrics } from "./lineListMetricsMockData";

export const LINE_LIST_METRICS_EMPTY = "—";

export function formatSuccessRate(metrics: LineListMetrics | null): string {
  if (!metrics) {
    return LINE_LIST_METRICS_EMPTY;
  }
  return `${Math.round(metrics.successRatePct)}%`;
}

/** Median cycle time. Example: `15m`. */
export function formatDuration(metrics: LineListMetrics | null): string {
  if (!metrics || metrics.durationMinutes == null) {
    return LINE_LIST_METRICS_EMPTY;
  }
  return compactDurationMinutes(metrics.durationMinutes);
}

export function formatCostPerSuccess(metrics: LineListMetrics | null): string {
  if (!metrics || metrics.costPerSuccessUsd == null) {
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
  if (metrics.successDeltaPts == null) {
    return "";
  }
  return `${signedNumber(metrics.successDeltaPts, 0)} pts`;
}

/** Duration change vs the prior window. Example: `−2m`. */
export function formatDurationDelta(metrics: LineListMetrics | null): string {
  if (!metrics) {
    return LINE_LIST_METRICS_EMPTY;
  }
  if (metrics.durationDeltaMinutes == null) {
    return "";
  }
  const minutes = metrics.durationDeltaMinutes;
  const formatted = compactDurationMinutes(Math.abs(minutes));
  if (minutes > 0) {
    return `+${formatted}`;
  }
  if (minutes < 0) {
    return `−${formatted}`;
  }
  return formatted;
}

/** Cost change vs the prior window. Example: `−$0.40`. */
export function formatCostDelta(metrics: LineListMetrics | null): string {
  if (!metrics) {
    return LINE_LIST_METRICS_EMPTY;
  }
  if (metrics.costDeltaUsd == null) {
    return "";
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

/** Compact minutes as `15m`, hours when 60 or more, days when 48 hours or more. */
function compactDurationMinutes(minutes: number): string {
  if (minutes < 60) {
    return `${Math.round(minutes)}m`;
  }
  const hours = minutes / 60;
  if (hours < 48) {
    return `${hours % 1 === 0 ? hours.toFixed(0) : hours.toFixed(1)}h`;
  }
  const days = hours / 24;
  return `${days % 1 === 0 ? days.toFixed(0) : days.toFixed(1)}d`;
}
