import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { createElement } from "react";
import type { ReactNode } from "react";
import { canvasKeys } from "@/hooks/useCanvasData";

const { useWebSocketMock, nodeExecutionStoreMock } = vi.hoisted(() => ({
  useWebSocketMock: vi.fn(),
  nodeExecutionStoreMock: {
    updateNodeEvent: vi.fn(),
    updateNodeExecution: vi.fn(),
    addNodeQueueItem: vi.fn(),
    removeNodeQueueItem: vi.fn(),
  },
}));

vi.mock("react-use-websocket", () => ({
  default: useWebSocketMock,
}));

vi.mock("@/stores/nodeExecutionStore", () => ({
  useNodeExecutionStore: () => nodeExecutionStoreMock,
}));

import { useCanvasWebsocket } from "@/hooks/useCanvasWebsocket";

const testCanvasId = "canvas-1";
const testOrganizationId = "org-1";
const testNodeId = "node-1";

function getOnMessageHandler() {
  const call = useWebSocketMock.mock.calls.at(-1);
  if (!call || !call[1]?.onMessage) {
    throw new Error("Websocket onMessage handler was not registered");
  }
  return call[1].onMessage as (event: MessageEvent<unknown>) => void;
}

function emitWebsocketMessage(event: string, payload: unknown) {
  const onMessage = getOnMessageHandler();

  act(() => {
    onMessage(
      new MessageEvent("message", {
        data: JSON.stringify({ event, payload }),
      }),
    );
  });
}

function renderCanvasWebsocketHook(queryClient: QueryClient) {
  return renderHook(() => useCanvasWebsocket(testCanvasId, testOrganizationId), {
    wrapper: ({ children }: { children: ReactNode }) =>
      createElement(QueryClientProvider, { client: queryClient }, children),
  });
}

afterEach(() => {
  vi.clearAllMocks();
});

describe("useCanvasWebsocket", () => {
  it("invalidates infinite events query for root workflow events", async () => {
    const queryClient = new QueryClient();
    const invalidateQueriesSpy = vi.spyOn(queryClient, "invalidateQueries").mockResolvedValue();

    renderCanvasWebsocketHook(queryClient);
    emitWebsocketMessage("event_created", {
      id: "event-1",
      nodeId: testNodeId,
      root: true,
    });

    await waitFor(() => {
      expect(invalidateQueriesSpy).toHaveBeenCalledWith({
        queryKey: canvasKeys.infiniteEvents(testCanvasId),
      });
    });
  });

  it("does not invalidate infinite events query for non-root workflow events", async () => {
    const queryClient = new QueryClient();
    const invalidateQueriesSpy = vi.spyOn(queryClient, "invalidateQueries").mockResolvedValue();

    renderCanvasWebsocketHook(queryClient);
    emitWebsocketMessage("workflow_event_created", {
      id: "event-1",
      nodeId: testNodeId,
      root: false,
    });

    await waitFor(() => {
      expect(invalidateQueriesSpy).not.toHaveBeenCalledWith({
        queryKey: canvasKeys.infiniteEvents(testCanvasId),
      });
    });
  });

  it("does not invalidate infinite events query for queue_item_created", async () => {
    const queryClient = new QueryClient();
    const invalidateQueriesSpy = vi.spyOn(queryClient, "invalidateQueries").mockResolvedValue();

    renderCanvasWebsocketHook(queryClient);
    emitWebsocketMessage("queue_item_created", {
      id: "queue-item-1",
      nodeId: testNodeId,
    });

    await waitFor(() => {
      expect(invalidateQueriesSpy).not.toHaveBeenCalledWith({
        queryKey: canvasKeys.infiniteEvents(testCanvasId),
      });
    });
  });

  it("does not invalidate infinite events query for queue_item_consumed", async () => {
    const queryClient = new QueryClient();
    const invalidateQueriesSpy = vi.spyOn(queryClient, "invalidateQueries").mockResolvedValue();

    renderCanvasWebsocketHook(queryClient);
    emitWebsocketMessage("queue_item_consumed", {
      id: "queue-item-1",
      nodeId: testNodeId,
    });

    await waitFor(() => {
      expect(invalidateQueriesSpy).not.toHaveBeenCalledWith({
        queryKey: canvasKeys.infiniteEvents(testCanvasId),
      });
    });
  });

  it("invalidates infinite events query for execution events", async () => {
    const queryClient = new QueryClient();
    const invalidateQueriesSpy = vi.spyOn(queryClient, "invalidateQueries").mockResolvedValue();

    renderCanvasWebsocketHook(queryClient);
    emitWebsocketMessage("execution_created", {
      id: "execution-1",
      nodeId: testNodeId,
    });

    await waitFor(() => {
      expect(invalidateQueriesSpy).toHaveBeenCalledWith({
        queryKey: canvasKeys.infiniteEvents(testCanvasId),
      });
    });
  });
});
