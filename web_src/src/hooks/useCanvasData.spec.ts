import type { CanvasesCanvasSummary } from "@/api-client";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook, waitFor } from "@testing-library/react";
import { createElement, type ReactNode } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const {
  canvasFoldersUpdateCanvasFolder,
  canvasesListRuns,
  canvasesDescribeRun,
  canvasesPutCanvasStaging,
  canvasesCommitCanvasStaging,
  canvasesDeleteCanvasStaging,
  canvasesDescribeCanvasVersion,
  canvasesListCanvasVersions,
  canvasesGetCanvasStaging,
} = vi.hoisted(() => ({
  canvasFoldersUpdateCanvasFolder: vi.fn(),
  canvasesListRuns: vi.fn(),
  canvasesDescribeRun: vi.fn(),
  canvasesPutCanvasStaging: vi.fn(),
  canvasesCommitCanvasStaging: vi.fn(),
  canvasesDeleteCanvasStaging: vi.fn(),
  canvasesDescribeCanvasVersion: vi.fn(),
  canvasesListCanvasVersions: vi.fn(),
  canvasesGetCanvasStaging: vi.fn(),
}));

vi.mock("../api-client/sdk.gen", async (importOriginal) => {
  const actual = await importOriginal();
  return {
    ...(actual as Record<string, unknown>),
    canvasFoldersUpdateCanvasFolder,
    canvasesListRuns,
    canvasesDescribeRun,
    canvasesPutCanvasStaging,
    canvasesCommitCanvasStaging,
    canvasesDeleteCanvasStaging,
    canvasesDescribeCanvasVersion,
    canvasesListCanvasVersions,
    canvasesGetCanvasStaging,
  };
});

import {
  canvasKeys,
  useDescribeRun,
  useInfiniteCanvasRuns,
  useUpdateCanvasConsole,
  useUpdateCanvasFolderMembership,
} from "@/hooks/useCanvasData";

type TestCanvasFolder = {
  metadata?: { id?: string };
  spec?: {
    title?: string;
    backgroundColor?: string;
    canvases?: Array<{ id?: string }>;
  };
};

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

function makeCanvas(id: string, folderId?: string): CanvasesCanvasSummary {
  return {
    id,
    name: id,
    folderId,
  } as CanvasesCanvasSummary;
}

function makeFolder(id: string, canvasIds: string[] = []): TestCanvasFolder {
  return {
    metadata: { id },
    spec: {
      title: id,
      backgroundColor: "blue",
      canvases: canvasIds.map((canvasId) => ({ id: canvasId })),
    },
  };
}

function createDeferred<T>() {
  let resolve!: (value: T) => void;
  let reject!: (reason?: unknown) => void;
  const promise = new Promise<T>((promiseResolve, promiseReject) => {
    resolve = promiseResolve;
    reject = promiseReject;
  });

  return { promise, resolve, reject };
}

function getCanvasFolderId(queryClient: QueryClient, organizationId: string, canvasId: string) {
  const canvases = queryClient.getQueryData<CanvasesCanvasSummary[]>(canvasKeys.list(organizationId)) || [];
  const canvas = canvases.find((item) => item.id === canvasId);
  return canvas?.folderId;
}

function getFolderCanvasIds(queryClient: QueryClient, organizationId: string, folderId: string) {
  const folders = queryClient.getQueryData<TestCanvasFolder[]>(canvasKeys.folderList(organizationId)) || [];
  const folder = folders.find((item) => item.metadata?.id === folderId);
  return folder?.spec?.canvases?.map((canvas) => canvas.id) || [];
}

describe("useUpdateCanvasFolderMembership", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("optimistically moves canvases between folders before the request finishes", async () => {
    const organizationId = "org-123";
    const queryClient = createQueryClient();
    const deferred = createDeferred<unknown>();
    canvasFoldersUpdateCanvasFolder.mockReturnValue(deferred.promise);

    queryClient.setQueryData(canvasKeys.list(organizationId), [
      makeCanvas("canvas-1", "folder-1"),
      makeCanvas("canvas-2", "folder-2"),
    ]);
    queryClient.setQueryData(canvasKeys.folderList(organizationId), [
      makeFolder("folder-1", ["canvas-1"]),
      makeFolder("folder-2", ["canvas-2"]),
    ]);

    const { result } = renderHook(() => useUpdateCanvasFolderMembership(organizationId), {
      wrapper: createWrapper(queryClient),
    });

    const mutation = result.current.mutateAsync({
      folderId: "folder-2",
      title: "Deployments",
      backgroundColor: "green",
      canvasIds: ["canvas-2", "canvas-1"],
    });

    await waitFor(() => {
      expect(getCanvasFolderId(queryClient, organizationId, "canvas-1")).toBe("folder-2");
    });
    expect(getFolderCanvasIds(queryClient, organizationId, "folder-1")).toEqual([]);
    expect(getFolderCanvasIds(queryClient, organizationId, "folder-2")).toEqual(["canvas-2", "canvas-1"]);

    deferred.resolve({});
    await mutation;
  });

  it("rolls back optimistic folder membership on request failure", async () => {
    const organizationId = "org-123";
    const queryClient = createQueryClient();
    canvasFoldersUpdateCanvasFolder.mockRejectedValue(new Error("request failed"));

    queryClient.setQueryData(canvasKeys.list(organizationId), [makeCanvas("canvas-1", "folder-1")]);
    queryClient.setQueryData(canvasKeys.folderList(organizationId), [
      makeFolder("folder-1", ["canvas-1"]),
      makeFolder("folder-2"),
    ]);

    const { result } = renderHook(() => useUpdateCanvasFolderMembership(organizationId), {
      wrapper: createWrapper(queryClient),
    });

    await expect(
      result.current.mutateAsync({
        folderId: "folder-2",
        title: "Deployments",
        backgroundColor: "green",
        canvasIds: ["canvas-1"],
      }),
    ).rejects.toThrow("request failed");

    expect(getCanvasFolderId(queryClient, organizationId, "canvas-1")).toBe("folder-1");
    expect(getFolderCanvasIds(queryClient, organizationId, "folder-1")).toEqual(["canvas-1"]);
    expect(getFolderCanvasIds(queryClient, organizationId, "folder-2")).toEqual([]);
  });
});

const emptyConsoleYaml =
  "apiVersion: v1\nkind: Console\nmetadata:\n  canvasId: canvas-1\nspec:\n  panels: []\n  layout: []\n";

const committedEmptyConsoleVersion = {
  metadata: { id: "version-1" },
  spec: { panels: [], layout: [] },
};

const committedConsoleVersionWithPanel = {
  metadata: { id: "version-1" },
  spec: {
    panels: [{ id: "panel-1", type: "markdown", content: { title: "Before" } }],
    layout: [{ i: "panel-1", x: 0, y: 0, w: 12, h: 6 }],
  },
};

function mockCommittedConsoleVersion(
  version: typeof committedEmptyConsoleVersion | typeof committedConsoleVersionWithPanel = committedEmptyConsoleVersion,
) {
  canvasesDescribeCanvasVersion.mockResolvedValue({
    data: { version },
  } as never);
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
      data: { staging: { hasStaging: false, stagedPaths: [] } },
    });
    mockCommittedConsoleVersion();
    canvasesGetCanvasStaging.mockResolvedValue({
      data: { staging: { hasStaging: false, stagedPaths: [] } },
    });

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
    mockCommittedConsoleVersion(committedConsoleVersionWithPanel);
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
    mockCommittedConsoleVersion();
    canvasesGetCanvasStaging.mockResolvedValue({
      data: {
        staging: { hasStaging: true, stagedPaths: ["console.yaml"], spec: committedConsoleVersionWithPanel.spec },
      },
    });

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
    mockCommittedConsoleVersion();
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

  it("uses staging from the write response instead of a warm React Query cache", async () => {
    const queryClient = createQueryClient();
    const dashboardKey = canvasKeys.stagedConsole("canvas-1");
    const staleStaging = {
      hasStaging: false,
      stagedPaths: [],
      spec: committedEmptyConsoleVersion.spec,
    };
    queryClient.setQueryData(canvasKeys.canvasStaging("canvas-1"), staleStaging);
    queryClient.setQueryData(dashboardKey, {
      canvasId: "canvas-1",
      versionId: "version-1",
      panels: [{ id: "panel-1", type: "markdown", content: { title: "Before", body: "Before body" } }],
      layout: [{ i: "panel-1", x: 0, y: 0, w: 12, h: 6 }],
      consoleYaml: emptyConsoleYaml,
    });

    const updatedPanels = [{ id: "panel-1", type: "markdown", content: { title: "After", body: "After body" } }];
    const writeResponseStaging = {
      hasStaging: true,
      stagedPaths: ["console.yaml"],
      spec: {
        panels: updatedPanels,
        layout: [{ i: "panel-1", x: 0, y: 0, w: 12, h: 6 }],
      },
    };
    canvasesPutCanvasStaging.mockResolvedValue({ data: { staging: writeResponseStaging } });
    mockCommittedConsoleVersion(committedConsoleVersionWithPanel);
    canvasesGetCanvasStaging.mockResolvedValue({
      data: { staging: staleStaging },
    });

    const { result } = renderHook(() => useUpdateCanvasConsole("canvas-1", "version-1"), {
      wrapper: createWrapper(queryClient),
    });

    await result.current.mutateAsync({
      panels: updatedPanels,
      layout: [{ i: "panel-1", x: 0, y: 0, w: 12, h: 6 }],
    });

    expect(canvasesGetCanvasStaging).not.toHaveBeenCalled();
    expect(queryClient.getQueryData(dashboardKey)).toMatchObject({
      panels: updatedPanels,
    });
    expect(queryClient.getQueryData(canvasKeys.canvasStaging("canvas-1"))).toMatchObject({
      hasStaging: true,
      stagedPaths: ["console.yaml"],
    });
  });
});
