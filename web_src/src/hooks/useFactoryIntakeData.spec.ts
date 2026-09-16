import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react";
import { createElement, type ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "bun:test";

const { factoriesCreateFactoryIntake, factoriesRefreshBacklog } = vi.hoisted(() => ({
  factoriesCreateFactoryIntake: vi.fn(),
  factoriesRefreshBacklog: vi.fn(),
}));

vi.mock("@/api-client", () => ({
  factoriesCreateFactoryIntake,
  factoriesRefreshBacklog,
}));

import { factoryQueryKeys } from "./useFactoryData";
import { useCreateFactoryIntake, useRefreshBacklog } from "./useFactoryIntakeData";

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
