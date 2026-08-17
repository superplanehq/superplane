import type { ReactNode } from "react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { factoriesListLineMetrics } from "@/api-client";
import { factoryQueryKeys } from "./useFactoryData";
import { useFactoryLineMetrics } from "./useFactoryLineMetrics";

vi.mock("@/api-client", () => ({
  factoriesListLineMetrics: vi.fn(),
}));

function wrapper(queryClient: QueryClient) {
  return function TestQueryClientProvider({ children }: { children: ReactNode }) {
    return <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>;
  };
}

describe("useFactoryLineMetrics", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("keys the query by organization + factory", () => {
    expect(factoryQueryKeys.lineMetrics("org-1", "factory-1")).toEqual([
      "factories",
      "org-1",
      "factory-1",
      "line-metrics",
    ]);
  });

  it("reshapes the response array into a record keyed by line id", async () => {
    vi.mocked(factoriesListLineMetrics).mockResolvedValue({
      data: {
        metrics: [
          { lineId: "line-a", successRatePct: 90 },
          { lineId: "line-b", successRatePct: 50 },
        ],
      },
    } as Awaited<ReturnType<typeof factoriesListLineMetrics>>);

    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    const { result } = renderHook(() => useFactoryLineMetrics("org-1", "factory-1"), {
      wrapper: wrapper(queryClient),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(result.current.data).toEqual({
      "line-a": { lineId: "line-a", successRatePct: 90 },
      "line-b": { lineId: "line-b", successRatePct: 50 },
    });
  });

  it("drops entries with no line id and defaults to an empty record when the response is empty", async () => {
    vi.mocked(factoriesListLineMetrics).mockResolvedValue({
      data: { metrics: [{ successRatePct: 100 }] },
    } as Awaited<ReturnType<typeof factoriesListLineMetrics>>);

    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    const { result } = renderHook(() => useFactoryLineMetrics("org-1", "factory-1"), {
      wrapper: wrapper(queryClient),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data).toEqual({});
  });

  it("stays disabled until both organization and factory ids are present", () => {
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    const { result } = renderHook(() => useFactoryLineMetrics("", "factory-1"), {
      wrapper: wrapper(queryClient),
    });

    expect(result.current.fetchStatus).toBe("idle");
    expect(factoriesListLineMetrics).not.toHaveBeenCalled();
  });
});
