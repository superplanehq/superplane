import { describe, expect, it } from "bun:test";

import { shouldUseFactoryRunLeafLayout } from "./factoryRunLeafLayoutGate";

describe("shouldUseFactoryRunLeafLayout", () => {
  it("keeps the saved layout for run inspection, display previews, and edit canvases", () => {
    expect(
      shouldUseFactoryRunLeafLayout({
        factoryEmbed: true,
        isRunInspectionMode: true,
      }),
    ).toBe(false);
    expect(
      shouldUseFactoryRunLeafLayout({
        factoryEmbed: true,
        isRunInspectionMode: false,
        factoryDisplayLayout: true,
      }),
    ).toBe(false);
    expect(
      shouldUseFactoryRunLeafLayout({
        factoryEmbed: false,
        isRunInspectionMode: true,
        factoryDisplayLayout: true,
      }),
    ).toBe(false);
  });
});
