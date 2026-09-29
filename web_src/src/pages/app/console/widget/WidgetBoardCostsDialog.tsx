import { Link } from "react-router";

import { formatCompactTokenLabel } from "@/lib/formatTokenCount";
import { cn } from "@/lib/utils";
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Loader2 } from "lucide-react";

import { normalizeBoardLaneValue } from "../boardPanelContent";
import { applyTableWhere } from "./evalTableWhere";
import { resolveCellValue } from "./resolveCellValue";
import type { WidgetBoardRender } from "./types";

const COST_COL_HEADER_CLASSES = "border-b border-slate-200 pb-1.5 text-[11px] font-semibold text-slate-500 dark:border-gray-700 dark:text-gray-400 text-left";
const COST_CELL_CLASSES = "border-b border-slate-100 py-1.5 text-[13px] dark:border-gray-800";

function formatCostCents(costCents: unknown): string {
  const cents = typeof costCents === "number" ? costCents : Number(costCents ?? 0);
  if (!Number.isFinite(cents) || cents <= 0) return "\u2014";
  return `$${(cents / 100).toFixed(2)}`;
}

function formatTokens(totalTokens: unknown): string {
  const tokens = typeof totalTokens === "number" ? totalTokens : Number(totalTokens ?? 0);
  if (!Number.isFinite(tokens) || tokens <= 0) return "\u2014";
  return formatCompactTokenLabel(tokens);
}

function taskTitle(row: Record<string, unknown>, render: WidgetBoardRender): string {
  const raw = resolveCellValue(render.card.titleField, row);
  if (raw != null && String(raw).trim() !== "") return String(raw);
  const laneValue = resolveCellValue(render.groupBy, row);
  if (laneValue != null && String(laneValue).trim() !== "") return String(laneValue);
  const id = row.id;
  if (typeof id === "string" || typeof id === "number") return String(id);
  return "(no title)";
}

function taskKey(row: Record<string, unknown>): string {
  const id = row.id;
  if (typeof id === "string" && id.length >= 8) return id.slice(0, 8);
  if (typeof id === "string") return id;
  if (typeof id === "number") return String(id);
  return "";
}

interface WidgetBoardCostsDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  render: WidgetBoardRender;
  rows: unknown[];
  isLoading: boolean;
}

export function WidgetBoardCostsDialog({
  open,
  onOpenChange,
  render,
  rows,
  isLoading,
}: WidgetBoardCostsDialogProps) {
  const doneLane = render.lanes.find((l) => normalizeBoardLaneValue(l.value) === "done");

  if (!doneLane) {
    return null;
  }

  const recordRows = rows.filter((r): r is Record<string, unknown> => Boolean(r) && typeof r === "object" && !Array.isArray(r));
  const whereFiltered = render.where ? applyTableWhere(recordRows, render.where) : recordRows;

  const normalizedLaneValue = normalizeBoardLaneValue(doneLane.value);
  const doneRows = whereFiltered
    .filter((row) => {
      const groupValue = resolveCellValue(render.groupBy, row);
      return normalizeBoardLaneValue(groupValue) === normalizedLaneValue;
    })
    .sort((a, b) => {
      const costA = typeof a.costCents === "number" ? a.costCents : Number(a.costCents ?? 0);
      const costB = typeof b.costCents === "number" ? b.costCents : Number(b.costCents ?? 0);
      return costB - costA;
    });

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="w-full max-w-2xl sm:max-w-2xl" size="large">
        <DialogHeader>
          <DialogTitle>Costs \u2014 {doneLane.label || doneLane.value}</DialogTitle>
        </DialogHeader>
        <div className="overflow-y-auto max-h-[32rem]">
          {isLoading ? (
            <div className="flex items-center justify-center gap-2 py-8">
              <Loader2 className="size-4 animate-spin text-slate-400 dark:text-gray-400" />
              <span className="text-[13px] text-slate-500 dark:text-gray-400">Loading costs...</span>
            </div>
          ) : doneRows.length === 0 ? (
            <div className="flex flex-col items-center gap-2 py-8 text-[13px] text-slate-500 dark:text-gray-400">
              <p>No completed tasks with cost data.</p>
            </div>
          ) : (
            <table className="w-full" data-testid="costs-table">
              <thead>
                <tr className={COST_COL_HEADER_CLASSES}>
                  <th className="px-2">Task</th>
                  <th className="px-2">Title</th>
                  <th className="px-2 text-right">Cost</th>
                  <th className="px-2 text-right">Tokens</th>
                </tr>
              </thead>
              <tbody>
                {doneRows.map((row) => {
                  const id = row.id;
                  const idStr = typeof id === "string" ? id : typeof id === "number" ? String(id) : "";
                  return (
                    <tr
                      key={idStr}
                      data-testid="costs-row"
                      className={cn(
                        COST_CELL_CLASSES,
                        "cursor-pointer hover:bg-slate-50 dark:hover:bg-gray-850",
                      )}
                    >
                      <td className="px-2 font-mono text-[12px] text-slate-500 dark:text-gray-400">
                        <Link
                          to={`?run=${idStr}`}
                          className="text-inherit no-underline hover:underline"
                        >
                          {taskKey(row)}
                        </Link>
                      </td>
                      <td className="px-2 font-medium">
                        <Link
                          to={`?run=${idStr}`}
                          className="text-inherit no-underline hover:underline"
                        >
                          {taskTitle(row, render)}
                        </Link>
                      </td>
                      <td className="px-2 text-right tabular-nums text-slate-700 dark:text-gray-300">
                        <Link
                          to={`?run=${idStr}`}
                          className="text-inherit no-underline"
                        >
                          {formatCostCents(row.costCents)}
                        </Link>
                      </td>
                      <td className="px-2 text-right tabular-nums text-slate-600 dark:text-gray-400">
                        <Link
                          to={`?run=${idStr}`}
                          className="text-inherit no-underline"
                        >
                          {formatTokens(row.totalTokens)}
                        </Link>
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          )}
        </div>
      </DialogContent>
    </Dialog>
  );
}