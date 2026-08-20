import { describe, expect, it } from "vitest";

import { toLineListMetrics } from "./lineListMetricsMockData";

describe("toLineListMetrics", () => {
  it("returns null when metrics are absent", () => {
    expect(toLineListMetrics(undefined)).toBeNull();
    expect(toLineListMetrics(null)).toBeNull();
  });

  it("fills omitted counts and omits optional duration and cost", () => {
    const mapped = toLineListMetrics({
      successRatePct: 50,
      mergedCount: 1,
      totalClosedCount: 2,
      throughputPerDay: 0.1,
    });
    expect(mapped?.successRatePct).toBe(50);
    expect(mapped?.durationMinutes).toBeUndefined();
    expect(mapped?.costPerSuccessUsd).toBeUndefined();
    expect(mapped?.successTrendPct).toEqual([]);
  });
});
