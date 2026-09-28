import { MANUAL_FILTER_VALUE } from "./workOrderCardSource";
import { sourceFilterLabel } from "./workOrderFilterOptions";

export type BacklogColumnSort = "updated" | "confidence" | "source" | "created";
export type BacklogColumnSortDirection = "asc" | "desc";
export type BacklogColumnAge = "any" | "last_7_days" | "last_30_days" | "last_90_days" | "older_than_90_days";

export type BacklogColumnQuery = {
  sort: BacklogColumnSort;
  direction: BacklogColumnSortDirection;
  sourceGroups: string[];
  minConfidence?: number;
  confidenceMissing: boolean;
  age: BacklogColumnAge;
};

export const DEFAULT_BACKLOG_COLUMN_QUERY: BacklogColumnQuery = {
  sort: "updated",
  direction: "desc",
  sourceGroups: [],
  confidenceMissing: false,
  age: "any",
};

const SOURCE_GROUP_IDS = [
  "github-issues",
  "jira-issues",
  "sentry-exceptions",
  "pagerduty-incidents",
  "productive-tasks",
  "slack",
] as const;

const SORT_PARAM = {
  updated: "SORT_UPDATED",
  confidence: "SORT_CONFIDENCE",
  source: "SORT_SOURCE",
  created: "SORT_CREATED",
} as const;

const DIRECTION_PARAM = {
  asc: "SORT_DIRECTION_ASC",
  desc: "SORT_DIRECTION_DESC",
} as const;

const AGE_PARAM = {
  last_7_days: "AGE_LAST_7_DAYS",
  last_30_days: "AGE_LAST_30_DAYS",
  last_90_days: "AGE_LAST_90_DAYS",
  older_than_90_days: "AGE_OLDER_THAN_90_DAYS",
} as const;

export type BacklogListRequestQuery = {
  sort?: (typeof SORT_PARAM)[BacklogColumnSort];
  sortDirection?: (typeof DIRECTION_PARAM)[BacklogColumnSortDirection];
  sourceGroups?: string[];
  minConfidence?: number;
  confidenceMissing?: boolean;
  age?: (typeof AGE_PARAM)[Exclude<BacklogColumnAge, "any">];
};

export function isBacklogColumnQueryActive(query: BacklogColumnQuery | undefined): boolean {
  if (!query) {
    return false;
  }
  return (
    query.sort !== DEFAULT_BACKLOG_COLUMN_QUERY.sort ||
    query.direction !== DEFAULT_BACKLOG_COLUMN_QUERY.direction ||
    query.sourceGroups.length > 0 ||
    query.minConfidence != null ||
    query.confidenceMissing ||
    query.age !== DEFAULT_BACKLOG_COLUMN_QUERY.age
  );
}

export function backlogListRequestQuery(query: BacklogColumnQuery | undefined): BacklogListRequestQuery {
  if (!query || !isBacklogColumnQueryActive(query)) {
    return {};
  }
  const params: BacklogListRequestQuery = {};
  if (query.sort !== "updated") {
    params.sort = SORT_PARAM[query.sort];
  }
  if (query.direction !== "desc") {
    params.sortDirection = DIRECTION_PARAM[query.direction];
  }
  if (query.sourceGroups.length > 0) {
    params.sourceGroups = [...query.sourceGroups].sort();
  }
  if (query.minConfidence != null) {
    params.minConfidence = query.minConfidence;
  }
  if (query.confidenceMissing) {
    params.confidenceMissing = true;
  }
  if (query.age !== "any") {
    params.age = AGE_PARAM[query.age];
  }
  return params;
}

export function backlogColumnQueryKey(query: BacklogColumnQuery | undefined): string {
  const params = backlogListRequestQuery(query);
  return Object.keys(params).length === 0 ? "" : JSON.stringify(params);
}

export function backlogSourceFilterOptions(): Array<{ value: string; label: string }> {
  const tools = SOURCE_GROUP_IDS.map((value) => ({ value, label: sourceFilterLabel(value) })).sort((a, b) =>
    a.label.localeCompare(b.label),
  );
  return [...tools, { value: MANUAL_FILTER_VALUE, label: sourceFilterLabel(MANUAL_FILTER_VALUE) }];
}

export function backlogSortDirectionLabels(sort: BacklogColumnSort): { asc: string; desc: string } {
  if (sort === "confidence") {
    return { asc: "Lowest score first", desc: "Highest score first" };
  }
  if (sort === "source") {
    return { asc: "Alphabetical", desc: "Reverse alphabetical" };
  }
  if (sort === "created") {
    return { asc: "Oldest issue first", desc: "Newest issue first" };
  }
  return { asc: "Oldest update first", desc: "Newest update first" };
}
