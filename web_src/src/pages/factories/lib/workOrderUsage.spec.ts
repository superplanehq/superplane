import { describe, expect, it } from "vitest";

import { formatWorkOrderExecutionUsage } from "./workOrderUsage";

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
