import { ArrowDownRight, ArrowUpRight } from "lucide-react";

import { cn } from "@/lib/utils";
import { factoryCardClassName } from "@/pages/factories/pages/factoryPageLayoutStyles";
import { Sparkline } from "@/pages/app/console/widget/Sparkline";
import type { HealthSnapshot } from "@/pages/factories/verification/types";

interface HealthScoreCardProps {
  label: string;
  snapshot: HealthSnapshot;
}

/**
 * Health score in the console scorecard visual language: status dot, large
 * value, change chip, sparkline, and progress toward the target.
 */
export function HealthScoreCard({ label, snapshot }: HealthScoreCardProps) {
  const improving = snapshot.change > 0;
  const declining = snapshot.change < 0;
  const statusDotClass = improving
    ? "bg-emerald-500 dark:bg-emerald-400"
    : declining
      ? "bg-red-500 dark:bg-red-400"
      : "bg-slate-400 dark:bg-gray-500";
  const changeTextClass = improving
    ? "text-emerald-600 dark:text-emerald-400"
    : declining
      ? "text-red-600 dark:text-red-400"
      : "text-slate-500 dark:text-gray-400";
  const sparklineClass = improving
    ? "text-emerald-500 dark:text-emerald-400"
    : declining
      ? "text-red-500 dark:text-red-400"
      : "text-slate-400 dark:text-gray-500";
  const ChangeIcon = improving ? ArrowUpRight : ArrowDownRight;
  const progressPercent =
    snapshot.target && snapshot.target > 0 ? Math.min(100, (snapshot.score / snapshot.target) * 100) : null;

  return (
    <div className={cn(factoryCardClassName, "flex flex-col items-start justify-center gap-2 p-4")}>
      <div className="flex items-center gap-2">
        <span className={cn("inline-block h-2 w-2 rounded-full", statusDotClass)} aria-hidden />
        <span className="text-[12px] font-medium uppercase tracking-wide text-muted-foreground">{label}</span>
      </div>
      <div className="flex flex-wrap items-baseline gap-x-3 gap-y-1">
        <div className="flex items-baseline gap-0.5">
          <span className="text-4xl font-medium text-slate-900 dark:text-gray-100">{snapshot.score}</span>
          <span className="text-xl font-semibold text-slate-900 dark:text-gray-100">/100</span>
        </div>
        {snapshot.change !== 0 ? (
          <span className={cn("inline-flex items-center gap-1 text-sm font-medium tabular-nums", changeTextClass)}>
            <ChangeIcon className="size-4" aria-hidden />
            {improving ? "+" : ""}
            {snapshot.change}
          </span>
        ) : null}
      </div>
      {snapshot.series.length > 1 ? <Sparkline values={snapshot.series} className={sparklineClass} /> : null}
      {progressPercent != null ? (
        <div className="flex w-full flex-col gap-1">
          <div className="h-1.5 w-full overflow-hidden rounded-full bg-slate-200 dark:bg-gray-700" aria-hidden>
            <div
              className={cn("h-full rounded-full", improving ? "bg-emerald-500 dark:bg-emerald-400" : "bg-sky-500")}
              style={{ width: `${progressPercent}%` }}
            />
          </div>
          <span className="text-xs text-slate-500 dark:text-gray-400">
            {Math.round(progressPercent)}% of target {snapshot.target}
          </span>
        </div>
      ) : null}
    </div>
  );
}
