import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook, waitFor } from "@testing-library/react";
import { createElement, type ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import type { FactoriesFactory, FactoriesWorkOrder } from "@/api-client";
import { buildLinePhaseBoard, collectLineBacklogOrders } from "@/pages/factories/lib/linePhaseRuns";

const { factoriesDispatchWorkOrder } = vi.hoisted(() => ({
  factoriesDispatchWorkOrder: vi.fn(),
}));

vi.mock("@/api-client", async (importOriginal) => {
  const actual = await importOriginal<Record<string, unknown>>();
  return {
    ...actual,
    factoriesDispatchWorkOrder,
  };
});

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

describe("useDispatchWorkOrder optimistic cache patch", () => {
  beforeEach(() => {
    vi.clearAllMocks();
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
    expect(orders.find((order) => order.id === "wo-1")).toEqual(serverOrder);

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
});
