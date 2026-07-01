import { QueryClient, QueryClientProvider, type InfiniteData } from "@tanstack/react-query";
import { act, renderHook, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { createElement } from "react";
import type { ReactNode } from "react";
import type { CanvasesCanvasRun } from "@/api-client";
import { canvasKeys } from "@/hooks/useCanvasData";
import type { InfiniteRunsPage } from "@/hooks/canvasInfiniteCache";

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

async function flushMessageQueue() {
  await act(async () => {
    await new Promise((resolve) => setTimeout(resolve, 0));
  });
}

function getInvalidationCalls(invalidateQueriesSpy: ReturnType<typeof vi.spyOn>, queryKey: readonly unknown[]) {
  return invalidateQueriesSpy.mock.calls.filter((call: unknown[]) => {
    const args = call[0] as { queryKey?: readonly unknown[] };
    return JSON.stringify(args.queryKey) === JSON.stringify(queryKey);
  });
}

type QueryPredicate = (query: { queryKey: readonly unknown[] }) => boolean;

function getInvalidationPredicates(invalidateQueriesSpy: ReturnType<typeof vi.spyOn>) {
  const predicates: unknown[] = invalidateQueriesSpy.mock.calls.map(
    (call: unknown[]) => (call[0] as { predicate?: unknown }).predicate,
  );
  return predicates.filter((predicate: unknown): predicate is QueryPredicate => typeof predicate === "function");
}

function seedInfiniteRuns(
  queryClient: QueryClient,
  runs: CanvasesCanvasRun[] = [],
  filters?: Parameters<typeof canvasKeys.infiniteRuns>[1],
) {
  queryClient.setQueryData<InfiniteData<InfiniteRunsPage>>(canvasKeys.infiniteRuns(testCanvasId, filters), {
    pages: [{ runs, totalCount: runs.length, hasNextPage: false }],
    pageParams: [undefined],
  });
}

afterEach(() => {
  vi.clearAllMocks();
});

describe("useCanvasWebsocket", () => {
  it("refreshes runs for root workflow events without patching them into the cache", async () => {
    const queryClient = new QueryClient();
    const invalidateQueriesSpy = vi.spyOn(queryClient, "invalidateQueries").mockResolvedValue();
    seedInfiniteRuns(queryClient, [
      {
        id: "run-old",
        canvasId: testCanvasId,
        rootEvent: { id: "event-old", nodeId: testNodeId },
        executions: [],
      },
    ]);

    renderCanvasWebsocketHook(queryClient);
    emitWebsocketMessage("event_created", {
      id: "event-new",
      nodeId: testNodeId,
      root: true,
      createdAt: "2026-06-01T12:00:00.000Z",
    });

    await flushMessageQueue();

    await waitFor(() => {
      const data = queryClient.getQueryData<InfiniteData<InfiniteRunsPage>>(canvasKeys.infiniteRuns(testCanvasId));
      expect(data?.pages[0]?.runs?.map((run) => run.rootEvent?.id)).toEqual(["event-old"]);
    });
    expect(getInvalidationCalls(invalidateQueriesSpy, canvasKeys.infiniteRuns(testCanvasId))).toHaveLength(1);
  });

  it("invalidates infinite runs query for queue_item_created", async () => {
    const queryClient = new QueryClient();
    const invalidateQueriesSpy = vi.spyOn(queryClient, "invalidateQueries").mockResolvedValue();

    renderCanvasWebsocketHook(queryClient);
    emitWebsocketMessage("queue_item_created", {
      id: "queue-item-1",
      nodeId: testNodeId,
    });

    await flushMessageQueue();

    expect(getInvalidationCalls(invalidateQueriesSpy, canvasKeys.infiniteRuns(testCanvasId))).toHaveLength(1);
  });

  it("does not invalidate infinite runs query for queue_item_consumed", async () => {
    const queryClient = new QueryClient();
    const invalidateQueriesSpy = vi.spyOn(queryClient, "invalidateQueries").mockResolvedValue();

    renderCanvasWebsocketHook(queryClient);
    emitWebsocketMessage("queue_item_consumed", {
      id: "queue-item-1",
      nodeId: testNodeId,
    });

    await flushMessageQueue();

    expect(getInvalidationCalls(invalidateQueriesSpy, canvasKeys.infiniteRuns(testCanvasId))).toHaveLength(0);
  });

  it("patches execution events into infinite runs cache", async () => {
    const queryClient = new QueryClient();
    const invalidateQueriesSpy = vi.spyOn(queryClient, "invalidateQueries").mockResolvedValue();
    seedInfiniteRuns(queryClient, [
      {
        id: "run-1",
        canvasId: testCanvasId,
        state: "STATE_STARTED",
        rootEvent: { id: "event-1", nodeId: testNodeId },
        executions: [],
      },
    ]);

    renderCanvasWebsocketHook(queryClient);
    emitWebsocketMessage("execution_created", {
      id: "execution-1",
      nodeId: testNodeId,
      state: "STATE_PENDING",
      updatedAt: "2026-06-01T12:00:00.000Z",
      rootEvent: { id: "event-1", nodeId: testNodeId },
    });

    await flushMessageQueue();

    await waitFor(() => {
      const runsData = queryClient.getQueryData<InfiniteData<InfiniteRunsPage>>(canvasKeys.infiniteRuns(testCanvasId));
      expect(runsData?.pages[0]?.runs?.[0]?.executions?.[0]?.id).toBe("execution-1");
    });
    expect(getInvalidationCalls(invalidateQueriesSpy, canvasKeys.infiniteRuns(testCanvasId))).toHaveLength(0);
  });

  it("invalidates infinite runs when execution events cannot be patched into cached runs", async () => {
    const queryClient = new QueryClient();
    const invalidateQueriesSpy = vi.spyOn(queryClient, "invalidateQueries").mockResolvedValue();
    seedInfiniteRuns(queryClient, []);

    renderCanvasWebsocketHook(queryClient);
    emitWebsocketMessage("execution_created", {
      id: "execution-1",
      nodeId: testNodeId,
      state: "STATE_PENDING",
      updatedAt: "2026-06-01T12:00:00.000Z",
      rootEvent: { id: "event-1", nodeId: testNodeId },
    });

    await flushMessageQueue();

    expect(getInvalidationCalls(invalidateQueriesSpy, canvasKeys.infiniteRuns(testCanvasId))).toHaveLength(1);
  });

  it("patches run events into all infinite runs cache variants", async () => {
    const queryClient = new QueryClient();
    const invalidateQueriesSpy = vi.spyOn(queryClient, "invalidateQueries").mockResolvedValue();
    seedInfiniteRuns(queryClient, []);
    seedInfiniteRuns(queryClient, [], { states: ["STATE_STARTED"] });

    renderCanvasWebsocketHook(queryClient);
    emitWebsocketMessage("run_finished", {
      id: "run-1",
      canvasId: testCanvasId,
      state: "STATE_FINISHED",
      result: "RESULT_PASSED",
      createdAt: "2026-06-01T12:00:00.000Z",
      updatedAt: "2026-06-01T12:01:00.000Z",
    });

    const allRuns = queryClient.getQueryData<InfiniteData<InfiniteRunsPage>>(canvasKeys.infiniteRuns(testCanvasId));
    const startedRuns = queryClient.getQueryData<InfiniteData<InfiniteRunsPage>>(
      canvasKeys.infiniteRuns(testCanvasId, { states: ["STATE_STARTED"] }),
    );

    expect(allRuns?.pages[0]?.runs?.[0]?.state).toBe("STATE_FINISHED");
    expect(startedRuns?.pages[0]?.runs).toEqual([]);
    expect(getInvalidationCalls(invalidateQueriesSpy, canvasKeys.infiniteRuns(testCanvasId))).toHaveLength(0);
  });

  it("rejects stale run events when patching the describe-run cache", () => {
    const queryClient = new QueryClient();
    queryClient.setQueryData(canvasKeys.run(testCanvasId, "run-1"), {
      run: {
        id: "run-1",
        canvasId: testCanvasId,
        state: "STATE_FINISHED",
        result: "RESULT_PASSED",
        updatedAt: "2026-06-01T12:01:00.000Z",
      },
    });

    renderCanvasWebsocketHook(queryClient);
    emitWebsocketMessage("run_started", {
      id: "run-1",
      canvasId: testCanvasId,
      state: "STATE_STARTED",
      updatedAt: "2026-06-01T12:01:00.000Z",
    });

    const describedRun = queryClient.getQueryData<{ run?: CanvasesCanvasRun }>(canvasKeys.run(testCanvasId, "run-1"));
    expect(describedRun?.run?.state).toBe("STATE_FINISHED");
    expect(describedRun?.run?.result).toBe("RESULT_PASSED");
  });

  it("seeds describe-run cache when websocket events arrive before describe loads", () => {
    const queryClient = new QueryClient();

    renderCanvasWebsocketHook(queryClient);
    emitWebsocketMessage("run_finished", {
      id: "run-1",
      canvasId: testCanvasId,
      state: "STATE_FINISHED",
      result: "RESULT_PASSED",
      updatedAt: "2026-06-01T12:01:00.000Z",
    });

    const describedRun = queryClient.getQueryData<{ run?: CanvasesCanvasRun }>(canvasKeys.run(testCanvasId, "run-1"));
    expect(describedRun?.run?.state).toBe("STATE_FINISHED");
    expect(describedRun?.run?.result).toBe("RESULT_PASSED");
  });

  it("does not invalidate runs on initial websocket connect", () => {
    const queryClient = new QueryClient();
    const invalidateQueriesSpy = vi.spyOn(queryClient, "invalidateQueries").mockResolvedValue();

    renderCanvasWebsocketHook(queryClient);
    emitWebSocketOpen();

    expect(getInvalidationCalls(invalidateQueriesSpy, canvasKeys.infiniteRuns(testCanvasId))).toHaveLength(0);
  });

  it("invalidates runs on websocket reconnect", () => {
    const queryClient = new QueryClient();
    const invalidateQueriesSpy = vi.spyOn(queryClient, "invalidateQueries").mockResolvedValue();

    renderCanvasWebsocketHook(queryClient);
    emitWebSocketOpen();
    emitWebSocketOpen();

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

  it("invalidates draft branch queries for canvas version deletions", () => {
    const queryClient = new QueryClient();
    const invalidateQueriesSpy = vi.spyOn(queryClient, "invalidateQueries").mockResolvedValue();

    renderCanvasWebsocketHook(queryClient);
    emitWebsocketMessage("canvas_version_deleted", {
      canvasId: testCanvasId,
      versionId: "version-1",
    });

    expect(invalidateQueriesSpy).toHaveBeenCalledWith({
      queryKey: canvasKeys.versionList(testCanvasId),
    });
    expect(invalidateQueriesSpy).toHaveBeenCalledWith({
      queryKey: canvasKeys.draftBranches(testCanvasId),
    });
    expect(invalidateQueriesSpy).not.toHaveBeenCalledWith({
      queryKey: canvasKeys.consoleAll(testCanvasId),
    });
  });

  it("skips lifecycle invalidation when canvas_version_updated echo is consumed", () => {
    const queryClient = new QueryClient();
    const invalidateQueriesSpy = vi.spyOn(queryClient, "invalidateQueries").mockResolvedValue();
    const onCanvasLifecycleEvent = vi.fn().mockReturnValue(false);

    renderHook(
      () =>
        useCanvasWebsocket(testCanvasId, testOrganizationId, undefined, undefined, undefined, onCanvasLifecycleEvent),
      {
        wrapper: ({ children }: { children: ReactNode }) =>
          createElement(QueryClientProvider, { client: queryClient }, children),
      },
    );

    emitWebsocketMessage("canvas_version_updated", {
      canvasId: testCanvasId,
      versionId: "version-1",
    });

    expect(onCanvasLifecycleEvent).toHaveBeenCalledWith(
      { canvasId: testCanvasId, versionId: "version-1" },
      "canvas_version_updated",
    );
    expect(getInvalidationCalls(invalidateQueriesSpy, canvasKeys.versionList(testCanvasId))).toHaveLength(0);
    expect(getInvalidationCalls(invalidateQueriesSpy, canvasKeys.consoleAll(testCanvasId))).toHaveLength(0);
  });

  it("invalidates staged caches for staging_updated events", () => {
    const queryClient = new QueryClient();
    const invalidateQueriesSpy = vi.spyOn(queryClient, "invalidateQueries").mockResolvedValue();
    const onCanvasStagingEvent = vi.fn();

    renderHook(
      () =>
        useCanvasWebsocket(
          testCanvasId,
          testOrganizationId,
          undefined,
          undefined,
          undefined,
          undefined,
          undefined,
          false,
          true,
          onCanvasStagingEvent,
        ),
      {
        wrapper: ({ children }: { children: ReactNode }) =>
          createElement(QueryClientProvider, { client: queryClient }, children),
      },
    );

    emitWebsocketMessage("staging_updated", {
      canvasId: testCanvasId,
      versionId: "version-1",
    });

    expect(onCanvasStagingEvent).toHaveBeenCalledWith(
      { canvasId: testCanvasId, versionId: "version-1" },
      "staging_updated",
    );
    expect(
      getInvalidationCalls(invalidateQueriesSpy, canvasKeys.versionStaging(testCanvasId, "version-1")),
    ).toHaveLength(1);
    expect(getInvalidationCalls(invalidateQueriesSpy, canvasKeys.repositoryFiles(testCanvasId))).toHaveLength(1);

    const [stagedPredicate] = getInvalidationPredicates(invalidateQueriesSpy);
    expect(stagedPredicate).toBeDefined();
    expect(stagedPredicate({ queryKey: canvasKeys.versionStagedDetail(testCanvasId, "version-1") })).toBe(true);
    expect(stagedPredicate({ queryKey: canvasKeys.consoleStaged(testCanvasId, "version-1") })).toBe(true);
    expect(stagedPredicate({ queryKey: canvasKeys.repositoryFile(testCanvasId, "README.md", "version-1", true) })).toBe(
      true,
    );
    expect(
      stagedPredicate({ queryKey: canvasKeys.repositoryFileContent(testCanvasId, "README.md", "version-1", true) }),
    ).toBe(true);
    expect(stagedPredicate({ queryKey: canvasKeys.repositoryFile(testCanvasId, "README.md") })).toBe(false);
    expect(stagedPredicate({ queryKey: canvasKeys.repositoryFile(testCanvasId, "README.md", "version-2") })).toBe(
      false,
    );
    expect(
      stagedPredicate({ queryKey: canvasKeys.repositoryFileContent(testCanvasId, "README.md", "version-1", false) }),
    ).toBe(false);
  });

  it("skips staging invalidation when onCanvasStagingEvent returns false", () => {
    const queryClient = new QueryClient();
    const invalidateQueriesSpy = vi.spyOn(queryClient, "invalidateQueries").mockResolvedValue();
    const onCanvasStagingEvent = vi.fn().mockReturnValue(false);

    renderHook(
      () =>
        useCanvasWebsocket(
          testCanvasId,
          testOrganizationId,
          undefined,
          undefined,
          undefined,
          undefined,
          undefined,
          false,
          true,
          onCanvasStagingEvent,
        ),
      {
        wrapper: ({ children }: { children: ReactNode }) =>
          createElement(QueryClientProvider, { client: queryClient }, children),
      },
    );

    emitWebsocketMessage("staging_updated", {
      canvasId: testCanvasId,
      versionId: "version-1",
    });

    expect(onCanvasStagingEvent).toHaveBeenCalledOnce();
    expect(
      getInvalidationCalls(invalidateQueriesSpy, canvasKeys.versionStaging(testCanvasId, "version-1")),
    ).toHaveLength(0);
  });
});
