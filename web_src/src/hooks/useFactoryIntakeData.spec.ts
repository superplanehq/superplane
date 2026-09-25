import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook, waitFor } from "@testing-library/react";
import { createElement, type ReactNode } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "bun:test";

const { factoriesCreateFactoryIntake, factoriesListFactoryIntakeRuns, factoriesRefreshBacklog, useCanvasWebsocket } =
  vi.hoisted(() => ({
    factoriesCreateFactoryIntake: vi.fn(),
    factoriesListFactoryIntakeRuns: vi.fn(),
    factoriesRefreshBacklog: vi.fn(),
    useCanvasWebsocket: vi.fn(),
  }));

vi.mock("@/api-client", () => ({
  factoriesCreateFactoryIntake,
  factoriesListFactoryIntakeRuns,
  factoriesRefreshBacklog,
}));

vi.mock("@/hooks/useCanvasWebsocket", () => ({
  useCanvasWebsocket,
}));

import { factoryQueryKeys } from "./useFactoryData";
import {
  factoryIntakeRunsKey,
  useCreateFactoryIntake,
  useFactoryIntakeRuns,
  useFactoryIntakeRunsWebsocket,
  useRefreshBacklog,
} from "./useFactoryIntakeData";

const ORGANIZATION_ID = "org-1";
const FACTORY_ID = "factory-1";

function createWrapper(queryClient: QueryClient) {
  return function Wrapper({ children }: { children: ReactNode }) {
    return createElement(QueryClientProvider, { client: queryClient }, children);
  };
}

beforeEach(() => {
  vi.clearAllMocks();
  factoriesCreateFactoryIntake.mockResolvedValue({ data: { intake: { id: "intake-1" } } });
  factoriesRefreshBacklog.mockResolvedValue({
    data: { archivedCount: 2, failedItemCount: 0, failedSourceCount: 0 },
  });
  factoriesListFactoryIntakeRuns.mockResolvedValue({ data: { runs: [] } });
});

afterEach(() => {
  vi.useRealTimers();
});

describe("useFactoryIntakeRuns", () => {
  it("loads once without a refresh interval", async () => {
    const queryClient = new QueryClient();
    const { result } = renderHook(() => useFactoryIntakeRuns(ORGANIZATION_ID, FACTORY_ID, "intake-1"), {
      wrapper: createWrapper(queryClient),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    const query = queryClient.getQueryCache().find({
      queryKey: factoryIntakeRunsKey(ORGANIZATION_ID, FACTORY_ID, "intake-1"),
    });
    expect((query?.options as { refetchInterval?: unknown } | undefined)?.refetchInterval).toBeUndefined();
    expect(factoriesListFactoryIntakeRuns).toHaveBeenCalledTimes(1);
  });

  it("batches canvas events into one WebSocket-driven refresh", async () => {
    vi.useFakeTimers();
    const queryClient = new QueryClient();
    const invalidate = vi.spyOn(queryClient, "invalidateQueries").mockResolvedValue();

    renderHook(
      () =>
        useFactoryIntakeRunsWebsocket({
          organizationId: ORGANIZATION_ID,
          factoryId: FACTORY_ID,
          intakeId: "intake-1",
          canvasId: "canvas-1",
        }),
      { wrapper: createWrapper(queryClient) },
    );

    const options = useCanvasWebsocket.mock.calls.at(-1)?.[0] as {
      onRunEvent: () => void;
      onExecutionEvent: () => void;
    };
    act(() => {
      options.onRunEvent();
      options.onExecutionEvent();
      vi.advanceTimersByTime(250);
    });

    expect(invalidate).toHaveBeenCalledTimes(1);
    expect(invalidate).toHaveBeenCalledWith({
      queryKey: factoryIntakeRunsKey(ORGANIZATION_ID, FACTORY_ID, "intake-1"),
    });
  });
});

describe("useCreateFactoryIntake", () => {
  // A new intake seeds the newest items of its source, so the tasks exist
  // before any card mutation runs. Without this the Backlog kept serving its
  // cached list and the seeded tasks looked lost.
  it("refetches the Backlog tasks a new intake seeds", async () => {
    const queryClient = new QueryClient();
    const invalidate = vi.spyOn(queryClient, "invalidateQueries");
    const { result } = renderHook(() => useCreateFactoryIntake(ORGANIZATION_ID, FACTORY_ID), {
      wrapper: createWrapper(queryClient),
    });

    await result.current.mutateAsync({ source: "SOURCE_PRODUCTIVE_TASKS" });

    await waitFor(() => {
      expect(invalidate).toHaveBeenCalledWith({
        queryKey: factoryQueryKeys.workOrders(ORGANIZATION_ID, FACTORY_ID),
      });
    });
  });
});

describe("useRefreshBacklog", () => {
  it("refetches backlog tasks after intake items close", async () => {
    const queryClient = new QueryClient();
    const invalidate = vi.spyOn(queryClient, "invalidateQueries");
    const { result } = renderHook(() => useRefreshBacklog(ORGANIZATION_ID, FACTORY_ID), {
      wrapper: createWrapper(queryClient),
    });

    await expect(result.current.mutateAsync()).resolves.toEqual({
      archivedCount: 2,
      failedItemCount: 0,
      failedSourceCount: 0,
    });

    await waitFor(() => {
      expect(invalidate).toHaveBeenCalledWith({
        queryKey: factoryQueryKeys.workOrders(ORGANIZATION_ID, FACTORY_ID),
      });
    });
  });
});
