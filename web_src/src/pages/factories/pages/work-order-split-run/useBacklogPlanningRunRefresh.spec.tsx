import type { CanvasesCanvasRun } from "@/api-client";
import { QueryClient, QueryClientProvider, useMutation, useQuery } from "@tanstack/react-query";
import { act, renderHook } from "@testing-library/react";
import { createElement, type ReactNode } from "react";
import { afterEach, describe, expect, it, vi } from "bun:test";

import { latestBacklogAnalysisRun, type BacklogAnalysisRun } from "../../lib/backlogAnalysis";
import { workOrderUpdateMutationKey } from "@/hooks/useFactoryData";
import { useBacklogPlanningRunRefresh } from "./useBacklogPlanningRunRefresh";

const queryKey = ["backlog-analysis-runs", "org-1", "backlog-canvas"];

function analysisRun(
  id: string,
  state: CanvasesCanvasRun["state"],
  result?: CanvasesCanvasRun["result"],
): BacklogAnalysisRun {
  return {
    canvasId: "backlog-canvas",
    workOrderId: "order-1",
    run: { id, canvasId: "backlog-canvas", state, result },
  };
}

function wrapper(queryClient: QueryClient) {
  return function Wrapper({ children }: { children: ReactNode }) {
    return createElement(QueryClientProvider, { client: queryClient }, children);
  };
}

async function advanceAndFlush(milliseconds: number) {
  await act(async () => {
    await vi.advanceTimersByTimeAsync(milliseconds);
    await vi.advanceTimersByTimeAsync(1);
  });
}

describe("useBacklogPlanningRunRefresh", () => {
  afterEach(() => vi.useRealTimers());

  it("finds a fast failure without a websocket event and detects a later retry", async () => {
    vi.useFakeTimers();
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    let serverRuns: BacklogAnalysisRun[] = [];
    const fetchRuns = vi.fn(async () => serverRuns);

    const { result, unmount } = renderHook(
      () => {
        const { data: runs = [] } = useQuery({ queryKey, queryFn: fetchRuns });
        const updateModel = useMutation({
          mutationKey: workOrderUpdateMutationKey("org-1", "factory-1"),
          mutationFn: async (_input: { orderId: string; refinementModel: string }) => undefined,
        });
        useBacklogPlanningRunRefresh({
          organizationId: "org-1",
          factoryId: "factory-1",
          orderId: "order-1",
          isDraft: true,
          hasSession: false,
          latestRun: latestBacklogAnalysisRun(runs)?.run,
        });
        return { runs, updateModel };
      },
      { wrapper: wrapper(queryClient) },
    );

    await advanceAndFlush(0);
    expect(fetchRuns).toHaveBeenCalledTimes(1);

    serverRuns = [analysisRun("run-first", "STATE_FINISHED", "RESULT_FAILED")];
    await advanceAndFlush(3_000);
    expect(fetchRuns).toHaveBeenCalledTimes(2);
    expect(queryClient.getQueryData<BacklogAnalysisRun[]>(queryKey)?.at(-1)?.run.id).toBe("run-first");
    expect(result.current.runs.at(-1)?.run.id).toBe("run-first");
    expect(result.current.runs.at(-1)?.run.result).toBe("RESULT_FAILED");

    const callsAfterFailure = fetchRuns.mock.calls.length;
    await advanceAndFlush(6_000);
    expect(fetchRuns).toHaveBeenCalledTimes(callsAfterFailure);

    await act(async () => {
      await result.current.updateModel.mutateAsync({ orderId: "order-1", refinementModel: "gpt-5.6-luna" });
    });
    await advanceAndFlush(0);
    await advanceAndFlush(3_000);
    expect(fetchRuns).toHaveBeenCalledTimes(callsAfterFailure + 1);

    serverRuns = [
      analysisRun("run-first", "STATE_FINISHED", "RESULT_FAILED"),
      analysisRun("run-retry", "STATE_STARTED"),
    ];
    await advanceAndFlush(3_000);
    expect(result.current.runs.at(-1)?.run.id).toBe("run-retry");

    serverRuns = [
      analysisRun("run-first", "STATE_FINISHED", "RESULT_FAILED"),
      analysisRun("run-retry", "STATE_FINISHED", "RESULT_FAILED"),
    ];
    await advanceAndFlush(3_000);
    expect(result.current.runs.at(-1)?.run.result).toBe("RESULT_FAILED");

    const callsAfterRetry = fetchRuns.mock.calls.length;
    await advanceAndFlush(6_000);
    expect(fetchRuns).toHaveBeenCalledTimes(callsAfterRetry);
    unmount();
  });
});
