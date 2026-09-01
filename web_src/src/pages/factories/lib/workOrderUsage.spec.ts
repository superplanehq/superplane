import { describe, expect, it } from "vitest";

import { firstPositiveWorkOrderMetric, formatWorkOrderExecutionUsage } from "./workOrderUsage";

describe("firstPositiveWorkOrderMetric", () => {
  it("skips zero and empty values", () => {
    expect(firstPositiveWorkOrderMetric("0", undefined, "45")).toBe("45");
    expect(firstPositiveWorkOrderMetric(0, "1200")).toBe("1200");
    expect(firstPositiveWorkOrderMetric("0", "0")).toBeUndefined();
  });
});

describe("formatWorkOrderExecutionUsage", () => {
  it("sums usage from every execution in the dispatch", () => {
    expect(
      formatWorkOrderExecutionUsage([
        { totalTokens: "1200", costCents: "45", durationSeconds: "12" },
        { totalTokens: "1800", costCents: "105", durationSeconds: "8" },
      ]),
    ).toBe("$1.50 · 3k tokens · 20 s");
  });

  it("ignores absent and invalid usage", () => {
    expect(formatWorkOrderExecutionUsage([{ totalTokens: "invalid" }, {}])).toBeNull();
  });
});
