import { useCallback, useEffect, useMemo, useRef, type UIEvent } from "react";
import { Loader2, Plus, Table2 } from "lucide-react";

import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";

import { useConsoleContext, resolveConsoleNode } from "../ConsoleContext";
import { CONSOLE_WIDGET_TABLE_HEAD_CLASSES, CONSOLE_TABLE_HEAD_BORDER_DARK_CLASSES } from "../consoleTableStyles";
import { isManualRunNode } from "../manualRunTriggers";
import { applyTableWhere } from "./evalTableWhere";
import { evaluateRowShow } from "./rowVisibility";
import { makeRowStyleResolver } from "./rowStyles";
import { applyFilters, applySort } from "./widgetData";
import { WidgetEmptyState } from "../WidgetEmptyState";
import { WidgetTableActionLockProvider } from "./WidgetTableActionLock";
import { WidgetTableCell } from "./WidgetTableCell";
import { WidgetRowActionButton } from "./WidgetRowActionButton";
import { rowKeyForRow } from "./rowKey";
import type { WidgetTableRender } from "./types";

interface WidgetTableProps {
  render: WidgetTableRender;
  rows: unknown[];
  isLoading: boolean;
  /**
   * Progressive-pagination affordances. When `hasMore` is true the footer
   * shows a "Load more" button and the scrollable area auto-fetches as the
   * user scrolls near the bottom. Wired only by the table panel; chart and
   * number panels render the full configured limit at once.
   */
  hasMore?: boolean;
  isFetchingMore?: boolean;
  onLoadMore?: () => void;
  /**
   * Progressive display window. When set, `rows` is the full loaded set and
   * only the first `displayCount` rows after filter+sort are rendered — so
   * trend columns can compare against loaded-but-hidden neighbors.
   */
  displayCount?: number;
}

/** Distance from the bottom (px) at which scrolling auto-requests more rows. */
const AUTO_LOAD_SCROLL_THRESHOLD_PX = 160;

export function WidgetTable({
  render,
  rows,
  isLoading,
  hasMore,
  isFetchingMore,
  onLoadMore,
  displayCount,
}: WidgetTableProps) {
  const ctx = useConsoleContext();
  const recordRows = useMemo(
    () => rows.filter((r): r is Record<string, unknown> => Boolean(r) && typeof r === "object" && !Array.isArray(r)),
    [rows],
  );

  const filteredAll = useMemo(() => {
    const afterWhere = applyTableWhere(recordRows, render.where);
    const afterFilters = applyFilters(afterWhere, render.filters);
    return applySort(afterFilters, render.sort);
  }, [recordRows, render.where, render.filters, render.sort]);

  // Slice after filter+sort so progressive windows and trend baselines share
  // the same ordered list. Without `displayCount`, render the full filtered set.
  const filtered = useMemo(() => {
    if (displayCount == null || displayCount >= filteredAll.length) return filteredAll;
    return filteredAll.slice(0, displayCount);
  }, [filteredAll, displayCount]);

  const resolveRowStyle = useMemo(() => makeRowStyleResolver(render.rowStyles), [render.rowStyles]);

  // Row actions whose configured node isn't manually runnable are hidden
  // downstream in `WidgetTableGrid`; unresolved actions still render with a
  // "Node not found" tooltip. We only need the trigger id set here to scope
  // the lock's runs subscription to the actual manual-runnable targets.
  const triggerNodeIds = useMemo(() => {
    const ids = new Set<string>();
    for (const action of render.rowActions ?? []) {
      const resolved = resolveConsoleNode(ctx, action.node);
      if (resolved?.node.id && isManualRunNode(resolved.node)) ids.add(resolved.node.id);
    }
    return Array.from(ids);
  }, [render.rowActions, ctx]);

  // Auto-load more rows as the user scrolls near the bottom. The table's
  // `loadMore()` is dual-mode: it usually just widens the display window over
  // rows already in the infinite-query cache (no network fetch), and only
  // sometimes triggers an actual page fetch. So we can't rely on a fetching
  // flag flipping to re-arm the guard (it would stay armed forever after a
  // cache-only reveal). Instead we re-arm whenever the rendered row set grows
  // or a fetch settles, which covers both modes.
  const loadMoreRequestedRef = useRef(false);
  useEffect(() => {
    loadMoreRequestedRef.current = false;
  }, [rows.length, isFetchingMore]);

  const onScroll = useCallback(
    (event: UIEvent<HTMLDivElement>) => {
      const el = event.currentTarget;
      if (!hasMore || !onLoadMore || isFetchingMore || loadMoreRequestedRef.current) return;
      const remainingScroll = el.scrollHeight - el.scrollTop - el.clientHeight;
      if (remainingScroll > AUTO_LOAD_SCROLL_THRESHOLD_PX) return;
      loadMoreRequestedRef.current = true;
      onLoadMore();
    },
    [hasMore, onLoadMore, isFetchingMore],
  );

  if (isLoading) return <WidgetSpinner />;
  if (render.columns.length === 0) {
    return (
      <WidgetEmptyState
        icon={Table2}
        testId="widget-table-no-columns"
        message={
          <>
            Configure columns in the panel editor.
            <br />
            Pick a memory namespace to see available fields.
          </>
        }
      />
    );
  }
  if (filtered.length === 0) {
    // The current pages produced no matching rows, but later server pages
    // might — so keep the "Load more" affordance available instead of
    // dead-ending on the empty message.
    return (
      <div data-testid="widget-table-empty-wrap">
        <div className="p-4 text-center text-xs text-slate-500 dark:text-gray-400" data-testid="widget-table-empty">
          {render.emptyMessage ?? "No data to display."}
        </div>
        {hasMore && onLoadMore ? (
          <WidgetLoadMoreFooter isFetchingMore={Boolean(isFetchingMore)} onLoadMore={onLoadMore} />
        ) : null}
      </div>
    );
  }

  return (
    <WidgetTableActionLockProvider triggerNodeIds={triggerNodeIds}>
      <WidgetTableGrid
        render={render}
        filtered={filtered}
        filteredAll={filteredAll}
        resolveRowStyle={resolveRowStyle}
        hasMore={hasMore}
        isFetchingMore={isFetchingMore}
        onLoadMore={onLoadMore}
        onScroll={onScroll}
      />
    </WidgetTableActionLockProvider>
  );
}

interface WidgetTableGridProps {
  render: WidgetTableRender;
  filtered: Record<string, unknown>[];
  /** Full filter+sort result; may be longer than `filtered` when displayCount slices. */
  filteredAll: Record<string, unknown>[];
  resolveRowStyle: ReturnType<typeof makeRowStyleResolver>;
  hasMore?: boolean;
  isFetchingMore?: boolean;
  onLoadMore?: () => void;
  onScroll: (event: UIEvent<HTMLDivElement>) => void;
}

function WidgetTableGrid({
  render,
  filtered,
  filteredAll,
  resolveRowStyle,
  hasMore,
  isFetchingMore,
  onLoadMore,
  onScroll,
}: WidgetTableGridProps) {
  const ctx = useConsoleContext();
  const rowActions = (render.rowActions ?? []).filter((action) => {
    const resolved = resolveConsoleNode(ctx, action.node);
    return !resolved || isManualRunNode(resolved.node);
  });
  const hasActions = rowActions.length > 0;
  const lastIdx = filtered.length - 1;
  return (
    <div className="overflow-auto" data-testid="widget-table" onScroll={onScroll}>
      <table className="w-full border-collapse text-[13px]">
        <thead>
          <tr>
            {render.columns.map((col, i) => (
              <th
                key={`${col.field}-${i}`}
                className={cn(CONSOLE_WIDGET_TABLE_HEAD_CLASSES, CONSOLE_TABLE_HEAD_BORDER_DARK_CLASSES, "text-left")}
              >
                {col.label ?? col.field}
              </th>
            ))}
            {hasActions ? (
              <th
                className={cn(CONSOLE_WIDGET_TABLE_HEAD_CLASSES, CONSOLE_TABLE_HEAD_BORDER_DARK_CLASSES, "text-right")}
              >
                Actions
              </th>
            ) : null}
          </tr>
        </thead>
        <tbody>
          {filtered.map((row, idx) => {
            const rowKey = rowKeyForRow(row, idx);
            const toneClass = resolveRowStyle?.(row);
            // Neighbor comes from the full ordered list so a progressive
            // display window still compares against the next sorted row even
            // when that row is loaded but not yet shown.
            const nextRow = filteredAll[idx + 1] as Record<string, unknown> | undefined;
            const hasMoreBelow = idx === lastIdx && !nextRow && Boolean(hasMore);
            return (
              <tr
                key={rowKey}
                data-row-tone={toneClass ? "true" : undefined}
                className={cn(
                  "border-b border-black/10 last:border-0 dark:border-gray-800",
                  // Drop the default hover wash when a tone is applied so the
                  // tint isn't overridden — the row already has a deliberate
                  // background and a hover bg would mask it.
                  toneClass ? toneClass : "hover:bg-slate-50/60 dark:hover:bg-gray-800/60",
                )}
              >
                {render.columns.map((col, ci) => (
                  <WidgetTableCell
                    key={`${col.field}-${ci}`}
                    col={col}
                    row={row}
                    nextRow={nextRow}
                    hasMoreBelow={hasMoreBelow}
                  />
                ))}
                {hasActions ? (
                  <td className="px-3 py-1.5 text-right">
                    <div className="inline-flex items-center gap-1">
                      {rowActions
                        .filter((action) => evaluateRowShow(action.show, row))
                        .map((action, ai) => (
                          <WidgetRowActionButton key={ai} action={action} row={row} rowKey={rowKey} />
                        ))}
                    </div>
                  </td>
                ) : null}
              </tr>
            );
          })}
        </tbody>
      </table>
      {hasMore && onLoadMore ? (
        <WidgetLoadMoreFooter isFetchingMore={Boolean(isFetchingMore)} onLoadMore={onLoadMore} />
      ) : null}
    </div>
  );
}

export function WidgetLoadMoreFooter({
  isFetchingMore,
  onLoadMore,
}: {
  isFetchingMore: boolean;
  onLoadMore: () => void;
}) {
  return (
    <div
      className="flex items-center justify-center border-t border-slate-100 bg-slate-50/60 px-3 py-2 dark:border-gray-800 dark:bg-gray-800/60"
      data-testid="widget-table-load-more"
    >
      <Button
        type="button"
        size="xs"
        variant="outline"
        onClick={onLoadMore}
        disabled={isFetchingMore}
        className="gap-1"
        data-testid="widget-table-load-more-button"
      >
        {isFetchingMore ? <Loader2 className="h-3 w-3 animate-spin" /> : <Plus className="h-3 w-3" />}
        {isFetchingMore ? "Loading…" : "Load more"}
      </Button>
    </div>
  );
}

function WidgetSpinner() {
  return (
    <div className="flex h-full items-center justify-center p-4">
      <Loader2 className="size-4 animate-spin text-slate-400 dark:text-gray-500" />
    </div>
  );
}
