import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react";
import { createElement, type ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

const { findPlanningSessionByWorkOrder } = vi.hoisted(() => ({
  findPlanningSessionByWorkOrder: vi.fn(),
}));

vi.mock("./planningSessionClient", () => ({
  findPlanningSessionByWorkOrder,
}));

import { useWorkOrderPlanningSurvey } from "./useWorkOrderPlanningSurvey";

function createWrapper(queryClient: QueryClient) {
  return function Wrapper({ children }: { children: ReactNode }) {
    return createElement(QueryClientProvider, { client: queryClient }, children);
  };
}

describe("useWorkOrderPlanningSurvey", () => {
  beforeEach(() => {
    findPlanningSessionByWorkOrder.mockReset();
  });

  it("is true when the session has a pending survey", async () => {
    findPlanningSessionByWorkOrder.mockResolvedValue({
      survey: { id: "survey-1", questions: [{ prompt: "Which API?", options: ["REST", "GraphQL"] }] },
    });
    const queryClient = new QueryClient();
    const { result } = renderHook(() => useWorkOrderPlanningSurvey("org-1", "factory-1", "wo-1", true), {
      wrapper: createWrapper(queryClient),
    });

    await waitFor(() => expect(result.current).toBe(true));
  });

  it("is false when analysis is not running", async () => {
    const queryClient = new QueryClient();
    const { result } = renderHook(() => useWorkOrderPlanningSurvey("org-1", "factory-1", "wo-1", false), {
      wrapper: createWrapper(queryClient),
    });

    expect(result.current).toBe(false);
    expect(findPlanningSessionByWorkOrder).not.toHaveBeenCalled();
  });
});
