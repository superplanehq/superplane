import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook } from "@testing-library/react";
import { createElement, type ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { CanvasesCanvasRun } from "@/api-client";
import { canvasKeys } from "@/hooks/useCanvasData";
import type { SidebarEvent } from "@/ui/componentSidebar/types";

const { canvasesListRuns } = vi.hoisted(() => ({
  canvasesListRuns: vi.fn(),
}));

vi.mock("@/api-client", async (importOriginal) => {
  const actual = await importOriginal<Record<string, unknown>>();
  return {
    ...actual,
    canvasesListRuns,
  };
});

import { useSidebarEventRunLookup } from "./useSidebarEventRunLookup";

const triggerEvent = {
  id: "root-1",
  title: "Trigger",
  state: "success",
  isOpen: false,
  kind: "trigger",
} satisfies SidebarEvent;

function createWrapper(queryClient: QueryClient) {
  return function Wrapper({ children }: { children: ReactNode }) {
    return createElement(QueryClientProvider, { client: queryClient }, children);
  };
}

describe("useSidebarEventRunLookup", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    canvasesListRuns.mockResolvedValue({ data: { runs: [] } });
  });

  it("does not cache failed paginated lookups so later cached runs can resolve", async () => {
    const queryClient = new QueryClient();
    const runsWithMatch: CanvasesCanvasRun[] = [
      {
        id: "run-1",
        rootEvent: { id: "root-1" },
      },
    ];

    const { result, rerender } = renderHook(
      (props: { runs: CanvasesCanvasRun[] }) =>
        useSidebarEventRunLookup({
          enabled: true,
          canvasId: "canvas-1",
          organizationId: "org-1",
          queryClient,
          runs: props.runs,
        }),
      {
        wrapper: createWrapper(queryClient),
        initialProps: { runs: [] as CanvasesCanvasRun[] },
      },
    );

    await act(async () => {
      expect(await result.current.fetchRunIdForSidebarEvent(triggerEvent)).toBeNull();
    });

    rerender({ runs: runsWithMatch });

    await act(async () => {
      expect(await result.current.fetchRunIdForSidebarEvent(triggerEvent)).toBe("run-1");
    });

    expect(canvasesListRuns).toHaveBeenCalledTimes(1);
  });

  it("paginates beyond the first three pages until a matching run is found", async () => {
    const queryClient = new QueryClient();
    const fillerRuns = Array.from({ length: 25 }, (_, index) => ({
      id: `run-filler-${index}`,
      rootEvent: { id: `root-filler-${index}` },
    }));

    canvasesListRuns
      .mockResolvedValueOnce({
        data: {
          runs: fillerRuns,
          totalCount: 100,
          hasNextPage: true,
          lastTimestamp: "2026-02-06T14:00:00.000Z",
        },
      })
      .mockResolvedValueOnce({
        data: {
          runs: fillerRuns,
          totalCount: 100,
          hasNextPage: true,
          lastTimestamp: "2026-02-06T13:00:00.000Z",
        },
      })
      .mockResolvedValueOnce({
        data: {
          runs: fillerRuns,
          totalCount: 100,
          hasNextPage: true,
          lastTimestamp: "2026-02-06T12:00:00.000Z",
        },
      })
      .mockResolvedValueOnce({
        data: {
          runs: [{ id: "run-1", rootEvent: { id: "root-1" } }],
          totalCount: 100,
          hasNextPage: false,
          lastTimestamp: "2026-02-06T11:00:00.000Z",
        },
      });

    const { result } = renderHook(
      () =>
        useSidebarEventRunLookup({
          enabled: true,
          canvasId: "canvas-1",
          organizationId: "org-1",
          queryClient,
          runs: [],
        }),
      {
        wrapper: createWrapper(queryClient),
      },
    );

    await act(async () => {
      expect(await result.current.fetchRunIdForSidebarEvent(triggerEvent)).toBe("run-1");
    });

    expect(canvasesListRuns).toHaveBeenCalledTimes(4);
  });

  it("can cap paginated lookup to the newest page", async () => {
    const queryClient = new QueryClient();
    const fillerRuns = Array.from({ length: 25 }, (_, index) => ({
      id: `run-filler-${index}`,
      rootEvent: { id: `root-filler-${index}` },
    }));

    canvasesListRuns.mockResolvedValueOnce({
      data: {
        runs: fillerRuns,
        totalCount: 50,
        hasNextPage: true,
        lastTimestamp: "2026-02-06T14:00:00.000Z",
      },
    });

    const { result } = renderHook(
      () =>
        useSidebarEventRunLookup({
          enabled: true,
          canvasId: "canvas-1",
          organizationId: "org-1",
          queryClient,
          runs: [],
        }),
      {
        wrapper: createWrapper(queryClient),
      },
    );

    await act(async () => {
      expect(await result.current.fetchRunIdForSidebarEvent(triggerEvent, { maxPages: 1 })).toBeNull();
    });

    expect(canvasesListRuns).toHaveBeenCalledTimes(1);
  });

  it("checks the current infinite runs cache before capped pagination", async () => {
    const queryClient = new QueryClient();

    const { result } = renderHook(
      () =>
        useSidebarEventRunLookup({
          enabled: true,
          canvasId: "canvas-1",
          organizationId: "org-1",
          queryClient,
          runs: [],
        }),
      {
        wrapper: createWrapper(queryClient),
      },
    );

    queryClient.setQueryData(canvasKeys.infiniteRuns("canvas-1", { states: ["STATE_STARTED"] }), {
      pages: [{ runs: [{ id: "run-1", rootEvent: { id: "root-1" } }] }],
      pageParams: [],
    });

    await act(async () => {
      expect(await result.current.fetchRunIdForSidebarEvent(triggerEvent, { maxPages: 1 })).toBe("run-1");
    });

    expect(canvasesListRuns).not.toHaveBeenCalled();
  });

  it("starts paginated lookup from the newest runs", async () => {
    const queryClient = new QueryClient();

    canvasesListRuns.mockResolvedValueOnce({
      data: {
        runs: [{ id: "run-1", rootEvent: { id: "root-1" } }],
        totalCount: 1,
        hasNextPage: false,
        lastTimestamp: "2026-02-06T15:00:00.000Z",
      },
    });

    const event = {
      ...triggerEvent,
      receivedAt: new Date("2026-02-06T14:00:00.000Z"),
    } satisfies SidebarEvent;

    const { result } = renderHook(
      () =>
        useSidebarEventRunLookup({
          enabled: true,
          canvasId: "canvas-1",
          organizationId: "org-1",
          queryClient,
          runs: [],
        }),
      {
        wrapper: createWrapper(queryClient),
      },
    );

    await act(async () => {
      expect(await result.current.fetchRunIdForSidebarEvent(event)).toBe("run-1");
    });

    expect(canvasesListRuns).toHaveBeenCalledWith(
      expect.objectContaining({
        query: { limit: 25 },
      }),
    );
  });

  it("does not reuse fetched run ids after cached lookup sources are invalidated", async () => {
    const queryClient = new QueryClient();

    canvasesListRuns
      .mockResolvedValueOnce({
        data: {
          runs: [{ id: "run-1", rootEvent: { id: "root-1" } }],
          totalCount: 1,
          hasNextPage: false,
          lastTimestamp: "2026-02-06T14:00:00.000Z",
        },
      })
      .mockResolvedValueOnce({
        data: {
          runs: [],
          totalCount: 0,
          hasNextPage: false,
        },
      });

    const { result, rerender } = renderHook(
      () =>
        useSidebarEventRunLookup({
          enabled: true,
          canvasId: "canvas-1",
          organizationId: "org-1",
          queryClient,
          runs: [],
        }),
      {
        wrapper: createWrapper(queryClient),
      },
    );

    await act(async () => {
      expect(await result.current.fetchRunIdForSidebarEvent(triggerEvent)).toBe("run-1");
    });

    queryClient.clear();
    rerender();

    await act(async () => {
      expect(await result.current.fetchRunIdForSidebarEvent(triggerEvent)).toBeNull();
    });

    expect(canvasesListRuns).toHaveBeenCalledTimes(2);
  });
});
