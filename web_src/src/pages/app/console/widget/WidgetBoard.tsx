import { useMemo } from "react";
import { Check, Kanban, Loader2, RefreshCcw, User, X } from "lucide-react";

import { cn } from "@/lib/utils";

import { normalizeBoardLaneValue } from "../boardPanelContent";
import { useConsoleContext, resolveConsoleNode } from "../ConsoleContext";
import { isManualRunNode } from "../manualRunTriggers";
import { WidgetEmptyState } from "../WidgetEmptyState";
import { applyTableWhere } from "./evalTableWhere";
import { resolveCellValue } from "./resolveCellValue";
import { evaluateRowShow } from "./rowVisibility";
import { applySort } from "./widgetData";
import { WidgetBoardCardField, isBoardCardHeaderField, isBoardCardMetaField } from "./WidgetBoardCardField";
import { WidgetRowActionButton } from "./WidgetRowActionButton";
import { rowKeyForRow } from "./rowKey";
import { WidgetLoadMoreFooter } from "./WidgetTable";
import { WidgetTableActionLockProvider } from "./WidgetTableActionLock";
import type { WidgetBoardLane, WidgetBoardRender, WidgetRowAction, WidgetTableColumn } from "./types";

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

  const filteredAll = useMemo(() => applyTableWhere(recordRows, render.where), [recordRows, render.where]);

  // Slice the progressive window on loaded order, *before* sorting: `sort`
  // orders cards inside each lane (see WidgetBoardRender), so it must not
  // act as a global selection criterion that starves lanes whose rows sort
  // lower than the window cutoff.
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
  const laneLabel = bucket.lane.label?.trim() ? bucket.lane.label : bucket.lane.value;
  const laneStatus = laneStatusFor(bucket.lane);
  return (
    <div
      className="flex h-full min-h-0 min-w-72 flex-1 flex-col rounded-md bg-slate-100 dark:bg-gray-900/40"
      data-testid="widget-board-lane"
      data-lane-key={bucket.key}
    >
      <div className="flex items-center justify-between rounded-t-md px-3 py-1.5 text-slate-500 dark:text-gray-200">
        <span className="truncate text-[13px] font-medium">{laneLabel}</span>
        <span
          className="ml-2 shrink-0 rounded-full bg-slate-200 px-1.5 py-0.5 text-[10px] font-medium tabular-nums text-slate-700 dark:bg-gray-700 dark:text-gray-200"
          data-testid="widget-board-lane-count"
        >
          {bucket.rows.length}
        </span>
      </div>
      <div className="flex-1 space-y-2 overflow-y-auto p-2" data-testid="widget-board-lane-body">
        {bucket.rows.length === 0 ? (
          <p className="p-2 text-center text-[11px] text-slate-400 dark:text-gray-500">Empty lane</p>
        ) : (
          bucket.rows.map((row, idx) => (
            <BoardCard
              key={rowKeyForRow(row, idx)}
              row={row}
              index={idx}
              rowActions={rowActions}
              render={render}
              laneStatus={laneStatus}
            />
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
  laneStatus,
}: {
  row: Record<string, unknown>;
  index: number;
  rowActions: WidgetRowAction[];
  render: WidgetBoardRender;
  laneStatus?: BoardLaneStatus;
}) {
  const rowKey = rowKeyForRow(row, index);
  const title = cardTitle(row, render);
  const visibleActions = rowActions.filter((action) => evaluateRowShow(action.show, row));
  const { headerFields, metaFields, bodyFields } = partitionCardFields(render.card.fields ?? []);
  const durationFields = metaFields.filter((field) => field.format === "duration");
  const relativeFields = metaFields.filter((field) => field.format === "relative");
  const showStatusIcon =
    laneStatus === "failed" || laneStatus === "done" || laneStatus === "in_progress" || laneStatus === "human_review";

  return (
    <div
      className={cn(
        "rounded-lg border border-slate-950/15 bg-clip-padding p-2 dark:border-gray-700/70",
        laneStatus === "failed"
          ? "bg-[radial-gradient(100px_circle_at_top_right,theme(colors.red.100)_0%,theme(colors.white)_100%)] dark:bg-[radial-gradient(100px_circle_at_top_right,rgb(153_27_27_/_0.55)_0%,rgb(255_255_255_/_0.05)_100%)]"
          : "bg-white dark:bg-white/5",
      )}
      data-testid="widget-board-card"
    >
      {headerFields.length > 0 || showStatusIcon ? (
        <div className="mb-1 flex w-full items-center justify-between gap-2" data-testid="board-card-header-fields">
          <div className="flex min-w-0 flex-wrap items-center gap-x-2 gap-y-0.5">
            {headerFields.map((field, fi) => (
              <WidgetBoardCardField key={`${field.field}-${fi}`} col={field} row={row} variant="header" />
            ))}
          </div>
          {laneStatus === "failed" ? (
            <X
              className="ml-auto size-3.5 shrink-0 text-red-600 dark:text-red-400"
              aria-hidden
              data-testid="board-card-failed-icon"
            />
          ) : null}
          {laneStatus === "done" ? (
            <Check
              className="ml-auto size-3.5 shrink-0 text-emerald-600"
              aria-hidden
              data-testid="board-card-done-icon"
            />
          ) : null}
          {laneStatus === "in_progress" ? (
            <RefreshCcw
              className="ml-auto size-3.5 shrink-0 animate-[spin_2.5s_linear_infinite] [animation-direction:reverse] text-blue-600 dark:text-blue-400"
              aria-hidden
              data-testid="board-card-in-progress-icon"
            />
          ) : null}
          {laneStatus === "human_review" ? (
            <User
              className="ml-auto size-3.5 shrink-0 text-yellow-800 dark:text-yellow-400"
              aria-hidden
              data-testid="board-card-human-review-icon"
            />
          ) : null}
        </div>
      ) : null}
      <div className="text-[13px] font-medium leading-tight text-slate-800 dark:text-gray-100">{title}</div>
      {metaFields.length > 0 ? (
        <div className="mt-1.5 flex w-full items-center justify-between gap-2" data-testid="board-card-meta-fields">
          <div className="flex min-w-0 flex-wrap items-center gap-x-2.5 gap-y-0.5">
            {durationFields.map((field, fi) => (
              <WidgetBoardCardField key={`${field.field}-${fi}`} col={field} row={row} variant="meta" />
            ))}
          </div>
          {relativeFields.length > 0 ? (
            <div className="ml-auto flex shrink-0 items-center justify-end gap-1.5">
              {relativeFields.map((field, fi) => (
                <WidgetBoardCardField key={`${field.field}-${fi}`} col={field} row={row} variant="meta" />
              ))}
            </div>
          ) : null}
        </div>
      ) : null}
      {bodyFields.length > 0 ? (
        <div className="mt-1.5 space-y-1">
          {bodyFields.map((field, fi) => (
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

/**
 * Split card fields into header links (above title), time meta (one row under
 * the title), and remaining body rows — preserving configured order in each
 * group.
 */
function partitionCardFields(fields: WidgetTableColumn[]): {
  headerFields: WidgetTableColumn[];
  metaFields: WidgetTableColumn[];
  bodyFields: WidgetTableColumn[];
} {
  const headerFields: WidgetTableColumn[] = [];
  const metaFields: WidgetTableColumn[] = [];
  const bodyFields: WidgetTableColumn[] = [];
  for (const field of fields) {
    if (isBoardCardHeaderField(field)) {
      headerFields.push(field);
      continue;
    }
    if (isBoardCardMetaField(field)) {
      metaFields.push(field);
      continue;
    }
    bodyFields.push(field);
  }
  return { headerFields, metaFields, bodyFields };
}

type BoardLaneStatus = "failed" | "done" | "in_progress" | "human_review";

/**
 * Match factory YAML (`failed`/`done`/`in_progress`/`human_review`) and
 * fixture labels (`Failed`/`Done`/`In progress`/`Human review`) for
 * lane-specific card chrome.
 */
function laneStatusFor(lane: WidgetBoardLane): BoardLaneStatus | undefined {
  const keys = [normalizeBoardLaneValue(lane.value), normalizeBoardLaneValue(lane.label)];
  if (keys.includes("failed")) return "failed";
  if (keys.includes("done")) return "done";
  if (keys.includes("in_progress") || keys.includes("in progress")) return "in_progress";
  if (keys.includes("human_review") || keys.includes("human review")) return "human_review";
  return undefined;
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
    ? { lane: { value: OTHER_LANE_KEY, label: "Other" }, rows: [], key: OTHER_LANE_KEY }
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
  // `sort` orders cards within each lane, per the board schema/PRD — it never
  // decides which rows make it onto the board.
  for (const bucket of buckets) {
    bucket.rows = applySort(bucket.rows, render.sort);
  }
  return buckets;
}

function BoardSpinner() {
  return (
    <div className="flex h-full items-center justify-center p-4">
      <Loader2 className="size-4 animate-spin text-slate-400 dark:text-gray-500" />
    </div>
  );
}
