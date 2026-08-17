import { describe, expect, it } from "vitest";

import {
  formatCostDelta,
  formatCostPerSuccess,
  formatReworkDelta,
  formatReworkRate,
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
  reworkPerWorkOrder: 1.4,
  costPerSuccessUsd: 3.2,
  successTrendPct: [70, 82],
  successDeltaPts: 6,
  reworkDelta: -0.2,
  costDeltaUsd: -0.4,
  throughputPerDay: 1.4,
  throughputTrend: [2, 3, 4],
};

describe("lineListMetricsFormat", () => {
  it("formats success rate, rework, cost, and completions", () => {
    expect(formatSuccessRate(sample)).toBe("82%");
    expect(formatReworkRate(sample)).toBe("1.4x");
    expect(formatCostPerSuccess(sample)).toBe("$3.20");
    expect(formatThroughput(sample)).toBe("1.4 per day");
  });

  it("formats change vs the prior window", () => {
    expect(formatSuccessDelta(sample)).toBe("+6 pts");
    expect(formatReworkDelta(sample)).toBe("−0.2x");
    expect(formatCostDelta(sample)).toBe("−$0.40");
  });

  it("uses an em dash when the line has no closed work orders", () => {
    expect(formatSuccessRate(null)).toBe(LINE_LIST_METRICS_EMPTY);
    expect(formatReworkRate(null)).toBe(LINE_LIST_METRICS_EMPTY);
    expect(formatCostPerSuccess(null)).toBe(LINE_LIST_METRICS_EMPTY);
    expect(formatThroughput(null)).toBe(LINE_LIST_METRICS_EMPTY);
    expect(formatSuccessDelta(null)).toBe(LINE_LIST_METRICS_EMPTY);
  });

  it("drops the decimal on whole rework counts", () => {
    expect(formatReworkRate({ ...sample, reworkPerWorkOrder: 2 })).toBe("2x");
  });
});
