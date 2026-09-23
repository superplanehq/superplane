import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react";
import { createElement, type ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "bun:test";

import { BOARD_BACKLOG_PAGE_SIZE, BOARD_BACKLOG_STATES } from "@/pages/factories/lib/workOrderListPagination";

const { factoriesListWorkOrders } = vi.hoisted(() => ({
  factoriesListWorkOrders: vi.fn(),
}));

vi.mock("@/api-client", () => ({
  factoriesListWorkOrders,
}));

import { useFactoryWorkOrdersPage } from "./useFactoryData";

function createWrapper(queryClient: QueryClient) {
  return function Wrapper({ children }: { children: ReactNode }) {
    return createElement(QueryClientProvider, { client: queryClient }, children);
  };
}

function ordersPage(orders: { id: string }[]) {
  return { data: { orders, hasNextPage: false } };
}

describe("useFactoryWorkOrdersPage", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  /**
   * Scope and owner are part of the query key, so a filter change starts a new
   * query. Without the previous pages to hold, the board has no data to render
   * and drops into its full-screen loading state.
   */
  it("keeps the previous pages while the next board filter loads", async () => {
    let resolveNextPage: ((page: unknown) => void) | undefined;
    factoriesListWorkOrders.mockResolvedValueOnce(ordersPage([{ id: "wo-1" }])).mockImplementationOnce(
      () =>
        new Promise((resolve) => {
          resolveNextPage = resolve;
        }),
    );

    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    const { result, rerender } = renderHook(
      ({ userId }: { userId?: string }) =>
        useFactoryWorkOrdersPage("org-1", "factory-1", BOARD_BACKLOG_STATES, BOARD_BACKLOG_PAGE_SIZE, { userId }),
      { initialProps: { userId: undefined as string | undefined }, wrapper: createWrapper(queryClient) },
    );

    await waitFor(() => expect(result.current.orders).toEqual([{ id: "wo-1" }]));

    rerender({ userId: "user-1" });

    await waitFor(() => expect(result.current.isPlaceholderData).toBe(true));
    expect(result.current.isLoading).toBe(false);
    expect(result.current.orders).toEqual([{ id: "wo-1" }]);

    resolveNextPage?.(ordersPage([{ id: "wo-2" }]));

    await waitFor(() => expect(result.current.orders).toEqual([{ id: "wo-2" }]));
    expect(result.current.isPlaceholderData).toBe(false);
  });
});
