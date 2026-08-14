import type { Edge, Node } from "@xyflow/react";
import { describe, expect, it } from "vitest";
import { interpolateLayoutNodes } from "./nodePositionAnimation";

describe("interpolateLayoutNodes", () => {
  it("moves existing nodes and fades a connected node from its source", () => {
    const current: Node[] = [{ id: "source", position: { x: 0, y: 0 }, data: {} }];
    const target: Node[] = [
      { id: "source", position: { x: 100, y: 100 }, data: {} },
      { id: "added", position: { x: 100, y: 300 }, data: {} },
    ];
    const edges: Edge[] = [{ id: "edge", source: "source", target: "added" }];

    const halfway = interpolateLayoutNodes(current, target, edges, 0.5);

    expect(halfway.find((node) => node.id === "source")?.position).toEqual({ x: 50, y: 50 });
    expect(halfway.find((node) => node.id === "added")).toMatchObject({
      position: { x: 50, y: 150 },
      style: { opacity: 0.5 },
    });
  });

  it("fades a disconnected node at its final position", () => {
    const target: Node[] = [{ id: "added", position: { x: 400, y: 200 }, data: {} }];

    expect(interpolateLayoutNodes([], target, [], 0.25)[0]).toMatchObject({
      position: { x: 400, y: 200 },
      style: { opacity: 0.25 },
    });
  });

  it("preserves stationary node references during animation", () => {
    const stationary: Node = { id: "stationary", position: { x: 20, y: 40 }, data: {} };

    const animated = interpolateLayoutNodes([stationary], [stationary], [], 0.5);

    expect(animated[0]).toBe(stationary);
  });
});
