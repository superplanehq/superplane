import { describe, expect, it } from "vitest";
import {
  CANVAS_FIT_VIEW_INCLUDE_HIDDEN,
  CANVAS_NODE_FOCUS_FIT_VIEW_OPTIONS,
  LIVE_CANVAS_FIT_VIEW_OPTIONS,
  RUN_CANVAS_FIT_VIEW_OPTIONS,
} from "./canvasFitOptions";

describe("canvasFitOptions", () => {
  it("includes hidden nodes in all canvas fitView presets", () => {
    expect(CANVAS_FIT_VIEW_INCLUDE_HIDDEN).toEqual({ includeHiddenNodes: true });
    expect(LIVE_CANVAS_FIT_VIEW_OPTIONS.includeHiddenNodes).toBe(true);
    expect(RUN_CANVAS_FIT_VIEW_OPTIONS.includeHiddenNodes).toBe(true);
    expect(CANVAS_NODE_FOCUS_FIT_VIEW_OPTIONS.includeHiddenNodes).toBe(true);
  });
});
