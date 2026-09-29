import type { CanvasesCanvasRun, FactoriesWorkOrder } from "@/api-client";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook, waitFor } from "@testing-library/react";
import { createElement, type ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "bun:test";

import {
  clearBacklogAnalysisPending,
  markBacklogAnalysisPending,
  pendingBacklogAnalysisIds,
} from "@/pages/factories/lib/backlogAnalysis";

const {
  canvasesListRuns,
  factoriesListFactoryAutomations,
  factoriesListFactoryIntakes,
  factoriesListWorkOrders,
  useCanvasWebsocket,
} = vi.hoisted(() => ({
  canvasesListRuns: vi.fn(),
  factoriesListFactoryAutomations: vi.fn(),
  factoriesListFactoryIntakes: vi.fn(),
  factoriesListWorkOrders: vi.fn(),
  useCanvasWebsocket: vi.fn(),
}));

vi.mock("@/api-client", () => ({
  canvasesListRuns,
  factoriesListFactoryAutomations,
  factoriesListFactoryIntakes,
  factoriesListWorkOrders,
}));

vi.mock("@/hooks/useCanvasWebsocket", () => ({
  useCanvasWebsocket,
}));

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

describe("useBacklogAnalysisRuns", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    canvasesListRuns.mockResolvedValue({ data: { runs: [] } });
  });

  it("does not poll while a pending id is set", async () => {
    const queryClient = new QueryClient();
    markBacklogAnalysisPending("wo-1");

    const { result } = renderHook(() => useBacklogAnalysisRuns("org-1", "canvas-1"), {
      wrapper: createWrapper(queryClient),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    const query = queryClient.getQueryCache().find({
      queryKey: ["backlog-analysis-runs", "org-1", "canvas-1"],
    });
    expect((query?.options as { refetchInterval?: unknown } | undefined)?.refetchInterval).toBeUndefined();
    expect(canvasesListRuns).toHaveBeenCalledTimes(1);

    act(() => {
      clearBacklogAnalysisPending("wo-1");
    });
  });

  it("does not poll when a run is active", async () => {
    const queryClient = new QueryClient();
    canvasesListRuns.mockResolvedValue({
      data: { runs: [analysisRun({ id: "run-1", workOrderId: "wo-1", state: "STATE_STARTED" })] },
    });

    const { result } = renderHook(() => useBacklogAnalysisRuns("org-1", "canvas-1"), {
      wrapper: createWrapper(queryClient),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    const query = queryClient.getQueryCache().find({
      queryKey: ["backlog-analysis-runs", "org-1", "canvas-1"],
    });
    expect((query?.options as { refetchInterval?: unknown } | undefined)?.refetchInterval).toBeUndefined();
    expect(canvasesListRuns).toHaveBeenCalledTimes(1);
  });
});

describe("useFactoryBacklogAnalysis", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    factoriesListFactoryAutomations.mockResolvedValue({
      data: { automations: [{ id: "app-analyzer", name: "Backlog" }] },
    });
    factoriesListFactoryIntakes.mockResolvedValue({ data: { intakes: [] } });
    factoriesListWorkOrders.mockResolvedValue({ data: { orders: [] } });
    canvasesListRuns.mockResolvedValue({ data: { runs: [] } });
  });

  it("merges a pending id into analyzingOrderIds and drops it once the real run appears", async () => {
    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    });
    markBacklogAnalysisPending("wo-1");

    const { result } = renderHook(() => useFactoryBacklogAnalysis("org-1", "factory-1"), {
      wrapper: createWrapper(queryClient),
    });

    await waitFor(() => {
      expect(result.current.analyzingOrderIds.has("wo-1")).toBe(true);
      expect(
        queryClient.getQueryCache().find({ queryKey: ["backlog-analysis-runs", "org-1", "app-analyzer"] }),
      ).toBeDefined();
    });

    canvasesListRuns.mockResolvedValue({
      data: { runs: [analysisRun({ id: "run-1", workOrderId: "wo-1", state: "STATE_STARTED" })] },
    });
    await act(async () => {
      await queryClient.invalidateQueries({ queryKey: ["backlog-analysis-runs", "org-1"] });
    });

    await waitFor(() => expect(result.current.runsByWorkOrder.has("wo-1")).toBe(true));
    await waitFor(() => expect(pendingBacklogAnalysisIds().has("wo-1")).toBe(false));
    expect(result.current.analyzingOrderIds.has("wo-1")).toBe(true);

    clearPending("wo-1");
  });

  it("subscribes to the analyzer canvas and applies run events", async () => {
    const queryClient = new QueryClient();
    factoriesListWorkOrders.mockResolvedValue({ data: { orders: [draftOrder("wo-api-1")] } });

    const { result } = renderHook(() => useFactoryBacklogAnalysis("org-1", "factory-1"), {
      wrapper: createWrapper(queryClient),
    });

    await waitFor(() => {
      expect(useCanvasWebsocket).toHaveBeenCalledWith(
        expect.objectContaining({
          canvasId: "app-analyzer",
          organizationId: "org-1",
          processRuntimeEvents: false,
          enabled: true,
        }),
      );
    });

    const websocketOptions = useCanvasWebsocket.mock.calls.at(-1)?.[0] as {
      onRunEvent: (run: CanvasesCanvasRun) => void;
    };
    act(() => {
      websocketOptions.onRunEvent(analysisRun({ id: "run-live", workOrderId: "wo-api-1", state: "STATE_STARTED" }));
    });

    await waitFor(() => expect(result.current.runsByWorkOrder.get("wo-api-1")?.[0]?.run.id).toBe("run-live"));
    expect(canvasesListRuns).toHaveBeenCalledTimes(1);
  });
});
