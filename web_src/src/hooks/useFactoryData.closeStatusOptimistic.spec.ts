import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { InfiniteData } from "@tanstack/react-query";
import { act, renderHook, waitFor } from "@testing-library/react";
import { createElement, type ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "bun:test";

import type { FactoriesWorkOrder, FactoriesWorkOrderSummary } from "@/api-client";
import {
  BOARD_BACKLOG_STATES,
  BOARD_OPEN_STATES,
  factoryWorkOrdersPageKey,
  flattenWorkOrdersPages,
  type WorkOrdersPage,
} from "@/pages/factories/lib/workOrderListPagination";

const { factoriesCloseWorkOrder, factoriesUpdateWorkOrderStatus } = vi.hoisted(() => ({
  factoriesCloseWorkOrder: vi.fn(),
  factoriesUpdateWorkOrderStatus: vi.fn(),
}));

vi.mock("@/api-client", () => ({
  factoriesCloseWorkOrder,
  factoriesUpdateWorkOrderStatus,
}));

import { useCloseWorkOrder, useUpdateWorkOrderStatus } from "./useFactoryData";

const ORGANIZATION_ID = "org-1";
const FACTORY_ID = "factory-1";

function createWrapper(queryClient: QueryClient) {
  return function Wrapper({ children }: { children: ReactNode }) {
    return createElement(QueryClientProvider, { client: queryClient }, children);
  };
}

function seedPage(
  states: readonly FactoriesWorkOrder["state"][],
  orders: FactoriesWorkOrderSummary[],
): { queryClient: QueryClient; pageKey: readonly unknown[] } {
  const queryClient = new QueryClient();
  const pageKey = factoryWorkOrdersPageKey(ORGANIZATION_ID, FACTORY_ID, states, { unassigned: false });
  const data: InfiniteData<WorkOrdersPage> = {
    pages: [{ orders, hasNextPage: false }],
    pageParams: [undefined],
  };
  queryClient.setQueryData(pageKey, data);
  return { queryClient, pageKey };
}

function ordersInPage(queryClient: QueryClient, pageKey: readonly unknown[]): FactoriesWorkOrderSummary[] {
  const data = queryClient.getQueryData<InfiniteData<WorkOrdersPage>>(pageKey);
  return flattenWorkOrdersPages(data?.pages);
}

function deferred<T>() {
  let resolve!: (value: T) => void;
  const promise = new Promise<T>((res) => {
    resolve = res;
  });
  return { promise, resolve };
}

describe("useCloseWorkOrder cache patch", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("removes the card from the cached backlog list as soon as the close response arrives", async () => {
    const draft: FactoriesWorkOrderSummary = { id: "wo-1", title: "Fix the outage", state: "STATE_DRAFT" };
    const { queryClient, pageKey } = seedPage(BOARD_BACKLOG_STATES, [draft]);

    // A request still in flight — the card must stay put until the server
    // actually confirms the close, but must not wait on a list refetch
    // once it does.
    const { promise: closeResponse, resolve: resolveClose } = deferred<{ data: { order: FactoriesWorkOrder } }>();
    factoriesCloseWorkOrder.mockReturnValue(closeResponse);

    const { result } = renderHook(() => useCloseWorkOrder(ORGANIZATION_ID, FACTORY_ID), {
      wrapper: createWrapper(queryClient),
    });

    act(() => {
      result.current.mutate({ orderId: "wo-1", result: "RESULT_REJECTED" });
    });

    expect(ordersInPage(queryClient, pageKey).map((order) => order.id)).toContain("wo-1");

    const closed: FactoriesWorkOrder = {
      id: "wo-1",
      title: "Fix the outage",
      state: "STATE_CLOSED",
      result: "RESULT_REJECTED",
    };

    await act(async () => {
      resolveClose({ data: { order: closed } });
      await closeResponse;
    });

    await waitFor(() => {
      expect(ordersInPage(queryClient, pageKey).map((order) => order.id)).not.toContain("wo-1");
    });

    expect(factoriesCloseWorkOrder).toHaveBeenCalledTimes(1);
  });
});

describe("useUpdateWorkOrderStatus cache patch", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("removes the card from the cached open list on a reject transition", async () => {
    const open: FactoriesWorkOrderSummary = { id: "wo-2", title: "Investigate outage", state: "STATE_OPEN" };
    const { queryClient, pageKey } = seedPage(BOARD_OPEN_STATES, [open]);

    const { promise: statusResponse, resolve: resolveStatus } = deferred<{ data: { order: FactoriesWorkOrder } }>();
    factoriesUpdateWorkOrderStatus.mockReturnValue(statusResponse);

    const { result } = renderHook(() => useUpdateWorkOrderStatus(ORGANIZATION_ID, FACTORY_ID), {
      wrapper: createWrapper(queryClient),
    });

    act(() => {
      result.current.mutate({ orderId: "wo-2", state: "STATE_CLOSED", result: "RESULT_REJECTED" });
    });

    expect(ordersInPage(queryClient, pageKey).map((order) => order.id)).toContain("wo-2");

    const closed: FactoriesWorkOrder = {
      id: "wo-2",
      title: "Investigate outage",
      state: "STATE_CLOSED",
      result: "RESULT_REJECTED",
    };

    await act(async () => {
      resolveStatus({ data: { order: closed } });
      await statusResponse;
    });

    await waitFor(() => {
      expect(ordersInPage(queryClient, pageKey).map((order) => order.id)).not.toContain("wo-2");
    });

    expect(factoriesUpdateWorkOrderStatus).toHaveBeenCalledTimes(1);
  });
});
