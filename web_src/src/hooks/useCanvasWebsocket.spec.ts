import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
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
const INVALIDATION_DEBOUNCE_MS = 1000;

function getWebsocketHandler<T extends (...args: never[]) => unknown>(handlerName: "onMessage" | "onOpen"): T {
  const call = useWebSocketMock.mock.calls.at(-1);
  if (!call || !call[1]?.[handlerName]) {
    throw new Error(`Websocket ${handlerName} handler was not registered`);
  }
  return call[1][handlerName] as T;
}

function emitWebsocketMessage(event: string, payload: unknown) {
  const onMessage = getWebsocketHandler<(event: MessageEvent<unknown>) => void>("onMessage");

  act(() => {
    onMessage(
      new MessageEvent("message", {
        data: JSON.stringify({ event, payload }),
      }),
    );
  });
}

function emitWebSocketOpen() {
  const onOpen = getWebsocketHandler<() => void>("onOpen");

  act(() => {
    onOpen();
  });
}

function renderCanvasWebsocketHook(queryClient: QueryClient) {
  return renderHook(() => useCanvasWebsocket(testCanvasId, testOrganizationId), {
    wrapper: ({ children }: { children: ReactNode }) =>
      createElement(QueryClientProvider, { client: queryClient }, children),
  });
}

async function flushMessageQueueAndDebouncedInvalidations() {
  await act(async () => {
    await vi.runAllTimersAsync();
  });
}

function getInvalidationCalls(invalidateQueriesSpy: ReturnType<typeof vi.spyOn>, queryKey: readonly unknown[]) {
  return invalidateQueriesSpy.mock.calls.filter((call: unknown[]) => {
    const args = call[0] as { queryKey?: readonly unknown[] };
    return JSON.stringify(args.queryKey) === JSON.stringify(queryKey);
  });
}

beforeEach(() => {
  vi.useFakeTimers();
});

afterEach(() => {
  vi.useRealTimers();
  vi.clearAllMocks();
});

describe("useCanvasWebsocket", () => {
  it("debounces infinite events invalidation for root workflow events", async () => {
    const queryClient = new QueryClient();
    const invalidateQueriesSpy = vi.spyOn(queryClient, "invalidateQueries").mockResolvedValue();

    renderCanvasWebsocketHook(queryClient);
    emitWebsocketMessage("event_created", {
      id: "event-1",
      nodeId: testNodeId,
      root: true,
    });

    expect(getInvalidationCalls(invalidateQueriesSpy, canvasKeys.infiniteEvents(testCanvasId))).toHaveLength(0);

    await flushMessageQueueAndDebouncedInvalidations();

    expect(getInvalidationCalls(invalidateQueriesSpy, canvasKeys.infiniteEvents(testCanvasId))).toHaveLength(1);
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

    await flushMessageQueueAndDebouncedInvalidations();

    expect(getInvalidationCalls(invalidateQueriesSpy, canvasKeys.infiniteEvents(testCanvasId))).toHaveLength(0);
  });

  it("does not invalidate infinite events query for queue_item_created", async () => {
    const queryClient = new QueryClient();
    const invalidateQueriesSpy = vi.spyOn(queryClient, "invalidateQueries").mockResolvedValue();

    renderCanvasWebsocketHook(queryClient);
    emitWebsocketMessage("queue_item_created", {
      id: "queue-item-1",
      nodeId: testNodeId,
    });

    await flushMessageQueueAndDebouncedInvalidations();

    expect(getInvalidationCalls(invalidateQueriesSpy, canvasKeys.infiniteEvents(testCanvasId))).toHaveLength(0);
  });

  it("does not invalidate infinite events query for queue_item_consumed", async () => {
    const queryClient = new QueryClient();
    const invalidateQueriesSpy = vi.spyOn(queryClient, "invalidateQueries").mockResolvedValue();

    renderCanvasWebsocketHook(queryClient);
    emitWebsocketMessage("queue_item_consumed", {
      id: "queue-item-1",
      nodeId: testNodeId,
    });

    await flushMessageQueueAndDebouncedInvalidations();

    expect(getInvalidationCalls(invalidateQueriesSpy, canvasKeys.infiniteEvents(testCanvasId))).toHaveLength(0);
  });

  it("debounces infinite events invalidation for execution events", async () => {
    const queryClient = new QueryClient();
    const invalidateQueriesSpy = vi.spyOn(queryClient, "invalidateQueries").mockResolvedValue();

    renderCanvasWebsocketHook(queryClient);
    emitWebsocketMessage("execution_created", {
      id: "execution-1",
      nodeId: testNodeId,
    });

    expect(getInvalidationCalls(invalidateQueriesSpy, canvasKeys.infiniteEvents(testCanvasId))).toHaveLength(0);

    await flushMessageQueueAndDebouncedInvalidations();

    expect(getInvalidationCalls(invalidateQueriesSpy, canvasKeys.infiniteEvents(testCanvasId))).toHaveLength(1);
  });

  it("invalidates infinite runs immediately for run events", () => {
    const queryClient = new QueryClient();
    const invalidateQueriesSpy = vi.spyOn(queryClient, "invalidateQueries").mockResolvedValue();

    renderCanvasWebsocketHook(queryClient);
    emitWebsocketMessage("run_finished", {
      id: "run-1",
      canvasId: testCanvasId,
      state: "STATE_FINISHED",
      result: "RESULT_PASSED",
    });

    expect(getInvalidationCalls(invalidateQueriesSpy, canvasKeys.infiniteRuns(testCanvasId))).toHaveLength(1);
  });

  it("coalesces debounced websocket invalidations into one flush per query", async () => {
    const queryClient = new QueryClient();
    const invalidateQueriesSpy = vi.spyOn(queryClient, "invalidateQueries").mockResolvedValue();

    renderCanvasWebsocketHook(queryClient);

    emitWebsocketMessage("event_created", {
      id: "event-1",
      nodeId: testNodeId,
      root: true,
    });
    emitWebsocketMessage("execution_created", {
      id: "execution-1",
      nodeId: testNodeId,
    });

    await flushMessageQueueAndDebouncedInvalidations();

    expect(getInvalidationCalls(invalidateQueriesSpy, canvasKeys.infiniteEvents(testCanvasId))).toHaveLength(1);
    expect(getInvalidationCalls(invalidateQueriesSpy, canvasKeys.infiniteRuns(testCanvasId))).toHaveLength(1);
  });

  it("does not duplicate runs invalidation when run events arrive during a debounce window", async () => {
    const queryClient = new QueryClient();
    const invalidateQueriesSpy = vi.spyOn(queryClient, "invalidateQueries").mockResolvedValue();

    renderCanvasWebsocketHook(queryClient);

    emitWebsocketMessage("execution_created", {
      id: "execution-1",
      nodeId: testNodeId,
    });
    emitWebsocketMessage("run_finished", {
      id: "run-1",
      canvasId: testCanvasId,
      state: "STATE_FINISHED",
      result: "RESULT_PASSED",
    });

    expect(getInvalidationCalls(invalidateQueriesSpy, canvasKeys.infiniteRuns(testCanvasId))).toHaveLength(1);

    await flushMessageQueueAndDebouncedInvalidations();

    expect(getInvalidationCalls(invalidateQueriesSpy, canvasKeys.infiniteRuns(testCanvasId))).toHaveLength(1);
  });

  it("resets debounce timer on subsequent invalidation requests", async () => {
    const queryClient = new QueryClient();
    const invalidateQueriesSpy = vi.spyOn(queryClient, "invalidateQueries").mockResolvedValue();

    renderCanvasWebsocketHook(queryClient);
    emitWebsocketMessage("event_created", {
      id: "event-1",
      nodeId: testNodeId,
      root: true,
    });

    await act(async () => {
      await vi.advanceTimersByTimeAsync(INVALIDATION_DEBOUNCE_MS - 100);
    });

    emitWebsocketMessage("execution_created", {
      id: "execution-1",
      nodeId: testNodeId,
    });

    await act(async () => {
      await vi.advanceTimersByTimeAsync(INVALIDATION_DEBOUNCE_MS - 100);
    });

    expect(getInvalidationCalls(invalidateQueriesSpy, canvasKeys.infiniteEvents(testCanvasId))).toHaveLength(0);

    await act(async () => {
      await vi.advanceTimersByTimeAsync(100);
    });

    expect(getInvalidationCalls(invalidateQueriesSpy, canvasKeys.infiniteEvents(testCanvasId))).toHaveLength(1);
    expect(getInvalidationCalls(invalidateQueriesSpy, canvasKeys.infiniteRuns(testCanvasId))).toHaveLength(1);
  });

  it("does not invalidate runs or events on initial websocket connect", () => {
    const queryClient = new QueryClient();
    const invalidateQueriesSpy = vi.spyOn(queryClient, "invalidateQueries").mockResolvedValue();

    renderCanvasWebsocketHook(queryClient);
    emitWebSocketOpen();

    expect(getInvalidationCalls(invalidateQueriesSpy, canvasKeys.infiniteEvents(testCanvasId))).toHaveLength(0);
    expect(getInvalidationCalls(invalidateQueriesSpy, canvasKeys.infiniteRuns(testCanvasId))).toHaveLength(0);
  });

  it("invalidates runs and events on websocket reconnect", () => {
    const queryClient = new QueryClient();
    const invalidateQueriesSpy = vi.spyOn(queryClient, "invalidateQueries").mockResolvedValue();

    renderCanvasWebsocketHook(queryClient);
    emitWebSocketOpen();
    emitWebSocketOpen();

    expect(getInvalidationCalls(invalidateQueriesSpy, canvasKeys.infiniteEvents(testCanvasId))).toHaveLength(1);
    expect(getInvalidationCalls(invalidateQueriesSpy, canvasKeys.infiniteRuns(testCanvasId))).toHaveLength(1);
  });

  it("invalidates version and console queries for canvas version updates", () => {
    const queryClient = new QueryClient();
    const invalidateQueriesSpy = vi.spyOn(queryClient, "invalidateQueries").mockResolvedValue();

    renderCanvasWebsocketHook(queryClient);
    emitWebsocketMessage("canvas_version_updated", {
      canvasId: testCanvasId,
      versionId: "version-1",
    });

    expect(invalidateQueriesSpy).toHaveBeenCalledWith({
      queryKey: canvasKeys.versionList(testCanvasId),
    });
    expect(invalidateQueriesSpy).toHaveBeenCalledWith({
      queryKey: canvasKeys.consoleAll(testCanvasId),
    });
  });
});
