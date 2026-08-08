import { QueryClient } from "@tanstack/react-query";
import { renderHook } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import type { ActionsAction, CanvasesCanvasVersion } from "@/api-client";
import { makeCanvas, makeComponentsNode, makeEdge } from "@/test/factories";
import { useDraftVisualDiff } from "./useDraftVisualDiff";

const noopComponent = {
  name: "noop",
  label: "Noop",
  icon: "circle",
  outputChannels: [{ name: "default" }],
} as ActionsAction;

function renderDraftVisualDiffWithRemovedNode() {
  const keptNode = makeComponentsNode({ id: "node-1", name: "First" });
  const removedNode = makeComponentsNode({ id: "node-2", name: "Second" });
  const liveCanvasVersion = {
    spec: {
      nodes: [keptNode, removedNode],
      edges: [makeEdge({ sourceId: "node-1", targetId: "node-2" })],
    },
  } as CanvasesCanvasVersion;

  return renderHook(() =>
    useDraftVisualDiff({
      isViewingDraftVersion: true,
      canvas: makeCanvas({ spec: { nodes: [keptNode], edges: [] } }),
      liveCanvasVersion,
      preparedNodes: [{ id: "node-1", position: { x: 0, y: 0 }, data: {} }],
      preparedEdges: [],
      allTriggers: [],
      allComponents: [noopComponent],
      canvasId: "canvas-1",
      queryClient: new QueryClient(),
    }),
  );
}

describe("useDraftVisualDiff", () => {
  it("renders removed nodes as ghosts that cannot be connected", () => {
    const { result } = renderDraftVisualDiffWithRemovedNode();

    const ghost = result.current.nodes.find((node) => node.id === "node-2");
    expect(ghost?.data._draftDiffStatus).toBe("removed");
    expect(ghost?.connectable).toBe(false);
  });
});
