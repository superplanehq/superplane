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

import {
  planningActivityPollInterval,
  useWorkOrderPlanningActivity,
  useWorkOrderPlanningSurvey,
} from "./useWorkOrderPlanningSurvey";

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

  it("does not keep a cached survey after polling stops", async () => {
    findPlanningSessionByWorkOrder.mockResolvedValue({
      survey: { id: "survey-1", questions: [{ prompt: "Which API?", options: ["REST", "GraphQL"] }] },
    });
    const queryClient = new QueryClient();
    const { result, rerender } = renderHook(
      ({ enabled }: { enabled: boolean }) => useWorkOrderPlanningSurvey("org-1", "factory-1", "wo-1", enabled),
      { wrapper: createWrapper(queryClient), initialProps: { enabled: true } },
    );

    await waitFor(() => expect(result.current).toBe(true));
    rerender({ enabled: false });
    expect(result.current).toBe(false);
  });
});

describe("useWorkOrderPlanningActivity", () => {
  beforeEach(() => {
    findPlanningSessionByWorkOrder.mockReset();
  });

  it("marks an open session as working", async () => {
    findPlanningSessionByWorkOrder.mockResolvedValue({
      executionId: "exec-1",
    });
    const queryClient = new QueryClient();
    const { result } = renderHook(() => useWorkOrderPlanningActivity("org-1", "factory-1", "wo-1", true), {
      wrapper: createWrapper(queryClient),
    });

    await waitFor(() => expect(result.current.isWorking).toBe(true));
    expect(result.current.isWaiting).toBe(false);
    expect(result.current.hasAgentQuestion).toBe(false);
  });

  it("marks a pending wait as waiting, not working", async () => {
    findPlanningSessionByWorkOrder.mockResolvedValue({
      executionId: "exec-1",
      waitState: "pending",
    });
    const queryClient = new QueryClient();
    const { result } = renderHook(() => useWorkOrderPlanningActivity("org-1", "factory-1", "wo-1", true), {
      wrapper: createWrapper(queryClient),
    });

    await waitFor(() => expect(result.current.isWaiting).toBe(true));
    expect(result.current.isWorking).toBe(false);
  });

  it("keeps a pending survey after the session ends", async () => {
    findPlanningSessionByWorkOrder.mockResolvedValue({
      state: "ended",
      survey: { id: "survey-1", questions: [{ prompt: "Which API?", options: ["REST", "GraphQL"] }] },
    });
    const queryClient = new QueryClient();
    const { result } = renderHook(() => useWorkOrderPlanningActivity("org-1", "factory-1", "wo-1", true, false), {
      wrapper: createWrapper(queryClient),
    });

    await waitFor(() => expect(result.current.hasAgentQuestion).toBe(true));
    expect(result.current.isWorking).toBe(false);
    expect(result.current.isWaiting).toBe(false);
  });

  it("stays false when an ended session has an empty survey", async () => {
    findPlanningSessionByWorkOrder.mockResolvedValue({
      state: "ended",
      survey: { questions: [] },
    });
    const queryClient = new QueryClient();
    const { result } = renderHook(() => useWorkOrderPlanningActivity("org-1", "factory-1", "wo-1", true, false), {
      wrapper: createWrapper(queryClient),
    });

    await waitFor(() => expect(findPlanningSessionByWorkOrder).toHaveBeenCalled());
    expect(result.current.hasAgentQuestion).toBe(false);
  });

  it("finds a pending survey when analysis ends after an empty lookup", async () => {
    findPlanningSessionByWorkOrder.mockResolvedValueOnce(null).mockResolvedValueOnce({
      state: "ended",
      survey: { id: "survey-1", questions: [{ prompt: "Which API?", options: ["REST", "GraphQL"] }] },
    });
    const queryClient = new QueryClient();
    const { result, rerender } = renderHook(
      ({ backlogAnalyzing }: { backlogAnalyzing: boolean }) =>
        useWorkOrderPlanningActivity("org-1", "factory-1", "wo-1", true, backlogAnalyzing),
      { wrapper: createWrapper(queryClient), initialProps: { backlogAnalyzing: true } },
    );

    await waitFor(() => expect(findPlanningSessionByWorkOrder).toHaveBeenCalledTimes(1));
    expect(result.current.hasAgentQuestion).toBe(false);

    rerender({ backlogAnalyzing: false });

    await waitFor(() => expect(result.current.hasAgentQuestion).toBe(true));
    expect(findPlanningSessionByWorkOrder).toHaveBeenCalledTimes(2);
  });
});

describe("planningActivityPollInterval", () => {
  it("polls while the agent works or a backlog run is active", () => {
    expect(planningActivityPollInterval(true, { state: "running" }, false)).toBe(1500);
    expect(planningActivityPollInterval(true, null, true)).toBe(1500);
    expect(planningActivityPollInterval(true, { state: "ended" }, true)).toBe(1500);
  });

  it("stops when the session waits or analysis is idle", () => {
    expect(planningActivityPollInterval(true, { state: "running", waitState: "pending" }, true)).toBe(false);
    expect(planningActivityPollInterval(true, { state: "ended" }, false)).toBe(false);
    expect(planningActivityPollInterval(true, null, false)).toBe(false);
    expect(planningActivityPollInterval(false, { state: "running" }, true)).toBe(false);
  });
});
