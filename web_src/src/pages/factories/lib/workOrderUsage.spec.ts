import { describe, expect, it } from "vitest";

import { firstPositiveWorkOrderMetric, formatDurationSeconds, formatWorkOrderExecutionUsage } from "./workOrderUsage";

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

describe("formatDurationSeconds", () => {
  it("shows seconds only under a minute", () => {
    expect(formatDurationSeconds(45)).toBe("45 s");
  });

  it("shows minutes at exactly 60 seconds", () => {
    expect(formatDurationSeconds(60)).toBe("1 min");
  });

  it("shows minutes and seconds remainder", () => {
    expect(formatDurationSeconds(90)).toBe("1 min 30 s");
  });

  it("shows hours at exactly 3600 seconds", () => {
    expect(formatDurationSeconds(3600)).toBe("1 h");
  });

  it("shows hours and minutes remainder", () => {
    expect(formatDurationSeconds(3900)).toBe("1 h 5 min");
    expect(formatDurationSeconds(5400)).toBe("1 h 30 min");
  });

  it("keeps hours growing for large values", () => {
    expect(formatDurationSeconds(7200)).toBe("2 h");
  });
});
