import { useCallback, useEffect, useMemo, useRef, useState, type UIEvent } from "react";
import { ExternalLink, Loader2, Play, Plus, RefreshCw, Square, Table2, Trash2 } from "lucide-react";

import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";

import { useConsoleContext, resolveConsoleNode } from "../ConsoleContext";
import { CONSOLE_WIDGET_TABLE_HEAD_CLASSES, CONSOLE_TABLE_HEAD_BORDER_DARK_CLASSES } from "../consoleTableStyles";
import { isManualRunNode } from "../manualRunTriggers";
import { applyTableWhere } from "./evalTableWhere";
import { mergeTriggerParameters } from "./mergeTriggerPayload";
import { RowActionConfirmDialog } from "./RowActionConfirmDialog";
import { evaluateRowShow } from "./rowVisibility";
import { makeRowStyleResolver } from "./rowStyles";
import { applyFilters, applySort } from "./widgetData";
import { WidgetEmptyState } from "../WidgetEmptyState";
import { WidgetTableActionLockProvider } from "./WidgetTableActionLock";
import { useWidgetTableActionLock } from "./WidgetTableActionLockContext";
import { WidgetTableCell } from "./WidgetTableCell";
import type { WidgetRowAction, WidgetTableRender } from "./types";

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
   * First already-loaded row beyond the progressive display window. Trend
   * columns on the last visible row compare against this peek when `rows`
   * has no neighbor below — `hasMore` alone is not enough, because it is
   * also true when more rows are loaded but still hidden.
   */
  nextLoadedRow?: Record<string, unknown>;
}

const ACTION_ICONS = {
  play: Play,
  stop: Square,
  trash: Trash2,
  refresh: RefreshCw,
  "external-link": ExternalLink,
} as const;

/** Distance from the bottom (px) at which scrolling auto-requests more rows. */
const AUTO_LOAD_SCROLL_THRESHOLD_PX = 160;

export function WidgetTable({
  render,
  rows,
  isLoading,
  hasMore,
  isFetchingMore,
  onLoadMore,
  nextLoadedRow,
}: WidgetTableProps) {
  const ctx = useConsoleContext();
  const recordRows = useMemo(
    () => rows.filter((r): r is Record<string, unknown> => Boolean(r) && typeof r === "object" && !Array.isArray(r)),
    [rows],
  );

  const filtered = useMemo(() => {
    const afterWhere = applyTableWhere(recordRows, render.where);
    const afterFilters = applyFilters(afterWhere, render.filters);
    return applySort(afterFilters, render.sort);
  }, [recordRows, render.where, render.filters, render.sort]);

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
          <LoadMoreFooter isFetchingMore={Boolean(isFetchingMore)} onLoadMore={onLoadMore} />
        ) : null}
      </div>
    );
  }

  return (
    <WidgetTableActionLockProvider triggerNodeIds={triggerNodeIds}>
      <WidgetTableGrid
        render={render}
        filtered={filtered}
        resolveRowStyle={resolveRowStyle}
        hasMore={hasMore}
        isFetchingMore={isFetchingMore}
        onLoadMore={onLoadMore}
        onScroll={onScroll}
        nextLoadedRow={nextLoadedRow}
      />
    </WidgetTableActionLockProvider>
  );
}

interface WidgetTableGridProps {
  render: WidgetTableRender;
  filtered: Record<string, unknown>[];
  resolveRowStyle: ReturnType<typeof makeRowStyleResolver>;
  hasMore?: boolean;
  isFetchingMore?: boolean;
  onLoadMore?: () => void;
  onScroll: (event: UIEvent<HTMLDivElement>) => void;
  nextLoadedRow?: Record<string, unknown>;
}

function WidgetTableGrid({
  render,
  filtered,
  resolveRowStyle,
  hasMore,
  isFetchingMore,
  onLoadMore,
  onScroll,
  nextLoadedRow,
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
            const neighborBelow = filtered[idx + 1] as Record<string, unknown> | undefined;
            // Prefer the next visible filtered row; fall back to the first
            // already-loaded row still hidden by the progressive window so
            // trend cells don't show pending `...` for data we already have.
            const nextRow = neighborBelow ?? (idx === lastIdx ? nextLoadedRow : undefined);
            const hasMoreBelow = idx === lastIdx && !neighborBelow && !nextLoadedRow && Boolean(hasMore);
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
                          <RowActionButton key={ai} action={action} row={row} rowKey={rowKey} />
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
        <LoadMoreFooter isFetchingMore={Boolean(isFetchingMore)} onLoadMore={onLoadMore} />
      ) : null}
    </div>
  );
}

function LoadMoreFooter({ isFetchingMore, onLoadMore }: { isFetchingMore: boolean; onLoadMore: () => void }) {
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

type ActionDisabledReason = "no-perm" | "no-node" | "not-manual-run" | "run-in-flight" | "submitting" | null;

// `not-manual-run` is defense in depth: `WidgetTableGrid` already hides
// non-manual-run actions upstream. This branch covers the transient case
// before the trigger catalog resolves — the action then renders disabled
// rather than as a button that would fail server-side.
function disabledReason(
  canRun: boolean,
  hasResolvedNode: boolean,
  isManualRun: boolean,
  runInFlight: boolean,
  submitting: boolean,
): ActionDisabledReason {
  if (!canRun) return "no-perm";
  if (!hasResolvedNode) return "no-node";
  if (!isManualRun) return "not-manual-run";
  if (runInFlight) return "run-in-flight";
  if (submitting) return "submitting";
  return null;
}

/**
 * Stable per-row key used to scope action locks. Prefers the row's `id`
 * when present (memory rows, executions, runs all expose one) and falls
 * back to a deterministic JSON encoding when the source rows don't carry
 * identifiers. The index is only used as a last-resort tiebreaker so
 * locks don't bleed across rows on re-render.
 */
function rowKeyForRow(row: Record<string, unknown>, index: number): string {
  const id = row.id;
  if (typeof id === "string" && id.length > 0) return id;
  if (typeof id === "number") return String(id);
  try {
    return `row:${index}:${JSON.stringify(row)}`;
  } catch {
    return `row:${index}`;
  }
}

function disabledTooltip(reason: ActionDisabledReason, node: string): string | undefined {
  switch (reason) {
    case "no-perm":
      return "You do not have permission to run actions in this canvas";
    case "no-node":
      return `Node "${node}" not found on this canvas`;
    case "not-manual-run":
      return "Only trigger nodes with a manual run can be fired from the console.";
    case "run-in-flight":
      return "A run for this trigger is already in progress.";
    case "submitting":
      return "Submitting trigger…";
    default:
      return undefined;
  }
}

type ResolvedNode = NonNullable<ReturnType<typeof resolveConsoleNode>>;

function useRowActionFire({
  action,
  row,
  rowKey,
  resolved,
  hookName,
  label,
  setConfirmOpen,
}: {
  action: WidgetRowAction;
  row: Record<string, unknown>;
  rowKey: string;
  resolved: ResolvedNode | undefined;
  hookName: string;
  label: string;
  setConfirmOpen: (open: boolean) => void;
}) {
  const ctx = useConsoleContext();
  const lock = useWidgetTableActionLock();
  const [error, setError] = useState<string | undefined>();
  const [pending, setPending] = useState(false);

  const fire = async () => {
    if (!ctx?.onTriggerNode || !resolved?.node.id) return;
    const triggerNodeId = resolved.node.id;
    setError(undefined);
    setPending(true);
    lock.beginSubmission(triggerNodeId, rowKey);
    let succeeded = false;
    try {
      const parameters = mergeTriggerParameters(resolved.node, hookName, action.template, row, action.payload);
      await ctx.onTriggerNode(triggerNodeId, {
        hookName,
        templateName: action.template,
        parameters,
        successLabel: label,
      });
      succeeded = true;
      setConfirmOpen(false);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to trigger");
    } finally {
      setPending(false);
      lock.endSubmission(triggerNodeId, rowKey, succeeded);
    }
  };

  return { fire, error, pending };
}

function useRowActionGate(action: WidgetRowAction, rowKey: string) {
  const ctx = useConsoleContext();
  const lock = useWidgetTableActionLock();
  const canRun = ctx?.canRunNodes ?? false;
  const resolved = resolveConsoleNode(ctx, action.node);
  // WidgetTable hides non-manual actions upstream; at this level `true` is
  // the normal case, unresolved nodes render disabled with a tooltip.
  const isManualRun = isManualRunNode(resolved?.node);
  const triggerNodeId = resolved?.node.id;
  // Per-row locking: a row's button is disabled by `runInFlight` only when
  // its own submission produced the in-flight run (i.e. the mapping points
  // back to this row's key). Other rows sharing the same trigger stay
  // clickable, matching the "lock only the affected row" model.
  const runInFlight = Boolean(
    triggerNodeId && lock.runInFlightIds.has(triggerNodeId) && lock.inFlightRowByTrigger.get(triggerNodeId) === rowKey,
  );
  const submitting = lock.pendingRowKeys.has(rowKey);
  const reason = disabledReason(canRun, Boolean(resolved), isManualRun, runInFlight, submitting);
  return {
    resolved,
    isManualRun,
    disabled: reason !== null,
    reason,
    tooltip: disabledTooltip(reason, action.node),
  };
}

function RowActionButton({
  action,
  row,
  rowKey,
}: {
  action: WidgetRowAction;
  row: Record<string, unknown>;
  rowKey: string;
}) {
  const [confirmOpen, setConfirmOpen] = useState(false);

  const { resolved, isManualRun, disabled, reason, tooltip } = useRowActionGate(action, rowKey);
  const label = action.label ?? "Run";
  const hookName = action.hook ?? "run";
  const Icon = action.icon ? ACTION_ICONS[action.icon] : undefined;

  const { fire, error, pending } = useRowActionFire({
    action,
    row,
    rowKey,
    resolved,
    hookName,
    label,
    setConfirmOpen,
  });

  const handleClick = () => {
    if (disabled) return;
    if (action.confirm?.trim()) {
      setConfirmOpen(true);
      return;
    }
    void fire();
  };

  const testId = `widget-row-action-${action.node || "trigger"}`;

  return (
    <div className="inline-flex flex-col items-end gap-0.5">
      <Button
        type="button"
        size="xs"
        variant="outline"
        onClick={handleClick}
        disabled={disabled || pending}
        aria-disabled={disabled}
        title={tooltip}
        data-testid={testId}
        data-variant={action.variant ?? "default"}
        data-disabled-reason={reason ?? undefined}
      >
        {Icon ? <Icon className="mr-1 h-3 w-3" /> : null}
        {label}
      </Button>
      {error ? (
        <span
          className="max-w-48 text-right text-[10px] text-red-600 dark:text-red-400"
          data-testid={`${testId}-error`}
        >
          {error}
        </span>
      ) : null}
      {action.confirm ? (
        <RowActionConfirmDialog
          action={action}
          row={row}
          resolved={resolved}
          isManualRun={isManualRun}
          hookName={hookName}
          label={label}
          open={confirmOpen}
          onOpenChange={setConfirmOpen}
          confirmDisabled={pending || disabled}
          onConfirm={() => void fire()}
          testId={testId}
        />
      ) : null}
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
