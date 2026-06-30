import { useCallback, useEffect, useMemo, useRef, useState, type UIEvent } from "react";
import { ExternalLink, Loader2, Play, Plus, RefreshCw, Square, Table2, Trash2 } from "lucide-react";

import { formatTimestampInUserTimezone } from "@/lib/timezone";
import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";

import { useConsoleContext, resolveConsoleNode } from "../ConsoleContext";
import { applyTableWhere } from "./evalTableWhere";
import { mergeTriggerParameters } from "./mergeTriggerPayload";
import { RowActionConfirmDialog } from "./RowActionConfirmDialog";
import { evaluateRowShow } from "./rowVisibility";
import { resolveCellValue } from "./resolveCellValue";
import { resolveHref } from "./resolveHref";
import { makeRowStyleResolver } from "./rowStyles";
import { applyFilters, applySort } from "./widgetData";
import { WidgetEmptyState } from "../WidgetEmptyState";
import { formatValue } from "./widgetFormat";
import { WidgetTableActionLockProvider } from "./WidgetTableActionLock";
import { useWidgetTableActionLock } from "./WidgetTableActionLockContext";
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
}

const STATUS_PILL_CLASS: Record<string, string> = {
  passed: "bg-emerald-500 text-white",
  ready: "bg-emerald-500 text-white",
  active: "bg-emerald-500 text-white",
  "very low": "bg-emerald-500 text-white",
  low: "bg-emerald-500 text-white",
  failed: "bg-red-500 text-white",
  critical: "bg-red-500 text-white",
  high: "bg-orange-500 text-white",
  running: "bg-blue-500 text-white",
  medium: "bg-yellow-500 text-white",
  cancelled: "bg-gray-500 text-white",
  pending: "bg-gray-500 text-white",
  idle: "bg-gray-500 text-white",
};

const STATUS_PILL_BASE_CLASS = "inline-flex rounded-full border-none px-2 py-0.5 text-[11px] font-medium";

const BADGE_PILL_CLASS =
  "inline-flex rounded-full bg-transparent px-2 py-0.5 text-[11px] font-medium text-slate-700 outline outline-1 -outline-offset-1 outline-slate-950/15";

const ACTION_ICONS = {
  play: Play,
  stop: Square,
  trash: Trash2,
  refresh: RefreshCw,
  "external-link": ExternalLink,
} as const;

/** Distance from the bottom (px) at which scrolling auto-requests more rows. */
const AUTO_LOAD_SCROLL_THRESHOLD_PX = 160;

export function WidgetTable({ render, rows, isLoading, hasMore, isFetchingMore, onLoadMore }: WidgetTableProps) {
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

  // Collect the unique trigger node ids referenced by this table's row actions
  // so the action lock can subscribe to the runs query only when needed.
  const triggerNodeIds = useMemo(() => {
    const ids = new Set<string>();
    for (const action of render.rowActions ?? []) {
      const resolved = resolveConsoleNode(ctx, action.node);
      if (resolved?.node.id && resolved.node.type === "TYPE_TRIGGER") ids.add(resolved.node.id);
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
        <div className="p-4 text-center text-xs text-slate-500" data-testid="widget-table-empty">
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
}

function WidgetTableGrid({
  render,
  filtered,
  resolveRowStyle,
  hasMore,
  isFetchingMore,
  onLoadMore,
  onScroll,
}: WidgetTableGridProps) {
  const hasActions = Boolean(render.rowActions && render.rowActions.length > 0);
  return (
    <div className="overflow-auto" data-testid="widget-table" onScroll={onScroll}>
      <table className="w-full border-collapse text-[13px]">
        <thead>
          <tr>
            {render.columns.map((col, i) => (
              <th
                key={`${col.field}-${i}`}
                className="border-b border-slate-200 px-3 py-1.5 text-left text-[11px] font-semibold uppercase tracking-wide text-slate-500"
              >
                {col.label ?? col.field}
              </th>
            ))}
            {hasActions ? (
              <th className="border-b border-slate-200 px-3 py-1.5 text-right text-[11px] font-semibold uppercase tracking-wide text-slate-500">
                Actions
              </th>
            ) : null}
          </tr>
        </thead>
        <tbody>
          {filtered.map((row, idx) => {
            const rowKey = rowKeyForRow(row, idx);
            const toneClass = resolveRowStyle?.(row);
            return (
              <tr
                key={rowKey}
                data-row-tone={toneClass ? "true" : undefined}
                className={cn(
                  "border-b border-black/10 last:border-0",
                  // Drop the default hover wash when a tone is applied so the
                  // tint isn't overridden — the row already has a deliberate
                  // background and a hover bg would mask it.
                  toneClass ? toneClass : "hover:bg-slate-50/60",
                )}
              >
                {render.columns.map((col, ci) => (
                  <Cell key={`${col.field}-${ci}`} col={col} row={row} />
                ))}
                {hasActions ? (
                  <td className="px-3 py-1.5 text-right">
                    <div className="inline-flex items-center gap-1">
                      {render.rowActions
                        ?.filter((action) => evaluateRowShow(action.show, row))
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
      className="flex items-center justify-center border-t border-slate-100 bg-slate-50/60 px-3 py-2"
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

function Cell({ col, row }: { col: WidgetTableRender["columns"][number]; row: Record<string, unknown> }) {
  const visible = evaluateRowShow(col.show, row);
  if (!visible) {
    return <td className="px-3 py-1.5 text-slate-300">—</td>;
  }
  const value = resolveCellValue(col.field, row);
  const formatted = formatValue(value, col.format);
  // `status` renders semantic values (passed, failed, risk levels) as colored
  // pills. `badge` is for neutral tags (service names, categories) with a
  // lighter outlined treatment.
  if (col.format === "badge") {
    return (
      <td className="px-3 py-1.5">
        <span className={BADGE_PILL_CLASS}>{formatted}</span>
      </td>
    );
  }
  if (col.format === "status") {
    const toneClass = STATUS_PILL_CLASS[formatted.toLowerCase()] ?? "bg-gray-500 text-white";
    return (
      <td className="px-3 py-1.5">
        <span className={cn(STATUS_PILL_BASE_CLASS, toneClass)}>{formatted}</span>
      </td>
    );
  }
  if (col.format === "relative") {
    const title = formatAbsoluteTitle(value);
    return (
      <td className="px-3 py-1.5 text-slate-700" title={title}>
        {formatted}
      </td>
    );
  }
  if (col.format === "link" || col.href) {
    const href = col.href ? resolveHref(col.href, row) : String(value ?? "");
    return (
      <td className="px-3 py-1.5">
        <a
          href={href}
          target="_blank"
          rel="noopener noreferrer"
          className="text-sky-600 no-underline hover:!underline underline-offset-2 decoration-current"
        >
          {formatted || href}
        </a>
      </td>
    );
  }
  if (col.format === "code") {
    return (
      <td className="px-3 py-1.5">
        <code className="rounded bg-slate-100 px-1 py-0.5 font-mono text-[13px] text-slate-800">{formatted}</code>
      </td>
    );
  }
  return <td className="px-3 py-1.5 text-slate-700">{formatted}</td>;
}

function formatAbsoluteTitle(value: unknown): string | undefined {
  if (value == null) return undefined;
  if (typeof value === "string" && value.trim() !== "") {
    const parsed = Date.parse(value);
    if (Number.isFinite(parsed)) return formatTimestampInUserTimezone(new Date(parsed).toISOString());
  }
  const n = typeof value === "number" ? value : Number(value);
  if (!Number.isFinite(n)) return undefined;
  const ms = n > 1e12 ? n : n * 1000;
  return formatTimestampInUserTimezone(new Date(ms).toISOString());
}

type ActionDisabledReason = "no-perm" | "no-node" | "not-trigger" | "run-in-flight" | "submitting" | null;

function disabledReason({
  canRun,
  hasResolvedNode,
  isTrigger,
  runInFlight,
  submitting,
}: {
  canRun: boolean;
  hasResolvedNode: boolean;
  isTrigger: boolean;
  runInFlight: boolean;
  submitting: boolean;
}): ActionDisabledReason {
  if (!canRun) return "no-perm";
  if (!hasResolvedNode) return "no-node";
  if (!isTrigger) return "not-trigger";
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
    case "not-trigger":
      return "Only trigger nodes can be run from the console. Pick the trigger that starts your flow.";
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
  const isTrigger = resolved?.node.type === "TYPE_TRIGGER";
  const triggerNodeId = resolved?.node.id;
  // Per-row locking: a row's button is disabled by `runInFlight` only when
  // its own submission produced the in-flight run (i.e. the mapping points
  // back to this row's key). Other rows sharing the same trigger stay
  // clickable, matching the "lock only the affected row" model.
  const runInFlight = Boolean(
    triggerNodeId && lock.runInFlightIds.has(triggerNodeId) && lock.inFlightRowByTrigger.get(triggerNodeId) === rowKey,
  );
  const submitting = lock.pendingRowKeys.has(rowKey);
  const reason = disabledReason({
    canRun,
    hasResolvedNode: Boolean(resolved),
    isTrigger,
    runInFlight,
    submitting,
  });
  return {
    resolved,
    isTrigger,
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

  const { resolved, isTrigger, disabled, reason, tooltip } = useRowActionGate(action, rowKey);
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
        <span className="max-w-48 text-right text-[10px] text-red-600" data-testid={`${testId}-error`}>
          {error}
        </span>
      ) : null}
      {action.confirm ? (
        <RowActionConfirmDialog
          action={action}
          row={row}
          resolved={resolved}
          isTrigger={isTrigger}
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
      <Loader2 className="size-4 animate-spin text-slate-400" />
    </div>
  );
}
