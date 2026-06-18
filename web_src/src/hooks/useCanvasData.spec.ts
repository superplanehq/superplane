import type { CanvasesCanvasSummary } from "@/api-client";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook, waitFor } from "@testing-library/react";
import { createElement, type ReactNode } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const {
  canvasFoldersUpdateCanvasFolder,
  canvasesListRuns,
  canvasesDescribeRun,
  canvasesStageCanvasRepositoryFile,
  canvasesCommitCanvasStaging,
  canvasesDiscardCanvasStaging,
  canvasesDescribeCanvasVersion,
  canvasesListCanvasVersions,
} = vi.hoisted(() => ({
  canvasFoldersUpdateCanvasFolder: vi.fn(),
  canvasesListRuns: vi.fn(),
  canvasesDescribeRun: vi.fn(),
  canvasesStageCanvasRepositoryFile: vi.fn(),
  canvasesCommitCanvasStaging: vi.fn(),
  canvasesDiscardCanvasStaging: vi.fn(),
  canvasesDescribeCanvasVersion: vi.fn(),
  canvasesListCanvasVersions: vi.fn(),
}));

vi.mock("../api-client/sdk.gen", async (importOriginal) => {
  const actual = await importOriginal();
  return {
    ...(actual as Record<string, unknown>),
    canvasFoldersUpdateCanvasFolder,
    canvasesListRuns,
    canvasesDescribeRun,
    canvasesStageCanvasRepositoryFile,
    canvasesCommitCanvasStaging,
    canvasesDiscardCanvasStaging,
    canvasesDescribeCanvasVersion,
    canvasesListCanvasVersions,
  };
});

import {
  canvasKeys,
  ensureDraftVersionExists,
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

const afterConsoleYaml =
  "apiVersion: v1\nkind: Console\nmetadata:\n  canvasId: canvas-1\nspec:\n  panels:\n    - id: panel-1\n      type: markdown\n      content:\n        title: After\n  layout:\n    - i: panel-1\n      x: 0\n      y: 0\n      w: 12\n      h: 6\n";

function mockConsoleRepositoryFileFetch(yamlBody: string, options?: { committedYaml?: string }) {
  vi.stubGlobal(
    "fetch",
    vi.fn(async (input: RequestInfo | URL) => {
      const url = typeof input === "string" ? input : input.toString();
      if (url.includes("/repository/file") && url.includes("console.yaml")) {
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

  it("registers a canvas version websocket echo before saving dashboard changes", async () => {
    const queryClient = createQueryClient();
    const registerIgnoredCanvasVersionUpdatedEcho = vi.fn(() => vi.fn());
    canvasesStageCanvasRepositoryFile.mockResolvedValue({ data: {} });
    canvasesCommitCanvasStaging.mockResolvedValue({ data: {} });
    canvasesDiscardCanvasStaging.mockResolvedValue({
      data: { stagingSummary: { hasStaging: false, stagedPaths: [] } },
    });
    canvasesDescribeCanvasVersion.mockResolvedValue({
      data: { stagingSummary: { hasStaging: false, stagedPaths: [] } },
    });
    mockConsoleRepositoryFileFetch(emptyConsoleYaml);

    const { result } = renderHook(
      () =>
        useUpdateCanvasConsole("canvas-1", "version-1", {
          registerIgnoredCanvasVersionUpdatedEcho,
        }),
      { wrapper: createWrapper(queryClient) },
    );

    await result.current.mutateAsync({ panels: [], layout: [] });

    expect(registerIgnoredCanvasVersionUpdatedEcho).toHaveBeenCalledWith("version-1");
    expect(canvasesDiscardCanvasStaging).toHaveBeenCalledOnce();
    expect(canvasesDiscardCanvasStaging).toHaveBeenCalledWith(
      expect.objectContaining({
        path: { canvasId: "canvas-1", versionId: "version-1" },
        body: { paths: ["console.yaml"] },
      }),
    );
    // Console edits stage only; committing into the version row is an explicit action.
    expect(canvasesCommitCanvasStaging).not.toHaveBeenCalled();
  });

  it("releases the ignored canvas version echo when dashboard save fails", async () => {
    const queryClient = createQueryClient();
    const releaseCanvasVersionUpdatedEcho = vi.fn();
    const registerIgnoredCanvasVersionUpdatedEcho = vi.fn(() => releaseCanvasVersionUpdatedEcho);
    canvasesStageCanvasRepositoryFile.mockRejectedValue(new Error("request failed"));

    const { result } = renderHook(
      () =>
        useUpdateCanvasConsole("canvas-1", "version-1", {
          registerIgnoredCanvasVersionUpdatedEcho,
        }),
      { wrapper: createWrapper(queryClient) },
    );

    await expect(result.current.mutateAsync({ panels: [], layout: [] })).rejects.toThrow("request failed");

    expect(registerIgnoredCanvasVersionUpdatedEcho).toHaveBeenCalledWith("version-1");
    expect(releaseCanvasVersionUpdatedEcho).toHaveBeenCalledOnce();
  });

  it("optimistically updates the dashboard cache while console changes are saving", async () => {
    const queryClient = createQueryClient();
    const dashboardKey = canvasKeys.consoleStaged("canvas-1", "version-1");
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
    canvasesStageCanvasRepositoryFile.mockReturnValue(savePromise);
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
    const dashboardKey = canvasKeys.consoleStaged("canvas-1", "version-1");
    queryClient.setQueryData(dashboardKey, {
      canvasId: "canvas-1",
      versionId: "version-1",
      panels: [{ id: "panel-1", type: "markdown", content: { title: "Before" } }],
      layout: [{ i: "panel-1", x: 0, y: 0, w: 12, h: 6 }],
    });
    canvasesStageCanvasRepositoryFile.mockRejectedValue(new Error("request failed"));

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

describe("ensureDraftVersionExists", () => {
  beforeEach(() => {
    canvasesListCanvasVersions.mockReset();
  });

  it("returns true when the draft version is still present", async () => {
    canvasesListCanvasVersions.mockResolvedValue({
      data: { versions: [{ metadata: { id: "v-1" } }, { metadata: { id: "v-2" } }] },
    });
    const queryClient = new QueryClient();

    const exists = await ensureDraftVersionExists(queryClient, "org-1", "canvas-1", "v-2");

    expect(exists).toBe(true);
    expect(canvasesListCanvasVersions).toHaveBeenCalledTimes(1);
  });

  it("returns false when the draft version is gone", async () => {
    canvasesListCanvasVersions.mockResolvedValue({
      data: { versions: [{ metadata: { id: "v-1" } }] },
    });
    const queryClient = new QueryClient();

    const exists = await ensureDraftVersionExists(queryClient, "org-1", "canvas-1", "deleted");

    expect(exists).toBe(false);
  });

  it("short-circuits to false without calling the API when args are missing", async () => {
    const queryClient = new QueryClient();

    const exists = await ensureDraftVersionExists(queryClient, "org-1", "canvas-1", "");

    expect(exists).toBe(false);
    expect(canvasesListCanvasVersions).not.toHaveBeenCalled();
  });
});
