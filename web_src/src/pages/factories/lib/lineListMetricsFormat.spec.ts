import { describe, expect, it } from "vitest";

import type { FactoriesLineMetrics } from "@/api-client";

import {
  costDeltaUsd,
  costPerSuccessUsd,
  formatCostDelta,
  formatCostPerSuccess,
  formatReworkDelta,
  formatReworkRate,
  formatSuccessDelta,
  formatSuccessRate,
  formatThroughput,
  LINE_LIST_METRICS_EMPTY,
  mergedCount,
  totalClosedCount,
} from "./lineListMetricsFormat";

const metrics: FactoriesLineMetrics = {
  lineId: "line-1",
  successRatePct: 92.4,
  mergedCount: "23",
  totalClosedCount: "25",
  reworkPerWorkOrder: 1.25,
  costPerSuccessCents: "1234",
  successTrendPct: [80, 85, 90],
  successDeltaPts: 4.2,
  reworkDelta: -0.3,
  costDeltaCents: "-150",
  throughputPerDay: 0.77,
  throughputTrend: [1, 2, 3],
};

describe("lineListMetricsFormat", () => {
  it("parses int64-as-string counters", () => {
    expect(mergedCount(metrics)).toBe(23);
    expect(totalClosedCount(metrics)).toBe(25);
    expect(costPerSuccessUsd(metrics)).toBeCloseTo(12.34);
    expect(costDeltaUsd(metrics)).toBeCloseTo(-1.5);
  });

  it("treats a missing counter as 0", () => {
    expect(mergedCount({})).toBe(0);
    expect(costPerSuccessUsd({})).toBe(0);
  });

  it("formats the success rate rounded to a whole percent", () => {
    expect(formatSuccessRate(metrics)).toBe("92%");
  });

  it("formats throughput per day", () => {
    expect(formatThroughput(metrics)).toBe("0.8 / day");
  });

  it("formats rework per work order", () => {
    expect(formatReworkRate(metrics)).toBe("1.3 / order");
  });

  it("formats cost per success in dollars", () => {
    expect(formatCostPerSuccess(metrics)).toBe("$12.34");
  });

  it("signs positive and negative deltas", () => {
    expect(formatSuccessDelta(metrics)).toBe("+4 pts");
    expect(formatReworkDelta(metrics)).toBe("-0.3");
    expect(formatCostDelta(metrics)).toBe("-$1.50");
  });

  it("renders the empty placeholder for a zero delta and for missing metrics", () => {
    expect(formatSuccessDelta({ ...metrics, successDeltaPts: 0 })).toBe(LINE_LIST_METRICS_EMPTY);
    expect(formatSuccessRate(null)).toBe(LINE_LIST_METRICS_EMPTY);
    expect(formatCostPerSuccess(undefined)).toBe(LINE_LIST_METRICS_EMPTY);
  });
});
