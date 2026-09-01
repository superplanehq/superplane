import { describe, expect, it } from "vitest";

import {
  compactCanvasFitKey,
  compactCanvasNodeFocusRequest,
  COMPACT_CANVAS_NODE_FOCUS_FIT_VIEW_OPTIONS,
  shouldFitCompactCanvas,
} from "./compactCanvasFit";

describe("compactCanvasFitKey", () => {
  it("uses empty when there are no nodes", () => {
    expect(compactCanvasFitKey([])).toBe("empty");
  });

  it("joins node ids in a stable order", () => {
    expect(compactCanvasFitKey(["step-1", "kickoff"])).toBe("kickoff|step-1");
    expect(compactCanvasFitKey(["kickoff", "step-1"])).toBe("kickoff|step-1");
  });

  it("fits only after live nodes arrive", () => {
    expect(shouldFitCompactCanvas("empty")).toBe(false);
    expect(shouldFitCompactCanvas("kickoff|step-1")).toBe(true);
  });
});

describe("compactCanvasNodeFocusRequest", () => {
  it("frames the selected node so the compact canvas zooms to it", () => {
    const node = { id: "create-pr" };
    expect(compactCanvasNodeFocusRequest(node)).toEqual({
      nodes: [node],
      ...COMPACT_CANVAS_NODE_FOCUS_FIT_VIEW_OPTIONS,
    });
    expect(COMPACT_CANVAS_NODE_FOCUS_FIT_VIEW_OPTIONS.maxZoom).toBe(1.2);
    expect(COMPACT_CANVAS_NODE_FOCUS_FIT_VIEW_OPTIONS.duration).toBe(400);
  });

  it("returns null when the selected node is not on the canvas", () => {
    expect(compactCanvasNodeFocusRequest(undefined)).toBeNull();
  });
});
