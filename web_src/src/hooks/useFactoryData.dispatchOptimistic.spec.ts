import { QueryClient, QueryClientProvider, type InfiniteData } from "@tanstack/react-query";
import { act, renderHook, waitFor } from "@testing-library/react";
import { createElement, type ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "bun:test";

import type { FactoriesFactory, FactoriesWorkOrder } from "@/api-client";
import { buildLinePhaseBoard, collectLineBacklogOrders } from "@/pages/factories/lib/linePhaseRuns";
import type { WorkOrdersPage } from "@/pages/factories/lib/workOrderListPagination";

const { factoriesDispatchWorkOrder, factoriesResumeHandOffWorkOrder } = vi.hoisted(() => ({
  factoriesDispatchWorkOrder: vi.fn(),
  factoriesResumeHandOffWorkOrder: vi.fn(),
}));

vi.mock("@/api-client", () => ({
  factoriesDispatchWorkOrder,
  factoriesResumeHandOffWorkOrder,
}));

import { factoryQueryKeys, useDispatchWorkOrder } from "./useFactoryData";

const ORGANIZATION_ID = "org-1";
const FACTORY_ID = "factory-1";

const LINE = { id: "line-1", name: "hotfix", steps: [{ app: { app: "app-build" } }] };

const FACTORY: FactoriesFactory = {
  id: FACTORY_ID,
  name: "Factory",
  lines: [LINE],
};

function draftOrder(): FactoriesWorkOrder {
  return { id: "wo-1", title: "Fix the outage", state: "STATE_DRAFT", lineDispatches: [] };
}

function createWrapper(queryClient: QueryClient) {
  return function Wrapper({ children }: { children: ReactNode }) {
    return createElement(QueryClientProvider, { client: queryClient }, children);
  };
}

function seedClient(orders: FactoriesWorkOrder[]) {
  const queryClient = new QueryClient();
  queryClient.setQueryData(factoryQueryKeys.detail(ORGANIZATION_ID, FACTORY_ID), FACTORY);
  queryClient.setQueryData(factoryQueryKeys.workOrders(ORGANIZATION_ID, FACTORY_ID), orders);
  return queryClient;
}

function ordersInCache(queryClient: QueryClient): FactoriesWorkOrder[] {
  return queryClient.getQueryData<FactoriesWorkOrder[]>(factoryQueryKeys.workOrders(ORGANIZATION_ID, FACTORY_ID)) ?? [];
}

function seedPendingHandoff(source: "detail" | "list" | "page") {
  const order = {
    ...draftOrder(),
    pendingHandoff: { lineId: LINE.id, column: "verify", pullRequestUrl: "https://github.com/acme/app/pull/42" },
  };
  const queryClient = seedClient(source === "list" ? [order] : []);
  if (source === "detail") {
    queryClient.setQueryData(factoryQueryKeys.workOrderDetail(ORGANIZATION_ID, FACTORY_ID, "wo-1"), order);
  }
  const pageKey = factoryQueryKeys.workOrdersPage(ORGANIZATION_ID, FACTORY_ID, ["STATE_DRAFT", "STATE_OPEN"]);
  const page: InfiniteData<WorkOrdersPage> = {
    pages: [{ orders: source === "page" ? [order] : [draftOrder()], hasNextPage: false, totalCount: 1 }],
    pageParams: [undefined],
  };
  queryClient.setQueryData(pageKey, page);
  return { queryClient, order, pageKey, page };
}

describe("useDispatchWorkOrder optimistic cache patch", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    factoriesDispatchWorkOrder.mockReset();
    factoriesResumeHandOffWorkOrder.mockReset();
  });

  it("moves the card to the first phase column before the dispatch request resolves", async () => {
    // A promise that never resolves during the assertions below stands in
    // for a dispatch request still in flight — the card must already be on
    // the board without any response, let alone a ListWorkOrders refetch.
    factoriesDispatchWorkOrder.mockReturnValue(new Promise(() => {}));

    const queryClient = seedClient([draftOrder()]);
    const { result } = renderHook(() => useDispatchWorkOrder(ORGANIZATION_ID, FACTORY_ID), {
      wrapper: createWrapper(queryClient),
    });

    act(() => {
      result.current.mutate({ orderId: "wo-1", lineName: "hotfix" });
    });

    // onMutate runs as a microtask, ahead of the (never-resolving) request
    // above; wait only for that patch, not for any response.
    await waitFor(() => {
      const orders = ordersInCache(queryClient);
      expect(collectLineBacklogOrders(orders).map((order) => order.id)).not.toContain("wo-1");
    });

    const orders = ordersInCache(queryClient);
    const board = buildLinePhaseBoard(LINE, orders);

    expect(board[0]?.runs.map((run) => run.workOrderId)).toContain("wo-1");
    expect(factoriesDispatchWorkOrder).toHaveBeenCalledTimes(1);
    expect(factoriesResumeHandOffWorkOrder).not.toHaveBeenCalled();
  });

  it("rolls the card back to Backlog when the dispatch request fails", async () => {
    factoriesDispatchWorkOrder.mockRejectedValue(new Error("boom"));

    const original = [draftOrder()];
    const queryClient = seedClient(original);
    const { result } = renderHook(() => useDispatchWorkOrder(ORGANIZATION_ID, FACTORY_ID), {
      wrapper: createWrapper(queryClient),
    });

    await act(async () => {
      await expect(result.current.mutateAsync({ orderId: "wo-1", lineName: "hotfix" })).rejects.toThrow("boom");
    });

    const orders = ordersInCache(queryClient);
    const board = buildLinePhaseBoard(LINE, orders);

    expect(board[0]?.runs.map((run) => run.workOrderId)).not.toContain("wo-1");
    expect(collectLineBacklogOrders(orders).map((order) => order.id)).toContain("wo-1");
    expect(orders).toEqual(original);
  });

  it("writes the server order into the cache once the dispatch resolves", async () => {
    const serverOrder: FactoriesWorkOrder = {
      id: "wo-1",
      title: "Fix the outage",
      state: "STATE_OPEN",
      lineDispatches: [
        {
          id: "dispatch-real-1",
          line: { id: LINE.id, name: LINE.name },
          state: "STATE_ACTIVE",
          createdAt: "2026-08-31T00:00:00.000Z",
          stepExecutions: [
            {
              id: "execution-real-1",
              stepIndex: 0,
              state: "STATE_PENDING",
              createdAt: "2026-08-31T00:00:00.000Z",
              updatedAt: "2026-08-31T00:00:00.000Z",
              run: { id: "run-real-1", appId: "app-build" },
            },
          ],
        },
      ],
    };
    factoriesDispatchWorkOrder.mockResolvedValue({ data: { order: serverOrder } });

    const queryClient = seedClient([draftOrder()]);
    const { result } = renderHook(() => useDispatchWorkOrder(ORGANIZATION_ID, FACTORY_ID), {
      wrapper: createWrapper(queryClient),
    });

    await act(async () => {
      await result.current.mutateAsync({ orderId: "wo-1", lineName: "hotfix" });
    });

    const orders = ordersInCache(queryClient);
    expect(orders.find((order) => order.id === "wo-1")).toMatchObject(serverOrder);

    const board = buildLinePhaseBoard(LINE, orders);
    expect(board[0]?.runs.map((run) => run.workOrderId)).toContain("wo-1");
  });

  it("falls back to invalidate-only when the line isn't in the factory-detail cache yet", async () => {
    factoriesDispatchWorkOrder.mockReturnValue(new Promise(() => {}));

    const queryClient = new QueryClient();
    queryClient.setQueryData(factoryQueryKeys.workOrders(ORGANIZATION_ID, FACTORY_ID), [draftOrder()]);
    const { result } = renderHook(() => useDispatchWorkOrder(ORGANIZATION_ID, FACTORY_ID), {
      wrapper: createWrapper(queryClient),
    });

    act(() => {
      result.current.mutate({ orderId: "wo-1", lineName: "hotfix" });
    });

    const orders = ordersInCache(queryClient);
    expect(orders).toEqual([draftOrder()]);
  });

  it.each(["detail", "list", "page"] as const)("resumes a pending handoff from the %s cache", async (source) => {
    const { queryClient, pageKey } = seedPendingHandoff(source);
    const serverOrder: FactoriesWorkOrder = {
      ...draftOrder(),
      state: "STATE_OPEN",
      lineDispatches: [{ id: "resumed-dispatch", line: { id: LINE.id, name: LINE.name }, state: "STATE_ACTIVE" }],
    };
    factoriesDispatchWorkOrder.mockResolvedValue({ data: { order: serverOrder } });
    factoriesResumeHandOffWorkOrder.mockResolvedValue({ data: { order: serverOrder } });
    const { result } = renderHook(() => useDispatchWorkOrder(ORGANIZATION_ID, FACTORY_ID), {
      wrapper: createWrapper(queryClient),
    });

    await act(async () => {
      await result.current.mutateAsync({
        orderId: "wo-1",
        lineName: "hotfix",
        model: "codex/model-1",
        thinkingLevel: "high",
      });
    });

    expect(factoriesDispatchWorkOrder).not.toHaveBeenCalled();
    expect(factoriesResumeHandOffWorkOrder).toHaveBeenCalledTimes(1);
    expect(factoriesResumeHandOffWorkOrder).toHaveBeenCalledWith(
      expect.objectContaining({
        path: { factoryId: FACTORY_ID, orderId: "wo-1" },
        body: { model: "codex/model-1", thinkingLevel: "high" },
      }),
    );
    const detail = queryClient.getQueryData<FactoriesWorkOrder>(
      factoryQueryKeys.workOrderDetail(ORGANIZATION_ID, FACTORY_ID, "wo-1"),
    );
    const pageOrder = queryClient.getQueryData<InfiniteData<WorkOrdersPage>>(pageKey)?.pages[0]?.orders[0];
    expect(detail).toEqual(serverOrder);
    for (const cachedOrder of [pageOrder, ordersInCache(queryClient)[0]]) {
      expect(cachedOrder).toMatchObject(serverOrder);
      expect(cachedOrder?.pendingHandoff).toBeUndefined();
    }
  });

  it.each(["detail", "list", "page"] as const)(
    "restores caches and keeps the pending handoff from %s after resume fails",
    async (source) => {
      const { queryClient, order, pageKey, page } = seedPendingHandoff(source);
      const previousOrders = ordersInCache(queryClient);
      factoriesDispatchWorkOrder.mockRejectedValue(new Error("ordinary dispatch used"));
      factoriesResumeHandOffWorkOrder.mockRejectedValue(new Error("connection unavailable"));
      const { result } = renderHook(() => useDispatchWorkOrder(ORGANIZATION_ID, FACTORY_ID), {
        wrapper: createWrapper(queryClient),
      });

      await act(async () => {
        await expect(result.current.mutateAsync({ orderId: "wo-1", lineName: "hotfix" })).rejects.toThrow(
          "connection unavailable",
        );
      });

      expect(factoriesDispatchWorkOrder).not.toHaveBeenCalled();
      expect(ordersInCache(queryClient)).toEqual(previousOrders);
      expect(queryClient.getQueryData(pageKey)).toEqual(page);
      const pendingOrder =
        source === "detail"
          ? queryClient.getQueryData<FactoriesWorkOrder>(
              factoryQueryKeys.workOrderDetail(ORGANIZATION_ID, FACTORY_ID, "wo-1"),
            )
          : source === "list"
            ? ordersInCache(queryClient)[0]
            : queryClient.getQueryData<InfiniteData<WorkOrdersPage>>(pageKey)?.pages[0]?.orders[0];
      expect(pendingOrder?.pendingHandoff).toEqual(order.pendingHandoff);
    },
  );
});
