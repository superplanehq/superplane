import { Position } from "@xyflow/react";
import { describe, expect, it } from "vitest";
import { getBackwardRouteCenterY, getCanvasEdgePath, isBackwardEdge } from "./edgePath";

describe("isBackwardEdge", () => {
  it("detects right-to-left edges that target an earlier node", () => {
    expect(
      isBackwardEdge({
        sourceX: 500,
        sourceY: 100,
        sourcePosition: Position.Right,
        targetX: 100,
        targetY: 50,
        targetPosition: Position.Left,
      }),
    ).toBe(true);
  });

  it("does not treat forward right-to-left edges as backward", () => {
    expect(
      isBackwardEdge({
        sourceX: 100,
        sourceY: 100,
        sourcePosition: Position.Right,
        targetX: 500,
        targetY: 100,
        targetPosition: Position.Left,
      }),
    ).toBe(false);
  });
});

describe("getBackwardRouteCenterY", () => {
  it("routes loop-back edges below both nodes", () => {
    expect(getBackwardRouteCenterY(300, 100)).toBe(380);
  });

  it("routes downward backward edges closer to the target node", () => {
    expect(getBackwardRouteCenterY(120, 420)).toBe(345);
  });

  it("never routes closer than the minimum distance above the target top", () => {
    expect(getBackwardRouteCenterY(300, 420)).toBe(352);
  });
});

describe("getCanvasEdgePath", () => {
  it("uses a bezier path for forward edges", () => {
    const [path] = getCanvasEdgePath({
      sourceX: 100,
      sourceY: 100,
      sourcePosition: Position.Right,
      targetX: 500,
      targetY: 100,
      targetPosition: Position.Left,
    });

    expect(path).toContain("C");
  });

  it("uses a smooth step path for loop-back edges below both nodes", () => {
    const [path] = getCanvasEdgePath({
      sourceX: 500,
      sourceY: 100,
      sourcePosition: Position.Right,
      targetX: 100,
      targetY: 50,
      targetPosition: Position.Left,
    });

    expect(path).not.toContain("C");
    expect(path).toContain("180");
  });

  it("routes downward backward edges through the gap between nodes", () => {
    const [path] = getCanvasEdgePath({
      sourceX: 900,
      sourceY: 120,
      sourcePosition: Position.Right,
      targetX: 200,
      targetY: 420,
      targetPosition: Position.Left,
    });

    expect(path).not.toContain("C");
    expect(path).toContain("345");
  });
});
