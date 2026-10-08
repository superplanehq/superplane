import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook } from "@testing-library/react";
import { createElement, type ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "bun:test";

import type { FactoriesWorkOrder, FactoriesWorkOrderSummary } from "@/api-client";
import {
  BOARD_BACKLOG_STATES,
  factoryWorkOrdersPageKey,
  flattenWorkOrdersPages,
} from "@/pages/factories/lib/workOrderListPagination";

const { factoriesUpdateWorkOrderAssignees } = vi.hoisted(() => ({
  factoriesUpdateWorkOrderAssignees: vi.fn(),
}));

vi.mock("@/api-client", () => ({
  factoriesUpdateWorkOrderAssignees,
}));

import { factoryQueryKeys, useUpdateWorkOrderAssignees } from "./useFactoryData";

const ORGANIZATION_ID = "org-1";
const FACTORY_ID = "factory-1";

function summary(
  id: string,
  updatedAt: string,
  assignees: FactoriesWorkOrderSummary["assignees"] = [],
): FactoriesWorkOrderSummary {
  return {
    id,
    title: id,
    state: "STATE_DRAFT",
    updatedAt,
    assignees,
    lineDispatches: [],
    checkScores: [],
  };
}

function createWrapper(queryClient: QueryClient) {
  return function Wrapper({ children }: { children: ReactNode }) {
    return createElement(QueryClientProvider, { client: queryClient }, children);
  };
}

function seedBacklogPage(queryClient: QueryClient, orders: FactoriesWorkOrderSummary[]) {
  const pageKey = factoryWorkOrdersPageKey(ORGANIZATION_ID, FACTORY_ID, BOARD_BACKLOG_STATES);
  queryClient.setQueryData(pageKey, {
    pages: [{ orders, hasNextPage: false }],
    pageParams: [undefined],
  });
  queryClient.setQueryData(factoryQueryKeys.workOrders(ORGANIZATION_ID, FACTORY_ID), orders);
}

describe("useUpdateWorkOrderAssignees list cache", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("keeps backlog order when assignees change and updated_at moves forward", async () => {
    const orders = [
      summary("wo-a", "2026-01-01T00:00:00.000Z"),
      summary("wo-b", "2026-01-02T00:00:00.000Z"),
      summary("wo-c", "2026-01-03T00:00:00.000Z"),
    ];
    const queryClient = new QueryClient();
    seedBacklogPage(queryClient, orders);

    const serverOrder: FactoriesWorkOrder = {
      id: "wo-b",
      title: "wo-b",
      state: "STATE_DRAFT",
      updatedAt: "2026-02-01T00:00:00.000Z",
      assignees: [{ id: "user-2", name: "Bob" }],
      lineDispatches: [],
    };
    factoriesUpdateWorkOrderAssignees.mockResolvedValue({ data: { order: serverOrder } });

    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries");
    const { result } = renderHook(() => useUpdateWorkOrderAssignees(ORGANIZATION_ID, FACTORY_ID), {
      wrapper: createWrapper(queryClient),
    });

    await act(async () => {
      await result.current.mutateAsync({ orderId: "wo-b", assigneeIds: ["user-2"] });
    });

    const pageKey = factoryWorkOrdersPageKey(ORGANIZATION_ID, FACTORY_ID, BOARD_BACKLOG_STATES);
    const ids = flattenWorkOrdersPages(queryClient.getQueryData(pageKey)?.pages).map((order) => order.id);
    expect(ids).toEqual(["wo-a", "wo-b", "wo-c"]);

    const patched = flattenWorkOrdersPages(queryClient.getQueryData(pageKey)?.pages).find(
      (order) => order.id === "wo-b",
    );
    expect(patched?.assignees).toEqual([{ id: "user-2", name: "Bob" }]);
    expect(patched?.updatedAt).toBe("2026-02-01T00:00:00.000Z");

    expect(
      invalidateSpy.mock.calls.some(([options]) => JSON.stringify(options?.queryKey).includes("work-orders-page")),
    ).toBe(false);
    expect(invalidateSpy.mock.calls.some(([options]) => JSON.stringify(options?.queryKey).includes("events"))).toBe(
      true,
    );
  });
});
