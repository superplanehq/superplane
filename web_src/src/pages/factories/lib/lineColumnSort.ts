import { useCallback, useEffect, useState } from "react";

import type { FactoriesWorkOrder } from "@/api-client";

import { confidenceScoreFromChecks } from "./confidenceScore";

export const LINE_COLUMN_SORT_STORAGE_KEY = "factories-line-column-sorts";

export type LineColumnKey = "backlog" | `phase-${number}` | "verify" | "done";

export const LINE_COLUMN_SORT_IDS = ["updated", "created", "completed", "result", "confidence"] as const;

export type LineColumnSortId = (typeof LINE_COLUMN_SORT_IDS)[number];

export const LINE_COLUMN_SORT_DIRECTIONS = ["desc", "asc"] as const;

export type LineColumnSortDirection = (typeof LINE_COLUMN_SORT_DIRECTIONS)[number];

export const LINE_COLUMN_FILTER_IDS = ["all", "completed", "failed", "rejected"] as const;

export type LineColumnFilterId = (typeof LINE_COLUMN_FILTER_IDS)[number];

export const DEFAULT_LINE_COLUMN_SORT: LineColumnSortId = "updated";

export const DEFAULT_LINE_COLUMN_SORT_DIRECTION: LineColumnSortDirection = "desc";

export const DEFAULT_LINE_COLUMN_FILTER: LineColumnFilterId = "all";

export type LineColumnView = {
  sort: LineColumnSortId;
  direction: LineColumnSortDirection;
  filter: LineColumnFilterId;
};

export const DEFAULT_LINE_COLUMN_VIEW: LineColumnView = {
  sort: DEFAULT_LINE_COLUMN_SORT,
  direction: DEFAULT_LINE_COLUMN_SORT_DIRECTION,
  filter: DEFAULT_LINE_COLUMN_FILTER,
};

export const LINE_COLUMN_SORT_LABELS: Record<LineColumnSortId, string> = {
  updated: "Newest activity",
  created: "Created time",
  completed: "Completed time",
  result: "Result",
  confidence: "Confidence score",
};

export const LINE_COLUMN_FILTER_LABELS: Record<LineColumnFilterId, string> = {
  all: "All results",
  completed: "Completed",
  failed: "Failed",
  rejected: "Rejected",
};

export const LINE_COLUMN_SORTS = {
  backlog: ["updated", "created", "confidence"],
  phase: ["updated", "created"],
  verify: ["updated", "created"],
  done: ["updated", "created", "completed", "result"],
} as const satisfies Record<string, readonly LineColumnSortId[]>;

export const LINE_COLUMN_FILTERS = {
  backlog: ["all"],
  phase: ["all"],
  verify: ["all"],
  done: ["all", "completed", "failed", "rejected"],
} as const satisfies Record<string, readonly LineColumnFilterId[]>;

const DONE_RESULT_RANK: Record<string, number> = {
  RESULT_FAILED: 0,
  RESULT_REJECTED: 1,
  RESULT_COMPLETED: 2,
};

const DONE_FILTER_RESULT: Record<Exclude<LineColumnFilterId, "all">, string> = {
  completed: "RESULT_COMPLETED",
  failed: "RESULT_FAILED",
  rejected: "RESULT_REJECTED",
};

export type LineSortableRun = {
  executionId: string;
  order: Pick<FactoriesWorkOrder, "id" | "createdAt">;
  execution: { createdAt?: string; updatedAt?: string };
};

export type LinePhaseColumnSort = {
  sort: LineColumnSortId;
  direction: LineColumnSortDirection;
};

type StoredColumnPref = {
  sort?: string;
  direction?: string;
  filter?: string;
};

type StoredLinePrefs = Record<string, Record<string, string | StoredColumnPref>>;

export function columnSortKind(columnKey: LineColumnKey): keyof typeof LINE_COLUMN_SORTS {
  if (columnKey === "backlog" || columnKey === "verify" || columnKey === "done") {
    return columnKey;
  }
  return "phase";
}

export function allowedSortsForColumn(columnKey: LineColumnKey): readonly LineColumnSortId[] {
  return LINE_COLUMN_SORTS[columnSortKind(columnKey)];
}

export function allowedFiltersForColumn(columnKey: LineColumnKey): readonly LineColumnFilterId[] {
  return LINE_COLUMN_FILTERS[columnSortKind(columnKey)];
}

export function isLineColumnSortId(value: unknown): value is LineColumnSortId {
  return typeof value === "string" && (LINE_COLUMN_SORT_IDS as readonly string[]).includes(value);
}

export function isLineColumnSortDirection(value: unknown): value is LineColumnSortDirection {
  return typeof value === "string" && (LINE_COLUMN_SORT_DIRECTIONS as readonly string[]).includes(value);
}

export function isLineColumnFilterId(value: unknown): value is LineColumnFilterId {
  return typeof value === "string" && (LINE_COLUMN_FILTER_IDS as readonly string[]).includes(value);
}

export function resolveLineColumnSort(columnKey: LineColumnKey, value: unknown): LineColumnSortId {
  if (!isLineColumnSortId(value)) {
    return DEFAULT_LINE_COLUMN_SORT;
  }
  return (allowedSortsForColumn(columnKey) as readonly string[]).includes(value) ? value : DEFAULT_LINE_COLUMN_SORT;
}

export function resolveLineColumnSortDirection(value: unknown): LineColumnSortDirection {
  return isLineColumnSortDirection(value) ? value : DEFAULT_LINE_COLUMN_SORT_DIRECTION;
}

export function resolveLineColumnFilter(columnKey: LineColumnKey, value: unknown): LineColumnFilterId {
  if (!isLineColumnFilterId(value)) {
    return DEFAULT_LINE_COLUMN_FILTER;
  }
  return (allowedFiltersForColumn(columnKey) as readonly string[]).includes(value) ? value : DEFAULT_LINE_COLUMN_FILTER;
}

export function resolveLineColumnView(columnKey: LineColumnKey, value: unknown): LineColumnView {
  const pref = storedPrefFromValue(value);
  return {
    sort: resolveLineColumnSort(columnKey, pref.sort),
    direction: resolveLineColumnSortDirection(pref.direction),
    filter: resolveLineColumnFilter(columnKey, pref.filter),
  };
}

export function lineColumnSortDirectionLabels(sort: LineColumnSortId): {
  desc: string;
  asc: string;
} {
  switch (sort) {
    case "confidence":
      return { desc: "High to low", asc: "Low to high" };
    case "result":
      return { desc: "Failed first", asc: "Completed first" };
    default:
      return { desc: "Newest first", asc: "Oldest first" };
  }
}

export function compareOrdersNewestActivity(left: FactoriesWorkOrder, right: FactoriesWorkOrder): number {
  return compareByTime(left.updatedAt ?? left.createdAt, right.updatedAt ?? right.createdAt, left.id, right.id, "desc");
}

export function compareLineOrders(
  sort: LineColumnSortId,
  left: FactoriesWorkOrder,
  right: FactoriesWorkOrder,
  confidenceByOrderId?: ReadonlyMap<string, number | undefined>,
  direction: LineColumnSortDirection = DEFAULT_LINE_COLUMN_SORT_DIRECTION,
): number {
  switch (sort) {
    case "created":
      return compareByTime(left.createdAt, right.createdAt, left.id, right.id, direction);
    case "completed":
      return compareByTime(left.updatedAt, right.updatedAt, left.id, right.id, direction);
    case "result":
      return compareOrdersByResult(left, right, direction);
    case "confidence":
      return compareOrdersByConfidence(left, right, confidenceByOrderId, direction);
    case "updated":
    default:
      return compareByTime(
        left.updatedAt ?? left.createdAt,
        right.updatedAt ?? right.createdAt,
        left.id,
        right.id,
        direction,
      );
  }
}

export function compareLinePhaseRuns(
  sort: LineColumnSortId,
  left: LineSortableRun,
  right: LineSortableRun,
  direction: LineColumnSortDirection = DEFAULT_LINE_COLUMN_SORT_DIRECTION,
): number {
  if (sort === "created") {
    return compareByTime(
      left.order.createdAt,
      right.order.createdAt,
      left.order.id ?? left.executionId,
      right.order.id ?? right.executionId,
      direction,
    );
  }
  return compareByTime(
    left.execution.updatedAt ?? left.execution.createdAt,
    right.execution.updatedAt ?? right.execution.createdAt,
    left.executionId,
    right.executionId,
    direction,
  );
}

export function filterLineColumnOrders(
  orders: readonly FactoriesWorkOrder[],
  filter: LineColumnFilterId,
): FactoriesWorkOrder[] {
  if (filter === "all") {
    return [...orders];
  }
  const result = DONE_FILTER_RESULT[filter];
  return orders.filter((order) => order.result === result);
}

export function confidenceScoresFromCheckQueries(
  orderIds: readonly string[],
  queries: ReadonlyArray<{ data?: Array<{ name?: string; score?: number }>; isPending: boolean }>,
): ReadonlyMap<string, number | undefined> | undefined {
  if (queries.length !== orderIds.length) {
    return undefined;
  }
  if (queries.some((query) => query.isPending)) {
    return undefined;
  }
  const scores = new Map<string, number | undefined>();
  orderIds.forEach((orderId, index) => {
    scores.set(orderId, confidenceScoreFromChecks(queries[index]?.data));
  });
  return scores;
}

export function readStoredLineColumnSorts(lineId: string | undefined): Record<string, LineColumnView> {
  if (!lineId) {
    return {};
  }
  const stored = readStoredPrefMap()[lineId];
  if (!stored || typeof stored !== "object" || Array.isArray(stored)) {
    return {};
  }
  const resolved: Record<string, LineColumnView> = {};
  for (const [columnKey, value] of Object.entries(stored)) {
    const view = resolveLineColumnView(columnKey as LineColumnKey, value);
    if (!isDefaultView(view)) {
      resolved[columnKey] = view;
    }
  }
  return resolved;
}

export function useLineColumnSortPreference(lineId: string | undefined): {
  viewFor: (columnKey: LineColumnKey) => LineColumnView;
  sortFor: (columnKey: LineColumnKey) => LineColumnSortId;
  setSort: (columnKey: LineColumnKey, sortId: LineColumnSortId) => void;
  setDirection: (columnKey: LineColumnKey, direction: LineColumnSortDirection) => void;
  setFilter: (columnKey: LineColumnKey, filter: LineColumnFilterId) => void;
} {
  const [views, setViews] = useState<Record<string, LineColumnView>>(() => readStoredLineColumnSorts(lineId));

  useEffect(() => {
    setViews(readStoredLineColumnSorts(lineId));
  }, [lineId]);

  const viewFor = useCallback(
    (columnKey: LineColumnKey): LineColumnView => resolveLineColumnView(columnKey, views[columnKey]),
    [views],
  );

  const sortFor = useCallback((columnKey: LineColumnKey): LineColumnSortId => viewFor(columnKey).sort, [viewFor]);

  const patchView = useCallback(
    (columnKey: LineColumnKey, patch: Partial<LineColumnView>) => {
      setViews((current) => {
        const nextView = resolveLineColumnView(columnKey, { ...current[columnKey], ...patch });
        const next = { ...current };
        if (isDefaultView(nextView)) {
          delete next[columnKey];
        } else {
          next[columnKey] = nextView;
        }
        persistLineColumnSorts(lineId, next);
        return next;
      });
    },
    [lineId],
  );

  const setSort = useCallback(
    (columnKey: LineColumnKey, sortId: LineColumnSortId) => {
      patchView(columnKey, { sort: resolveLineColumnSort(columnKey, sortId) });
    },
    [patchView],
  );

  const setDirection = useCallback(
    (columnKey: LineColumnKey, direction: LineColumnSortDirection) => {
      patchView(columnKey, { direction: resolveLineColumnSortDirection(direction) });
    },
    [patchView],
  );

  const setFilter = useCallback(
    (columnKey: LineColumnKey, filter: LineColumnFilterId) => {
      patchView(columnKey, { filter: resolveLineColumnFilter(columnKey, filter) });
    },
    [patchView],
  );

  return { viewFor, sortFor, setSort, setDirection, setFilter };
}

function persistLineColumnSorts(lineId: string | undefined, views: Record<string, LineColumnView>): void {
  if (!lineId) {
    return;
  }
  try {
    const all = readStoredPrefMap();
    const stored = storedPrefsFromViews(views);
    if (Object.keys(stored).length === 0) {
      delete all[lineId];
    } else {
      all[lineId] = stored;
    }
    if (Object.keys(all).length === 0) {
      window.localStorage.removeItem(LINE_COLUMN_SORT_STORAGE_KEY);
      return;
    }
    window.localStorage.setItem(LINE_COLUMN_SORT_STORAGE_KEY, JSON.stringify(all));
  } catch {
    // Ignore storage failures (private mode, quota, etc.).
  }
}

function readStoredPrefMap(): StoredLinePrefs {
  try {
    const raw = window.localStorage.getItem(LINE_COLUMN_SORT_STORAGE_KEY);
    if (!raw) {
      return {};
    }
    const parsed: unknown = JSON.parse(raw);
    if (!parsed || typeof parsed !== "object" || Array.isArray(parsed)) {
      return {};
    }
    return parsed as StoredLinePrefs;
  } catch {
    return {};
  }
}

function storedPrefFromValue(value: unknown): StoredColumnPref {
  if (typeof value === "string") {
    return { sort: value };
  }
  if (!value || typeof value !== "object" || Array.isArray(value)) {
    return {};
  }
  const pref = value as StoredColumnPref;
  return {
    sort: typeof pref.sort === "string" ? pref.sort : undefined,
    direction: typeof pref.direction === "string" ? pref.direction : undefined,
    filter: typeof pref.filter === "string" ? pref.filter : undefined,
  };
}

function storedPrefsFromViews(views: Record<string, LineColumnView>): Record<string, StoredColumnPref> {
  const stored: Record<string, StoredColumnPref> = {};
  for (const [columnKey, view] of Object.entries(views)) {
    const pref: StoredColumnPref = {};
    if (view.sort !== DEFAULT_LINE_COLUMN_SORT) {
      pref.sort = view.sort;
    }
    if (view.direction !== DEFAULT_LINE_COLUMN_SORT_DIRECTION) {
      pref.direction = view.direction;
    }
    if (view.filter !== DEFAULT_LINE_COLUMN_FILTER) {
      pref.filter = view.filter;
    }
    if (Object.keys(pref).length > 0) {
      stored[columnKey] = pref;
    }
  }
  return stored;
}

function isDefaultView(view: LineColumnView): boolean {
  return (
    view.sort === DEFAULT_LINE_COLUMN_SORT &&
    view.direction === DEFAULT_LINE_COLUMN_SORT_DIRECTION &&
    view.filter === DEFAULT_LINE_COLUMN_FILTER
  );
}

function compareOrdersByResult(
  left: FactoriesWorkOrder,
  right: FactoriesWorkOrder,
  direction: LineColumnSortDirection,
): number {
  const leftRank = DONE_RESULT_RANK[left.result ?? ""] ?? Number.MAX_SAFE_INTEGER;
  const rightRank = DONE_RESULT_RANK[right.result ?? ""] ?? Number.MAX_SAFE_INTEGER;
  if (leftRank !== rightRank) {
    return direction === "asc" ? rightRank - leftRank : leftRank - rightRank;
  }
  return compareIds(left.id, right.id);
}

function compareOrdersByConfidence(
  left: FactoriesWorkOrder,
  right: FactoriesWorkOrder,
  confidenceByOrderId: ReadonlyMap<string, number | undefined> | undefined,
  direction: LineColumnSortDirection,
): number {
  const leftScore = left.id ? confidenceByOrderId?.get(left.id) : undefined;
  const rightScore = right.id ? confidenceByOrderId?.get(right.id) : undefined;
  const leftMissing = leftScore == null;
  const rightMissing = rightScore == null;
  if (leftMissing && rightMissing) {
    return compareIds(left.id, right.id);
  }
  if (leftMissing) {
    return 1;
  }
  if (rightMissing) {
    return -1;
  }
  if (leftScore !== rightScore) {
    return direction === "asc" ? leftScore - rightScore : rightScore - leftScore;
  }
  return compareIds(left.id, right.id);
}

function compareByTime(
  leftTime: string | undefined,
  rightTime: string | undefined,
  leftId: string | undefined,
  rightId: string | undefined,
  direction: LineColumnSortDirection,
): number {
  const leftMs = Date.parse(leftTime ?? "") || 0;
  const rightMs = Date.parse(rightTime ?? "") || 0;
  if (leftMs !== rightMs) {
    return direction === "asc" ? leftMs - rightMs : rightMs - leftMs;
  }
  return compareIds(leftId, rightId);
}

function compareIds(left: string | undefined, right: string | undefined): number {
  return (right ?? "").localeCompare(left ?? "");
}
