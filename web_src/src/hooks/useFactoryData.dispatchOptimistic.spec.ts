import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook, waitFor } from "@testing-library/react";
import { createElement, type ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "bun:test";

import type { FactoriesFactory, FactoriesWorkOrder } from "@/api-client";
import { buildLinePhaseBoard, collectLineBacklogOrders } from "@/pages/factories/lib/linePhaseRuns";

const { factoriesDispatchWorkOrder } = vi.hoisted(() => ({
  factoriesDispatchWorkOrder: vi.fn(),
}));

vi.mock("@/api-client", () => ({
  factoriesDispatchWorkOrder,
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

describe("useDispatchWorkOrder cache updates", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("keeps the card in Backlog while the dispatch request is in flight", async () => {
    factoriesDispatchWorkOrder.mockReturnValue(new Promise(() => {}));

    const queryClient = seedClient([draftOrder()]);
    const { result } = renderHook(() => useDispatchWorkOrder(ORGANIZATION_ID, FACTORY_ID), {
      wrapper: createWrapper(queryClient),
    });

    act(() => {
      result.current.mutate({ orderId: "wo-1", lineName: "hotfix" });
    });

    await waitFor(() => {
      expect(factoriesDispatchWorkOrder).toHaveBeenCalledTimes(1);
    });

    const orders = ordersInCache(queryClient);
    expect(collectLineBacklogOrders(orders).map((order) => order.id)).toContain("wo-1");
    expect(buildLinePhaseBoard(LINE, orders)[0]?.runs.map((run) => run.workOrderId)).not.toContain("wo-1");
  });

  it("leaves the card in Backlog when the dispatch request fails", async () => {
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

  it("keeps a first-step queued task in Backlog after dispatch resolves", async () => {
    const serverOrder: FactoriesWorkOrder = {
      id: "wo-1",
      title: "Fix the outage",
      state: "STATE_OPEN",
      lineDispatches: [
        {
          id: "dispatch-queued-1",
          line: { id: LINE.id, name: LINE.name },
          state: "STATE_ACTIVE",
          createdAt: "2026-08-31T00:00:00.000Z",
          stepExecutions: [],
          queueItem: {
            id: "queue-1",
            stepName: "hotfix",
            stepIndex: 0,
            position: 1,
            appId: "app-build",
          },
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
    expect(collectLineBacklogOrders(orders, LINE.id).map((order) => order.id)).toContain("wo-1");
    expect(buildLinePhaseBoard(LINE, orders)[0]?.runs.map((run) => run.workOrderId)).not.toContain("wo-1");
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
