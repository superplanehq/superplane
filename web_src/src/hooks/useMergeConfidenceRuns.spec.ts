import type { CanvasesCanvasRun } from "@/api-client";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook, waitFor } from "@testing-library/react";
import { createElement, type ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "bun:test";

const { canvasesListRuns, factoriesListFactoryAutomations } = vi.hoisted(() => ({
  canvasesListRuns: vi.fn(),
  factoriesListFactoryAutomations: vi.fn(),
}));

vi.mock("@/api-client", () => ({
  canvasesListRuns,
  factoriesListFactoryAutomations,
}));

import { mergeConfidenceRunsKey, useFactoryMergeConfidenceRuns } from "./useMergeConfidenceRuns";

function scoreRun(overrides: {
  id: string;
  number: number;
  repository: string;
  createdAt?: string;
}): CanvasesCanvasRun {
  return {
    id: overrides.id,
    state: "STATE_FINISHED",
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
  });

  it("follows the run-list cursor to load a score past the newest page", async () => {
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
    canvasesListRuns
      .mockResolvedValueOnce({
        data: {
          runs: newest,
          totalCount: 26,
          hasNextPage: true,
          lastTimestamp: "2026-08-26T12:00:00Z",
        },
      })
      .mockResolvedValueOnce({
        data: {
          runs: [older],
          totalCount: 26,
          hasNextPage: false,
          lastTimestamp: "2026-08-01T00:00:00Z",
        },
      });

    const { result } = renderHook(() => useFactoryMergeConfidenceRuns("org-1", "factory-1", pullRequests), {
      wrapper: createWrapper(queryClient),
    });

    await waitFor(() => expect(result.current.runs.map((entry) => entry.run.id)).toEqual(["run-older"]));
    expect(canvasesListRuns).toHaveBeenNthCalledWith(
      2,
      expect.objectContaining({
        path: { canvasId: "app-merge" },
        query: { limit: 25, before: "2026-08-26T12:00:00Z" },
      }),
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

    act(() => {
      void queryClient.invalidateQueries({ queryKey: mergeConfidenceRunsKey("org-1", "app-merge") });
    });
    await waitFor(() => expect(canvasesListRuns).toHaveBeenCalledTimes(2));

    const live = scoreRun({
      id: "run-live",
      number: 12,
      repository: "acme/app",
      createdAt: "2026-08-26T12:00:00Z",
    });
    act(() => {
      queryClient.setQueryData<CanvasesCanvasRun[]>(mergeConfidenceRunsKey("org-1", "app-merge"), (current) => [
        ...(current ?? []),
        live,
      ]);
    });

    await act(async () => {
      releaseFetch({ data: { runs: [kept], totalCount: 1, hasNextPage: false } });
      await pendingFetch;
    });

    await waitFor(() => expect(result.current.runs.map((entry) => entry.run.id)).toEqual(["run-kept", "run-live"]));
    const cached = queryClient.getQueryData<CanvasesCanvasRun[]>(mergeConfidenceRunsKey("org-1", "app-merge"));
    expect(cached?.map((run) => run.id)).toEqual(["run-kept", "run-live"]);
  });
});
