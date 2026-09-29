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

import { useFactoryBoardWorkOrders, useFactoryWorkOrdersPage } from "./useFactoryData";

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

  it("sends close results on the list query", async () => {
    factoriesListWorkOrders.mockResolvedValueOnce(ordersPage([{ id: "wo-failed" }]));
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    const { result } = renderHook(
      () =>
        useFactoryWorkOrdersPage("org-1", "factory-1", ["STATE_CLOSED"], BOARD_BACKLOG_PAGE_SIZE, {
          results: ["RESULT_FAILED"],
        }),
      { wrapper: createWrapper(queryClient) },
    );

    await waitFor(() => expect(result.current.orders).toEqual([{ id: "wo-failed" }]));
    expect(factoriesListWorkOrders).toHaveBeenCalledWith(
      expect.objectContaining({
        query: expect.objectContaining({
          states: ["STATE_CLOSED"],
          results: ["RESULT_FAILED"],
        }),
      }),
    );
  });

  it("sends the line on the list query", async () => {
    factoriesListWorkOrders.mockResolvedValueOnce(ordersPage([{ id: "wo-failed" }]));
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    const { result } = renderHook(
      () =>
        useFactoryWorkOrdersPage("org-1", "factory-1", ["STATE_CLOSED"], BOARD_BACKLOG_PAGE_SIZE, {
          results: ["RESULT_FAILED"],
          lineId: "line-1",
        }),
      { wrapper: createWrapper(queryClient) },
    );

    await waitFor(() => expect(result.current.orders).toEqual([{ id: "wo-failed" }]));
    expect(factoriesListWorkOrders).toHaveBeenCalledWith(
      expect.objectContaining({
        query: expect.objectContaining({
          lineId: "line-1",
          results: ["RESULT_FAILED"],
        }),
      }),
    );
  });
});

describe("useFactoryBoardWorkOrders", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("scopes only the Done page to the line and the selected close result", async () => {
    factoriesListWorkOrders.mockResolvedValue(ordersPage([]));
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    renderHook(
      () =>
        useFactoryBoardWorkOrders("org-1", "factory-1", {
          userId: "user-1",
          done: { lineId: "line-1", results: ["RESULT_FAILED"] },
        }),
      { wrapper: createWrapper(queryClient) },
    );

    await waitFor(() => expect(factoriesListWorkOrders).toHaveBeenCalledTimes(3));
    const queries = factoriesListWorkOrders.mock.calls.map(
      (call) => call[0].query as { states?: string[]; lineId?: string; results?: string[] },
    );
    const closed = queries.find((query) => query.states?.[0] === "STATE_CLOSED");
    const draft = queries.find((query) => query.states?.[0] === "STATE_DRAFT");
    const open = queries.find((query) => query.states?.[0] === "STATE_OPEN");

    expect(closed).toEqual(
      expect.objectContaining({
        states: ["STATE_CLOSED"],
        results: ["RESULT_FAILED"],
        lineId: "line-1",
        userId: "user-1",
      }),
    );
    expect(draft).toEqual(expect.objectContaining({ states: ["STATE_DRAFT"], userId: "user-1" }));
    expect(draft?.lineId).toBeUndefined();
    expect(draft?.results).toBeUndefined();
    expect(open?.lineId).toBeUndefined();
    expect(open?.results).toBeUndefined();
  });

  it("does not request Done while the line is unknown", async () => {
    factoriesListWorkOrders.mockResolvedValue(ordersPage([]));
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    renderHook(() => useFactoryBoardWorkOrders("org-1", "factory-1", { done: { enabled: false } }), {
      wrapper: createWrapper(queryClient),
    });

    await waitFor(() => expect(factoriesListWorkOrders).toHaveBeenCalledTimes(2));
    const queries = factoriesListWorkOrders.mock.calls.map((call) => call[0].query as { states?: string[] });
    expect(queries.some((query) => query.states?.[0] === "STATE_CLOSED")).toBe(false);
    expect(queries.some((query) => query.states?.[0] === "STATE_DRAFT")).toBe(true);
    expect(queries.some((query) => query.states?.[0] === "STATE_OPEN")).toBe(true);
  });
});
