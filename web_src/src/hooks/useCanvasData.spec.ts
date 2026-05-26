import type { CanvasesCanvas } from "@/api-client";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react";
import { createElement, type ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

const { canvasFoldersUpdateCanvasFolder, canvasesListRuns } = vi.hoisted(() => ({
  canvasFoldersUpdateCanvasFolder: vi.fn(),
  canvasesListRuns: vi.fn(),
}));

vi.mock("../api-client/sdk.gen", async (importOriginal) => {
  const actual = await importOriginal();
  return {
    ...(actual as Record<string, unknown>),
    canvasFoldersUpdateCanvasFolder,
    canvasesListRuns,
  };
});

import { canvasKeys, useInfiniteCanvasRuns, useUpdateCanvasFolderMembership } from "@/hooks/useCanvasData";

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

function makeCanvas(id: string, folderId?: string): CanvasesCanvas {
  return {
    metadata: {
      id,
      name: id,
      folderId,
    },
    spec: {},
  } as CanvasesCanvas;
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
  const canvases = queryClient.getQueryData<CanvasesCanvas[]>(canvasKeys.list(organizationId)) || [];
  const canvas = canvases.find((item) => item.metadata?.id === canvasId);
  return (canvas?.metadata as { folderId?: string } | undefined)?.folderId;
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
