import type { CanvasesCanvasRun } from "@/api-client";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook, waitFor } from "@testing-library/react";
import { createElement, type ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "bun:test";

const { canvasesDescribeRun, canvasesListRuns, factoriesListFactoryAutomations, factoriesListWorkOrderEvents } =
  vi.hoisted(() => ({
    canvasesDescribeRun: vi.fn(),
    canvasesListRuns: vi.fn(),
    factoriesListFactoryAutomations: vi.fn(),
    factoriesListWorkOrderEvents: vi.fn(),
  }));

vi.mock("@/api-client", () => ({
  canvasesDescribeRun,
  canvasesListRuns,
  factoriesListFactoryAutomations,
  factoriesListWorkOrderEvents,
}));

import { mergeConfidenceRunsKey, useFactoryMergeConfidenceRuns } from "./useMergeConfidenceRuns";

function scoreRun(overrides: {
  id: string;
  number: number;
  repository: string;
  createdAt?: string;
  state?: CanvasesCanvasRun["state"];
}): CanvasesCanvasRun {
  return {
    id: overrides.id,
    state: overrides.state ?? "STATE_FINISHED",
    result: "RESULT_PASSED",
    createdAt: overrides.createdAt ?? "2026-08-26T11:00:00Z",
    rootEvent: {
      data: {
        type: "github.pullRequest",
        data: {
          number: overrides.number,
          pull_request: {
            number: overrides.number,
            base: { repo: { full_name: overrides.repository } },
          },
          repository: { full_name: overrides.repository },
        },
      },
    },
  };
}

function createWrapper(queryClient: QueryClient) {
  return function Wrapper({ children }: { children: ReactNode }) {
    return createElement(QueryClientProvider, { client: queryClient }, children);
  };
}

const pullRequests = [{ number: "12", repository: "acme/app", url: "https://github.com/acme/app/pull/12" }];

describe("useFactoryMergeConfidenceRuns", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    factoriesListFactoryAutomations.mockResolvedValue({
      data: { automations: [{ id: "app-merge", name: "Merge confidence", columnKey: "verify" }] },
    });
    factoriesListWorkOrderEvents.mockResolvedValue({ data: { events: [], totalCount: 0, hasNextPage: false } });
    canvasesDescribeRun.mockResolvedValue({ data: {} });
  });

  it("does not list canvas runs when the task has no pull request", async () => {
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    const { result } = renderHook(() => useFactoryMergeConfidenceRuns("org-1", "factory-1", [], "order-1"), {
      wrapper: createWrapper(queryClient),
    });

    await waitFor(() => expect(factoriesListFactoryAutomations).toHaveBeenCalled());
    expect(result.current.runs).toEqual([]);
    expect(result.current.canvasIds).toEqual([]);
    expect(canvasesListRuns).not.toHaveBeenCalled();
    expect(factoriesListWorkOrderEvents).not.toHaveBeenCalled();
    expect(canvasesDescribeRun).not.toHaveBeenCalled();
  });

  it("describes a recorded older score and does not page through unrelated runs", async () => {
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    const newest = Array.from({ length: 25 }, (_, index) =>
      scoreRun({
        id: `run-new-${index}`,
        number: 99,
        repository: "acme/other",
        createdAt: `2026-08-26T12:${String(index).padStart(2, "0")}:00Z`,
      }),
    );
    const older = scoreRun({
      id: "run-older",
      number: 12,
      repository: "acme/app",
      createdAt: "2026-08-01T00:00:00Z",
    });
    const inProgress = scoreRun({
      id: "run-live",
      number: 12,
      repository: "acme/app",
      createdAt: "2026-08-26T12:30:00Z",
      state: "STATE_STARTED",
    });
    canvasesListRuns.mockResolvedValue({
      data: {
        runs: [inProgress, ...newest.slice(0, 24)],
        totalCount: 200,
        hasNextPage: true,
        lastTimestamp: "2026-08-26T12:00:00Z",
      },
    });
    factoriesListWorkOrderEvents
      .mockResolvedValueOnce({
        data: {
          events: [{ type: "order.comment.added", event: {} }],
          totalCount: 2,
          hasNextPage: true,
          lastTimestamp: "2026-08-20T00:00:00Z",
        },
      })
      .mockResolvedValueOnce({
        data: {
          events: [
            {
              type: "order.check.reported",
              event: { run: { id: "run-older" }, automation: { appId: "app-merge" } },
            },
          ],
          totalCount: 2,
          hasNextPage: false,
          lastTimestamp: "2026-08-01T00:00:00Z",
        },
      });
    canvasesDescribeRun.mockImplementation((request: { path?: { runId?: string } }) =>
      Promise.resolve({
        data: {
          run:
            request.path?.runId === "run-older"
              ? older
              : scoreRun({ id: "run-check", number: 12, repository: "acme/app" }),
        },
      }),
    );

    const { result } = renderHook(
      () =>
        useFactoryMergeConfidenceRuns("org-1", "factory-1", pullRequests, "order-1", [
          { runId: "run-check", automation: { appId: "app-merge" } },
        ]),
      { wrapper: createWrapper(queryClient) },
    );

    await waitFor(() =>
      expect(result.current.runs.map((entry) => entry.run.id)).toEqual(["run-older", "run-check", "run-live"]),
    );
    expect(canvasesListRuns).toHaveBeenCalledTimes(1);
    expect(canvasesListRuns).toHaveBeenCalledWith(
      expect.objectContaining({
        path: { canvasId: "app-merge" },
        query: { limit: 25 },
      }),
    );
    expect(factoriesListWorkOrderEvents).toHaveBeenNthCalledWith(
      2,
      expect.objectContaining({
        path: { factoryId: "factory-1", orderId: "order-1" },
        query: { limit: 50, before: "2026-08-20T00:00:00Z" },
      }),
    );
    expect(canvasesDescribeRun).toHaveBeenCalledWith(
      expect.objectContaining({ path: { canvasId: "app-merge", runId: "run-older" } }),
    );
  });

  it("drops a cached run the new snapshot omits and keeps a run that arrives during the fetch", async () => {
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    const kept = scoreRun({ id: "run-kept", number: 12, repository: "acme/app" });
    const dropped = scoreRun({ id: "run-dropped", number: 4, repository: "acme/other" });
    canvasesListRuns.mockResolvedValueOnce({
      data: { runs: [kept, dropped], totalCount: 2, hasNextPage: false },
    });

    const { result } = renderHook(() => useFactoryMergeConfidenceRuns("org-1", "factory-1", pullRequests), {
      wrapper: createWrapper(queryClient),
    });

    await waitFor(() => expect(result.current.runs.map((entry) => entry.run.id)).toEqual(["run-kept"]));

    let releaseFetch: (value: unknown) => void = () => {};
    const pendingFetch = new Promise((resolve) => {
      releaseFetch = resolve;
    });
    canvasesListRuns.mockReturnValueOnce(pendingFetch);

    const taskKey = result.current.taskKey;
    act(() => {
      void queryClient.invalidateQueries({ queryKey: mergeConfidenceRunsKey("org-1", "app-merge", taskKey) });
    });
    await waitFor(() => expect(canvasesListRuns).toHaveBeenCalledTimes(2));

    const live = scoreRun({
      id: "run-live",
      number: 12,
      repository: "acme/app",
      createdAt: "2026-08-26T12:00:00Z",
    });
    act(() => {
      queryClient.setQueriesData<CanvasesCanvasRun[]>(
        { queryKey: mergeConfidenceRunsKey("org-1", "app-merge", taskKey) },
        (current) => [...(current ?? []), live],
      );
    });

    await act(async () => {
      releaseFetch({ data: { runs: [kept], totalCount: 1, hasNextPage: false } });
      await pendingFetch;
    });

    await waitFor(() => expect(result.current.runs.map((entry) => entry.run.id)).toEqual(["run-kept", "run-live"]));
    const cached = queryClient.getQueriesData<CanvasesCanvasRun[]>({
      queryKey: mergeConfidenceRunsKey("org-1", "app-merge", taskKey),
    });
    expect(cached.some(([, runs]) => runs?.map((run) => run.id).join() === "run-kept,run-live")).toBe(true);
  });
});
