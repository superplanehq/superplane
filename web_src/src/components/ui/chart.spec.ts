import { describe, expect, it } from "vitest";

import { toChartColorVarName } from "./chartColorVarName";

describe("toChartColorVarName", () => {
  it("slugifies display names for CSS custom properties", () => {
    expect(toChartColorVarName("Claude Haiku 4.5")).toBe("claude-haiku-4-5");
    expect(toChartColorVarName("Passed")).toBe("passed");
    expect(toChartColorVarName("(empty)")).toBe("empty");
  });

  it("returns empty for blank keys", () => {
    expect(toChartColorVarName("   ")).toBe("empty");
  });
});
