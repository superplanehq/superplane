import { describe, expect, it } from "vitest";
import {
  CANVAS_FIT_VIEW_INCLUDE_HIDDEN,
  CANVAS_NODE_FOCUS_FIT_VIEW_OPTIONS,
  FACTORY_CONFIGURE_FIT_VIEW_OPTIONS,
  LIVE_CANVAS_FIT_VIEW_OPTIONS,
  NATIVE_ZOOM_FIT_VIEW_OPTIONS,
  RUN_CANVAS_FIT_VIEW_OPTIONS,
  resolveInitialCanvasFitViewOptions,
} from "./canvasFitOptions";

describe("canvasFitOptions", () => {
  it("includes hidden nodes in all canvas fitView presets", () => {
    expect(CANVAS_FIT_VIEW_INCLUDE_HIDDEN).toEqual({ includeHiddenNodes: true });
    expect(LIVE_CANVAS_FIT_VIEW_OPTIONS.includeHiddenNodes).toBe(true);
    expect(RUN_CANVAS_FIT_VIEW_OPTIONS.includeHiddenNodes).toBe(true);
    expect(CANVAS_NODE_FOCUS_FIT_VIEW_OPTIONS.includeHiddenNodes).toBe(true);
  });

  it("does not clamp run participant fitting to a minimum zoom", () => {
    expect(RUN_CANVAS_FIT_VIEW_OPTIONS).not.toHaveProperty("minZoom");
  });

  it("locks Factory Configure fitting to 100% zoom", () => {
    expect(FACTORY_CONFIGURE_FIT_VIEW_OPTIONS.minZoom).toBe(1);
    expect(FACTORY_CONFIGURE_FIT_VIEW_OPTIONS.maxZoom).toBe(1);
    expect(FACTORY_CONFIGURE_FIT_VIEW_OPTIONS.includeHiddenNodes).toBe(true);
    expect(FACTORY_CONFIGURE_FIT_VIEW_OPTIONS).toBe(NATIVE_ZOOM_FIT_VIEW_OPTIONS);
  });

  it("locks the first-load fit to 100% zoom when the preview asks for native zoom", () => {
    expect(resolveInitialCanvasFitViewOptions(true)).toBe(NATIVE_ZOOM_FIT_VIEW_OPTIONS);
    expect(resolveInitialCanvasFitViewOptions(false)).toBe(LIVE_CANVAS_FIT_VIEW_OPTIONS);
  });
});
