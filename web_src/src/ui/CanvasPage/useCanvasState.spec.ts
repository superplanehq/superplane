import { act, renderHook } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import type { Edge, Node } from "@xyflow/react";
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

function makeEdge(id: string, source: string, target: string): Edge {
  return {
    id,
    source,
    target,
  };
}

function makeProps(nodes: Node[], edges: Edge[] = []): CanvasPageProps {
  return { nodes, edges } as unknown as CanvasPageProps;
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

  it("preserves dropped position until the saved position catches up", () => {
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
      result.current.onNodesChange([{ id: "a", type: "position", position: { x: 50.4, y: 50.2 }, dragging: false }]);
    });

    // A parent rerender can still carry the old cache position before autosave finishes.
    rerender({ props: makeProps([makeNode("a", 0, 0)]) });
    expect(result.current.nodes.find((n) => n.id === "a")?.position).toEqual({ x: 50.4, y: 50.2 });

    // Once the saved rounded position arrives from the parent, normal syncing resumes.
    rerender({ props: makeProps([makeNode("a", 50, 50)]) });
    expect(result.current.nodes.find((n) => n.id === "a")?.position).toEqual({ x: 50, y: 50 });
  });

  it("does not resync edges when only node positions change", () => {
    const initialNodes = [makeNode("a", 0, 0), makeNode("b", 100, 100)];
    const initialEdges = [makeEdge("edge-a-b", "a", "b")];
    const { result } = renderHook(({ props }) => useCanvasState(props), {
      initialProps: { props: makeProps(initialNodes, initialEdges) },
    });

    const edgesBeforeDrag = result.current.edges;

    act(() => {
      result.current.onNodesChange([{ id: "a", type: "position", position: { x: 50, y: 50 }, dragging: true }]);
    });

    expect(result.current.edges).toBe(edgesBeforeDrag);
  });
});
