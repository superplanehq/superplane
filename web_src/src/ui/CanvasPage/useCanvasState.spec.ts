import { act, renderHook } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import type { Edge, Node } from "@xyflow/react";
import { FACTORY_COMPACT_LAYER_STRIDE, FACTORY_EDIT_VERTICAL_EXTRA_PER_LAYER } from "@/lib/factoryEditVerticalSpacing";
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
  afterEach(() => {
    vi.restoreAllMocks();
  });

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

  it("stretches stacked factory ranks in factory-auto edit layout only", () => {
    const compactStride = FACTORY_COMPACT_LAYER_STRIDE;
    const stacked = [makeNode("top", 0, 0), makeNode("bottom", 0, compactStride)];

    const { result, rerender } = renderHook(({ props }: { props: CanvasPageProps }) => useCanvasState(props), {
      initialProps: { props: { ...makeProps(stacked), layoutMode: "factory-auto" } },
    });

    expect(result.current.nodes.find((node) => node.id === "top")?.position.y).toBe(0);
    expect(result.current.nodes.find((node) => node.id === "bottom")?.position.y).toBe(
      compactStride + FACTORY_EDIT_VERTICAL_EXTRA_PER_LAYER,
    );

    rerender({ props: { ...makeProps(stacked), layoutMode: "manual" } });
    expect(result.current.nodes.find((node) => node.id === "bottom")?.position.y).toBe(compactStride);
  });

  it("snaps the first factory-auto layout without animation", () => {
    const requestAnimationFrame = vi.spyOn(window, "requestAnimationFrame");
    const { result, rerender } = renderHook(({ props }) => useCanvasState(props), {
      initialProps: { props: { ...makeProps([makeNode("a", 0, 0)]), layoutMode: "factory-auto" as const } },
    });

    rerender({
      props: { ...makeProps([makeNode("a", 100, 100)]), layoutMode: "factory-auto" as const },
    });

    expect(requestAnimationFrame).not.toHaveBeenCalled();
    expect(result.current.nodes.find((node) => node.id === "a")?.position).toEqual({ x: 100, y: 100 });
  });

  it("does not restart an active factory layout animation when the same layout refreshes", () => {
    const animationFrames = new Map<number, FrameRequestCallback>();
    let nextFrameId = 1;
    const requestAnimationFrame = vi.spyOn(window, "requestAnimationFrame").mockImplementation((callback) => {
      const frameId = nextFrameId++;
      animationFrames.set(frameId, callback);
      return frameId;
    });
    const cancelAnimationFrame = vi.spyOn(window, "cancelAnimationFrame").mockImplementation((frameId) => {
      animationFrames.delete(frameId);
    });
    const initialProps = {
      ...makeProps([makeNode("a", 0, 0)]),
      layoutMode: "factory-auto" as const,
    };
    const { rerender } = renderHook(({ props }) => useCanvasState(props), {
      initialProps: { props: initialProps },
    });

    rerender({
      props: {
        ...makeProps([makeNode("a", 100, 100)]),
        layoutMode: "factory-auto" as const,
      },
    });
    rerender({
      props: {
        ...makeProps([makeNode("a", 200, 200)]),
        layoutMode: "factory-auto" as const,
      },
    });
    expect(requestAnimationFrame).toHaveBeenCalledTimes(1);

    rerender({
      props: {
        ...makeProps([{ ...makeNode("a", 200, 200), data: { status: "saved" } }]),
        layoutMode: "factory-auto" as const,
      },
    });

    expect(cancelAnimationFrame).not.toHaveBeenCalled();
    expect(requestAnimationFrame).toHaveBeenCalledTimes(1);
  });

  it("keeps the open sidebar node selected through factory layout animation", () => {
    const animationFrames = new Map<number, FrameRequestCallback>();
    let nextFrameId = 1;
    vi.spyOn(window, "requestAnimationFrame").mockImplementation((callback) => {
      const frameId = nextFrameId++;
      animationFrames.set(frameId, callback);
      return frameId;
    });
    vi.spyOn(window, "cancelAnimationFrame").mockImplementation((frameId) => {
      animationFrames.delete(frameId);
    });

    const { result, rerender } = renderHook(({ props }: { props: CanvasPageProps }) => useCanvasState(props), {
      initialProps: {
        props: {
          ...makeProps([makeNode("a", 0, 0)]),
          layoutMode: "factory-auto" as const,
          initialSidebar: { isOpen: true, nodeId: "a" },
        } as unknown as CanvasPageProps,
      },
    });

    expect(result.current.nodes.find((node) => node.id === "a")?.selected).toBe(true);

    rerender({
      props: {
        ...makeProps([makeNode("a", 100, 100)]),
        layoutMode: "factory-auto" as const,
        initialSidebar: { isOpen: true, nodeId: "a" },
      } as unknown as CanvasPageProps,
    });
    rerender({
      props: {
        ...makeProps([makeNode("a", 200, 200)]),
        layoutMode: "factory-auto" as const,
        initialSidebar: { isOpen: true, nodeId: "a" },
      } as unknown as CanvasPageProps,
    });

    act(() => {
      const frame = [...animationFrames.values()].at(-1);
      frame?.(performance.now() + 1_000);
    });

    expect(result.current.nodes.find((node) => node.id === "a")?.selected).toBe(true);
  });

  it("does not re-push sidebar params when onSidebarChange identity changes", () => {
    const onSidebarChange = vi.fn();
    const props = {
      ...makeProps([makeNode("a", 0, 0)]),
      initialSidebar: { isOpen: true, nodeId: "a" },
      onSidebarChange,
    } as unknown as CanvasPageProps;

    const { rerender } = renderHook(({ hookProps }) => useCanvasState(hookProps), {
      initialProps: { hookProps: props },
    });

    onSidebarChange.mockClear();

    const nextOnSidebarChange = vi.fn();
    rerender({
      hookProps: {
        ...props,
        onSidebarChange: nextOnSidebarChange,
      } as unknown as CanvasPageProps,
    });

    expect(nextOnSidebarChange).not.toHaveBeenCalled();
  });

  it("does not notify onSidebarChange on mount when the sidebar opens from initial state", () => {
    const onSidebarChange = vi.fn();
    const props = {
      ...makeProps([makeNode("a", 0, 0)]),
      initialSidebar: { isOpen: true, nodeId: "a" },
      onSidebarChange,
    } as unknown as CanvasPageProps;

    renderHook(() => useCanvasState(props));

    expect(onSidebarChange).not.toHaveBeenCalled();
  });

  it("does not echo externally-applied sidebar selection back to onSidebarChange", () => {
    const onSidebarChange = vi.fn();
    const baseProps = {
      ...makeProps([makeNode("a", 0, 0)]),
      onSidebarChange,
    };

    const { rerender } = renderHook(({ hookProps }) => useCanvasState(hookProps), {
      initialProps: {
        hookProps: { ...baseProps, initialSidebar: { isOpen: true, nodeId: "a" } } as unknown as CanvasPageProps,
      },
    });

    onSidebarChange.mockClear();

    rerender({
      hookProps: { ...baseProps, initialSidebar: { isOpen: true, nodeId: "b" } } as unknown as CanvasPageProps,
    });

    expect(onSidebarChange).not.toHaveBeenCalled();
  });

  it("notifies onSidebarChange when the user opens a node", () => {
    const onSidebarChange = vi.fn();
    const props = {
      ...makeProps([makeNode("a", 0, 0)]),
      initialSidebar: { isOpen: false, nodeId: null },
      onSidebarChange,
    } as unknown as CanvasPageProps;

    const { result } = renderHook(() => useCanvasState(props));

    act(() => {
      result.current.componentSidebar.open("a");
    });

    expect(onSidebarChange).toHaveBeenCalledWith(true, "a");
  });
});
