import { Loader2 } from "lucide-react";
import type { CanvasesCanvasRun } from "@/api-client";
import { Timestamp } from "@/components/Timestamp";
import { Button } from "@/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { formatMinutesSecondsDuration } from "@/lib/duration";
import { cn } from "@/lib/utils";
import { calculateRunDuration } from "./runNodeDetailModel";
import { getRunStatus } from "./runPresentation";
import { RunStatusBadge } from "./RunStatusBadge";

function getActionTooltip(status: string) {
  switch (status) {
    case "running":
      return "Stop all running steps and cancel queued ones";
    case "cancelling":
      return "Cancelling all running steps and cancelling queued ones";
    default:
      return "Restart this whole run from trigger event";
  }
}

function getActionLabel(status: string) {
  switch (status) {
    case "running":
      return "Stop";
    case "cancelling":
      return "Cancelling";
    default:
      return "Rerun";
  }
}

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
  const duration = calculateRunDuration(run);
  const durationText = duration !== null ? formatMinutesSecondsDuration(duration) : "";
  const actionLabel = getActionLabel(status);
  const actionTooltip = getActionTooltip(status);
  const isStopAction = status === "running";

  return (
    <div className="sticky top-0 z-20 border-b border-slate-950/10 bg-white px-4 py-4 dark:border-gray-800 dark:bg-gray-950">
      <div className="flex flex-col gap-1.5">
        <div className="flex min-w-0 flex-wrap items-center gap-2">
          <RunStatusBadge status={status} />
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
              <span className="inline-flex">
                <Button
                  type="button"
                  variant="outline"
                  size="xs"
                  disabled={actionDisabled || actionPending}
                  onClick={onAction}
                  className={cn(
                    isStopAction &&
                      "border-red-200 text-red-600 hover:bg-red-50 hover:text-red-700 dark:border-red-900/70 dark:text-red-300 dark:hover:bg-red-950/50 dark:hover:text-red-200",
                  )}
                >
                  {actionPending ? <Loader2 className="h-3.5 w-3.5 shrink-0 animate-spin" /> : null}
                  {actionPending ? `${actionLabel}...` : actionLabel}
                </Button>
              </span>
            </TooltipTrigger>
            <TooltipContent side="bottom">{actionTooltip}</TooltipContent>
          </Tooltip>
        </div>
      </div>
    </div>
  );
}
