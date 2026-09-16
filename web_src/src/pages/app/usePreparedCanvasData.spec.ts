import { renderHook } from "@testing-library/react";
import { describe, expect, it, vi } from "bun:test";
import type { QueryClient } from "@tanstack/react-query";
import type { CanvasesCanvas } from "@/api-client";
import { usePreparedCanvasData } from "./usePreparedCanvasData";
import { prepareData } from "./workflowPageHelpers";

vi.mock("./workflowPageHelpers", () => ({
  prepareData: vi.fn(() => ({ nodes: [{ id: "node-1" }], edges: [] })),
}));

describe("usePreparedCanvasData", () => {
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

    expect(prepareData).toHaveBeenCalledWith(
      canvas,
      [],
      [],
      {},
      {},
      {},
      "canvas-1",
      queryClient,
      null,
      "live",
      undefined,
      "org-1",
    );
  });
});
