import { Loader2, Square } from "lucide-react";
import type { CanvasesCanvasRun } from "@/api-client";
import { Timestamp } from "@/components/Timestamp";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { formatMinutesSecondsDuration } from "@/lib/duration";
import { cn } from "@/lib/utils";
import { calculateRunDuration } from "./runNodeDetailModel";
import { getRunStatus, RUN_STATUS_META } from "./runPresentation";

export function RunInspectorHeader({
  run,
  title,
  stepCount,
  actionPending,
  actionDisabled,
  onAction,
}: {
  run: CanvasesCanvasRun;
  title: string;
  stepCount: number;
  actionPending: boolean;
  actionDisabled: boolean;
  onAction: () => void;
}) {
  const status = getRunStatus(run);
  const meta = RUN_STATUS_META[status];
  const Icon = meta.icon;
  const duration = calculateRunDuration(run);
  const durationText = duration !== null ? formatMinutesSecondsDuration(duration) : "";
  const actionLabel = status === "running" ? "Stop" : "Rerun";
  const actionTooltip =
    status === "running"
      ? "Stop all running steps and cancel queued ones"
      : "Restart this whole run from trigger event";

  return (
    <div className="sticky top-0 z-20 border-b border-slate-950/10 bg-white px-4 py-4 dark:border-gray-800 dark:bg-gray-950">
      <div className="flex flex-col gap-1.5">
        <div className="flex min-w-0 flex-wrap items-center gap-2">
          <span
            className={cn(
              "inline-flex shrink-0 items-center gap-1.5 rounded-full px-2 py-0.5 text-xs font-medium ring-1 ring-inset",
              meta.badgeClassName,
            )}
          >
            <Icon className="h-3.5 w-3.5" />
            {meta.label}
          </span>
          <h2 className="min-w-0 flex-1 truncate text-base font-semibold leading-tight text-gray-900 dark:text-gray-100">
            {title}
          </h2>
        </div>
        <div className="flex items-center justify-between gap-2">
          <div className="flex flex-wrap items-center gap-x-1.5 gap-y-0.5 text-xs text-gray-600 dark:text-gray-400">
            {run.createdAt ? <Timestamp date={run.createdAt} display="relative" relativeStyle="abbreviated" /> : null}
            {durationText ? (
              <>
                <span className="text-gray-300" aria-hidden>
                  ·
                </span>
                <span>{durationText}</span>
              </>
            ) : null}
            <span className="text-gray-300" aria-hidden>
              ·
            </span>
            <span>
              {stepCount} {stepCount === 1 ? "step" : "steps"}
            </span>
          </div>
          <Tooltip>
            <TooltipTrigger asChild>
              <span className="shrink-0">
                <button
                  type="button"
                  disabled={actionDisabled || actionPending}
                  onClick={onAction}
                  className={cn(
                    "inline-flex shrink-0 items-center rounded border px-2 py-0.5 text-xs font-medium transition-colors disabled:cursor-not-allowed disabled:opacity-60",
                    status === "running"
                      ? "border-red-200 bg-white text-red-600 hover:bg-red-50 dark:border-red-900/70 dark:bg-gray-950 dark:text-red-300"
                      : "border-slate-200 bg-white text-slate-700 hover:bg-slate-50 hover:text-slate-900 dark:border-gray-700 dark:bg-gray-950 dark:text-gray-200",
                  )}
                >
                  <span className="inline-flex items-center gap-2">
                    {actionPending ? (
                      <Loader2 className="h-3.5 w-3.5 animate-spin" />
                    ) : status === "running" ? (
                      <Square className="h-3.5 w-3.5" />
                    ) : null}
                    <span>{actionPending ? `${actionLabel}...` : actionLabel}</span>
                  </span>
                </button>
              </span>
            </TooltipTrigger>
            <TooltipContent side="bottom">{actionTooltip}</TooltipContent>
          </Tooltip>
        </div>
      </div>
    </div>
  );
}
