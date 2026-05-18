import { useMemo } from "react";
import { Loader2 } from "lucide-react";

import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";

import { useDashboardContext, resolveDashboardNode } from "../DashboardContext";
import { DASHBOARD_EXECUTION_ACTION_EVENT, DASHBOARD_TRIGGER_NODE_EVENT } from "../dashboardEvents";
import { getValueAtPath, interpolate } from "./fieldPath";
import { evaluateShow } from "./showExpression";
import { applyFilters } from "./widgetData";
import { formatValue } from "./widgetFormat";
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
};

export function WidgetTable({ render, rows, isLoading }: WidgetTableProps) {
  const filtered = useMemo(() => applyFilters(rows, render.filters), [rows, render.filters]);

  if (isLoading) return <WidgetSpinner />;
  if (filtered.length === 0) {
    return (
      <div className="p-4 text-center text-xs text-slate-500" data-testid="widget-table-empty">
        {render.emptyMessage ?? "No data to display."}
      </div>
    );
  }

  return (
    <div className="overflow-auto" data-testid="widget-table">
      <table className="w-full border-collapse text-xs">
        <thead className="bg-slate-50">
          <tr>
            {render.columns.map((col) => (
              <th
                key={col.field}
                className="border-b border-slate-200 px-3 py-1.5 text-left font-semibold text-slate-700"
              >
                {col.label ?? col.field}
              </th>
            ))}
            {render.rowActions && render.rowActions.length > 0 ? (
              <th className="border-b border-slate-200 px-3 py-1.5 text-right font-semibold text-slate-700">Actions</th>
            ) : null}
          </tr>
        </thead>
        <tbody>
          {filtered.map((row, idx) => (
            <tr key={idx} className="border-b border-slate-100 last:border-0 hover:bg-slate-50/60">
              {render.columns.map((col) => {
                const value = getValueAtPath(row, col.field);
                const visible = evaluateShow(col.show, row);
                if (!visible) {
                  return (
                    <td key={col.field} className="px-3 py-1.5 text-slate-300">
                      —
                    </td>
                  );
                }
                const formatted = formatValue(value, col.format);
                if (col.format === "status") {
                  const classes =
                    STATUS_PILL_CLASS[formatted.toLowerCase()] ?? "bg-slate-100 text-slate-600 ring-slate-300";
                  return (
                    <td key={col.field} className="px-3 py-1.5">
                      <span
                        className={cn(
                          "inline-flex rounded-full px-2 py-0.5 text-[10px] font-medium ring-1 ring-inset",
                          classes,
                        )}
                      >
                        {formatted}
                      </span>
                    </td>
                  );
                }
                if (col.format === "link" || col.href) {
                  const href = col.href ? interpolate(col.href, row) : String(value ?? "");
                  return (
                    <td key={col.field} className="px-3 py-1.5">
                      <a
                        href={href}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="text-sky-600 underline underline-offset-2"
                      >
                        {formatted || href}
                      </a>
                    </td>
                  );
                }
                if (col.format === "code") {
                  return (
                    <td key={col.field} className="px-3 py-1.5">
                      <code className="rounded bg-slate-100 px-1 py-0.5 font-mono text-[11px] text-slate-800">
                        {formatted}
                      </code>
                    </td>
                  );
                }
                return (
                  <td key={col.field} className="px-3 py-1.5 text-slate-700">
                    {formatted}
                  </td>
                );
              })}
              {render.rowActions && render.rowActions.length > 0 ? (
                <td className="px-3 py-1.5 text-right">
                  <div className="inline-flex items-center gap-1">
                    {render.rowActions
                      .filter((action) => evaluateShow(action.show, row))
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
  );
}

function RowActionButton({ action, row }: { action: WidgetRowAction; row: unknown }) {
  const ctx = useDashboardContext();
  const targetField = action.target ?? defaultTargetField(action);
  const targetValue = targetField ? String(getValueAtPath(row, targetField) ?? "") : undefined;
  // All four row action kinds map to `canvases:update` on the backend
  // (InvokeNodeTriggerHook / InvokeNodeExecutionHook). Mirror that here so
  // viewers without the permission see disabled buttons instead of clicks
  // that fail with PermissionDenied.
  const canRun = ctx?.canRunNodes ?? false;

  const handleClick = () => {
    if (!targetValue || !canRun) return;
    if (action.kind === "trigger") {
      const resolved = resolveDashboardNode(ctx, targetValue);
      if (!resolved) return;
      if (ctx?.onTriggerNode) {
        ctx.onTriggerNode(resolved.node.id!, { templateName: action.triggerName });
        return;
      }
      window.dispatchEvent(
        new CustomEvent(DASHBOARD_TRIGGER_NODE_EVENT, {
          detail: { nodeId: resolved.node.id, triggerName: action.triggerName },
        }),
      );
      return;
    }
    // approve / cancel / push-through go through an execution-hook event for now.
    window.dispatchEvent(
      new CustomEvent(DASHBOARD_EXECUTION_ACTION_EVENT, {
        detail: { executionId: targetValue, kind: action.kind },
      }),
    );
  };

  const label = action.label ?? defaultActionLabel(action.kind);
  const disabled = !targetValue || !canRun;
  return (
    <Button
      type="button"
      size="sm"
      variant="outline"
      onClick={handleClick}
      disabled={disabled}
      aria-disabled={disabled}
      title={canRun ? undefined : "You do not have permission to run actions in this canvas"}
      data-testid={`widget-row-action-${action.kind}`}
    >
      {label}
    </Button>
  );
}

function defaultTargetField(action: WidgetRowAction): string {
  if (action.kind === "trigger") return "nodeId";
  return "id";
}

function defaultActionLabel(kind: WidgetRowAction["kind"]): string {
  switch (kind) {
    case "trigger":
      return "Trigger";
    case "approve":
      return "Approve";
    case "cancel":
      return "Cancel";
    case "push-through":
      return "Push through";
    default:
      return "Run";
  }
}

function WidgetSpinner() {
  return (
    <div className="flex h-full items-center justify-center p-4">
      <Loader2 className="size-4 animate-spin text-slate-400" />
    </div>
  );
}
