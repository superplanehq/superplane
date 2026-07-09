import { useMemo } from "react";
import { ArrowDown, ArrowUp, Loader2, Minus, Trophy } from "lucide-react";

import { cn } from "@/lib/utils";

import { buildRankingData, type RankingRow, type WidgetRankingRender } from "./rankingData";
import { WidgetEmptyState } from "../WidgetEmptyState";
import { formatValue } from "./widgetFormat";

interface WidgetRankingProps {
  render: WidgetRankingRender;
  rows: unknown[];
  isLoading: boolean;
}

const HEAD_CELL_CLASS =
  "border-b border-slate-200 px-3 py-1.5 text-left text-[11px] font-semibold uppercase tracking-wide text-slate-500 dark:border-gray-600 dark:text-gray-400";

const ROW_CLASS =
  "border-b border-black/10 last:border-0 hover:bg-slate-50/60 dark:border-gray-800 dark:hover:bg-gray-800/60";

/**
 * Prototype ranking / leaderboard panel. Groups the incoming rows by
 * `render.groupField`, aggregates a metric per group, sorts into a ranking,
 * and — when `render.trend` is set — shows the metric's direction and percent
 * change vs the previous rolling window. Styled to match `WidgetTable`.
 */
export function WidgetRanking({ render, rows, isLoading }: WidgetRankingProps) {
  const recordRows = useMemo(
    () => rows.filter((r): r is Record<string, unknown> => Boolean(r) && typeof r === "object" && !Array.isArray(r)),
    [rows],
  );

  const ranked = useMemo(() => buildRankingData(recordRows, render), [recordRows, render]);
  const showTrend = Boolean(render.trend);

  if (isLoading) {
    return (
      <div className="flex h-full items-center justify-center p-4">
        <Loader2 className="size-4 animate-spin text-slate-400 dark:text-gray-500" />
      </div>
    );
  }

  if (ranked.length === 0) {
    return <WidgetEmptyState icon={Trophy} testId="widget-ranking-empty" message="No data to rank yet." />;
  }

  const groupHeader = render.groupLabel ?? "Name";
  const valueHeader = render.label ?? "Value";

  return (
    <div className="overflow-auto" data-testid="widget-ranking">
      <table className="w-full border-collapse text-[13px]">
        <thead>
          <tr>
            <th className={cn(HEAD_CELL_CLASS, "w-10 text-right tabular-nums")}>#</th>
            <th className={HEAD_CELL_CLASS}>{groupHeader}</th>
            <th className={cn(HEAD_CELL_CLASS, "text-right")}>{valueHeader}</th>
            {showTrend ? <th className={cn(HEAD_CELL_CLASS, "w-24 text-right")}>Trend</th> : null}
          </tr>
        </thead>
        <tbody>
          {ranked.map((row) => (
            <tr key={row.group} className={ROW_CLASS} data-testid="widget-ranking-row">
              <td className="px-3 py-1.5 text-right tabular-nums text-slate-400 dark:text-gray-500">{row.rank}</td>
              <td className="px-3 py-1.5 font-medium text-slate-700 dark:text-gray-200">{row.group}</td>
              <td className="px-3 py-1.5 text-right tabular-nums text-slate-700 dark:text-gray-300">
                {formatValue(row.value, render.format)}
              </td>
              {showTrend ? (
                <td className="px-3 py-1.5 text-right">
                  <TrendCell row={row} />
                </td>
              ) : null}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function TrendCell({ row }: { row: RankingRow }) {
  if (row.direction === "new") {
    return (
      <span className="inline-flex rounded-full bg-emerald-500/10 px-2 py-0.5 text-[11px] font-medium text-emerald-600 dark:text-emerald-400">
        New
      </span>
    );
  }

  const Icon = row.direction === "up" ? ArrowUp : row.direction === "down" ? ArrowDown : Minus;
  const tone =
    row.direction === "up"
      ? "text-emerald-600 dark:text-emerald-400"
      : row.direction === "down"
        ? "text-red-600 dark:text-red-400"
        : "text-slate-400 dark:text-gray-500";

  return (
    <span
      className={cn("inline-flex items-center justify-end gap-0.5 tabular-nums", tone)}
      data-direction={row.direction}
    >
      <Icon className="size-3.5" aria-hidden />
      {formatDeltaPct(row.deltaPct)}
    </span>
  );
}

/**
 * Render a fractional delta as a signed percent. Uses one decimal only when it
 * changes the displayed value so common cases stay compact (`+40%`, `-12%`).
 * `null` (no baseline) collapses to a neutral dash.
 */
function formatDeltaPct(deltaPct: number | null): string {
  if (deltaPct === null) return "—";
  const pct = deltaPct * 100;
  const rounded = Math.abs(pct) < 100 && !Number.isInteger(pct) ? Number(pct.toFixed(1)) : Math.round(pct);
  const sign = rounded > 0 ? "+" : "";
  return `${sign}${rounded}%`;
}
