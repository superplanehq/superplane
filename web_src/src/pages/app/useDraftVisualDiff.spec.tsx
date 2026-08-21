import { QueryClient } from "@tanstack/react-query";
import { renderHook } from "@testing-library/react";
import { beforeEach, describe, expect, it } from "vitest";

import { useDraftVisualDiff } from "./useDraftVisualDiff";

describe("useDraftVisualDiff", () => {
  beforeEach(() => {
    localStorage.clear();
  });

  it("does not add removed-node ghosts when the context disables them", () => {
    const { result } = renderHook(() =>
      useDraftVisualDiff({
        isViewingDraftVersion: true,
        canvas: { metadata: { id: "canvas-1" }, spec: { nodes: [], edges: [] } },
        liveCanvasVersion: {
          metadata: { id: "version-live" },
          spec: {
            nodes: [
              {
                id: "removed-node",
                name: "Removed node",
                type: "TYPE_ACTION",
                component: "noop",
                position: { x: 100, y: 100 },
              },
            ],
            edges: [],
          },
        },
        preparedNodes: [],
        preparedEdges: [],
        allTriggers: [],
        allComponents: [],
        canvasId: "canvas-1",
        queryClient: new QueryClient(),
        includeRemovedNodeGhosts: false,
      }),
    );

    expect(result.current.nodes).toEqual([]);
    expect(result.current.diffCounts.removed).toBe(1);
    expect(result.current.diffToggles.showDeletedNodesControl).toBe(false);
  });
});
