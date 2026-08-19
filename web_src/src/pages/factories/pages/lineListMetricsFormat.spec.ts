import { describe, expect, it } from "vitest";

import {
  formatCostDelta,
  formatCostPerSuccess,
  formatDuration,
  formatDurationDelta,
  formatSuccessDelta,
  formatSuccessRate,
  formatThroughput,
  LINE_LIST_METRICS_EMPTY,
} from "./lineListMetricsFormat";
import type { LineListMetrics } from "./lineListMetricsMockData";

const sample: LineListMetrics = {
  successRatePct: 82,
  mergedCount: 41,
  totalClosedCount: 50,
  durationMinutes: 15,
  costPerSuccessUsd: 3.2,
  successTrendPct: [70, 82],
  successDeltaPts: 6,
  durationDeltaMinutes: -2,
  costDeltaUsd: -0.4,
  throughputPerDay: 1.4,
  throughputTrend: [2, 3, 4],
};

describe("lineListMetricsFormat", () => {
  it("formats success rate, duration, cost, and completions", () => {
    expect(formatSuccessRate(sample)).toBe("82%");
    expect(formatDuration(sample)).toBe("15m");
    expect(formatCostPerSuccess(sample)).toBe("$3.20");
    expect(formatThroughput(sample)).toBe("1.4 per day");
  });

  it("formats change vs the prior window", () => {
    expect(formatSuccessDelta(sample)).toBe("+6 pts");
    expect(formatDurationDelta(sample)).toBe("−2m");
    expect(formatCostDelta(sample)).toBe("−$0.40");
  });

  it("uses an em dash when the line has no closed work orders", () => {
    expect(formatSuccessRate(null)).toBe(LINE_LIST_METRICS_EMPTY);
    expect(formatDuration(null)).toBe(LINE_LIST_METRICS_EMPTY);
    expect(formatCostPerSuccess(null)).toBe(LINE_LIST_METRICS_EMPTY);
    expect(formatThroughput(null)).toBe(LINE_LIST_METRICS_EMPTY);
    expect(formatSuccessDelta(null)).toBe(LINE_LIST_METRICS_EMPTY);
  });

  it("formats 60 minutes or more as hours", () => {
    expect(formatDuration({ ...sample, durationMinutes: 90 })).toBe("1.5h");
  });

  it("dashes duration and cost when those fields are omitted", () => {
    const withoutOptional: LineListMetrics = {
      successRatePct: 0,
      mergedCount: 0,
      totalClosedCount: 2,
      successTrendPct: [0, 0],
      throughputPerDay: 0,
      throughputTrend: [0, 0],
    };
    expect(formatDuration(withoutOptional)).toBe(LINE_LIST_METRICS_EMPTY);
    expect(formatCostPerSuccess(withoutOptional)).toBe(LINE_LIST_METRICS_EMPTY);
    expect(formatSuccessDelta(withoutOptional)).toBe("");
    expect(formatDurationDelta(withoutOptional)).toBe("");
    expect(formatCostDelta(withoutOptional)).toBe("");
    expect(formatSuccessRate(withoutOptional)).toBe("0%");
  });
});
