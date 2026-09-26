import type {
  FactoriesListWorkOrdersResponse,
  FactoriesWorkOrderResult,
  FactoriesWorkOrderState,
  FactoriesWorkOrderSummary,
} from "@/api-client";

export const BOARD_BACKLOG_PAGE_SIZE = 20;
export const BOARD_OPEN_PAGE_SIZE = 50;
export const BOARD_DONE_PAGE_SIZE = 20;
export const WORK_ORDER_LIST_PAGE_SIZE = 100;

export const BOARD_BACKLOG_STATES = ["STATE_DRAFT"] as const satisfies readonly FactoriesWorkOrderState[];
export const BOARD_OPEN_STATES = ["STATE_OPEN"] as const satisfies readonly FactoriesWorkOrderState[];
export const BOARD_DONE_STATES = ["STATE_CLOSED"] as const satisfies readonly FactoriesWorkOrderState[];
export const BOARD_DONE_RESULTS = ["RESULT_COMPLETED"] as const satisfies readonly FactoriesWorkOrderResult[];

export type WorkOrdersPageCursor = {
  beforeId: string;
};

export type WorkOrdersPage = {
  orders: FactoriesWorkOrderSummary[];
  hasNextPage: boolean;
};

export function uniqueWorkOrdersById<T extends { id?: string }>(orders: T[]): T[] {
  const seen = new Set<string>();
  const unique: T[] = [];
  for (const order of orders) {
    const id = order.id;
    if (id) {
      if (seen.has(id)) {
        continue;
      }
      seen.add(id);
    }
    unique.push(order);
  }
  return unique;
}

export function flattenWorkOrdersPages(
  pages: Array<WorkOrdersPage | undefined> | undefined,
): FactoriesWorkOrderSummary[] {
  return uniqueWorkOrdersById(pages?.flatMap((page) => page?.orders ?? []) ?? []);
}

export function getWorkOrdersNextPageParam(lastPage: WorkOrdersPage | undefined): WorkOrdersPageCursor | undefined {
  if (!lastPage?.hasNextPage) {
    return undefined;
  }
  const last = lastPage.orders.at(-1);
  if (!last?.id) {
    return undefined;
  }
  return { beforeId: last.id };
}

export function workOrdersPageFromResponse(response: FactoriesListWorkOrdersResponse | undefined): WorkOrdersPage {
  return {
    orders: response?.orders ?? [],
    hasNextPage: Boolean(response?.hasNextPage),
  };
}

export function factoryWorkOrdersPagePrefix(organizationId: string, factoryId: string) {
  return ["factories", organizationId, factoryId, "work-orders-page"] as const;
}

/** Server-side filters for one board /orders page query. */
export type WorkOrdersPageQuery = {
  userId?: string;
  unassigned: boolean;
  results: readonly FactoriesWorkOrderResult[];
};

export function normalizeWorkOrdersPageQuery(query?: Partial<WorkOrdersPageQuery>): WorkOrdersPageQuery {
  return {
    userId: query?.userId,
    unassigned: Boolean(query?.unassigned),
    results: [...(query?.results ?? [])].sort(),
  };
}

export function factoryWorkOrdersPageKey(
  organizationId: string,
  factoryId: string,
  states: readonly FactoriesWorkOrderState[],
  query?: Partial<WorkOrdersPageQuery>,
) {
  const normalized = normalizeWorkOrdersPageQuery(query);
  return [
    ...factoryWorkOrdersPagePrefix(organizationId, factoryId),
    [...states].sort().join(","),
    normalized.userId ?? "",
    normalized.unassigned ? "unassigned" : "",
    normalized.results.join(","),
  ] as const;
}

export function workOrdersPageQueryFromKey(queryKey: readonly unknown[]): WorkOrdersPageQuery {
  const userId = queryKey[5];
  const resultsJoined = queryKey[7];
  return normalizeWorkOrdersPageQuery({
    userId: typeof userId === "string" && userId.length > 0 ? userId : undefined,
    unassigned: queryKey[6] === "unassigned",
    results:
      typeof resultsJoined === "string" && resultsJoined.length > 0
        ? (resultsJoined.split(",") as FactoriesWorkOrderResult[])
        : [],
  });
}

export function workOrderMatchesUser(
  order: Pick<FactoriesWorkOrderSummary, "assignees" | "createdBy">,
  userId: string,
): boolean {
  if (order.createdBy?.user?.id === userId) {
    return true;
  }
  return (order.assignees ?? []).some((assignee) => assignee.id === userId);
}

export function workOrderMatchesPageQuery(
  order: Pick<FactoriesWorkOrderSummary, "assignees" | "createdBy">,
  query: WorkOrdersPageQuery,
): boolean {
  const isUnassigned = (order.assignees ?? []).every((assignee) => !assignee.id);
  if (query.userId && query.unassigned) {
    return isUnassigned || workOrderMatchesUser(order, query.userId);
  }
  if (query.unassigned) {
    return isUnassigned;
  }
  if (query.userId) {
    return workOrderMatchesUser(order, query.userId);
  }
  return true;
}
