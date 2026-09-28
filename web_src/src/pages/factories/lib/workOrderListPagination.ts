import type {
  FactoriesListWorkOrdersResponse,
  FactoriesWorkOrder,
  FactoriesWorkOrderResult,
  FactoriesWorkOrderState,
  FactoriesWorkOrderSummary,
} from "@/api-client";

import { CONFIDENCE_CHECK_KEY } from "./confidenceScore";
import { belongsToLineBoard } from "./linePhaseRuns";
import { workOrderListSource } from "./workOrderCardSource";
import type { WorkOrderDisplayStatus } from "./workOrderProgress";

export const BOARD_BACKLOG_PAGE_SIZE = 20;
export const BOARD_OPEN_PAGE_SIZE = 50;
export const BOARD_DONE_PAGE_SIZE = 20;
export const WORK_ORDER_LIST_PAGE_SIZE = 100;

export const BOARD_BACKLOG_STATES = ["STATE_DRAFT"] as const satisfies readonly FactoriesWorkOrderState[];
export const BOARD_OPEN_STATES = ["STATE_OPEN"] as const satisfies readonly FactoriesWorkOrderState[];
export const BOARD_DONE_STATES = ["STATE_CLOSED"] as const satisfies readonly FactoriesWorkOrderState[];
export const BOARD_DONE_RESULTS = [
  "RESULT_COMPLETED",
  "RESULT_FAILED",
] as const satisfies readonly FactoriesWorkOrderResult[];

/** Done-page results. Failed or Completed alone narrows the page so that filter can see those tasks. */
export function boardDoneResultsForStatuses(
  statuses: readonly WorkOrderDisplayStatus[],
): readonly FactoriesWorkOrderResult[] {
  const wantsCompleted = statuses.includes("completed");
  const wantsFailed = statuses.includes("failed");
  if (wantsCompleted === wantsFailed) {
    return BOARD_DONE_RESULTS;
  }
  return wantsFailed ? ["RESULT_FAILED"] : ["RESULT_COMPLETED"];
}

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
  lineId?: string;
  backlog: BacklogColumnQuery;
};

export type BacklogColumnSort = "updated" | "confidence" | "source" | "created";
export type BacklogColumnSortDirection = "asc" | "desc";
export type BacklogColumnAge = "any" | "last7" | "last30" | "last90" | "older90";

/** Column sort and filter for the backlog list. Not stored after the visit. */
export type BacklogColumnQuery = {
  sort: BacklogColumnSort;
  sortDirection: BacklogColumnSortDirection;
  sources: string[];
  minConfidence?: number;
  confidenceMissing: boolean;
  age: BacklogColumnAge;
};

export const DEFAULT_BACKLOG_COLUMN_QUERY: BacklogColumnQuery = {
  sort: "updated",
  sortDirection: "desc",
  sources: [],
  confidenceMissing: false,
  age: "any",
};

export function normalizeBacklogColumnQuery(query?: Partial<BacklogColumnQuery>): BacklogColumnQuery {
  return {
    sort: query?.sort ?? DEFAULT_BACKLOG_COLUMN_QUERY.sort,
    sortDirection: query?.sortDirection ?? DEFAULT_BACKLOG_COLUMN_QUERY.sortDirection,
    sources: [...(query?.sources ?? [])].sort(),
    minConfidence: query?.confidenceMissing ? undefined : query?.minConfidence,
    confidenceMissing: Boolean(query?.confidenceMissing),
    age: query?.age ?? DEFAULT_BACKLOG_COLUMN_QUERY.age,
  };
}

export function isDefaultBacklogColumnQuery(query: BacklogColumnQuery): boolean {
  return (
    query.sort === DEFAULT_BACKLOG_COLUMN_QUERY.sort &&
    query.sortDirection === DEFAULT_BACKLOG_COLUMN_QUERY.sortDirection &&
    query.sources.length === 0 &&
    query.minConfidence == null &&
    !query.confidenceMissing &&
    query.age === DEFAULT_BACKLOG_COLUMN_QUERY.age
  );
}

export function backlogColumnQueryKey(query: BacklogColumnQuery): string {
  if (isDefaultBacklogColumnQuery(query)) {
    return "";
  }
  return [
    query.sort,
    query.sortDirection,
    query.sources.join(","),
    query.minConfidence ?? "",
    query.confidenceMissing ? "missing" : "",
    query.age,
  ].join("|");
}

export function normalizeWorkOrdersPageQuery(query?: Partial<WorkOrdersPageQuery>): WorkOrdersPageQuery {
  return {
    userId: query?.userId,
    unassigned: Boolean(query?.unassigned),
    results: [...(query?.results ?? [])].sort(),
    lineId: query?.lineId,
    backlog: normalizeBacklogColumnQuery(query?.backlog),
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
    normalized.lineId ?? "",
    backlogColumnQueryKey(normalized.backlog),
  ] as const;
}

export function workOrdersPageQueryFromKey(queryKey: readonly unknown[]): WorkOrdersPageQuery {
  const userId = queryKey[5];
  const resultsJoined = queryKey[7];
  const lineId = queryKey[8];
  return normalizeWorkOrdersPageQuery({
    userId: typeof userId === "string" && userId.length > 0 ? userId : undefined,
    unassigned: queryKey[6] === "unassigned",
    results:
      typeof resultsJoined === "string" && resultsJoined.length > 0
        ? (resultsJoined.split(",") as FactoriesWorkOrderResult[])
        : [],
    lineId: typeof lineId === "string" && lineId.length > 0 ? lineId : undefined,
    backlog: backlogColumnQueryFromKey(queryKey[9]),
  });
}

function backlogColumnQueryFromKey(value: unknown): BacklogColumnQuery {
  if (typeof value !== "string" || value.length === 0) {
    return DEFAULT_BACKLOG_COLUMN_QUERY;
  }
  const [sort, sortDirection, sources, minConfidence, confidenceMissing, age] = value.split("|");
  return normalizeBacklogColumnQuery({
    sort: sort as BacklogColumnSort,
    sortDirection: sortDirection as BacklogColumnSortDirection,
    sources: sources ? sources.split(",") : [],
    minConfidence: minConfidence ? Number(minConfidence) : undefined,
    confidenceMissing: confidenceMissing === "missing",
    age: age as BacklogColumnAge,
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
  order: Pick<FactoriesWorkOrderSummary, "assignees" | "createdBy" | "lineDispatches"> &
    Partial<Pick<FactoriesWorkOrderSummary, "origin" | "createdAt" | "checkScores">>,
  query: WorkOrdersPageQuery,
): boolean {
  if (query.lineId && !belongsToLineBoard(order, query.lineId)) {
    return false;
  }
  if (query.backlog && !workOrderMatchesBacklogQuery(order, query.backlog)) {
    return false;
  }
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

const BACKLOG_SORT_API = {
  updated: "SORT_UPDATED",
  confidence: "SORT_CONFIDENCE",
  source: "SORT_SOURCE",
  created: "SORT_CREATED",
} as const;

const BACKLOG_DIRECTION_API = {
  asc: "SORT_DIRECTION_ASC",
  desc: "SORT_DIRECTION_DESC",
} as const;

const BACKLOG_AGE_API = {
  last7: "AGE_LAST_7_DAYS",
  last30: "AGE_LAST_30_DAYS",
  last90: "AGE_LAST_90_DAYS",
  older90: "AGE_OLDER_THAN_90_DAYS",
} as const;

export type BacklogListQuery = {
  sort?: (typeof BACKLOG_SORT_API)[BacklogColumnSort];
  sortDirection?: (typeof BACKLOG_DIRECTION_API)[BacklogColumnSortDirection];
  sources?: string[];
  minConfidence?: number;
  confidenceMissing?: boolean;
  age?: (typeof BACKLOG_AGE_API)[Exclude<BacklogColumnAge, "any">];
};

/** Query fields for one backlog page. Omitted when the column still uses the default order. */
export function backlogListQuery(query: BacklogColumnQuery | undefined): BacklogListQuery {
  const normalized = normalizeBacklogColumnQuery(query);
  if (isDefaultBacklogColumnQuery(normalized)) {
    return {};
  }
  return {
    sort: BACKLOG_SORT_API[normalized.sort],
    sortDirection: BACKLOG_DIRECTION_API[normalized.sortDirection],
    ...(normalized.sources.length > 0 ? { sources: normalized.sources } : {}),
    ...(normalized.minConfidence != null ? { minConfidence: normalized.minConfidence } : {}),
    ...(normalized.confidenceMissing ? { confidenceMissing: true } : {}),
    ...(normalized.age !== "any" ? { age: BACKLOG_AGE_API[normalized.age] } : {}),
  };
}

function workOrderMatchesBacklogQuery(
  order: Partial<Pick<FactoriesWorkOrderSummary, "origin" | "createdAt" | "createdBy" | "checkScores">>,
  query: BacklogColumnQuery,
): boolean {
  const normalized = normalizeBacklogColumnQuery(query);
  if (isDefaultBacklogColumnQuery(normalized)) {
    return true;
  }
  if (
    normalized.sources.length > 0 &&
    !normalized.sources.includes(workOrderListSource(order as FactoriesWorkOrder).id)
  ) {
    return false;
  }
  const score = confidenceScore(order);
  if (normalized.confidenceMissing) {
    return score == null;
  }
  if (normalized.minConfidence != null && (score == null || score < normalized.minConfidence)) {
    return false;
  }
  return matchesBacklogAge(order.createdAt, normalized.age);
}

function confidenceScore(
  order: Partial<Pick<FactoriesWorkOrderSummary, "checkScores">>,
): number | undefined {
  const score = order.checkScores?.find((check) => check.key === CONFIDENCE_CHECK_KEY)?.score;
  return score == null ? undefined : score;
}

function matchesBacklogAge(createdAt: string | undefined, age: BacklogColumnAge): boolean {
  if (age === "any") {
    return true;
  }
  const createdAtMs = Date.parse(createdAt ?? "");
  if (!createdAtMs) {
    return false;
  }
  const ageMs = Date.now() - createdAtMs;
  const dayMs = 24 * 60 * 60 * 1000;
  if (age === "older90") {
    return ageMs > 90 * dayMs;
  }
  const days = age === "last7" ? 7 : age === "last30" ? 30 : 90;
  return ageMs <= days * dayMs;
}
