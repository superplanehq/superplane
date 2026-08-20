import { describe, expect, it } from "vitest";
import { normalizeRunErrors, shouldShowFactoryCanvasRunErrors } from "./runErrors";

describe("normalizeRunErrors", () => {
  it("returns an empty list when errors are missing", () => {
    expect(normalizeRunErrors(undefined)).toEqual([]);
    expect(normalizeRunErrors(null)).toEqual([]);
    expect(normalizeRunErrors([])).toEqual([]);
  });

  it("drops blank messages", () => {
    expect(normalizeRunErrors(["pipeline failed", "  ", "", undefined, "tests failed"])).toEqual([
      "pipeline failed",
      "tests failed",
    ]);
  });
});

describe("shouldShowFactoryCanvasRunErrors", () => {
  const visible = {
    factoryEmbed: true,
    isRunInspectionMode: true,
    runInspectorOpen: false,
    errorCount: 1,
  };

  it("shows the canvas banner only for a factory run with errors and a closed inspector", () => {
    expect(shouldShowFactoryCanvasRunErrors(visible)).toBe(true);
  });

  it("hides the canvas banner when the inspector already shows the errors", () => {
    expect(shouldShowFactoryCanvasRunErrors({ ...visible, runInspectorOpen: true })).toBe(false);
  });

  it("hides the canvas banner outside factory run inspection", () => {
    expect(shouldShowFactoryCanvasRunErrors({ ...visible, factoryEmbed: false })).toBe(false);
    expect(shouldShowFactoryCanvasRunErrors({ ...visible, isRunInspectionMode: false })).toBe(false);
    expect(shouldShowFactoryCanvasRunErrors({ ...visible, errorCount: 0 })).toBe(false);
  });
});
