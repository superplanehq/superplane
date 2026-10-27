import { QueryClient, QueryClientProvider, type InfiniteData } from "@tanstack/react-query";
import { act, renderHook, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "bun:test";
import { createElement } from "react";
import type { ReactNode } from "react";
import type { CanvasesCanvasRun } from "@/api-client";
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

vi.mock("@/lib/reactUseWebsocket", () => ({
  useWebSocket: useWebSocketMock,
}));

vi.mock("@/stores/nodeExecutionStore", () => ({
  useNodeExecutionStore: (selector?: (state: typeof nodeExecutionStoreMock) => unknown) =>
    selector ? selector(nodeExecutionStoreMock) : nodeExecutionStoreMock,
}));

import type { InfiniteRunsPage } from "./canvasInfiniteCache";
import { useCanvasWebsocket } from "@/hooks/useCanvasWebsocket";

const testCanvasId = "canvas-1";
const testNodeId = "node-1";

function emitWebsocketMessage(event: string, payload: unknown) {
  const onMessage = useWebSocketMock.mock.calls.at(-1)?.[1]?.onMessage as (event: MessageEvent<unknown>) => void;
  act(() => {
    onMessage(new MessageEvent("message", { data: JSON.stringify({ event, payload }) }));
  });
}

afterEach(() => {
  vi.clearAllMocks();
});

describe("useCanvasWebsocket task cache", () => {
  it("keeps and inserts matching pending child runs before their root events exist", async () => {
    const queryClient = new QueryClient();
    const queryKey = canvasKeys.infiniteRuns(testCanvasId, {
      triggerIds: ["onRun"],
      states: ["STATE_PENDING"],
    });
    const updatedAt = Date.now();
    const pendingRun = {
      id: "cached-child",
      canvasId: testCanvasId,
      nodeId: "onRun",
      state: "STATE_PENDING",
      updatedAt: new Date(updatedAt).toISOString(),
    };
    queryClient.setQueryData(queryKey, {
      pages: [{ runs: [pendingRun], totalCount: 1 }],
      pageParams: [undefined],
    });
    const { unmount } = renderHook(() => useCanvasWebsocket({ canvasId: testCanvasId, organizationId: "org-1" }), {
      wrapper: ({ children }: { children: ReactNode }) =>
        createElement(QueryClientProvider, { client: queryClient }, children),
    });

    // A sparse lifecycle update must retain the node ID from the REST snapshot.
    emitWebsocketMessage("run_pending", {
      id: pendingRun.id,
      canvasId: testCanvasId,
      state: "STATE_PENDING",
      updatedAt: new Date(updatedAt + 1).toISOString(),
    });
    emitWebsocketMessage("run_pending", { ...pendingRun, id: "new-child" });
    emitWebsocketMessage("run_pending", { ...pendingRun, id: "other-trigger", nodeId: "other-node" });

    await waitFor(() => {
      const cached = queryClient.getQueryData<InfiniteData<InfiniteRunsPage>>(queryKey);
      expect(cached?.pages[0]?.runs?.map((run) => run.id).sort()).toEqual(["cached-child", "new-child"]);
      expect(cached?.pages[0]?.totalCount).toBe(2);
      expect(cached?.pages[0]?.runs?.find((run) => run.id === pendingRun.id)?.nodeId).toBe("onRun");
    });
    unmount();
    queryClient.clear();
  });

  it("keeps live runs inside the selected title, trigger, and status filters", () => {
    const queryClient = new QueryClient();
    const filters = {
      search: "deploy",
      triggerIds: ["trigger-a"],
      states: ["STATE_STARTED" as const],
      results: ["RESULT_PASSED" as const],
      includeTask: true,
    };
    queryClient.setQueryData(canvasKeys.infiniteRuns(testCanvasId, filters), {
      pages: [{ runs: [], totalCount: 0 }],
      pageParams: [undefined],
    });
    const { unmount } = renderHook(() => useCanvasWebsocket({ canvasId: testCanvasId, organizationId: "org-1" }), {
      wrapper: ({ children }: { children: ReactNode }) =>
        createElement(QueryClientProvider, { client: queryClient }, children),
    });
    for (const [id, nodeId, customName, result] of [
      ["matching", "trigger-a", "Deploy API", "RESULT_PASSED"],
      ["wrong-title", "trigger-a", "Build API", "RESULT_PASSED"],
      ["wrong-trigger", "trigger-b", "Deploy API", "RESULT_PASSED"],
      ["wrong-status", "trigger-a", "Deploy API", "RESULT_FAILED"],
    ] as const) {
      emitWebsocketMessage("run_finished", {
        id,
        canvasId: testCanvasId,
        state: "STATE_FINISHED",
        result,
        rootEvent: { nodeId, customName },
      });
    }
    const cached = queryClient.getQueryData<InfiniteData<InfiniteRunsPage>>(
      canvasKeys.infiniteRuns(testCanvasId, filters),
    );
    expect(cached?.pages[0]?.runs?.map((run) => run.id)).toEqual(["matching"]);
    expect(cached?.pages[0]?.totalCount).toBe(1);
    unmount();
    queryClient.clear();
  });

  it("patches the include-task describe cache and keeps the task label", async () => {
    const queryClient = new QueryClient();
    const task = { id: "task-1", key: "SP-1", title: "Linked task" };
    queryClient.setQueryData(canvasKeys.run(testCanvasId, "run-1", true), {
      run: {
        id: "run-1",
        canvasId: testCanvasId,
        state: "STATE_STARTED",
        rootEvent: { id: "event-1", nodeId: testNodeId },
        executions: [],
        task,
      },
    });

    renderHook(() => useCanvasWebsocket({ canvasId: testCanvasId, organizationId: "org-1" }), {
      wrapper: ({ children }: { children: ReactNode }) =>
        createElement(QueryClientProvider, { client: queryClient }, children),
    });
    emitWebsocketMessage("execution_started", {
      id: "execution-1",
      nodeId: testNodeId,
      state: "STATE_STARTED",
      updatedAt: "2026-06-01T12:00:00.000Z",
      rootEvent: { id: "event-1", nodeId: testNodeId },
    });
    emitWebsocketMessage("run_finished", {
      id: "run-1",
      canvasId: testCanvasId,
      state: "STATE_FINISHED",
      result: "RESULT_PASSED",
      updatedAt: "2026-06-01T12:01:00.000Z",
    });
    await act(async () => {
      await new Promise((resolve) => setTimeout(resolve, 0));
    });

    await waitFor(() => {
      const described = queryClient.getQueryData<{ run?: CanvasesCanvasRun }>(
        canvasKeys.run(testCanvasId, "run-1", true),
      );
      expect(described?.run?.state).toBe("STATE_FINISHED");
      expect(described?.run?.executions?.[0]?.id).toBe("execution-1");
      expect(described?.run?.task).toEqual(task);
    });
  });
});
