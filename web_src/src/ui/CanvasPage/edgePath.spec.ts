import { Position } from "@xyflow/react";
import { describe, expect, it } from "vitest";
import {
  doesEdgePathIntersectRect,
  estimatePathLength,
  findTouchContrastEdgeIds,
  findTouchingEdgeIds,
  getBackwardRouteCenterY,
  getCanvasEdgePath,
  getFlowArrowPoints,
  getPointAlongPath,
  getRightGutterEdgePath,
  getUpwardBackwardGutterY,
  isBackwardEdge,
  isVerticalFlowEdge,
  pathsTouch,
} from "./edgePath";

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

describe("getUpwardBackwardGutterY", () => {
  it("places the gutter just below the target row", () => {
    expect(getUpwardBackwardGutterY(268, 118)).toBe(388);
  });
});

describe("getBackwardRouteCenterY", () => {
  it("routes same-row loop-back edges below both nodes", () => {
    expect(getBackwardRouteCenterY(200, 180)).toBe(280);
  });

  it("routes downward backward edges closer to the target node", () => {
    expect(getBackwardRouteCenterY(120, 420)).toBe(345);
  });

  it("never routes closer than the minimum distance above the target top", () => {
    expect(getBackwardRouteCenterY(300, 420)).toBe(352);
  });
});

describe("isVerticalFlowEdge", () => {
  it("detects top-to-bottom handle pairs", () => {
    expect(
      isVerticalFlowEdge({
        sourcePosition: Position.Bottom,
        targetPosition: Position.Top,
      }),
    ).toBe(true);
  });

  it("rejects horizontal handle pairs", () => {
    expect(
      isVerticalFlowEdge({
        sourcePosition: Position.Right,
        targetPosition: Position.Left,
      }),
    ).toBe(false);
  });
});

describe("getCanvasEdgePath", () => {
  it("uses a bezier path for forward horizontal edges", () => {
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

  it("uses a bezier path for forward vertical (factory) edges", () => {
    const [path] = getCanvasEdgePath({
      sourceX: 100,
      sourceY: 100,
      sourcePosition: Position.Bottom,
      targetX: 100,
      targetY: 400,
      targetPosition: Position.Top,
    });

    expect(path).toContain("C");
  });

  it("uses rounded bends for vertical factory loop-back edges", () => {
    const [path] = getCanvasEdgePath({
      sourceX: 200,
      sourceY: 400,
      sourcePosition: Position.Bottom,
      targetX: 200,
      targetY: 100,
      targetPosition: Position.Top,
    });

    // Soft corners (Q) — not sharp rectangle joins.
    expect(path).toContain("Q");
  });

  it("routes upward factory loops left of the target without a U-hook above it", () => {
    const [path] = getCanvasEdgePath({
      sourceX: 496,
      sourceY: 520,
      sourcePosition: Position.Bottom,
      targetX: 260,
      targetY: 200,
      targetPosition: Position.Top,
    });

    expect(path).toContain("Q");
    // Left gutter clears the Loop card (target center 260 − ≥160).
    expect(path).toContain("100");
    // Ends at the Top handle — no segment above targetY.
    expect(path).toContain("260 200");
    expect(path).not.toMatch(/260,\s*1\d\d/);
  });

  it("uses a smooth step path for same-row loop-back edges below both nodes", () => {
    const [path] = getCanvasEdgePath({
      sourceX: 500,
      sourceY: 200,
      sourcePosition: Position.Right,
      targetX: 100,
      targetY: 190,
      targetPosition: Position.Left,
    });

    expect(path).not.toContain("C");
    expect(path).toContain("280");
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

  it("routes upward backward edges right, through the gutter, then up to the target", () => {
    const [path] = getCanvasEdgePath({
      sourceX: 900,
      sourceY: 268,
      sourcePosition: Position.Right,
      targetX: 200,
      targetY: 118,
      targetPosition: Position.Left,
    });

    expect(path).not.toContain("C");
    expect(path).toContain("924,268");
    expect(path).toContain("388");
    expect(path).toContain("200 118");
  });

  it("uses a right gutter path for long cross-column factory edges", () => {
    const [path] = getCanvasEdgePath({
      sourceX: 520,
      sourceY: 100,
      sourcePosition: Position.Bottom,
      targetX: 120,
      targetY: 500,
      targetPosition: Position.Top,
      routeGutterX: 900,
    });

    expect(path).toContain("900");
    expect(path).toContain("Q");
  });

  it("keeps gutter paths clear of a blocking node between columns", () => {
    const [path] = getRightGutterEdgePath({
      sourceX: 520,
      sourceY: 120,
      sourcePosition: Position.Bottom,
      targetX: 120,
      targetY: 520,
      targetPosition: Position.Top,
      routeGutterX: 900,
    });

    const blockingNode = { x: 200, y: 250, width: 280, height: 104 };
    expect(doesEdgePathIntersectRect(path, blockingNode)).toBe(false);
  });

  it("keeps short vertical edges as bezier when no gutter is set", () => {
    const [path] = getCanvasEdgePath({
      sourceX: 120,
      sourceY: 100,
      sourcePosition: Position.Bottom,
      targetX: 120,
      targetY: 300,
      targetPosition: Position.Top,
    });

    expect(path).toContain("C");
  });

  it("adds sparse flow chevrons on long edges and skips short ones", () => {
    const [longPath] = getRightGutterEdgePath({
      sourceX: 520,
      sourceY: 120,
      sourcePosition: Position.Bottom,
      targetX: 120,
      targetY: 720,
      targetPosition: Position.Top,
      routeGutterX: 900,
    });
    const shortPath = "M 0 0 L 0 80";

    expect(estimatePathLength(longPath)).toBeGreaterThan(320);
    const arrows = getFlowArrowPoints(longPath);
    expect(arrows.length).toBeGreaterThan(0);
    expect(arrows.length).toBeLessThanOrEqual(3);
    expect(getFlowArrowPoints(shortPath)).toEqual([]);
  });

  it("keeps near-source label points on a bezier stroke (not on control-point chords)", () => {
    const [path, midX, midY] = getCanvasEdgePath({
      sourceX: 100,
      sourceY: 100,
      sourcePosition: Position.Right,
      targetX: 300,
      targetY: 200,
      targetPosition: Position.Left,
    });

    const nearSource = getPointAlongPath(path, 0.16)!;
    const midpoint = getPointAlongPath(path, 0.5)!;

    // On-curve midpoint should stay close to xyflow's label point.
    expect(Math.hypot(midpoint.x - midX, midpoint.y - midY)).toBeLessThan(12);
    // Near-source point stays closer to the source than the path midpoint.
    expect(Math.hypot(nearSource.x - 100, nearSource.y - 100)).toBeLessThan(
      Math.hypot(midpoint.x - 100, midpoint.y - 100),
    );
    // Must not sit on the naive control-point polyline far above/below the curve.
    expect(path).toContain("C");
    expect(nearSource.y).toBeGreaterThan(95);
    expect(nearSource.y).toBeLessThan(midY + 20);
  });

  it("detects crossing polylines and ignores far parallel paths", () => {
    expect(pathsTouch("M 0 50 L 100 50", "M 50 0 L 50 100")).toBe(true);
    expect(pathsTouch("M 0 0 L 100 0", "M 0 40 L 100 40")).toBe(false);
  });

  it("does not treat a shared endpoint join as a touch", () => {
    expect(pathsTouch("M 0 0 L 0 100", "M 0 100 L 80 100")).toBe(false);
  });

  it("marks both ids when a gutter path crosses a vertical fork", () => {
    const [gutterPath] = getRightGutterEdgePath({
      sourceX: 520,
      sourceY: 100,
      sourcePosition: Position.Bottom,
      targetX: 120,
      targetY: 400,
      targetPosition: Position.Top,
      routeGutterX: 900,
    });
    // Vertical through the gutter's lower horizontal run (y ≈ 376).
    const forkPath = "M 260 200 L 260 500";

    const touching = findTouchingEdgeIds([
      { id: "gutter", path: gutterPath },
      { id: "fork", path: forkPath },
      { id: "lonely", path: "M 0 0 L 0 40" },
    ]);

    expect(touching.has("gutter")).toBe(true);
    expect(touching.has("fork")).toBe(true);
    expect(touching.has("lonely")).toBe(false);
  });

  it("darkens only the longer edge in a touch pair", () => {
    const [gutterPath] = getRightGutterEdgePath({
      sourceX: 520,
      sourceY: 100,
      sourcePosition: Position.Bottom,
      targetX: 120,
      targetY: 400,
      targetPosition: Position.Top,
      routeGutterX: 900,
    });
    const forkPath = "M 260 200 L 260 500";

    const { contrastIds, involvedIds } = findTouchContrastEdgeIds([
      { id: "gutter", path: gutterPath, source: "a", target: "merge" },
      { id: "fork", path: forkPath, source: "b", target: "c" },
    ]);

    expect(involvedIds.has("gutter")).toBe(true);
    expect(involvedIds.has("fork")).toBe(true);
    expect(contrastIds.has("gutter")).toBe(true);
    expect(contrastIds.has("fork")).toBe(false);
  });

  it("does not contrast edges that only meet at a shared merge target", () => {
    const { contrastIds, involvedIds } = findTouchContrastEdgeIds([
      { id: "true", path: "M 120 100 L 120 400", source: "if", target: "noop4" },
      { id: "merge", path: "M 400 300 L 120 300 L 120 400", source: "noop3", target: "noop4" },
    ]);

    expect(involvedIds.size).toBe(0);
    expect(contrastIds.size).toBe(0);
  });
});
