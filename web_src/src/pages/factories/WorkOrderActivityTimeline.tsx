import type { FactoriesWorkOrder, FactoriesWorkOrderExecution } from "@/api-client";
import { Link } from "@/components/Link/link";
import { formatTimeAgo } from "@/lib/date";
import { cn } from "@/lib/utils";
import {
  Check,
  CircleDashed,
  CircleDot,
  CornerDownRight,
  GitBranch,
  Loader2,
  MinusCircle,
  UserRound,
  XCircle,
} from "lucide-react";
import {
  buildWorkOrderTimelineEvents,
  describeWorkOrderClosed,
  type WorkOrderTimelineEvent,
  type WorkOrderTimelineStep,
} from "./workOrderTimelineEvents";
import { getWorkOrderExecutionDisplayMeta, getWorkOrderExecutionRunHref } from "./workOrderExecutions";

interface WorkOrderTimelineProps {
  organizationId: string;
  order: FactoriesWorkOrder;
}

export function WorkOrderActivityTimeline({ organizationId, order }: WorkOrderTimelineProps) {
  const events = buildWorkOrderTimelineEvents(order);
  const closedLabel = describeWorkOrderClosed(order);

  if (events.length === 0 && !closedLabel) {
    return <p className="text-sm text-gray-500 dark:text-gray-400">No activity yet.</p>;
  }

  return (
    <ol className="relative space-y-0">
      {events.map((event, index) => (
        <TimelineItem
          key={event.id}
          event={event}
          organizationId={organizationId}
          isLast={index === events.length - 1 && !closedLabel}
        />
      ))}

      {closedLabel && order.updatedAt ? (
        <li className="relative flex gap-4 pb-2 pl-8">
          <TimelineMarker icon={CircleDot} className="text-gray-400" />
          <div className="min-w-0 flex-1 pb-8">
            <p className="text-sm text-gray-900 dark:text-gray-100">{closedLabel}</p>
            <time className="mt-1 block text-xs text-gray-500 dark:text-gray-400">
              {formatTimeAgo(new Date(order.updatedAt))}
            </time>
          </div>
        </li>
      ) : null}
    </ol>
  );
}

function TimelineItem({
  event,
  organizationId,
  isLast,
}: {
  event: WorkOrderTimelineEvent;
  organizationId: string;
  isLast: boolean;
}) {
  const Icon = event.kind === "created" ? UserRound : GitBranch;

  return (
    <li className="relative flex gap-4 pl-8">
      {!isLast ? (
        <span className="absolute left-[11px] top-6 bottom-0 w-px bg-gray-200 dark:bg-gray-700/70" aria-hidden />
      ) : null}
      <TimelineMarker icon={Icon} className={event.kind === "created" ? "text-gray-500" : "text-violet-500"} />
      <div className={cn("min-w-0 flex-1", isLast ? "pb-2" : "pb-8")}>
        {event.kind === "created" ? (
          <p className="text-sm text-gray-900 dark:text-gray-100">
            {event.actorName ? (
              <>
                <span className="font-semibold">{event.actorName}</span>{" "}
              </>
            ) : null}
            {event.title}
          </p>
        ) : (
          <>
            <p className="text-sm text-gray-900 dark:text-gray-100">{event.title}</p>
            {event.steps?.length ? (
              <ul className="mt-2 space-y-1">
                {event.steps.map((step) => (
                  <DispatchStepRow key={step.id} organizationId={organizationId} step={step} />
                ))}
              </ul>
            ) : null}
          </>
        )}
        {event.kind === "created" ? (
          <time className="mt-2 block text-xs text-gray-500 dark:text-gray-400">
            {formatTimeAgo(new Date(event.at))}
          </time>
        ) : null}
      </div>
    </li>
  );
}

function DispatchStepRow({ organizationId, step }: { organizationId: string; step: WorkOrderTimelineStep }) {
  const runHref = getWorkOrderExecutionRunHref(organizationId, step.execution);
  const linkClassName =
    "pointer-events-auto inline-flex w-fit max-w-full items-center gap-2 rounded-md px-1 py-0.5 text-sm text-gray-700 no-underline transition-colors hover:bg-gray-50 hover:text-gray-900 dark:text-gray-300 dark:hover:bg-gray-800/40 dark:hover:text-gray-100";
  const stepContent = (
    <>
      <StepStatusIcon execution={step.execution} />
      <span className="min-w-0">
        {step.stepName}
        {step.at ? (
          <>
            {" "}
            <time className="text-xs text-gray-500 dark:text-gray-400">{formatTimeAgo(new Date(step.at))}</time>
          </>
        ) : null}
      </span>
    </>
  );

  return (
    <li className="flex min-w-0 items-center gap-2 text-sm text-gray-700 dark:text-gray-300">
      <CornerDownRight className="h-3.5 w-3.5 shrink-0 text-gray-400 dark:text-gray-500" aria-hidden />
      {runHref ? (
        <Link href={runHref} className={linkClassName}>
          {stepContent}
        </Link>
      ) : (
        <span className="inline-flex w-fit max-w-full items-center gap-2">{stepContent}</span>
      )}
    </li>
  );
}

function StepStatusIcon({ execution }: { execution: FactoriesWorkOrderExecution }) {
  const meta = getWorkOrderExecutionDisplayMeta(execution);
  const iconClassName = "h-3.5 w-3.5 shrink-0";

  if (execution.result === "RESULT_PASSED") {
    return <Check className={cn(iconClassName, "text-emerald-500")} aria-label="Passed" />;
  }

  if (execution.result === "RESULT_FAILED") {
    return <XCircle className={cn(iconClassName, "text-red-500")} aria-label="Failed" />;
  }

  if (meta.isActive) {
    return <Loader2 className={cn(iconClassName, "animate-spin text-violet-500")} aria-label={meta.label} />;
  }

  if (execution.result === "RESULT_CANCELLED") {
    return <MinusCircle className={cn(iconClassName, "text-gray-400")} aria-label="Cancelled" />;
  }

  if (execution.state === "STATE_PENDING") {
    return <CircleDashed className={cn(iconClassName, "text-amber-500")} aria-label="Pending" />;
  }

  return <CircleDashed className={cn(iconClassName, "text-gray-400")} aria-label={meta.label} />;
}

function TimelineMarker({ icon: Icon, className }: { icon: typeof UserRound; className?: string }) {
  return (
    <span
      className={cn(
        "absolute left-0 flex h-6 w-6 items-center justify-center rounded-full border border-gray-200 bg-white dark:border-gray-700/70 dark:bg-gray-900",
        className,
      )}
    >
      <Icon className="h-3.5 w-3.5" aria-hidden />
    </span>
  );
}
