import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook, waitFor } from "@testing-library/react";
import { createElement, type ReactNode } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "bun:test";

const {
  canvasesListRuns,
  canvasesDescribeRun,
  canvasesPutCanvasStaging,
  canvasesCommitCanvasStaging,
  canvasesDeleteCanvasStaging,
  canvasesDescribeCanvasVersion,
  canvasesListCanvasVersions,
  canvasesGetCanvasStaging,
} = vi.hoisted(() => ({
  canvasesListRuns: vi.fn(),
  canvasesDescribeRun: vi.fn(),
  canvasesPutCanvasStaging: vi.fn(),
  canvasesCommitCanvasStaging: vi.fn(),
  canvasesDeleteCanvasStaging: vi.fn(),
  canvasesDescribeCanvasVersion: vi.fn(),
  canvasesListCanvasVersions: vi.fn(),
  canvasesGetCanvasStaging: vi.fn(),
}));

vi.mock("../api-client/sdk.gen", () => ({
  canvasesListRuns,
  canvasesDescribeRun,
  canvasesPutCanvasStaging,
  canvasesCommitCanvasStaging,
  canvasesDeleteCanvasStaging,
  canvasesDescribeCanvasVersion,
  canvasesListCanvasVersions,
  canvasesGetCanvasStaging,
}));

import { canvasKeys, useDescribeRun, useInfiniteCanvasRuns, useUpdateCanvasConsole } from "@/hooks/useCanvasData";

describe("canvasKeys.nodeExecution", () => {
  it("omits a trailing undefined when limit is not provided", () => {
    expect(canvasKeys.nodeExecution("canvas-1", "node-1")).toEqual([
      "canvases",
      "nodeExecutions",
      "canvas-1",
      "node-1",
    ]);
  });

  it("matches limited cached queries when invalidating by prefix", async () => {
    const queryClient = new QueryClient();
    const cachedKey = canvasKeys.nodeExecution("canvas-1", "node-1", undefined, 10);

    queryClient.setQueryData(cachedKey, { executions: [] });

    await queryClient.invalidateQueries({
      queryKey: canvasKeys.nodeExecution("canvas-1", "node-1"),
    });

    expect(queryClient.getQueryState(cachedKey)?.isInvalidated).toBe(true);
  });
});

describe("useInfiniteCanvasRuns", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("fetches runs and paginates with the previous page last timestamp", async () => {
    const queryClient = createQueryClient();
    canvasesListRuns
      .mockResolvedValueOnce({
        data: {
          runs: [{ id: "run-1", state: "STATE_FINISHED", result: "RESULT_PASSED" }],
          totalCount: 2,
          hasNextPage: true,
          lastTimestamp: "2026-05-01T00:00:00Z",
        },
      })
      .mockResolvedValueOnce({
        data: {
          runs: [{ id: "run-2", state: "STATE_STARTED" }],
          totalCount: 2,
          hasNextPage: false,
          lastTimestamp: "2026-04-30T00:00:00Z",
        },
      });

    const { result } = renderHook(() => useInfiniteCanvasRuns("canvas-1"), {
      wrapper: createWrapper(queryClient),
    });

    await waitFor(() => {
      expect(result.current.data?.pages[0]?.runs?.[0]?.id).toBe("run-1");
    });

    await result.current.fetchNextPage();

    expect(canvasesListRuns).toHaveBeenNthCalledWith(
      1,
      expect.objectContaining({
        path: { canvasId: "canvas-1" },
        query: { limit: 25 },
      }),
    );
    expect(canvasesListRuns).toHaveBeenNthCalledWith(
      2,
      expect.objectContaining({
        path: { canvasId: "canvas-1" },
        query: { limit: 25, before: "2026-05-01T00:00:00Z" },
      }),
    );
  });

  it("passes run filters to the list runs request", async () => {
    const queryClient = createQueryClient();
    canvasesListRuns.mockResolvedValueOnce({
      data: {
        runs: [],
        totalCount: 0,
        hasNextPage: false,
      },
    });

    renderHook(
      () =>
        useInfiniteCanvasRuns("canvas-1", {
          states: ["STATE_FINISHED"],
          results: ["RESULT_FAILED", "RESULT_CANCELLED"],
        }),
      {
        wrapper: createWrapper(queryClient),
      },
    );

    await waitFor(() => {
      expect(canvasesListRuns).toHaveBeenCalledWith(
        expect.objectContaining({
          path: { canvasId: "canvas-1" },
          query: {
            limit: 25,
            states: ["STATE_FINISHED"],
            results: ["RESULT_FAILED", "RESULT_CANCELLED"],
          },
        }),
      );
    });
  });
});

describe("useDescribeRun", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("fetches a run by id", async () => {
    const queryClient = createQueryClient();
    canvasesDescribeRun.mockResolvedValueOnce({
      data: {
        run: { id: "run-42", state: "STATE_FINISHED", result: "RESULT_PASSED" },
      },
    });

    const { result } = renderHook(() => useDescribeRun("canvas-1", "run-42"), {
      wrapper: createWrapper(queryClient),
    });

    await waitFor(() => {
      expect(result.current.data?.run?.id).toBe("run-42");
    });

    expect(canvasesDescribeRun).toHaveBeenCalledWith(
      expect.objectContaining({
        path: { canvasId: "canvas-1", runId: "run-42" },
      }),
    );
  });

  it("does not overwrite fresher websocket run state in the describe cache", async () => {
    const queryClient = createQueryClient();
    queryClient.setQueryData(canvasKeys.run("canvas-1", "run-42"), {
      run: {
        id: "run-42",
        state: "STATE_FINISHED",
        result: "RESULT_PASSED",
        updatedAt: "2026-06-01T12:01:00.000Z",
      },
    });
    canvasesDescribeRun.mockResolvedValueOnce({
      data: {
        run: {
          id: "run-42",
          state: "STATE_STARTED",
          result: "RESULT_UNKNOWN",
          updatedAt: "2026-06-01T12:00:00.000Z",
        },
      },
    });

    const { result } = renderHook(() => useDescribeRun("canvas-1", "run-42"), {
      wrapper: createWrapper(queryClient),
    });

    await waitFor(() => {
      expect(result.current.data?.run?.state).toBe("STATE_FINISHED");
    });
    expect(result.current.data?.run?.result).toBe("RESULT_PASSED");
  });
});

function createQueryClient() {
  return new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  });
}

function createWrapper(queryClient: QueryClient) {
  return function Wrapper({ children }: { children: ReactNode }) {
    return createElement(QueryClientProvider, { client: queryClient }, children);
  };
}

const emptyConsoleYaml =
  "apiVersion: v1\nkind: Console\nmetadata:\n  canvasId: canvas-1\nspec:\n  panels: []\n  layout: []\n";

const afterConsoleYaml =
  "apiVersion: v1\nkind: Console\nmetadata:\n  canvasId: canvas-1\nspec:\n  panels:\n    - id: panel-1\n      type: markdown\n      content:\n        title: After\n  layout:\n    - i: panel-1\n      x: 0\n      y: 0\n      w: 12\n      h: 6\n";

function mockConsoleRepositoryFileFetch(yamlBody: string, options?: { committedYaml?: string }) {
  vi.stubGlobal(
    "fetch",
    vi.fn(async (input: RequestInfo | URL) => {
      const url = typeof input === "string" ? input : input.toString();
      if (url.includes("/file") && url.includes("console.yaml")) {
        const committedYaml = options?.committedYaml ?? yamlBody;
        const body = url.includes("stage=true") ? yamlBody : committedYaml;
        return new Response(body, { status: 200 });
      }
      return new Response("not found", { status: 404 });
    }),
  );
}

describe("useUpdateCanvasConsole", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("stages dashboard changes through the canvas staging API", async () => {
    const queryClient = createQueryClient();
    canvasesPutCanvasStaging.mockResolvedValue({ data: {} });
    canvasesCommitCanvasStaging.mockResolvedValue({ data: {} });
    canvasesDeleteCanvasStaging.mockResolvedValue({
      data: { stagingSummary: { hasStaging: false, stagedPaths: [] } },
    });
    canvasesDescribeCanvasVersion.mockResolvedValue({
      data: { stagingSummary: { hasStaging: false, stagedPaths: [] } },
    });
    canvasesGetCanvasStaging.mockResolvedValue({
      data: { stagingSummary: { hasStaging: false, stagedPaths: [] } },
    });
    mockConsoleRepositoryFileFetch(emptyConsoleYaml);

    const { result } = renderHook(() => useUpdateCanvasConsole("canvas-1", "version-1"), {
      wrapper: createWrapper(queryClient),
    });

    await result.current.mutateAsync({ panels: [], layout: [] });

    expect(canvasesDeleteCanvasStaging).toHaveBeenCalledOnce();
    expect(canvasesDeleteCanvasStaging).toHaveBeenCalledWith(
      expect.objectContaining({
        path: { canvasId: "canvas-1" },
        query: { paths: ["console.yaml"] },
      }),
    );
    expect(canvasesCommitCanvasStaging).not.toHaveBeenCalled();
  });

  it("surfaces dashboard save failures", async () => {
    const queryClient = createQueryClient();
    canvasesPutCanvasStaging.mockRejectedValue(new Error("request failed"));

    const { result } = renderHook(() => useUpdateCanvasConsole("canvas-1", "version-1"), {
      wrapper: createWrapper(queryClient),
    });

    await expect(result.current.mutateAsync({ panels: [], layout: [] })).rejects.toThrow("request failed");
  });

  it("optimistically updates the dashboard cache while console changes are saving", async () => {
    const queryClient = createQueryClient();
    const dashboardKey = canvasKeys.stagedConsole("canvas-1");
    let resolveSave: (value: unknown) => void = () => {};
    const savePromise = new Promise((resolve) => {
      resolveSave = resolve;
    });
    queryClient.setQueryData(dashboardKey, {
      canvasId: "canvas-1",
      versionId: "version-1",
      panels: [{ id: "panel-1", type: "markdown", content: { title: "Before" } }],
      layout: [{ i: "panel-1", x: 0, y: 0, w: 12, h: 6 }],
      consoleYaml: emptyConsoleYaml,
    });
    canvasesPutCanvasStaging.mockReturnValue(savePromise);
    canvasesCommitCanvasStaging.mockResolvedValue({ data: {} });
    canvasesDescribeCanvasVersion.mockResolvedValue({
      data: { stagingSummary: { hasStaging: true, stagedPaths: ["console.yaml"] } },
    });
    mockConsoleRepositoryFileFetch(afterConsoleYaml, { committedYaml: emptyConsoleYaml });

    const { result } = renderHook(() => useUpdateCanvasConsole("canvas-1", "version-1"), {
      wrapper: createWrapper(queryClient),
    });

    act(() => {
      result.current.mutate({
        panels: [{ id: "panel-1", type: "markdown", content: { title: "After" } }],
        layout: [{ i: "panel-1", x: 0, y: 0, w: 12, h: 6 }],
      });
    });

    await waitFor(() => {
      expect(queryClient.getQueryData(dashboardKey)).toMatchObject({
        panels: [{ id: "panel-1", type: "markdown", content: { title: "After" } }],
      });
    });

    resolveSave({ data: {} });

    await waitFor(() => expect(result.current.isPending).toBe(false));
  });

  it("rolls back the dashboard cache when console save fails", async () => {
    const queryClient = createQueryClient();
    const dashboardKey = canvasKeys.stagedConsole("canvas-1");
    queryClient.setQueryData(dashboardKey, {
      canvasId: "canvas-1",
      versionId: "version-1",
      panels: [{ id: "panel-1", type: "markdown", content: { title: "Before" } }],
      layout: [{ i: "panel-1", x: 0, y: 0, w: 12, h: 6 }],
    });
    canvasesPutCanvasStaging.mockRejectedValue(new Error("request failed"));

    const { result } = renderHook(() => useUpdateCanvasConsole("canvas-1", "version-1"), {
      wrapper: createWrapper(queryClient),
    });

    await expect(
      result.current.mutateAsync({
        panels: [{ id: "panel-1", type: "markdown", content: { title: "After" } }],
        layout: [{ i: "panel-1", x: 0, y: 0, w: 12, h: 6 }],
      }),
    ).rejects.toThrow("request failed");

    expect(queryClient.getQueryData(dashboardKey)).toMatchObject({
      panels: [{ id: "panel-1", type: "markdown", content: { title: "Before" } }],
    });
  });
});
