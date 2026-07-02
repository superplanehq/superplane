import { describe, expect, it } from "vitest";
import type { InternalNode, Node } from "@xyflow/react";
import {
  CANVAS_VIEWPORT_CULL_PADDING_PX,
  getPaddedViewportScreenRect,
  getVisibleEdgeIdsInPaddedViewport,
  getVisibleNodeIdsInPaddedViewport,
  shouldKeepCanvasNodeVisible,
} from "./canvasViewportCulling";

function internalNode(
  id: string,
  position: { x: number; y: number },
  options: Partial<InternalNode<Node>> = {},
): InternalNode<Node> {
  const userNode: Node = {
    id,
    position,
    data: {},
  };

  return {
    ...userNode,
    measured: { width: 240, height: 120 },
    internals: {
      positionAbsolute: position,
      handleBounds: { source: [], target: [] },
      userNode,
      z: 0,
    },
    ...options,
  };
}

describe("canvasViewportCulling", () => {
  it("expands the viewport rect by the configured padding", () => {
    expect(getPaddedViewportScreenRect(1000, 800)).toEqual({
      x: -CANVAS_VIEWPORT_CULL_PADDING_PX,
      y: -CANVAS_VIEWPORT_CULL_PADDING_PX,
      width: 1000 + CANVAS_VIEWPORT_CULL_PADDING_PX * 2,
      height: 800 + CANVAS_VIEWPORT_CULL_PADDING_PX * 2,
    });
  });

  it("keeps nodes inside the padded viewport visible", () => {
    const nodeLookup = new Map([
      ["on-screen", internalNode("on-screen", { x: 100, y: 100 })],
      ["near-edge", internalNode("near-edge", { x: 980, y: 100 })],
      ["far-offscreen", internalNode("far-offscreen", { x: 5000, y: 5000 })],
    ]);

    const visibleNodeIds = getVisibleNodeIdsInPaddedViewport(nodeLookup, 1000, 800, [0, 0, 1]);

    expect(visibleNodeIds.has("on-screen")).toBe(true);
    expect(visibleNodeIds.has("near-edge")).toBe(true);
    expect(visibleNodeIds.has("far-offscreen")).toBe(false);
  });

  it("allows previously hidden nodes to become visible again", () => {
    const nodeLookup = new Map([
      ["hidden-on-screen", internalNode("hidden-on-screen", { x: 100, y: 100 }, { hidden: true })],
    ]);

    const visibleNodeIds = getVisibleNodeIdsInPaddedViewport(nodeLookup, 1000, 800, [0, 0, 1]);

    expect(visibleNodeIds.has("hidden-on-screen")).toBe(true);
  });

  it("keeps edges visible when either endpoint is visible", () => {
    const visibleNodeIds = new Set(["a", "b"]);
    const visibleEdgeIds = getVisibleEdgeIdsInPaddedViewport(
      [
        { id: "a-b", source: "a", target: "b" },
        { id: "b-c", source: "b", target: "c" },
        { id: "c-d", source: "c", target: "d" },
      ],
      visibleNodeIds,
    );

    expect(visibleEdgeIds.has("a-b")).toBe(true);
    expect(visibleEdgeIds.has("b-c")).toBe(true);
    expect(visibleEdgeIds.has("c-d")).toBe(false);
  });

  it("always keeps interactive nodes visible", () => {
    expect(shouldKeepCanvasNodeVisible({ id: "dragging", dragging: true })).toBe(true);
    expect(shouldKeepCanvasNodeVisible({ id: "selected", selected: true })).toBe(true);
    expect(
      shouldKeepCanvasNodeVisible({
        id: "template",
        data: { isTemplate: true },
      }),
    ).toBe(true);
    expect(shouldKeepCanvasNodeVisible({ id: "plain" })).toBe(false);
  });
});
