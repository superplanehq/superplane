import type { FactoriesWorkOrder, FactoriesWorkOrderState, FactoriesWorkOrderSummary } from "@/api-client";
import type { InfiniteData, QueryClient } from "@tanstack/react-query";

import {
  factoryWorkOrdersPagePrefix,
  flattenWorkOrdersPages,
  workOrderMatchesPageQuery,
  workOrdersPageQueryFromKey,
  type WorkOrdersPage,
  type WorkOrdersPageQuery,
} from "@/pages/factories/lib/workOrderListPagination";

function workOrdersListKey(organizationId: string, factoryId: string) {
  return ["factories", organizationId, factoryId, "work-orders"] as const;
}

const SUMMARY_FIELDS = [
  "id",
  "title",
  "description",
  "state",
  "result",
  "createdAt",
  "updatedAt",
  "assignees",
  "createdBy",
  "totalTokens",
  "totalCostCents",
  "number",
  "key",
  "lineDispatches",
  "statusNotes",
  "origin",
  "totalDurationSeconds",
  "pullRequests",
] as const;

function checkScoresFromChecks(order: FactoriesWorkOrder): FactoriesWorkOrderSummary["checkScores"] {
  return (order.checks ?? []).map((check) => ({
    key: check.key,
    name: check.name,
    score: check.score,
    maxScore: check.maxScore,
  }));
}

function summaryFromDescribedOrder(order: FactoriesWorkOrder): FactoriesWorkOrderSummary {
  const summary: FactoriesWorkOrderSummary = { checkScores: checkScoresFromChecks(order) };
  for (const field of SUMMARY_FIELDS) {
    if (order[field] !== undefined) {
      summary[field] = order[field] as never;
    }
  }
  return summary;
}

export function patchedWorkOrder(
  order: FactoriesWorkOrderSummary,
  described: FactoriesWorkOrder,
): FactoriesWorkOrderSummary {
  return { ...order, ...summaryFromDescribedOrder(described), planningSession: described.planningSession };
}

export function patchCachedWorkOrderList(
  orders: FactoriesWorkOrderSummary[] | undefined,
  orderId: string,
  described: FactoriesWorkOrder,
): FactoriesWorkOrderSummary[] | undefined {
  if (!orders) {
    return orders;
  }
  const index = orders.findIndex((order) => order.id === orderId);
  if (index < 0) {
    return [patchedWorkOrder({ id: described.id ?? orderId }, described), ...orders];
  }
  return orders.map((order) => (order.id === orderId ? patchedWorkOrder(order, described) : order));
}

function pageIncludesState(states: readonly string[], state: FactoriesWorkOrderState | undefined): boolean {
  return Boolean(state && states.includes(state));
}

export function workOrdersPageStatesFromKey(queryKey: readonly unknown[]): FactoriesWorkOrderState[] {
  const joined = queryKey[4];
  if (typeof joined !== "string" || joined.length === 0) {
    return [];
  }
  return joined.split(",") as FactoriesWorkOrderState[];
}

export function patchCachedWorkOrderPages(
  data: InfiniteData<WorkOrdersPage> | undefined,
  orderId: string,
  described: FactoriesWorkOrder,
  states: readonly FactoriesWorkOrderState[],
  query: WorkOrdersPageQuery = { unassigned: false },
): InfiniteData<WorkOrdersPage> | undefined {
  if (!data) {
    return data;
  }

  const belongs = pageIncludesState(states, described.state) && workOrderMatchesPageQuery(described, query);
  const exists = data.pages.some((page) => page.orders.some((order) => order.id === orderId));

  if (!belongs) {
    if (!exists) {
      return data;
    }
    return {
      ...data,
      pages: data.pages.map((page) => ({
        ...page,
        orders: page.orders.filter((order) => order.id !== orderId),
      })),
    };
  }

  if (exists) {
    return {
      ...data,
      pages: data.pages.map((page) => ({
        ...page,
        orders: page.orders.map((order) => (order.id === orderId ? patchedWorkOrder(order, described) : order)),
      })),
    };
  }

  const [first, ...rest] = data.pages;
  if (!first) {
    return {
      ...data,
      pages: [{ orders: [patchedWorkOrder({ id: described.id ?? orderId }, described)], hasNextPage: false }],
    };
  }

  return {
    ...data,
    pages: [
      {
        ...first,
        orders: [patchedWorkOrder({ id: described.id ?? orderId }, described), ...first.orders],
      },
      ...rest,
    ],
  };
}

export function applyWorkOrderToListCaches(
  queryClient: QueryClient,
  organizationId: string,
  factoryId: string,
  orderId: string,
  described: FactoriesWorkOrder,
): void {
  queryClient.setQueryData<FactoriesWorkOrderSummary[]>(workOrdersListKey(organizationId, factoryId), (orders) =>
    patchCachedWorkOrderList(orders, orderId, described),
  );

  for (const query of queryClient.getQueryCache().findAll({
    queryKey: factoryWorkOrdersPagePrefix(organizationId, factoryId),
  })) {
    queryClient.setQueryData<InfiniteData<WorkOrdersPage>>(query.queryKey, (data) =>
      patchCachedWorkOrderPages(
        data,
        orderId,
        described,
        workOrdersPageStatesFromKey(query.queryKey),
        workOrdersPageQueryFromKey(query.queryKey),
      ),
    );
  }
}

export function cachedWorkOrderFromLists(
  queryClient: QueryClient,
  organizationId: string,
  factoryId: string,
  orderId: string,
): FactoriesWorkOrderSummary | undefined {
  const list =
    queryClient.getQueryData<FactoriesWorkOrderSummary[]>(workOrdersListKey(organizationId, factoryId)) ?? [];
  const fromList = list.find((order) => order.id === orderId);
  if (fromList) {
    return fromList;
  }
  for (const query of queryClient.getQueryCache().findAll({
    queryKey: factoryWorkOrdersPagePrefix(organizationId, factoryId),
  })) {
    const data = query.state.data as InfiniteData<WorkOrdersPage> | undefined;
    const match = flattenWorkOrdersPages(data?.pages).find((order) => order.id === orderId);
    if (match) {
      return match;
    }
  }
  return undefined;
}

export function cachedWorkOrderIdsFromLists(
  queryClient: QueryClient,
  organizationId: string,
  factoryId: string,
): string[] {
  const ids = new Set<string>();
  const list =
    queryClient.getQueryData<FactoriesWorkOrderSummary[]>(workOrdersListKey(organizationId, factoryId)) ?? [];
  for (const order of list) {
    if (order.id) {
      ids.add(order.id);
    }
  }
  for (const query of queryClient.getQueryCache().findAll({
    queryKey: factoryWorkOrdersPagePrefix(organizationId, factoryId),
  })) {
    const data = query.state.data as InfiniteData<WorkOrdersPage> | undefined;
    for (const order of flattenWorkOrdersPages(data?.pages)) {
      if (order.id) {
        ids.add(order.id);
      }
    }
  }
  return [...ids];
}
