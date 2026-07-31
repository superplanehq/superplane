import { CornerLeftUp, Loader2 } from "lucide-react";
import { Link, useParams } from "react-router-dom";
import type { CanvasesCanvasRun } from "@/api-client";
import { Timestamp } from "@/components/Timestamp";
import { Button } from "@/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { appRunPath } from "@/lib/appPaths";
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
  organizationId,
  actionPending,
  actionDisabled,
  onAction,
}: {
  run: CanvasesCanvasRun;
  title: string;
  stepCount: number;
  organizationId?: string;
  actionPending: boolean;
  actionDisabled: boolean;
  onAction: () => void;
}) {
  const { organizationId: routeOrganizationId } = useParams<{ organizationId: string }>();
  const resolvedOrganizationId = organizationId ?? routeOrganizationId;
  const parentRun = run.parent;
  const parentRunHref =
    parentRun?.id && parentRun.canvasId && resolvedOrganizationId
      ? appRunPath(resolvedOrganizationId, parentRun.canvasId, parentRun.id)
      : null;
  const status = getRunStatus(run);
  const duration = calculateRunDuration(run);
  const durationText = duration !== null ? formatMinutesSecondsDuration(duration) : "";
  const actionLabel = getActionLabel(status);
  const actionTooltip = getActionTooltip(status);
  const isStopAction = status === "running";

  return (
    <div className="sticky top-0 z-20 border-b border-edge-subtle bg-surface-raised px-4 py-4">
      <div className="flex flex-col gap-1.5">
        <div className="flex min-w-0 flex-wrap items-center gap-2">
          <RunStatusBadge status={status} />
          <h2 className="min-w-0 flex-1 truncate text-base font-semibold leading-tight text-content-primary">
            {title}
          </h2>
        </div>
        {parentRunHref ? (
          <Link
            to={parentRunHref}
            className="inline-flex w-fit items-center gap-1 text-xs font-medium text-content-secondary underline decoration-edge-default underline-offset-2 transition-colors hover:text-content-primary hover:decoration-edge-strong"
          >
            <CornerLeftUp className="h-3.5 w-3.5 shrink-0" aria-hidden />
            See parent
          </Link>
        ) : null}
        <div className="flex items-center justify-between gap-2">
          <div className="flex flex-wrap items-center gap-x-1.5 gap-y-0.5 text-xs text-content-secondary">
            {run.createdAt ? <Timestamp date={run.createdAt} display="relative" relativeStyle="abbreviated" /> : null}
            {durationText ? (
              <>
                <span className="text-content-muted" aria-hidden>
                  ·
                </span>
                <span>{durationText}</span>
              </>
            ) : null}
            <span className="text-content-muted" aria-hidden>
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
