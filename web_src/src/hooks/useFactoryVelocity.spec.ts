import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react";
import { createElement, type ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

const { factoriesDescribeFactoryVelocity } = vi.hoisted(() => ({
  factoriesDescribeFactoryVelocity: vi.fn(),
}));

vi.mock("@/api-client", async (importOriginal) => {
  const actual = await importOriginal<Record<string, unknown>>();
  return {
    ...actual,
    factoriesDescribeFactoryVelocity,
  };
});

import { useFactoryVelocity } from "./useFactoryVelocity";

function createWrapper(queryClient: QueryClient) {
  return function Wrapper({ children }: { children: ReactNode }) {
    return createElement(QueryClientProvider, { client: queryClient }, children);
  };
}

function reportPage(people: { id: string }[]) {
  return { data: { peopleTotal: 12, peopleHasMore: true, people } };
}

describe("useFactoryVelocity", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  /**
   * The People offset is part of the query key, so "Show more" starts a new
   * query. Without the previous report to hold, the page has no data to render
   * and drops into its loading state, which reads as a page reload.
   */
  it("keeps the previous report while the next People page loads", async () => {
    let resolveNextPage: ((page: unknown) => void) | undefined;
    factoriesDescribeFactoryVelocity.mockResolvedValueOnce(reportPage([{ id: "person-1" }])).mockImplementationOnce(
      () =>
        new Promise((resolve) => {
          resolveNextPage = resolve;
        }),
    );

    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    const { result, rerender } = renderHook(
      ({ peopleOffset }: { peopleOffset: number }) =>
        useFactoryVelocity("org-1", "factory-1", { periodDays: 14, peopleOffset }),
      { initialProps: { peopleOffset: 0 }, wrapper: createWrapper(queryClient) },
    );

    await waitFor(() => expect(result.current.data?.people).toEqual([{ id: "person-1" }]));
    expect(factoriesDescribeFactoryVelocity.mock.calls[0]?.[0].query.peoplePageSize).toBe(5);

    rerender({ peopleOffset: 5 });

    await waitFor(() => expect(result.current.isFetching).toBe(true));
    expect(result.current.isLoading).toBe(false);
    expect(result.current.data?.people).toEqual([{ id: "person-1" }]);
    expect(result.current.isPlaceholderData).toBe(true);

    resolveNextPage?.(reportPage([{ id: "person-7" }]));

    await waitFor(() => expect(result.current.data?.people).toEqual([{ id: "person-7" }]));
    expect(result.current.isPlaceholderData).toBe(false);
    expect(factoriesDescribeFactoryVelocity.mock.calls[1]?.[0].query.peoplePageSize).toBe(20);
  });
});
