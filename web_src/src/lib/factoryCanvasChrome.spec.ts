import { describe, expect, it } from "vitest";
import { factoryCanvasBackground, factoryEdgePalette } from "./factoryCanvasChrome";

describe("factoryCanvasChrome", () => {
  it("matches Storybook dark canvas dots and background", () => {
    expect(factoryCanvasBackground(true)).toEqual({
      gap: 22,
      size: 1,
      color: "#33312b",
      bgColor: "#14120b",
    });
  });

  it("uses thin slate edges in light mode", () => {
    expect(factoryEdgePalette(false).default).toEqual({ stroke: "#cbd5e1", strokeWidth: 1.5 });
  });
});
