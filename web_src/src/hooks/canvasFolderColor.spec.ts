import { describe, expect, it } from "vitest";
import { DEFAULT_CANVAS_FOLDER_COLOR, normalizeCanvasFolderColor } from "./useCanvasData";

describe("normalizeCanvasFolderColor", () => {
  it("maps legacy yellow folders to slate", () => {
    expect(normalizeCanvasFolderColor("yellow")).toBe("slate");
  });

  it("returns supported colors unchanged", () => {
    expect(normalizeCanvasFolderColor("orange")).toBe("orange");
  });

  it("falls back to the default color for unknown values", () => {
    expect(normalizeCanvasFolderColor("unknown")).toBe(DEFAULT_CANVAS_FOLDER_COLOR);
  });
});
