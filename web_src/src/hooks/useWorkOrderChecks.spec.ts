import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react";
import { createElement, type ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { ANALYZING_WORK_ORDER_CHECKS_POLL_MS, useWorkOrderChecks } from "./useWorkOrderChecks";
import { factoryQueryKeys } from "./useFactoryData";

const { factoriesListWorkOrderChecks } = vi.hoisted(() => ({
  factoriesListWorkOrderChecks: vi.fn(),
}));

vi.mock("@/api-client", async (importOriginal) => {
  const actual = await importOriginal<Record<string, unknown>>();
  return {
    ...actual,
    factoriesListWorkOrderChecks,
  };
});

function createWrapper(queryClient: QueryClient) {
  return function Wrapper({ children }: { children: ReactNode }) {
    return createElement(QueryClientProvider, { client: queryClient }, children);
  };
}

describe("useWorkOrderChecks", () => {
  beforeEach(() => {
    factoriesListWorkOrderChecks.mockReset();
    factoriesListWorkOrderChecks.mockResolvedValue({ data: { checks: [] } });
  });

  it("does not poll after the first fetch", async () => {
    const queryClient = new QueryClient();
    const { result } = renderHook(() => useWorkOrderChecks("org-1", "factory-1", "wo-1"), {
      wrapper: createWrapper(queryClient),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    const query = queryClient
      .getQueryCache()
      .find({ queryKey: factoryQueryKeys.workOrderChecks("org-1", "factory-1", "wo-1") });
    expect(query?.options.refetchInterval).toBeUndefined();
  });

  it("polls while analysis is in flight", async () => {
    const queryClient = new QueryClient();
    const { result } = renderHook(
      () => useWorkOrderChecks("org-1", "factory-1", "wo-1", { refetchInterval: ANALYZING_WORK_ORDER_CHECKS_POLL_MS }),
      { wrapper: createWrapper(queryClient) },
    );

    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    const query = queryClient
      .getQueryCache()
      .find({ queryKey: factoryQueryKeys.workOrderChecks("org-1", "factory-1", "wo-1") });
    expect(query?.options.refetchInterval).toBe(ANALYZING_WORK_ORDER_CHECKS_POLL_MS);
  });
});
