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

function pageIncludesResults(results: readonly string[], result: FactoriesWorkOrder["result"]): boolean {
  if (results.length === 0) {
    return true;
  }
  return Boolean(result && results.includes(result));
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
  query: WorkOrdersPageQuery = { unassigned: false, results: [] },
): InfiniteData<WorkOrdersPage> | undefined {
  if (!data) {
    return data;
  }

  const belongs =
    pageIncludesState(states, described.state) &&
    pageIncludesResults(query.results, described.result) &&
    workOrderMatchesPageQuery(described, query);
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

function withoutWorkOrder<T extends { id?: string }>(orders: T[], orderId: string): T[] {
  return orders.filter((order) => order.id !== orderId);
}

function pageContainsWorkOrder(page: WorkOrdersPage, orderId: string): boolean {
  return page.orders.some((order) => order.id === orderId);
}

function pageHasNextCursor(page: WorkOrdersPage | undefined): boolean {
  return Boolean(page?.hasNextPage && page.orders.at(-1)?.id);
}

function withoutWorkOrderPages(data: InfiniteData<WorkOrdersPage>, orderId: string): InfiniteData<WorkOrdersPage> {
  const pages = data.pages.map((page) => ({
    ...page,
    orders: withoutWorkOrder(page.orders, orderId),
  }));
  const pageParams = data.pageParams.slice(0, pages.length);

  while (pages.length > 1 && (pages[pages.length - 1]?.orders.length ?? 0) === 0) {
    const dropped = pages.pop();
    pageParams.pop();
    const previous = pages[pages.length - 1];
    if (!previous || !dropped) {
      break;
    }
    pages[pages.length - 1] = { ...previous, hasNextPage: dropped.hasNextPage };
  }

  return { pages, pageParams };
}

function pagedListLostNextCursor(before: InfiniteData<WorkOrdersPage>, after: InfiniteData<WorkOrdersPage>): boolean {
  const nextPageRemains = Boolean(after.pages.at(-1)?.hasNextPage);
  return pageHasNextCursor(before.pages.at(-1)) && !pageHasNextCursor(after.pages.at(-1)) && nextPageRemains;
}

export function removeWorkOrderFromListCaches(
  queryClient: QueryClient,
  organizationId: string,
  factoryId: string,
  orderId: string,
): void {
  queryClient.setQueryData<FactoriesWorkOrderSummary[]>(workOrdersListKey(organizationId, factoryId), (orders) => {
    if (!orders) {
      return orders;
    }
    return withoutWorkOrder(orders, orderId);
  });

  for (const query of queryClient.getQueryCache().findAll({
    queryKey: factoryWorkOrdersPagePrefix(organizationId, factoryId),
  })) {
    const current = queryClient.getQueryData<InfiniteData<WorkOrdersPage>>(query.queryKey);
    if (!current || !current.pages.some((page) => pageContainsWorkOrder(page, orderId))) {
      continue;
    }

    const next = withoutWorkOrderPages(current, orderId);
    queryClient.setQueryData(query.queryKey, next);
    if (pagedListLostNextCursor(current, next)) {
      void queryClient.invalidateQueries({ queryKey: query.queryKey });
    }
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
