import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook, waitFor } from "@testing-library/react";
import { createElement, type ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import type { FactoriesWorkOrder } from "@/api-client";

const { factoriesReorderWorkOrder } = vi.hoisted(() => ({
  factoriesReorderWorkOrder: vi.fn(),
}));

vi.mock("@/api-client", async (importOriginal) => {
  const actual = await importOriginal<Record<string, unknown>>();
  return {
    ...actual,
    factoriesReorderWorkOrder,
  };
});

import { factoryQueryKeys, useReorderWorkOrder } from "./useFactoryData";

const ORGANIZATION_ID = "org-1";
const FACTORY_ID = "factory-1";

function draftOrder(id: string): FactoriesWorkOrder {
  return { id, title: id, state: "STATE_DRAFT", lineDispatches: [] };
}

function createWrapper(queryClient: QueryClient) {
  return function Wrapper({ children }: { children: ReactNode }) {
    return createElement(QueryClientProvider, { client: queryClient }, children);
  };
}

function seedClient(orders: FactoriesWorkOrder[]) {
  const queryClient = new QueryClient();
  queryClient.setQueryData(factoryQueryKeys.workOrders(ORGANIZATION_ID, FACTORY_ID), orders);
  return queryClient;
}

function orderIdsInCache(queryClient: QueryClient): (string | undefined)[] {
  const orders =
    queryClient.getQueryData<FactoriesWorkOrder[]>(factoryQueryKeys.workOrders(ORGANIZATION_ID, FACTORY_ID)) ?? [];
  return orders.map((order) => order.id);
}

describe("useReorderWorkOrder optimistic cache patch", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("moves the card in the cache before the request resolves", async () => {
    factoriesReorderWorkOrder.mockReturnValue(new Promise(() => {}));

    const queryClient = seedClient([draftOrder("a"), draftOrder("b"), draftOrder("c")]);
    const { result } = renderHook(() => useReorderWorkOrder(ORGANIZATION_ID, FACTORY_ID), {
      wrapper: createWrapper(queryClient),
    });

    act(() => {
      result.current.mutate({ orderId: "a", previousOrderId: "c" });
    });

    await waitFor(() => {
      expect(orderIdsInCache(queryClient)).toEqual(["b", "c", "a"]);
    });
  });

  it("rolls back to the original order when the request fails", async () => {
    factoriesReorderWorkOrder.mockRejectedValue(new Error("boom"));

    const original = [draftOrder("a"), draftOrder("b"), draftOrder("c")];
    const queryClient = seedClient(original);
    const { result } = renderHook(() => useReorderWorkOrder(ORGANIZATION_ID, FACTORY_ID), {
      wrapper: createWrapper(queryClient),
    });

    await act(async () => {
      await expect(result.current.mutateAsync({ orderId: "a", previousOrderId: "c" })).rejects.toThrow("boom");
    });

    expect(orderIdsInCache(queryClient)).toEqual(["a", "b", "c"]);
  });

  it("writes the server order into the cache once the reorder resolves", async () => {
    const serverOrder: FactoriesWorkOrder = { id: "a", title: "a", state: "STATE_DRAFT", position: 42 };
    factoriesReorderWorkOrder.mockResolvedValue({ data: { order: serverOrder } });

    const queryClient = seedClient([draftOrder("a"), draftOrder("b"), draftOrder("c")]);
    const { result } = renderHook(() => useReorderWorkOrder(ORGANIZATION_ID, FACTORY_ID), {
      wrapper: createWrapper(queryClient),
    });

    await act(async () => {
      await result.current.mutateAsync({ orderId: "a", previousOrderId: "c" });
    });

    const orders =
      queryClient.getQueryData<FactoriesWorkOrder[]>(factoryQueryKeys.workOrders(ORGANIZATION_ID, FACTORY_ID)) ?? [];
    expect(orders.find((order) => order.id === "a")?.position).toBe(42);
  });
});
