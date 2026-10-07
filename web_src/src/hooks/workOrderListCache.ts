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

function orderIsOnLoadedPages(pages: readonly WorkOrdersPage[], orderId: string): boolean {
  return pages.some((page) => page.orders.some((order) => order.id === orderId));
}

function loadedWorkOrderPagesAreComplete(pages: readonly WorkOrdersPage[]): boolean {
  const last = pages.at(-1);
  return last !== undefined && !last.hasNextPage;
}

function describedOrderBelongsOnPage(
  described: FactoriesWorkOrder,
  states: readonly FactoriesWorkOrderState[],
  query: WorkOrdersPageQuery,
): boolean {
  return (
    pageIncludesState(states, described.state) &&
    pageIncludesResults(query.results, described.result) &&
    workOrderMatchesPageQuery(described, query)
  );
}

function shiftPageTotalCount(page: WorkOrdersPage, delta: number): WorkOrdersPage {
  if (page.totalCount === undefined || delta === 0) {
    return page;
  }
  return { ...page, totalCount: Math.max(0, page.totalCount + delta) };
}

function shiftPagesTotalCount(pages: WorkOrdersPage[], delta: number): WorkOrdersPage[] {
  if (delta === 0) {
    return pages;
  }
  return pages.map((page) => shiftPageTotalCount(page, delta));
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

  const belongs = describedOrderBelongsOnPage(described, states, query);
  const exists = orderIsOnLoadedPages(data.pages, orderId);

  if (!belongs) {
    if (!exists) {
      return data;
    }
    return {
      ...data,
      pages: shiftPagesTotalCount(
        data.pages.map((page) => ({
          ...page,
          orders: page.orders.filter((order) => order.id !== orderId),
        })),
        -1,
      ),
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

  const totalDelta = loadedWorkOrderPagesAreComplete(data.pages) ? 1 : 0;
  return {
    ...data,
    pages: shiftPagesTotalCount(
      [
        {
          ...first,
          orders: [patchedWorkOrder({ id: described.id ?? orderId }, described), ...first.orders],
        },
        ...rest,
      ],
      totalDelta,
    ),
  };
}

export function pageKeysWithUnknownWorkOrderMembership(
  queryClient: QueryClient,
  organizationId: string,
  factoryId: string,
  orderId: string,
  described: FactoriesWorkOrder,
): ReadonlyArray<readonly unknown[]> {
  const keys: Array<readonly unknown[]> = [];
  for (const query of queryClient.getQueryCache().findAll({
    queryKey: factoryWorkOrdersPagePrefix(organizationId, factoryId),
  })) {
    const data = queryClient.getQueryData<InfiniteData<WorkOrdersPage>>(query.queryKey);
    if (!data || data.pages.length === 0) {
      continue;
    }
    if (loadedWorkOrderPagesAreComplete(data.pages) || orderIsOnLoadedPages(data.pages, orderId)) {
      continue;
    }
    if (
      !describedOrderBelongsOnPage(
        described,
        workOrdersPageStatesFromKey(query.queryKey),
        workOrdersPageQueryFromKey(query.queryKey),
      )
    ) {
      continue;
    }
    keys.push(query.queryKey);
  }
  return keys;
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
  const removed = data.pages.some((page) => pageContainsWorkOrder(page, orderId));
  const pages = shiftPagesTotalCount(
    data.pages.map((page) => ({
      ...page,
      orders: withoutWorkOrder(page.orders, orderId),
    })),
    removed ? -1 : 0,
  );
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
