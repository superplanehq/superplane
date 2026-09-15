import { useCallback, useEffect, useState } from "react";

import type { FactoriesWorkOrder } from "@/api-client";

import { confidenceScoreFromChecks } from "./confidenceScore";

export const LINE_COLUMN_SORT_STORAGE_KEY = "factories-line-column-sorts";

export type LineColumnKey = "backlog" | `phase-${number}` | "verify" | "done";

export const LINE_COLUMN_SORT_IDS = ["updated", "created", "completed", "result", "confidence"] as const;

export type LineColumnSortId = (typeof LINE_COLUMN_SORT_IDS)[number];

export const DEFAULT_LINE_COLUMN_SORT: LineColumnSortId = "updated";

export const LINE_COLUMN_SORT_LABELS: Record<LineColumnSortId, string> = {
  updated: "Newest activity",
  created: "Created time",
  completed: "Completed time",
  result: "Result",
  confidence: "Confidence score",
};

export const LINE_COLUMN_SORTS = {
  backlog: ["updated", "created", "confidence"],
  phase: ["updated", "created"],
  verify: ["updated", "created"],
  done: ["updated", "created", "completed", "result"],
} as const satisfies Record<string, readonly LineColumnSortId[]>;

const DONE_RESULT_RANK: Record<string, number> = {
  RESULT_FAILED: 0,
  RESULT_REJECTED: 1,
  RESULT_COMPLETED: 2,
};

export type LineSortableRun = {
  executionId: string;
  order: Pick<FactoriesWorkOrder, "id" | "createdAt">;
  execution: { createdAt?: string; updatedAt?: string };
};

type StoredLineSorts = Record<string, Record<string, string>>;

export function columnSortKind(columnKey: LineColumnKey): keyof typeof LINE_COLUMN_SORTS {
  if (columnKey === "backlog" || columnKey === "verify" || columnKey === "done") {
    return columnKey;
  }
  return "phase";
}

export function allowedSortsForColumn(columnKey: LineColumnKey): readonly LineColumnSortId[] {
  return LINE_COLUMN_SORTS[columnSortKind(columnKey)];
}

export function isLineColumnSortId(value: unknown): value is LineColumnSortId {
  return typeof value === "string" && (LINE_COLUMN_SORT_IDS as readonly string[]).includes(value);
}

export function resolveLineColumnSort(columnKey: LineColumnKey, value: unknown): LineColumnSortId {
  if (!isLineColumnSortId(value)) {
    return DEFAULT_LINE_COLUMN_SORT;
  }
  return (allowedSortsForColumn(columnKey) as readonly string[]).includes(value) ? value : DEFAULT_LINE_COLUMN_SORT;
}

export function compareOrdersNewestActivity(left: FactoriesWorkOrder, right: FactoriesWorkOrder): number {
  return compareNewestFirst(left.updatedAt ?? left.createdAt, right.updatedAt ?? right.createdAt, left.id, right.id);
}

export function compareLineOrders(
  sort: LineColumnSortId,
  left: FactoriesWorkOrder,
  right: FactoriesWorkOrder,
  confidenceByOrderId?: ReadonlyMap<string, number | undefined>,
): number {
  switch (sort) {
    case "created":
      return compareNewestFirst(left.createdAt, right.createdAt, left.id, right.id);
    case "completed":
      return compareNewestFirst(left.updatedAt, right.updatedAt, left.id, right.id);
    case "result":
      return compareOrdersByResult(left, right);
    case "confidence":
      return compareOrdersByConfidence(left, right, confidenceByOrderId);
    case "updated":
    default:
      return compareOrdersNewestActivity(left, right);
  }
}

export function compareLinePhaseRuns(sort: LineColumnSortId, left: LineSortableRun, right: LineSortableRun): number {
  if (sort === "created") {
    return compareNewestFirst(
      left.order.createdAt,
      right.order.createdAt,
      left.order.id ?? left.executionId,
      right.order.id ?? right.executionId,
    );
  }
  return compareNewestFirst(
    left.execution.updatedAt ?? left.execution.createdAt,
    right.execution.updatedAt ?? right.execution.createdAt,
    left.executionId,
    right.executionId,
  );
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

export function readStoredLineColumnSorts(lineId: string | undefined): Record<string, LineColumnSortId> {
  if (!lineId) {
    return {};
  }
  const stored = readStoredSortMap()[lineId];
  if (!stored || typeof stored !== "object" || Array.isArray(stored)) {
    return {};
  }
  const resolved: Record<string, LineColumnSortId> = {};
  for (const [columnKey, value] of Object.entries(stored)) {
    const sort = resolveLineColumnSort(columnKey as LineColumnKey, value);
    if (sort !== DEFAULT_LINE_COLUMN_SORT) {
      resolved[columnKey] = sort;
    }
  }
  return resolved;
}

export function useLineColumnSortPreference(lineId: string | undefined): {
  sortFor: (columnKey: LineColumnKey) => LineColumnSortId;
  setSort: (columnKey: LineColumnKey, sortId: LineColumnSortId) => void;
} {
  const [sorts, setSorts] = useState<Record<string, LineColumnSortId>>(() => readStoredLineColumnSorts(lineId));

  useEffect(() => {
    setSorts(readStoredLineColumnSorts(lineId));
  }, [lineId]);

  const sortFor = useCallback(
    (columnKey: LineColumnKey): LineColumnSortId => resolveLineColumnSort(columnKey, sorts[columnKey]),
    [sorts],
  );

  const setSort = useCallback(
    (columnKey: LineColumnKey, sortId: LineColumnSortId) => {
      const nextSort = resolveLineColumnSort(columnKey, sortId);
      setSorts((current) => {
        const next = { ...current };
        if (nextSort === DEFAULT_LINE_COLUMN_SORT) {
          delete next[columnKey];
        } else {
          next[columnKey] = nextSort;
        }
        persistLineColumnSorts(lineId, next);
        return next;
      });
    },
    [lineId],
  );

  return { sortFor, setSort };
}

function persistLineColumnSorts(lineId: string | undefined, sorts: Record<string, LineColumnSortId>): void {
  if (!lineId) {
    return;
  }
  try {
    const all = readStoredSortMap();
    if (Object.keys(sorts).length === 0) {
      delete all[lineId];
    } else {
      all[lineId] = sorts;
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

function readStoredSortMap(): StoredLineSorts {
  try {
    const raw = window.localStorage.getItem(LINE_COLUMN_SORT_STORAGE_KEY);
    if (!raw) {
      return {};
    }
    const parsed: unknown = JSON.parse(raw);
    if (!parsed || typeof parsed !== "object" || Array.isArray(parsed)) {
      return {};
    }
    return parsed as StoredLineSorts;
  } catch {
    return {};
  }
}

function compareOrdersByResult(left: FactoriesWorkOrder, right: FactoriesWorkOrder): number {
  const leftRank = DONE_RESULT_RANK[left.result ?? ""] ?? Number.MAX_SAFE_INTEGER;
  const rightRank = DONE_RESULT_RANK[right.result ?? ""] ?? Number.MAX_SAFE_INTEGER;
  if (leftRank !== rightRank) {
    return leftRank - rightRank;
  }
  return compareIds(left.id, right.id);
}

function compareOrdersByConfidence(
  left: FactoriesWorkOrder,
  right: FactoriesWorkOrder,
  confidenceByOrderId?: ReadonlyMap<string, number | undefined>,
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
    return rightScore - leftScore;
  }
  return compareIds(left.id, right.id);
}

function compareNewestFirst(
  leftTime: string | undefined,
  rightTime: string | undefined,
  leftId: string | undefined,
  rightId: string | undefined,
): number {
  const leftMs = Date.parse(leftTime ?? "") || 0;
  const rightMs = Date.parse(rightTime ?? "") || 0;
  if (leftMs !== rightMs) {
    return rightMs - leftMs;
  }
  return compareIds(leftId, rightId);
}

function compareIds(left: string | undefined, right: string | undefined): number {
  return (right ?? "").localeCompare(left ?? "");
}
