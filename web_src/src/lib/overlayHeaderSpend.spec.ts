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

it.each([
  { state: "measured" as const, tokens: 0, expected: "0 tokens" },
  { state: "pending" as const, tokens: 0, expected: "Waiting for token usage" },
  { state: "unavailable" as const, tokens: 0, expected: "Tokens unavailable" },
  { state: "partial" as const, tokens: 40, expected: "40 tokens (partial)" },
])("shows $state measurement honestly after reloading saved usage", ({ state, tokens, expected }) => {
  expect(overlayHeaderSpend("$0.00", `${tokens} tokens`, { tokens: 0, cents: 0, usageState: state }).tokensLabel).toBe(
    expected,
  );
});
