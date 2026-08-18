import { describe, expect, it } from "vitest";

import { lineListMetricsFromApi } from "./lineListMetrics";

describe("lineListMetricsFromApi", () => {
  it("returns null when the API omitted metrics", () => {
    expect(lineListMetricsFromApi(undefined)).toBeNull();
  });

  it("fills missing numeric fields with zero", () => {
    expect(lineListMetricsFromApi({ successRatePct: 82, mergedCount: 41 })).toEqual({
      successRatePct: 82,
      mergedCount: 41,
      totalClosedCount: 0,
      reworkPerWorkOrder: 0,
      costPerSuccessUsd: 0,
      successTrendPct: [],
      successDeltaPts: 0,
      reworkDelta: 0,
      costDeltaUsd: 0,
      throughputPerDay: 0,
      throughputTrend: [],
    });
  });
});
