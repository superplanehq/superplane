import { describe, expect, it } from "vitest";
import type { InternalNode, Node } from "@xyflow/react";
import {
  CANVAS_VIEWPORT_CULL_PADDING_PX,
  getPaddedViewportRendererRect,
  getPaddedViewportScreenRect,
  getVisibleEdgeIdsInPaddedViewport,
  getVisibleNodeIdsInPaddedViewport,
  includeCanvasNodesThatMustStayMounted,
  includeEndpointsOfVisibleEdges,
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

  it("keeps nodes visible while they are waiting for measurement", () => {
    const nodeLookup = new Map([
      ["unmeasured", internalNode("unmeasured", { x: 5000, y: 5000 }, { measured: { width: 0, height: 0 } })],
    ]);

    const visibleNodeIds = getVisibleNodeIdsInPaddedViewport(nodeLookup, 1000, 800, [0, 0, 1]);

    expect(visibleNodeIds.has("unmeasured")).toBe(true);
  });

  it("keeps prop nodes visible until React Flow adds them to the lookup", () => {
    const visibleNodeIds = includeCanvasNodesThatMustStayMounted(
      new Set(["known-visible"]),
      new Map([["known-hidden", internalNode("known-hidden", { x: 5000, y: 5000 })]]),
      [
        { id: "known-hidden", position: { x: 5000, y: 5000 }, data: {} },
        { id: "not-yet-measured", position: { x: 100, y: 100 }, data: {} },
      ],
    );

    expect(visibleNodeIds.has("known-visible")).toBe(true);
    expect(visibleNodeIds.has("known-hidden")).toBe(false);
    expect(visibleNodeIds.has("not-yet-measured")).toBe(true);
  });

  it("keeps interactive prop nodes visible even when they are outside the viewport", () => {
    const visibleNodeIds = includeCanvasNodesThatMustStayMounted(
      new Set<string>(),
      new Map([["selected", internalNode("selected", { x: 5000, y: 5000 })]]),
      [{ id: "selected", position: { x: 5000, y: 5000 }, data: {}, selected: true }],
    );

    expect(visibleNodeIds.has("selected")).toBe(true);
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

  it("keeps edges visible when both endpoints are off-screen but the edge crosses the viewport", () => {
    const nodeLookup = new Map([
      ["left", internalNode("left", { x: -2000, y: 400 })],
      ["right", internalNode("right", { x: 2000, y: 400 })],
    ]);
    const viewportRect = getPaddedViewportRendererRect(1000, 800, [0, 0, 1]);

    const visibleEdgeIds = getVisibleEdgeIdsInPaddedViewport(
      [{ id: "left-right", source: "left", target: "right" }],
      new Set<string>(),
      nodeLookup,
      viewportRect,
    );

    expect(visibleEdgeIds.has("left-right")).toBe(true);
  });

  it("hides edges when both endpoints and their span are outside the viewport", () => {
    const nodeLookup = new Map([
      ["far-a", internalNode("far-a", { x: 5000, y: 5000 })],
      ["far-b", internalNode("far-b", { x: 6000, y: 5000 })],
    ]);
    const viewportRect = getPaddedViewportRendererRect(1000, 800, [0, 0, 1]);

    const visibleEdgeIds = getVisibleEdgeIdsInPaddedViewport(
      [{ id: "far-a-far-b", source: "far-a", target: "far-b" }],
      new Set<string>(),
      nodeLookup,
      viewportRect,
    );

    expect(visibleEdgeIds.has("far-a-far-b")).toBe(false);
  });

  it("keeps both endpoint nodes of a visible edge mounted even when they are off-screen", () => {
    const visibleNodeIds = includeEndpointsOfVisibleEdges(
      new Set(["on-screen"]),
      [
        { id: "on-screen-off-screen", source: "on-screen", target: "off-screen" },
        { id: "crossing", source: "far-left", target: "far-right" },
        { id: "hidden-edge", source: "hidden-a", target: "hidden-b" },
      ],
      new Set(["on-screen-off-screen", "crossing"]),
    );

    // Endpoint that shares an edge with an on-screen node must stay mounted.
    expect(visibleNodeIds.has("off-screen")).toBe(true);
    // Both endpoints of an edge that only crosses the viewport must stay mounted.
    expect(visibleNodeIds.has("far-left")).toBe(true);
    expect(visibleNodeIds.has("far-right")).toBe(true);
    // Endpoints of hidden edges are not forced to stay mounted.
    expect(visibleNodeIds.has("hidden-a")).toBe(false);
    expect(visibleNodeIds.has("hidden-b")).toBe(false);
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
