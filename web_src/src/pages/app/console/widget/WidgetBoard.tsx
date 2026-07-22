import { useMemo } from "react";
import { Kanban, Loader2 } from "lucide-react";

import { cn } from "@/lib/utils";

import { normalizeBoardLaneValue } from "../boardPanelContent";
import { useConsoleContext, resolveConsoleNode } from "../ConsoleContext";
import { isManualRunNode } from "../manualRunTriggers";
import { WidgetEmptyState } from "../WidgetEmptyState";
import { applyTableWhere } from "./evalTableWhere";
import { laneStyleFor } from "./boardLaneStyles";
import { resolveCellValue } from "./resolveCellValue";
import { evaluateRowShow } from "./rowVisibility";
import { applySort } from "./widgetData";
import { WidgetBoardCardField } from "./WidgetBoardCardField";
import { WidgetRowActionButton } from "./WidgetRowActionButton";
import { rowKeyForRow } from "./rowKey";
import { WidgetLoadMoreFooter } from "./WidgetTable";
import { WidgetTableActionLockProvider } from "./WidgetTableActionLock";
import type { WidgetBoardLane, WidgetBoardRender, WidgetRowAction } from "./types";

interface WidgetBoardProps {
  render: WidgetBoardRender;
  rows: unknown[];
  isLoading: boolean;
  hasMore?: boolean;
  isFetchingMore?: boolean;
  onLoadMore?: () => void;
  displayCount?: number;
}

interface LaneBucket {
  lane: WidgetBoardLane;
  rows: Record<string, unknown>[];
  /**
   * Stable identifier for the lane. Configured lanes use `lane:<value>` so
   * two lanes sharing a display label don't collide; the trailing "Other"
   * lane always uses the sentinel below.
   */
  key: string;
}

/** Stable react key + data attribute value for the trailing "Other" lane. */
const OTHER_LANE_KEY = "__other__";

export function WidgetBoard({
  render,
  rows,
  isLoading,
  hasMore,
  isFetchingMore,
  onLoadMore,
  displayCount,
}: WidgetBoardProps) {
  const ctx = useConsoleContext();

  const recordRows = useMemo(
    () => rows.filter((r): r is Record<string, unknown> => Boolean(r) && typeof r === "object" && !Array.isArray(r)),
    [rows],
  );

  const filteredAll = useMemo(() => {
    const afterWhere = applyTableWhere(recordRows, render.where);
    return applySort(afterWhere, render.sort);
  }, [recordRows, render.where, render.sort]);

  const filtered = useMemo(() => {
    if (displayCount == null || displayCount >= filteredAll.length) return filteredAll;
    return filteredAll.slice(0, displayCount);
  }, [filteredAll, displayCount]);

  const lanes = useMemo(() => groupIntoLanes(filtered, render), [filtered, render]);

  const rowActions = useMemo(
    () =>
      (render.rowActions ?? []).filter((action) => {
        const resolved = resolveConsoleNode(ctx, action.node);
        return !resolved || isManualRunNode(resolved.node);
      }),
    [render.rowActions, ctx],
  );

  // Only trigger nodes actually reachable by a card's row actions need the
  // shared run-in-flight subscription; everything else stays lightweight.
  const triggerNodeIds = useMemo(() => {
    const ids = new Set<string>();
    for (const action of rowActions) {
      const resolved = resolveConsoleNode(ctx, action.node);
      if (resolved?.node.id && isManualRunNode(resolved.node)) ids.add(resolved.node.id);
    }
    return Array.from(ids);
  }, [rowActions, ctx]);

  if (isLoading) return <BoardSpinner />;
  if (render.lanes.length === 0) {
    return (
      <WidgetEmptyState
        icon={Kanban}
        testId="widget-board-no-lanes"
        message={
          <>
            Configure lanes in the panel editor.
            <br />
            Pick a data source and a groupBy field to see rows grouped into columns.
          </>
        }
      />
    );
  }
  const hasVisibleRows = lanes.some((lane) => lane.rows.length > 0);
  if (!hasVisibleRows) {
    return (
      <div className="flex h-full flex-col">
        <div
          className="flex-1 p-4 text-center text-xs text-slate-500 dark:text-gray-400"
          data-testid="widget-board-empty"
        >
          {render.emptyMessage ?? "No data to display."}
        </div>
        {hasMore && onLoadMore ? (
          <WidgetLoadMoreFooter isFetchingMore={Boolean(isFetchingMore)} onLoadMore={onLoadMore} />
        ) : null}
      </div>
    );
  }

  return (
    <div className="flex h-full min-h-0 flex-col">
      <WidgetTableActionLockProvider triggerNodeIds={triggerNodeIds}>
        <BoardLanes lanes={lanes} rowActions={rowActions} render={render} />
      </WidgetTableActionLockProvider>
      {hasMore && onLoadMore ? (
        <WidgetLoadMoreFooter isFetchingMore={Boolean(isFetchingMore)} onLoadMore={onLoadMore} />
      ) : null}
    </div>
  );
}

function BoardLanes({
  lanes,
  rowActions,
  render,
}: {
  lanes: LaneBucket[];
  rowActions: WidgetRowAction[];
  render: WidgetBoardRender;
}) {
  return (
    <div
      className="flex h-full min-h-0 gap-3 overflow-x-auto overflow-y-hidden p-3"
      data-testid="widget-board"
      data-groupby={render.groupBy}
    >
      {lanes.map((bucket) => (
        <BoardLane key={bucket.key} bucket={bucket} rowActions={rowActions} render={render} />
      ))}
    </div>
  );
}

function BoardLane({
  bucket,
  rowActions,
  render,
}: {
  bucket: LaneBucket;
  rowActions: WidgetRowAction[];
  render: WidgetBoardRender;
}) {
  const laneStyle = laneStyleFor(bucket.lane.color);
  const laneLabel = bucket.lane.label?.trim() ? bucket.lane.label : bucket.lane.value;
  return (
    <div
      className="flex h-full min-h-0 w-64 shrink-0 flex-col rounded-md border border-slate-200 bg-white dark:border-gray-800 dark:bg-gray-900/40"
      data-testid="widget-board-lane"
      data-lane-key={bucket.key}
    >
      <div
        className={cn(
          "flex items-center justify-between rounded-t-md border-b border-slate-200 px-3 py-1.5 dark:border-gray-800",
          laneStyle.header,
        )}
      >
        <span className="truncate text-xs font-medium">{laneLabel}</span>
        <span
          className={cn(
            "ml-2 shrink-0 rounded-full px-1.5 py-0.5 text-[10px] font-medium tabular-nums",
            laneStyle.badge,
          )}
          data-testid="widget-board-lane-count"
        >
          {bucket.rows.length}
        </span>
      </div>
      <div
        className={cn("flex-1 space-y-2 overflow-y-auto border-l-2 p-2", laneStyle.strip)}
        data-testid="widget-board-lane-body"
      >
        {bucket.rows.length === 0 ? (
          <p className="p-2 text-center text-[11px] text-slate-400 dark:text-gray-500">Empty lane</p>
        ) : (
          bucket.rows.map((row, idx) => (
            <BoardCard key={rowKeyForRow(row, idx)} row={row} index={idx} rowActions={rowActions} render={render} />
          ))
        )}
      </div>
    </div>
  );
}

function BoardCard({
  row,
  index,
  rowActions,
  render,
}: {
  row: Record<string, unknown>;
  index: number;
  rowActions: WidgetRowAction[];
  render: WidgetBoardRender;
}) {
  const rowKey = rowKeyForRow(row, index);
  const title = cardTitle(row, render);
  const visibleActions = rowActions.filter((action) => evaluateRowShow(action.show, row));

  return (
    <div
      className="rounded-md border border-slate-200 bg-white p-2 shadow-sm hover:border-slate-300 dark:border-gray-800 dark:bg-gray-900 dark:hover:border-gray-700"
      data-testid="widget-board-card"
    >
      <div className="text-[13px] font-medium leading-tight text-slate-800 dark:text-gray-100">{title}</div>
      {(render.card.fields ?? []).length > 0 ? (
        <div className="mt-1.5 space-y-1">
          {(render.card.fields ?? []).map((field, fi) => (
            <WidgetBoardCardField key={`${field.field}-${fi}`} col={field} row={row} />
          ))}
        </div>
      ) : null}
      {visibleActions.length > 0 ? (
        <div className="mt-2 flex flex-wrap justify-end gap-1">
          {visibleActions.map((action, ai) => (
            <WidgetRowActionButton key={ai} action={action} row={row} rowKey={rowKey} />
          ))}
        </div>
      ) : null}
    </div>
  );
}

function cardTitle(row: Record<string, unknown>, render: WidgetBoardRender): string {
  const raw = resolveCellValue(render.card.titleField, row);
  if (raw != null && String(raw).trim() !== "") return String(raw);
  // Fallback to the groupBy value so cards missing a title still label
  // themselves usefully, and finally to the row id / a numeric placeholder.
  const laneValue = resolveCellValue(render.groupBy, row);
  if (laneValue != null && String(laneValue).trim() !== "") return String(laneValue);
  const id = row.id;
  if (typeof id === "string" || typeof id === "number") return String(id);
  return "(no title)";
}

function groupIntoLanes(rows: Record<string, unknown>[], render: WidgetBoardRender): LaneBucket[] {
  const buckets: LaneBucket[] = render.lanes.map((lane) => ({
    lane,
    rows: [],
    key: `lane:${lane.value}`,
  }));
  const laneByNormalizedValue = new Map<string, LaneBucket>();
  for (const bucket of buckets) {
    laneByNormalizedValue.set(normalizeBoardLaneValue(bucket.lane.value), bucket);
  }

  const otherBucket: LaneBucket | undefined = render.otherLane
    ? { lane: { value: OTHER_LANE_KEY, label: "Other", color: "neutral" }, rows: [], key: OTHER_LANE_KEY }
    : undefined;

  for (const row of rows) {
    const groupValue = resolveCellValue(render.groupBy, row);
    const bucket = laneByNormalizedValue.get(normalizeBoardLaneValue(groupValue));
    if (bucket) {
      bucket.rows.push(row);
      continue;
    }
    if (otherBucket) otherBucket.rows.push(row);
  }

  if (otherBucket) buckets.push(otherBucket);
  return buckets;
}

function BoardSpinner() {
  return (
    <div className="flex h-full items-center justify-center p-4">
      <Loader2 className="size-4 animate-spin text-slate-400 dark:text-gray-500" />
    </div>
  );
}
