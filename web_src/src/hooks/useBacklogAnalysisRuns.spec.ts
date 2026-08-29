import type { CanvasesCanvasRun, FactoriesWorkOrder } from "@/api-client";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook, waitFor } from "@testing-library/react";
import { createElement, type ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import {
  clearBacklogAnalysisPending,
  markBacklogAnalysisPending,
  pendingBacklogAnalysisIds,
} from "@/pages/factories/lib/backlogAnalysis";

const { canvasesListRuns, factoriesListFactoryApps, factoriesListFactoryIntakes, factoriesListWorkOrders } = vi.hoisted(
  () => ({
    canvasesListRuns: vi.fn(),
    factoriesListFactoryApps: vi.fn(),
    factoriesListFactoryIntakes: vi.fn(),
    factoriesListWorkOrders: vi.fn(),
  }),
);

vi.mock("@/api-client", async (importOriginal) => {
  const actual = await importOriginal<Record<string, unknown>>();
  return {
    ...actual,
    canvasesListRuns,
    factoriesListFactoryApps,
    factoriesListFactoryIntakes,
    factoriesListWorkOrders,
  };
});

import { useBacklogAnalysisRuns, useFactoryBacklogAnalysis } from "./useBacklogAnalysisRuns";

function analysisRun(overrides: {
  id: string;
  workOrderId?: string;
  state?: CanvasesCanvasRun["state"];
}): CanvasesCanvasRun {
  return {
    id: overrides.id,
    state: overrides.state ?? "STATE_STARTED",
    createdAt: "2026-08-28T10:00:00Z",
    rootEvent: overrides.workOrderId
      ? { data: { type: "factory.workOrder", data: { workOrder: { id: overrides.workOrderId } } } }
      : undefined,
  };
}

function draftOrder(id: string, createdAt = new Date().toISOString()): FactoriesWorkOrder {
  return { id, state: "STATE_DRAFT", createdAt };
}

function createWrapper(queryClient: QueryClient) {
  return function Wrapper({ children }: { children: ReactNode }) {
    return createElement(QueryClientProvider, { client: queryClient }, children);
  };
}

function clearPending(...ids: string[]) {
  for (const id of ids) {
    clearBacklogAnalysisPending(id);
  }
}

/** `refetchInterval` is a client-only query option, not part of the cached `QueryOptions` type. */
function readRefetchInterval(query: { options: unknown }): (query: unknown) => number | false {
  return (query.options as { refetchInterval: (query: unknown) => number | false }).refetchInterval;
}

describe("useBacklogAnalysisRuns", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    canvasesListRuns.mockResolvedValue({ data: { runs: [] } });
  });

  it("polls while a pending id is set even with an empty run cache", async () => {
    const queryClient = new QueryClient();
    markBacklogAnalysisPending("wo-1");

    const { result } = renderHook(() => useBacklogAnalysisRuns("org-1", "canvas-1"), {
      wrapper: createWrapper(queryClient),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    const getQuery = () =>
      queryClient.getQueryCache().find({ queryKey: ["backlog-analysis-runs", "org-1", "canvas-1"] })!;
    const refetchIntervalOf = (query: ReturnType<typeof getQuery>) => readRefetchInterval(query)(query);

    expect(refetchIntervalOf(getQuery())).toBe(4000);

    act(() => {
      clearBacklogAnalysisPending("wo-1");
    });

    await waitFor(() => expect(refetchIntervalOf(getQuery())).toBe(false));
  });

  it("stops polling once no run is active and no id is pending", async () => {
    const queryClient = new QueryClient();

    const { result } = renderHook(() => useBacklogAnalysisRuns("org-1", "canvas-1"), {
      wrapper: createWrapper(queryClient),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    const query = queryClient.getQueryCache().find({ queryKey: ["backlog-analysis-runs", "org-1", "canvas-1"] });
    const refetchInterval = readRefetchInterval(query!);
    expect(refetchInterval(query!)).toBe(false);
  });
});

describe("useFactoryBacklogAnalysis", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    factoriesListFactoryApps.mockResolvedValue({ data: { apps: [{ id: "app-analyzer", name: "Backlog" }] } });
    factoriesListFactoryIntakes.mockResolvedValue({ data: { intakes: [] } });
    factoriesListWorkOrders.mockResolvedValue({ data: { orders: [] } });
    canvasesListRuns.mockResolvedValue({ data: { runs: [] } });
  });

  it("merges a pending id into analyzingOrderIds and drops it once the real run appears", async () => {
    const queryClient = new QueryClient();
    markBacklogAnalysisPending("wo-1");

    const { result, rerender } = renderHook(() => useFactoryBacklogAnalysis("org-1", "factory-1"), {
      wrapper: createWrapper(queryClient),
    });

    await waitFor(() => expect(result.current.analyzingOrderIds.has("wo-1")).toBe(true));

    canvasesListRuns.mockResolvedValue({
      data: { runs: [analysisRun({ id: "run-1", workOrderId: "wo-1", state: "STATE_STARTED" })] },
    });
    await queryClient.invalidateQueries({ queryKey: ["backlog-analysis-runs", "org-1"] });
    rerender();

    await waitFor(() => expect(result.current.runsByWorkOrder.has("wo-1")).toBe(true));
    await waitFor(() => expect(pendingBacklogAnalysisIds().has("wo-1")).toBe(false));
    expect(result.current.analyzingOrderIds.has("wo-1")).toBe(true);

    clearPending("wo-1");
  });

  it("keeps polling for a recent draft with no run yet", async () => {
    const queryClient = new QueryClient();
    factoriesListWorkOrders.mockResolvedValue({ data: { orders: [draftOrder("wo-api-1")] } });

    renderHook(() => useFactoryBacklogAnalysis("org-1", "factory-1"), {
      wrapper: createWrapper(queryClient),
    });

    await waitFor(() => {
      const query = queryClient.getQueryCache().find({ queryKey: ["backlog-analysis-runs", "org-1", "app-analyzer"] });
      expect(query).toBeDefined();
      expect(readRefetchInterval(query!)(query!)).toBe(4000);
    });
  });
});
