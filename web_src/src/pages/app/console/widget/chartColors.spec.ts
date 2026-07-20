import { describe, expect, it } from "vitest";

import { CHART_COLOR, DEFAULT_CHART_PALETTE, DEFAULT_DARK_CHART_PALETTE, resolveChartColor } from "./chartColors";

describe("resolveChartColor", () => {
  it("maps passed and failed to emerald and red", () => {
    expect(resolveChartColor("Passed", 3)).toBe(CHART_COLOR.emerald500);
    expect(resolveChartColor("failed", 3)).toBe(CHART_COLOR.red500);
  });

  it("falls back to the default palette by index", () => {
    expect(resolveChartColor("api", 0)).toBe(DEFAULT_CHART_PALETTE[0]);
    expect(resolveChartColor("web", 2)).toBe(DEFAULT_CHART_PALETTE[2]);
    expect(resolveChartColor("wrap", 6)).toBe(DEFAULT_CHART_PALETTE[0]);
  });

  it("uses sky in light mode and indigo in dark mode for series colors", () => {
    expect(resolveChartColor("api", 0, false)).toBe(CHART_COLOR.sky600);
    expect(resolveChartColor("api", 0, true)).toBe(DEFAULT_DARK_CHART_PALETTE[0]);
    expect(DEFAULT_DARK_CHART_PALETTE[0]).toBe("#818cf8");
  });
});
