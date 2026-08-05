import { Text } from "@/components/Text/text";
import type { FactoriesWorkOrder, FactoriesWorkOrderEvent, FactoriesWorkOrderExecution } from "@/api-client";
import { Link } from "@/components/Link/link";
import { useOrganizationUsers } from "@/hooks/useOrganizationData";
import type { OrgUserDisplayLookup } from "@/lib/orgUserDisplay";
import { formatTimeAgo } from "@/lib/date";
import { appRunPath } from "@/lib/appPaths";
import { cn } from "@/lib/utils";
import { Fragment, useMemo, type ReactNode } from "react";
import {
  Check,
  CircleDashed,
  CircleDot,
  GitBranch,
  Loader2,
  MinusCircle,
  UserRound,
  UsersRound,
  XCircle,
} from "lucide-react";
import {
  buildWorkOrderTimelineView,
  buildWorkOrderUserDisplayLookup,
  buildWorkOrderUserNameLookup,
  formatStepExecutionDuration,
  type WorkOrderTimelineEvent,
  type WorkOrderTimelineEventKind,
  type WorkOrderTimelineStep,
} from "./lib/workOrderTimelineEvents";
import { getWorkOrderExecutionDisplayMeta, getWorkOrderExecutionRunHref } from "./lib/workOrderExecutions";
import { OrgUserReference } from "./OrgUserReference";

interface WorkOrderTimelineProps {
  organizationId: string;
  order: FactoriesWorkOrder;
  events?: FactoriesWorkOrderEvent[];
  eventsError?: Error | null;
  isLoading?: boolean;
  hasMoreEvents?: boolean;
  isLoadingMoreEvents?: boolean;
  onLoadMoreEvents?: () => void;
  onRetryEvents?: () => void;
}

export function WorkOrderActivityTimeline({
  organizationId,
  order,
  events,
  eventsError = null,
  isLoading = false,
  hasMoreEvents = false,
  isLoadingMoreEvents = false,
  onLoadMoreEvents,
  onRetryEvents,
}: WorkOrderTimelineProps) {
  const { data: users = [] } = useOrganizationUsers(organizationId);
  const resolveUserName = useMemo(() => buildWorkOrderUserNameLookup(users, order), [users, order]);
  const resolveUserDisplay = useMemo(() => buildWorkOrderUserDisplayLookup(users, order), [users, order]);
  const pendingView = renderTimelinePendingView({ events, eventsError, isLoading, onRetryEvents });

  if (pendingView) {
    return pendingView;
  }

  const timeline = buildWorkOrderTimelineView(events, resolveUserName);

  if (timeline.events.length === 0) {
    return <TimelineActivityEmpty />;
  }

  return (
    <WorkOrderTimelineList
      timeline={timeline}
      organizationId={organizationId}
      resolveUserDisplay={resolveUserDisplay}
      hasMoreEvents={hasMoreEvents}
      isLoadingMoreEvents={isLoadingMoreEvents}
      onLoadMoreEvents={onLoadMoreEvents}
    />
  );
}

function renderTimelinePendingView({
  events,
  eventsError,
  isLoading,
  onRetryEvents,
}: {
  events?: FactoriesWorkOrderEvent[];
  eventsError: Error | null;
  isLoading: boolean;
  onRetryEvents?: () => void;
}) {
  if (eventsError && !events?.length) {
    return <TimelineActivityError onRetryEvents={onRetryEvents} />;
  }

  if (isLoading && !events?.length) {
    return <TimelineActivityLoading />;
  }

  return null;
}

function TimelineActivityError({ onRetryEvents }: { onRetryEvents?: () => void }) {
  return (
    <div role="alert" className="rounded-lg border border-red-300 px-4 py-3 dark:border-red-800">
      <Text className="text-red-500 dark:text-red-400">Failed to load activity.</Text>
      {onRetryEvents ? (
        <button
          type="button"
          onClick={onRetryEvents}
          className="mt-2 text-sm font-medium text-violet-600 hover:text-violet-700 dark:text-violet-400 dark:hover:text-violet-300"
        >
          Try again
        </button>
      ) : null}
    </div>
  );
}

function TimelineActivityLoading() {
  return <Text className="text-sm text-gray-500 dark:text-gray-400">Loading activity…</Text>;
}

function TimelineActivityEmpty() {
  return <p className="text-sm text-gray-500 dark:text-gray-400">No activity yet.</p>;
}

function WorkOrderTimelineList({
  timeline,
  organizationId,
  resolveUserDisplay,
  hasMoreEvents,
  isLoadingMoreEvents,
  onLoadMoreEvents,
}: {
  timeline: ReturnType<typeof buildWorkOrderTimelineView>;
  organizationId: string;
  resolveUserDisplay: OrgUserDisplayLookup;
  hasMoreEvents: boolean;
  isLoadingMoreEvents: boolean;
  onLoadMoreEvents?: () => void;
}) {
  return (
    <ol className="relative space-y-0">
      {hasMoreEvents ? (
        <li className="relative pb-4 pl-8">
          <button
            type="button"
            onClick={onLoadMoreEvents}
            disabled={isLoadingMoreEvents || !onLoadMoreEvents}
            className="text-sm font-medium text-violet-600 hover:text-violet-700 disabled:cursor-not-allowed disabled:opacity-60 dark:text-violet-400 dark:hover:text-violet-300"
          >
            {isLoadingMoreEvents ? "Loading…" : "Load more events"}
          </button>
        </li>
      ) : null}

      {timeline.events.map((event, index) => (
        <TimelineItem
          key={event.id}
          event={event}
          organizationId={organizationId}
          resolveUserDisplay={resolveUserDisplay}
          isLast={index === timeline.events.length - 1}
        />
      ))}
    </ol>
  );
}

function TimelineItem({
  event,
  organizationId,
  resolveUserDisplay,
  isLast,
}: {
  event: WorkOrderTimelineEvent;
  organizationId: string;
  resolveUserDisplay: OrgUserDisplayLookup;
  isLast: boolean;
}) {
  const { icon: Icon } = getTimelineEventPresentation(event.kind);

  return (
    <li className="relative flex gap-4 pl-8">
      {!isLast ? (
        <span className="absolute left-[11px] top-6 bottom-0 w-px bg-gray-200 dark:bg-gray-700/70" aria-hidden />
      ) : null}
      <TimelineMarker icon={Icon} />
      <div className={cn("min-w-0 flex-1", isLast ? "pb-2" : "pb-8")}>
        <TimelineItemContent
          event={event}
          organizationId={organizationId}
          resolveUserDisplay={resolveUserDisplay}
        />
      </div>
    </li>
  );
}

function TimelineItemContent({
  event,
  organizationId,
  resolveUserDisplay,
}: {
  event: WorkOrderTimelineEvent;
  organizationId: string;
  resolveUserDisplay: OrgUserDisplayLookup;
}) {
  const isUserActionEvent = event.kind === "created" || event.kind === "assigned" || event.kind === "closed";

  if (!isUserActionEvent) {
    return (
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
    );
  }

  return (
    <>
      <UserActionEventDescription
        event={event}
        organizationId={organizationId}
        resolveUserDisplay={resolveUserDisplay}
      />
      <time className="mt-2 block text-xs text-gray-500 dark:text-gray-400">
        {formatTimeAgo(new Date(event.at))}
      </time>
    </>
  );
}

function UserActionEventDescription({
  event,
  organizationId,
  resolveUserDisplay,
}: {
  event: WorkOrderTimelineEvent;
  organizationId: string;
  resolveUserDisplay: OrgUserDisplayLookup;
}) {
  const actorDisplay = resolveUserDisplay(event.actorUserId, event.actorName);
  const isAutomationCreated = event.kind === "created" && Boolean(event.sourceRunId);
  const showSomeoneFallback =
    (event.kind === "created" || event.kind === "closed") && !actorDisplay && !isAutomationCreated;

  return (
    <p className="flex flex-wrap items-center gap-x-1 gap-y-1 text-sm text-gray-900 dark:text-gray-100">
      {actorDisplay ? (
        <OrgUserReference display={actorDisplay} size="sm" emphasizeName />
      ) : showSomeoneFallback ? (
        <span className="font-semibold">Someone</span>
      ) : null}
      {event.kind === "assigned" && event.assigneeChange ? (
        <AssigneeChangeDescription
          actorUserId={event.actorUserId}
          assigneeChange={event.assigneeChange}
          resolveUserDisplay={resolveUserDisplay}
        />
      ) : isAutomationCreated ? (
        <AutomationCreatedDescription event={event} organizationId={organizationId} />
      ) : (
        <span>{event.title}</span>
      )}
    </p>
  );
}

function AutomationCreatedDescription({
  event,
  organizationId,
}: {
  event: WorkOrderTimelineEvent;
  organizationId: string;
}) {
  const runHref =
    event.sourceAppId && event.sourceRunId ? appRunPath(organizationId, event.sourceAppId, event.sourceRunId) : null;

  return (
    <span className="inline-flex flex-wrap items-center gap-x-1 gap-y-1">
      <span>{event.title}</span>
      {runHref ? (
        <Link href={runHref} className={timelineRunLinkClassName}>
          <GitBranch className="h-3.5 w-3.5 shrink-0" aria-hidden />
          run
        </Link>
      ) : (
        <span>run</span>
      )}
    </span>
  );
}

const timelineRunLinkClassName =
  "pointer-events-auto inline-flex w-fit max-w-full items-center gap-1 rounded-md px-1 py-0.5 text-sm text-gray-700 underline decoration-gray-300 underline-offset-2 transition-colors hover:bg-gray-50 hover:text-gray-900 hover:decoration-gray-500 dark:text-gray-300 dark:decoration-gray-600 dark:hover:bg-gray-800/40 dark:hover:text-gray-100 dark:hover:decoration-gray-400";

function AssigneeChangeDescription({
  actorUserId,
  assigneeChange,
  resolveUserDisplay,
}: {
  actorUserId?: string;
  assigneeChange: NonNullable<WorkOrderTimelineEvent["assigneeChange"]>;
  resolveUserDisplay: OrgUserDisplayLookup;
}) {
  const { assignedUserIds, unassignedUserIds } = assigneeChange;
  const parts: ReactNode[] = [];

  const assignedDescription = buildAssignedChangeDescription(actorUserId, assignedUserIds, resolveUserDisplay);
  if (assignedDescription) {
    parts.push(<span key="assigned">{assignedDescription}</span>);
  }

  if (unassignedUserIds.length > 0) {
    parts.push(
      <span key="unassigned" className="inline-flex flex-wrap items-center gap-x-1 gap-y-1">
        unassigned <InlineUserList userIds={unassignedUserIds} resolveUserDisplay={resolveUserDisplay} />
      </span>,
    );
  }

  if (parts.length === 0) {
    return <span>updated assignees</span>;
  }

  return (
    <>
      {parts.map((part, index) => (
        <Fragment key={index}>
          {index > 0 ? <span> and </span> : null}
          {part}
        </Fragment>
      ))}
    </>
  );
}

function buildAssignedChangeDescription(
  actorUserId: string | undefined,
  assignedUserIds: string[],
  resolveUserDisplay: OrgUserDisplayLookup,
): ReactNode | null {
  if (assignedUserIds.length === 0) {
    return null;
  }

  const assignedOthers = actorUserId ? assignedUserIds.filter((userId) => userId !== actorUserId) : assignedUserIds;
  const selfAssigned = Boolean(actorUserId && assignedUserIds.includes(actorUserId));

  if (selfAssigned && assignedOthers.length === 0) {
    return <span>self-assigned</span>;
  }

  if (selfAssigned && assignedOthers.length > 0) {
    return (
      <span className="inline-flex flex-wrap items-center gap-x-1 gap-y-1">
        self-assigned and assigned <InlineUserList userIds={assignedOthers} resolveUserDisplay={resolveUserDisplay} />
      </span>
    );
  }

  return (
    <span className="inline-flex flex-wrap items-center gap-x-1 gap-y-1">
      assigned <InlineUserList userIds={assignedUserIds} resolveUserDisplay={resolveUserDisplay} />
    </span>
  );
}

function InlineUserList({
  userIds,
  resolveUserDisplay,
}: {
  userIds: string[];
  resolveUserDisplay: OrgUserDisplayLookup;
}) {
  if (userIds.length === 0) {
    return null;
  }

  if (userIds.length === 1) {
    return <OrgUserReference display={resolveUserDisplay(userIds[0])} size="sm" emphasizeName />;
  }

  if (userIds.length === 2) {
    return (
      <>
        <OrgUserReference display={resolveUserDisplay(userIds[0])} size="sm" emphasizeName />
        <span> and </span>
        <OrgUserReference display={resolveUserDisplay(userIds[1])} size="sm" emphasizeName />
      </>
    );
  }

  return (
    <>
      {userIds.slice(0, -1).map((userId, index) => (
        <Fragment key={userId}>
          {index > 0 ? <span>, </span> : null}
          <OrgUserReference display={resolveUserDisplay(userId)} size="sm" emphasizeName />
        </Fragment>
      ))}
      <span>, and </span>
      <OrgUserReference display={resolveUserDisplay(userIds[userIds.length - 1])} size="sm" emphasizeName />
    </>
  );
}

function DispatchStepRow({ organizationId, step }: { organizationId: string; step: WorkOrderTimelineStep }) {
  const runHref = getWorkOrderExecutionRunHref(organizationId, step.execution);
  const durationLabel = formatStepExecutionDuration(step);
  const linkClassName = timelineRunLinkClassName;
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
        {durationLabel ? <span className="text-xs text-gray-500 dark:text-gray-400"> · {durationLabel}</span> : null}
      </span>
    </>
  );

  return (
    <li className="flex min-w-0 items-center gap-2 text-sm text-gray-700 dark:text-gray-300">
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

function TimelineMarker({ icon: Icon }: { icon: typeof UserRound }) {
  return (
    <span className="absolute left-0 flex h-6 w-6 items-center justify-center rounded-full border border-gray-200 bg-white dark:border-gray-700/70 dark:bg-gray-900">
      <Icon className="h-3.5 w-3.5 text-gray-500 dark:text-gray-400" aria-hidden />
    </span>
  );
}

type TimelineMarkerIcon = typeof UserRound;

function getTimelineEventPresentation(kind: WorkOrderTimelineEventKind): {
  icon: TimelineMarkerIcon;
} {
  switch (kind) {
    case "created":
      return { icon: UserRound };
    case "assigned":
      return { icon: UsersRound };
    case "dispatched":
      return { icon: GitBranch };
    case "closed":
      return { icon: CircleDot };
  }
}
