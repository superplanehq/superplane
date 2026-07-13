import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react";
import { createElement, type ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

const { canvasesListRuns } = vi.hoisted(() => ({
  canvasesListRuns: vi.fn(),
}));

vi.mock("../api-client/sdk.gen", async (importOriginal) => {
  const actual = await importOriginal();
  return {
    ...(actual as Record<string, unknown>),
    canvasesListRuns,
  };
});

import { useInfiniteCanvasRuns } from "@/hooks/useCanvasData";

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

describe("useInfiniteCanvasRuns cache reuse", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("reuses cached pages on full refetch and only re-fetches page 1", async () => {
    const queryClient = createQueryClient();
    canvasesListRuns
      .mockResolvedValueOnce({
        data: {
          runs: [{ id: "run-1", state: "STATE_FINISHED" }],
          totalCount: 3,
          hasNextPage: true,
          lastTimestamp: "2026-05-02T00:00:00Z",
        },
      })
      .mockResolvedValueOnce({
        data: {
          runs: [{ id: "run-2", state: "STATE_FINISHED" }],
          totalCount: 3,
          hasNextPage: true,
          lastTimestamp: "2026-05-01T00:00:00Z",
        },
      })
      .mockResolvedValueOnce({
        data: {
          runs: [{ id: "run-1", state: "STATE_FINISHED" }],
          totalCount: 3,
          hasNextPage: true,
          lastTimestamp: "2026-05-02T00:00:00Z",
        },
      });

    const { result } = renderHook(() => useInfiniteCanvasRuns("canvas-1"), {
      wrapper: createWrapper(queryClient),
    });

    await waitFor(() => {
      expect(result.current.data?.pages).toHaveLength(1);
    });

    await result.current.fetchNextPage();
    await waitFor(() => {
      expect(result.current.data?.pages).toHaveLength(2);
    });
    expect(canvasesListRuns).toHaveBeenCalledTimes(2);

    await result.current.refetch();
    // Full refetch: page 1 hits the network, page 2 is reused from cache.
    // The third mock (index 2) is consumed by the page-1 network call; the
    // page-2 cursor is served from the existing infinite cache.
    expect(canvasesListRuns).toHaveBeenCalledTimes(3);
    expect(canvasesListRuns).toHaveBeenNthCalledWith(
      3,
      expect.objectContaining({
        path: { canvasId: "canvas-1" },
        query: { limit: 25 },
      }),
    );
    expect(result.current.data?.pages[1]?.runs?.[0]?.id).toBe("run-2");
  });

  it("still hits the network for genuinely new fetchNextPage cursors after a refetch", async () => {
    const queryClient = createQueryClient();
    canvasesListRuns
      .mockResolvedValueOnce({
        data: {
          runs: [{ id: "run-1", state: "STATE_FINISHED" }],
          totalCount: 3,
          hasNextPage: true,
          lastTimestamp: "2026-05-02T00:00:00Z",
        },
      })
      .mockResolvedValueOnce({
        data: {
          runs: [{ id: "run-1", state: "STATE_FINISHED" }],
          totalCount: 3,
          hasNextPage: true,
          lastTimestamp: "2026-05-02T00:00:00Z",
        },
      })
      .mockResolvedValueOnce({
        data: {
          runs: [{ id: "run-2", state: "STATE_FINISHED" }],
          totalCount: 3,
          hasNextPage: false,
          lastTimestamp: "2026-05-01T00:00:00Z",
        },
      });

    const { result } = renderHook(() => useInfiniteCanvasRuns("canvas-1"), {
      wrapper: createWrapper(queryClient),
    });

    await waitFor(() => {
      expect(result.current.data?.pages).toHaveLength(1);
    });

    await result.current.refetch();
    expect(canvasesListRuns).toHaveBeenCalledTimes(2);

    await result.current.fetchNextPage();
    // fetchNextPage introduces a cursor not yet in the cache, so the queryFn
    // must hit the network for it (never silently reuse a stale page).
    expect(canvasesListRuns).toHaveBeenCalledTimes(3);
    expect(canvasesListRuns).toHaveBeenNthCalledWith(
      3,
      expect.objectContaining({
        path: { canvasId: "canvas-1" },
        query: { limit: 25, before: "2026-05-02T00:00:00Z" },
      }),
    );
  });
});
