import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook, waitFor } from "@testing-library/react";
import { createElement, type ReactNode } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "bun:test";

import { clearBacklogAnalysisPending, pendingBacklogAnalysisIds } from "@/pages/factories/lib/backlogAnalysis";

const { factoriesCreateWorkOrder, factoriesForkWorkOrder } = vi.hoisted(() => ({
  factoriesCreateWorkOrder: vi.fn(),
  factoriesForkWorkOrder: vi.fn(),
}));

vi.mock("@/api-client", () => ({
  factoriesCreateWorkOrder,
  factoriesForkWorkOrder,
}));

import { mergeFactoryBoardWorkOrders, useCreateWorkOrder, useForkWorkOrder } from "./useFactoryData";

function createWrapper(queryClient: QueryClient) {
  return function Wrapper({ children }: { children: ReactNode }) {
    return createElement(QueryClientProvider, { client: queryClient }, children);
  };
}

describe("mergeFactoryBoardWorkOrders", () => {
  it("keeps one row when backlog, open, and done share an id", () => {
    const shared = { id: "wo-1", title: "first" };
    expect(
      mergeFactoryBoardWorkOrders([shared], [{ id: "wo-1", title: "open copy" }], [{ id: "wo-1", title: "done copy" }]),
    ).toEqual([shared]);
  });
});

describe("useCreateWorkOrder", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  afterEach(() => {
    clearBacklogAnalysisPending("wo-created-1");
  });

  it("marks the new order pending analysis and invalidates backlog-analysis-runs", async () => {
    factoriesCreateWorkOrder.mockResolvedValue({ data: { order: { id: "wo-created-1" } } });
    const queryClient = new QueryClient();
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries");

    const { result } = renderHook(() => useCreateWorkOrder("org-1", "factory-1"), {
      wrapper: createWrapper(queryClient),
    });

    await act(async () => {
      await result.current.mutateAsync({ title: "New task", description: "" });
    });

    await waitFor(() => expect(pendingBacklogAnalysisIds().has("wo-created-1")).toBe(true));
    expect(invalidateSpy).toHaveBeenCalledWith({
      queryKey: ["backlog-analysis-runs", "org-1"],
    });
  });
});

describe("useForkWorkOrder", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  afterEach(() => {
    clearBacklogAnalysisPending("wo-forked-1");
  });

  it("marks a request fork pending only when planning is on", async () => {
    factoriesForkWorkOrder.mockResolvedValue({ data: { order: { id: "wo-forked-1" } } });
    const queryClient = new QueryClient();
    queryClient.setQueryData(["factories", "org-1", "factory-1"], {
      id: "factory-1",
      planning: { enabled: true },
    });

    const { result } = renderHook(() => useForkWorkOrder("org-1", "factory-1"), {
      wrapper: createWrapper(queryClient),
    });

    await act(async () => {
      await result.current.mutateAsync({ orderId: "wo-source", mode: "MODE_INTAKE" });
    });

    await waitFor(() => expect(pendingBacklogAnalysisIds().has("wo-forked-1")).toBe(true));
  });

  it("does not mark a request fork pending when planning is off", async () => {
    factoriesForkWorkOrder.mockResolvedValue({ data: { order: { id: "wo-forked-1" } } });
    const queryClient = new QueryClient();
    queryClient.setQueryData(["factories", "org-1"], [{ id: "factory-1", planning: { enabled: false } }]);

    const { result } = renderHook(() => useForkWorkOrder("org-1", "factory-1"), {
      wrapper: createWrapper(queryClient),
    });

    await act(async () => {
      await result.current.mutateAsync({ orderId: "wo-source", mode: "MODE_INTAKE" });
    });

    expect(pendingBacklogAnalysisIds().has("wo-forked-1")).toBe(false);
  });

  it("does not mark a plan fork pending", async () => {
    factoriesForkWorkOrder.mockResolvedValue({ data: { order: { id: "wo-forked-1" } } });
    const queryClient = new QueryClient();
    queryClient.setQueryData(["factories", "org-1", "factory-1"], {
      id: "factory-1",
      planning: { enabled: true },
    });

    const { result } = renderHook(() => useForkWorkOrder("org-1", "factory-1"), {
      wrapper: createWrapper(queryClient),
    });

    await act(async () => {
      await result.current.mutateAsync({ orderId: "wo-source", mode: "MODE_PLAN" });
    });

    expect(pendingBacklogAnalysisIds().has("wo-forked-1")).toBe(false);
  });
});
