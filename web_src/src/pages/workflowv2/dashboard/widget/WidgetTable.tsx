import { useMemo, useState } from "react";
import { ExternalLink, Loader2, Play, RefreshCw, Square, Trash2 } from "lucide-react";

import { formatTimestampInUserTimezone } from "@/lib/timezone";
import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";

import { useDashboardContext, resolveDashboardNode } from "../DashboardContext";
import { applyTableWhere } from "./evalTableWhere";
import { interpolate } from "./fieldPath";
import { mergeTriggerParameters } from "./mergeTriggerPayload";
import { RowActionConfirmDialog } from "./RowActionConfirmDialog";
import { evaluateRowShow } from "./rowVisibility";
import { resolveCellValue } from "./resolveCellValue";
import { applyFilters } from "./widgetData";
import { formatValue } from "./widgetFormat";
import { WidgetTableActionLockProvider } from "./WidgetTableActionLock";
import { useWidgetTableActionLock } from "./WidgetTableActionLockContext";
import type { WidgetRowAction, WidgetTableRender } from "./types";

interface WidgetTableProps {
  render: WidgetTableRender;
  rows: unknown[];
  isLoading: boolean;
}

const STATUS_PILL_CLASS: Record<string, string> = {
  passed: "bg-emerald-100 text-emerald-700 ring-emerald-300",
  failed: "bg-red-100 text-red-700 ring-red-300",
  cancelled: "bg-slate-200 text-slate-600 ring-slate-300",
  running: "bg-sky-100 text-sky-700 ring-sky-300",
  pending: "bg-amber-100 text-amber-700 ring-amber-300",
  ready: "bg-emerald-100 text-emerald-700 ring-emerald-300",
  active: "bg-emerald-100 text-emerald-700 ring-emerald-300",
  idle: "bg-slate-100 text-slate-600 ring-slate-300",
};

const ACTION_ICONS = {
  play: Play,
  stop: Square,
  trash: Trash2,
  refresh: RefreshCw,
  "external-link": ExternalLink,
} as const;

export function WidgetTable({ render, rows, isLoading }: WidgetTableProps) {
  const ctx = useDashboardContext();
  const recordRows = useMemo(
    () => rows.filter((r): r is Record<string, unknown> => Boolean(r) && typeof r === "object" && !Array.isArray(r)),
    [rows],
  );

  const filtered = useMemo(() => {
    const afterWhere = applyTableWhere(recordRows, render.where);
    return applyFilters(afterWhere, render.filters);
  }, [recordRows, render.where, render.filters]);

  // Collect the unique trigger node ids referenced by this table's row actions
  // so the action lock can subscribe to the runs query only when needed.
  const triggerNodeIds = useMemo(() => {
    const ids = new Set<string>();
    for (const action of render.rowActions ?? []) {
      const resolved = resolveDashboardNode(ctx, action.node);
      if (resolved?.node.id && resolved.node.type === "TYPE_TRIGGER") ids.add(resolved.node.id);
    }
    return Array.from(ids);
  }, [render.rowActions, ctx]);

  if (isLoading) return <WidgetSpinner />;
  if (render.columns.length === 0) {
    return (
      <div className="p-4 text-center text-xs text-slate-500" data-testid="widget-table-no-columns">
        Configure columns in the panel editor. Pick a memory namespace to see available fields.
      </div>
    );
  }
  if (filtered.length === 0) {
    return (
      <div className="p-4 text-center text-xs text-slate-500" data-testid="widget-table-empty">
        {render.emptyMessage ?? "No data to display."}
      </div>
    );
  }

  return (
    <WidgetTableActionLockProvider triggerNodeIds={triggerNodeIds}>
      <div className="overflow-auto" data-testid="widget-table">
        <table className="w-full border-collapse text-xs">
          <thead className="bg-slate-50">
            <tr>
              {render.columns.map((col, i) => (
                <th
                  key={`${col.field}-${i}`}
                  className="border-b border-slate-200 px-3 py-1.5 text-left font-semibold text-slate-700"
                >
                  {col.label ?? col.field}
                </th>
              ))}
              {render.rowActions && render.rowActions.length > 0 ? (
                <th className="border-b border-slate-200 px-3 py-1.5 text-right font-semibold text-slate-700">
                  Actions
                </th>
              ) : null}
            </tr>
          </thead>
          <tbody>
            {filtered.map((row, idx) => (
              <tr
                key={(row.id as string | undefined) ?? idx}
                className="border-b border-slate-100 last:border-0 hover:bg-slate-50/60"
              >
                {render.columns.map((col, ci) => (
                  <Cell key={`${col.field}-${ci}`} col={col} row={row} />
                ))}
                {render.rowActions && render.rowActions.length > 0 ? (
                  <td className="px-3 py-1.5 text-right">
                    <div className="inline-flex items-center gap-1">
                      {render.rowActions
                        .filter((action) => evaluateRowShow(action.show, row))
                        .map((action, ai) => (
                          <RowActionButton key={ai} action={action} row={row} />
                        ))}
                    </div>
                  </td>
                ) : null}
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </WidgetTableActionLockProvider>
  );
}

function Cell({ col, row }: { col: WidgetTableRender["columns"][number]; row: Record<string, unknown> }) {
  const visible = evaluateRowShow(col.show, row);
  if (!visible) {
    return <td className="px-3 py-1.5 text-slate-300">—</td>;
  }
  const value = resolveCellValue(col.field, row);
  const formatted = formatValue(value, col.format);
  if (col.format === "status") {
    const classes = STATUS_PILL_CLASS[formatted.toLowerCase()] ?? "bg-slate-100 text-slate-600 ring-slate-300";
    return (
      <td className="px-3 py-1.5">
        <span className={cn("inline-flex rounded-full px-2 py-0.5 text-[10px] font-medium ring-1 ring-inset", classes)}>
          {formatted}
        </span>
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
    const href = col.href ? interpolate(col.href, row) : String(value ?? "");
    return (
      <td className="px-3 py-1.5">
        <a href={href} target="_blank" rel="noopener noreferrer" className="text-sky-600 underline underline-offset-2">
          {formatted || href}
        </a>
      </td>
    );
  }
  if (col.format === "code") {
    return (
      <td className="px-3 py-1.5">
        <code className="rounded bg-slate-100 px-1 py-0.5 font-mono text-[11px] text-slate-800">{formatted}</code>
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

function actionVariantClass(variant: WidgetRowAction["variant"]): string | undefined {
  if (variant === "danger") return "border-red-200 text-red-700 hover:bg-red-50";
  if (variant === "primary") return "border-sky-200 bg-sky-50 text-sky-800 hover:bg-sky-100";
  return undefined;
}

type ResolvedNode = NonNullable<ReturnType<typeof resolveDashboardNode>>;

function useRowActionFire({
  action,
  row,
  resolved,
  hookName,
  label,
  setConfirmOpen,
}: {
  action: WidgetRowAction;
  row: Record<string, unknown>;
  resolved: ResolvedNode | undefined;
  hookName: string;
  label: string;
  setConfirmOpen: (open: boolean) => void;
}) {
  const ctx = useDashboardContext();
  const lock = useWidgetTableActionLock();
  const [error, setError] = useState<string | undefined>();
  const [pending, setPending] = useState(false);

  const fire = async () => {
    if (!ctx?.onTriggerNode || !resolved?.node.id) return;
    const triggerNodeId = resolved.node.id;
    setError(undefined);
    setPending(true);
    lock.beginSubmission(triggerNodeId);
    try {
      const parameters = mergeTriggerParameters(resolved.node, hookName, action.template, row, action.payload);
      await ctx.onTriggerNode(triggerNodeId, {
        hookName,
        templateName: action.template,
        parameters,
        successLabel: label,
      });
      setConfirmOpen(false);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to trigger");
    } finally {
      setPending(false);
      lock.endSubmission(triggerNodeId);
    }
  };

  return { fire, error, pending };
}

function useRowActionGate(action: WidgetRowAction) {
  const ctx = useDashboardContext();
  const lock = useWidgetTableActionLock();
  const canRun = ctx?.canRunNodes ?? false;
  const resolved = resolveDashboardNode(ctx, action.node);
  const isTrigger = resolved?.node.type === "TYPE_TRIGGER";
  const runInFlight = Boolean(resolved?.node.id && lock.runInFlightIds.has(resolved.node.id));
  const reason = disabledReason({
    canRun,
    hasResolvedNode: Boolean(resolved),
    isTrigger,
    runInFlight,
    submitting: lock.submitting,
  });
  return {
    resolved,
    isTrigger,
    disabled: reason !== null,
    reason,
    tooltip: disabledTooltip(reason, action.node),
  };
}

function RowActionButton({ action, row }: { action: WidgetRowAction; row: Record<string, unknown> }) {
  const [confirmOpen, setConfirmOpen] = useState(false);

  const { resolved, isTrigger, disabled, reason, tooltip } = useRowActionGate(action);
  const label = action.label ?? "Run";
  const hookName = action.hook ?? "run";
  const Icon = action.icon ? ACTION_ICONS[action.icon] : undefined;
  const variantClass = actionVariantClass(action.variant);

  const { fire, error, pending } = useRowActionFire({ action, row, resolved, hookName, label, setConfirmOpen });

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
        size="sm"
        variant="outline"
        onClick={handleClick}
        disabled={disabled || pending}
        aria-disabled={disabled}
        title={tooltip}
        className={variantClass}
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
