import { describe, expect, it } from "vitest";

import { shouldUseFactoryRunLeafLayout } from "./factoryRunLeafLayoutGate";

describe("shouldUseFactoryRunLeafLayout", () => {
  it("enables leaf layout for factory run inspection", () => {
    expect(
      shouldUseFactoryRunLeafLayout({
        factoryEmbed: true,
        isRunInspectionMode: true,
      }),
    ).toBe(true);
  });

  it("enables leaf layout for a factory display preview", () => {
    expect(
      shouldUseFactoryRunLeafLayout({
        factoryEmbed: true,
        isRunInspectionMode: false,
        factoryDisplayLayout: true,
      }),
    ).toBe(true);
  });

  it("stays off for a factory embed that is not a run or a display preview", () => {
    expect(
      shouldUseFactoryRunLeafLayout({
        factoryEmbed: true,
        isRunInspectionMode: false,
      }),
    ).toBe(false);
  });

  it("stays off outside factory embed", () => {
    expect(
      shouldUseFactoryRunLeafLayout({
        factoryEmbed: false,
        isRunInspectionMode: true,
        factoryDisplayLayout: true,
      }),
    ).toBe(false);
  });
});
