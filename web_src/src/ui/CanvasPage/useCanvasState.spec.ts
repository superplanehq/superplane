import { act, renderHook } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import type { Node } from "@xyflow/react";
import type { CanvasPageProps } from ".";
import { useCanvasState } from "./useCanvasState";

function makeNode(id: string, x: number, y: number): Node {
  return {
    id,
    position: { x, y },
    data: { type: "component", component: { collapsed: false } },
    type: "custom",
  };
}

function makeProps(nodes: Node[]): CanvasPageProps {
  return { nodes, edges: [] } as unknown as CanvasPageProps;
}

describe("useCanvasState", () => {
  it("preserves position of actively dragged nodes when props update", () => {
    const initial = [makeNode("a", 0, 0), makeNode("b", 100, 100)];
    const { result, rerender } = renderHook(({ props }) => useCanvasState(props), {
      initialProps: { props: makeProps(initial) },
    });

    // Simulate dragging node "a" to (50, 50) — ReactFlow sets dragging: true
    act(() => {
      result.current.onNodesChange([{ id: "a", type: "position", position: { x: 50, y: 50 }, dragging: true }]);
    });

    expect(result.current.nodes.find((n) => n.id === "a")?.position).toEqual({ x: 50, y: 50 });

    // Simulate a cache refetch that delivers stale server positions
    const refetched = [makeNode("a", 0, 0), makeNode("b", 100, 100)];
    rerender({ props: makeProps(refetched) });

    // Node "a" should keep its drag position, not snap back to (0, 0)
    const nodeA = result.current.nodes.find((n) => n.id === "a");
    expect(nodeA?.position).toEqual({ x: 50, y: 50 });

    // Node "b" (not dragging) should accept the new prop position
    const nodeB = result.current.nodes.find((n) => n.id === "b");
    expect(nodeB?.position).toEqual({ x: 100, y: 100 });
  });

  it("accepts new positions after drag ends", () => {
    const initial = [makeNode("a", 0, 0)];
    const { result, rerender } = renderHook(({ props }) => useCanvasState(props), {
      initialProps: { props: makeProps(initial) },
    });

    // Start drag
    act(() => {
      result.current.onNodesChange([{ id: "a", type: "position", position: { x: 50, y: 50 }, dragging: true }]);
    });

    // End drag
    act(() => {
      result.current.onNodesChange([{ id: "a", type: "position", position: { x: 50, y: 50 }, dragging: false }]);
    });

    // Now a prop update should apply normally
    const updated = [makeNode("a", 200, 200)];
    rerender({ props: makeProps(updated) });

    expect(result.current.nodes.find((n) => n.id === "a")?.position).toEqual({ x: 200, y: 200 });
  });
});
