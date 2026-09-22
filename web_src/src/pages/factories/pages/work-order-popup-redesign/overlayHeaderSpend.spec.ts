import { describe, expect, it } from "bun:test";

import { overlayHeaderSpend } from "./overlayHeaderSpend";

describe("overlayHeaderSpend", () => {
  it("keeps saved totals when there is no live overlay", () => {
    expect(overlayHeaderSpend("$0.73", "2.7k tokens", undefined)).toEqual({
      costUsd: "$0.73",
      tokensLabel: "2.7k tokens",
    });
  });

  it("uses live figures when saved totals are still zero", () => {
    expect(overlayHeaderSpend("$0.00", "0 tokens", { tokens: 2100, cents: 45 })).toEqual({
      costUsd: "$0.45",
      tokensLabel: "2.1k tokens",
    });
  });

  it("keeps saved totals when they exceed the live overlay", () => {
    expect(overlayHeaderSpend("$0.73", "2.7k tokens", { tokens: 2100, cents: 45 })).toEqual({
      costUsd: "$0.73",
      tokensLabel: "2.7k tokens",
    });
  });
});
