import { renderHook } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "bun:test";
import type { QueryClient } from "@tanstack/react-query";
import type { CanvasesCanvas } from "@/api-client";
import { usePreparedCanvasData } from "./usePreparedCanvasData";
import { prepareData } from "./workflowPageHelpers";

vi.mock("./workflowPageHelpers", () => ({
  prepareData: vi.fn(() => ({ nodes: [{ id: "node-1" }], edges: [] })),
}));

describe("usePreparedCanvasData", () => {
  beforeEach(() => {
    vi.mocked(prepareData).mockClear();
  });

  it("accepts a null canvas from the live page", () => {
    const queryClient = {} as QueryClient;

    const { result } = renderHook(() =>
      usePreparedCanvasData({
        canvas: null,
        triggers: [],
        components: [],
        nodeEventsMap: {},
        nodeExecutionsMap: {},
        nodeQueueItemsMap: {},
        canvasId: "canvas-1",
        queryClient,
        user: null,
        canvasMode: "live",
        organizationId: "org-1",
        enabled: true,
      }),
    );

    expect(result.current).toEqual({ nodes: [], edges: [] });
    expect(prepareData).not.toHaveBeenCalled();
  });

  it("passes the organization id so runner logs can stream", () => {
    const queryClient = {} as QueryClient;
    const canvas = { metadata: { id: "canvas-1" } } as CanvasesCanvas;

    renderHook(() =>
      usePreparedCanvasData({
        canvas,
        triggers: [],
        components: [],
        nodeEventsMap: {},
        nodeExecutionsMap: {},
        nodeQueueItemsMap: {},
        canvasId: "canvas-1",
        queryClient,
        user: null,
        canvasMode: "live",
        organizationId: "org-1",
        enabled: true,
      }),
    );

    expect(prepareData).toHaveBeenCalledWith({
      workflow: canvas,
      triggers: [],
      components: [],
      nodeEventsMap: {},
      nodeExecutionsMap: {},
      nodeQueueItemsMap: {},
      workflowId: "canvas-1",
      queryClient,
      user: null,
      canvasMode: "live",
      openModal: undefined,
      organizationId: "org-1",
    });
  });
});
