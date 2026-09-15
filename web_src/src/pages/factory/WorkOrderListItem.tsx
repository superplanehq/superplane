import { AlertTriangle, CheckCircle2, GitPullRequest, Loader2, PenLine, XCircle } from "lucide-react";

import { formatTimeAgo } from "@/lib/date";
import { cn } from "@/lib/utils";

import { factoryCardClassName, mutedTextClassName } from "./factoryStyles";
import type { PullRequestRef, WorkOrder, WorkOrderState } from "./factoryTypes";

const STATE_META = {
  draft: { label: "Draft", icon: PenLine, className: "bg-slate-100 text-gray-700 dark:bg-gray-800 dark:text-gray-300" },
  ready: {
    label: "Ready",
    icon: Loader2,
    className: "bg-blue-100 text-blue-800 dark:bg-blue-950/70 dark:text-blue-300",
  },
  successful: {
    label: "Successful",
    icon: CheckCircle2,
    className: "bg-emerald-100 text-emerald-800 dark:bg-emerald-950/70 dark:text-emerald-300",
  },
  unsuccessful: {
    label: "Unsuccessful",
    icon: XCircle,
    className: "bg-red-100 text-red-800 dark:bg-red-950/70 dark:text-red-300",
  },
} as const satisfies Record<WorkOrderState, { label: string; icon: typeof PenLine; className: string }>;

export function WorkOrderStateBadge({ state }: { state: WorkOrderState }) {
  const meta = STATE_META[state];
  const Icon = meta.icon;
  return (
    <span className={cn("inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-xs font-medium", meta.className)}>
      <Icon className="size-3.5" aria-hidden />
      {meta.label}
    </span>
  );
}

export function PullRequestLink({ pullRequest }: { pullRequest: PullRequestRef }) {
  return (
    <a
      href={pullRequest.url}
      target="_blank"
      rel="noreferrer"
      className={cn(
        "inline-flex items-center gap-1 underline decoration-slate-300 underline-offset-4",
        "hover:text-slate-900 dark:decoration-gray-600 dark:hover:text-gray-100",
      )}
    >
      <GitPullRequest className="size-3.5" aria-hidden />
      {pullRequest.repository}#{pullRequest.number}
    </a>
  );
}

interface WorkOrderListItemProps {
  workOrder: WorkOrder;
  onOpen: (workOrder: WorkOrder) => void;
}

/**
 * One row in a Work Order list. Shared by the Overview and Work Orders tabs so
 * a Work Order reads identically wherever it appears.
 */
export function WorkOrderListItem({ workOrder, onOpen }: WorkOrderListItemProps) {
  const primaryPullRequest = workOrder.pullRequests[0];
  const extraPullRequests = workOrder.pullRequests.length - 1;

  return (
    <li className={cn("px-4 py-3", factoryCardClassName)}>
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div className="min-w-0">
          <button
            type="button"
            onClick={() => onOpen(workOrder)}
            className="text-left text-sm font-medium text-slate-900 underline-offset-4 hover:underline dark:text-gray-100"
          >
            {workOrder.title}
          </button>

          {workOrder.attention ? (
            <p className="mt-1 flex items-start gap-1.5 text-sm text-amber-800 dark:text-amber-300">
              <AlertTriangle className="mt-0.5 size-3.5 shrink-0" aria-hidden />
              {workOrder.attention.reason}
            </p>
          ) : (
            workOrder.activity && <p className={cn("mt-1 text-sm", mutedTextClassName)}>{workOrder.activity}</p>
          )}

          <div className={cn("mt-1.5 flex flex-wrap items-center gap-x-3 gap-y-1 text-xs", mutedTextClassName)}>
            {workOrder.currentAutomation && <span>{workOrder.currentAutomation}</span>}
            {primaryPullRequest && <PullRequestLink pullRequest={primaryPullRequest} />}
            {extraPullRequests > 0 && <span>+{extraPullRequests} more</span>}
            <span>
              {workOrder.attention
                ? `Waiting ${formatTimeAgo(new Date(workOrder.attention.since), false)}`
                : `Updated ${formatTimeAgo(new Date(workOrder.updatedAt))}`}
            </span>
          </div>
        </div>
        <WorkOrderStateBadge state={workOrder.state} />
      </div>
    </li>
  );
}
