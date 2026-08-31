import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook, waitFor } from "@testing-library/react";
import { createElement, type ReactNode } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { clearBacklogAnalysisPending, pendingBacklogAnalysisIds } from "@/pages/factories/lib/backlogAnalysis";

const { factoriesCreateWorkOrder } = vi.hoisted(() => ({
  factoriesCreateWorkOrder: vi.fn(),
}));

vi.mock("@/api-client", async (importOriginal) => {
  const actual = await importOriginal<Record<string, unknown>>();
  return {
    ...actual,
    factoriesCreateWorkOrder,
  };
});

import { useCreateWorkOrder } from "./useFactoryData";

function createWrapper(queryClient: QueryClient) {
  return function Wrapper({ children }: { children: ReactNode }) {
    return createElement(QueryClientProvider, { client: queryClient }, children);
  };
}

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
