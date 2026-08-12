import { describe, expect, it } from "vitest";

import { formatWorkOrderExecutionUsage } from "./workOrderUsage";

describe("formatWorkOrderExecutionUsage", () => {
  it("sums usage from every execution in the dispatch", () => {
    expect(
      formatWorkOrderExecutionUsage([
        { totalTokens: "1200", costCents: "45" },
        { totalTokens: "1800", costCents: "105" },
      ]),
    ).toBe("3k tokens · $1.50");
  });

  it("ignores absent and invalid usage", () => {
    expect(formatWorkOrderExecutionUsage([{ totalTokens: "invalid" }, {}])).toBeNull();
  });
});
