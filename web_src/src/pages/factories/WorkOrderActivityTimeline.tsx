import { Text } from "@/components/Text/text";
import type { FactoriesWorkOrder, FactoriesWorkOrderEvent, FactoriesWorkOrderExecution } from "@/api-client";
import { Link } from "@/components/Link/link";
import { useOrganizationUsers } from "@/hooks/useOrganizationData";
import type { OrgUserDisplayLookup } from "@/lib/orgUserDisplay";
import { formatTimeAgo } from "@/lib/date";
import { cn } from "@/lib/utils";
import { useMemo } from "react";
import {
  Check,
  CircleDashed,
  CircleDot,
  FileText,
  GitBranch,
  Loader2,
  MessageSquareText,
  MinusCircle,
  ShieldCheck,
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
import { factoryAppRunPath } from "./lib/factoryPagePaths";
import { OrgUserReference } from "./OrgUserReference";
import { ArtifactTimelineBody, CommentTimelineBody, TimelineAutomationActor } from "./timeline";
import { AssigneeChangeDescription } from "./workOrderTimelineAssignee";

interface WorkOrderTimelineProps {
  organizationId: string;
  factoryId: string;
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
  factoryId,
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
      factoryId={factoryId}
      orderId={order.id}
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
  factoryId,
  orderId,
  resolveUserDisplay,
  hasMoreEvents,
  isLoadingMoreEvents,
  onLoadMoreEvents,
}: {
  timeline: ReturnType<typeof buildWorkOrderTimelineView>;
  organizationId: string;
  factoryId: string;
  orderId?: string;
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
          factoryId={factoryId}
          orderId={orderId}
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
  factoryId,
  orderId,
  resolveUserDisplay,
  isLast,
}: {
  event: WorkOrderTimelineEvent;
  organizationId: string;
  factoryId: string;
  orderId?: string;
  resolveUserDisplay: OrgUserDisplayLookup;
  isLast: boolean;
}) {
  const { icon: Icon } = getTimelineEventPresentation(event.kind);

  return (
    <li className="relative flex gap-4 pl-8">
      {!isLast ? (
        <span className="absolute left-2.75 top-6 bottom-0 w-px bg-gray-200 dark:bg-gray-700/70" aria-hidden />
      ) : null}
      <TimelineMarker icon={Icon} />
      <div className={cn("min-w-0 flex-1", isLast ? "pb-2" : "pb-8")}>
        <TimelineItemContent
          event={event}
          organizationId={organizationId}
          factoryId={factoryId}
          orderId={orderId}
          resolveUserDisplay={resolveUserDisplay}
        />
      </div>
    </li>
  );
}

const USER_ACTION_EVENT_KINDS: WorkOrderTimelineEventKind[] = ["created", "assigned", "statusChanged", "closed"];

function TimelineItemContent({
  event,
  organizationId,
  factoryId,
  orderId,
  resolveUserDisplay,
}: {
  event: WorkOrderTimelineEvent;
  organizationId: string;
  factoryId: string;
  orderId?: string;
  resolveUserDisplay: OrgUserDisplayLookup;
}) {
  const actorDisplay = resolveUserDisplay(event.actorUserId, event.actorName);

  if (event.kind === "commented") {
    return (
      <>
        <CommentTimelineBody event={event} actorDisplay={actorDisplay} />
        <time className="mt-2 block text-xs text-gray-500 dark:text-gray-400">{formatTimeAgo(new Date(event.at))}</time>
      </>
    );
  }

  if (event.kind === "artifactAdded") {
    return (
      <>
        <ArtifactTimelineBody event={event} actorDisplay={actorDisplay} />
        <time className="mt-2 block text-xs text-gray-500 dark:text-gray-400">{formatTimeAgo(new Date(event.at))}</time>
      </>
    );
  }

  if (USER_ACTION_EVENT_KINDS.includes(event.kind)) {
    return (
      <>
        <UserActionEventDescription
          event={event}
          organizationId={organizationId}
          factoryId={factoryId}
          orderId={orderId}
          resolveUserDisplay={resolveUserDisplay}
        />
        <time className="mt-2 block text-xs text-gray-500 dark:text-gray-400">{formatTimeAgo(new Date(event.at))}</time>
      </>
    );
  }

  return (
    <>
      <p className="text-sm text-gray-900 dark:text-gray-100">{event.title}</p>
      {event.steps?.length ? (
        <ul className="mt-2 space-y-1">
          {event.steps.map((step) => (
            <DispatchStepRow
              key={step.id}
              organizationId={organizationId}
              factoryId={factoryId}
              orderId={orderId}
              step={step}
            />
          ))}
        </ul>
      ) : null}
    </>
  );
}

// Attribution kinds we may show for an event. Order matters (highest priority
// first): a matched org user always wins; otherwise we look for an app-run
// source (main), then a factory-line automation actor (ours), and finally
// fall back to a generic "Someone" label.
const AUTOMATION_ATTRIBUTABLE_KINDS: WorkOrderTimelineEventKind[] = ["created", "statusChanged", "closed"];
const SOMEONE_FALLBACK_KINDS: WorkOrderTimelineEventKind[] = ["created", "statusChanged", "closed"];

function UserActionEventDescription({
  event,
  organizationId,
  factoryId,
  orderId,
  resolveUserDisplay,
}: {
  event: WorkOrderTimelineEvent;
  organizationId: string;
  factoryId: string;
  orderId?: string;
  resolveUserDisplay: OrgUserDisplayLookup;
}) {
  const actorDisplay = resolveUserDisplay(event.actorUserId, event.actorName);
  // Attribution preference: user → factory-line automation → originating
  // canvas run + app → "Someone" fallback. Automation wins over the source
  // run because the line/step tells the operator far more than a raw run
  // link (e.g. "closed by Plan / step-01" vs. "closed via run"). Only fall
  // back to sourceRunId when no automation ref is attached (typical for
  // user-issued transitions triggered by a run — the initial creation).
  const automationActor =
    !actorDisplay && AUTOMATION_ATTRIBUTABLE_KINDS.includes(event.kind) ? event.actorAutomation : undefined;
  const hasSourceRun = Boolean(event.sourceRunId) && !automationActor;
  const showSomeoneFallback =
    !actorDisplay && !hasSourceRun && !automationActor && SOMEONE_FALLBACK_KINDS.includes(event.kind);

  return (
    <p className="flex flex-wrap items-center gap-x-1 gap-y-1 text-sm text-gray-900 dark:text-gray-100">
      {actorDisplay ? (
        <OrgUserReference display={actorDisplay} size="sm" emphasizeName />
      ) : automationActor ? (
        <TimelineAutomationActor actor={automationActor} />
      ) : showSomeoneFallback ? (
        <span className="font-semibold">Someone</span>
      ) : null}
      {event.kind === "assigned" && event.assigneeChange ? (
        <AssigneeChangeDescription
          actorUserId={event.actorUserId}
          assigneeChange={event.assigneeChange}
          resolveUserDisplay={resolveUserDisplay}
        />
      ) : hasSourceRun ? (
        <SourceRunAttribution event={event} organizationId={organizationId} factoryId={factoryId} orderId={orderId} />
      ) : (
        <span>{event.title}</span>
      )}
      {automationActor ? (
        <span className="ml-1 inline-flex items-center gap-1 rounded border border-violet-200 bg-violet-50 px-1.5 py-0.5 text-[10px] font-medium uppercase tracking-wide text-violet-700 dark:border-violet-500/30 dark:bg-violet-500/10 dark:text-violet-200">
          Automation
        </span>
      ) : null}
    </p>
  );
}

// Renders an event title followed by "run" (linked to the originating canvas
// run when the app+run ids are known). Used for events attributed to a run
// rather than to a user or a factory-line automation.
function SourceRunAttribution({
  event,
  organizationId,
  factoryId,
  orderId,
}: {
  event: WorkOrderTimelineEvent;
  organizationId: string;
  factoryId: string;
  orderId?: string;
}) {
  const runHref =
    event.sourceAppId && event.sourceRunId
      ? factoryAppRunPath(organizationId, factoryId, event.sourceAppId, event.sourceRunId, {
          from: "work-order",
          orderId,
        })
      : null;

  // `created` events read "Work order created from run"; other kinds (e.g.
  // `statusChanged` for the `draft → open` marker) read "opened this work
  // order via run". We choose the connector accordingly.
  const connector = event.kind === "created" ? "from" : "via";

  return (
    <span className="inline-flex flex-wrap items-center gap-x-1 gap-y-1">
      <span>{event.title}</span>
      <span>{connector}</span>
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

function DispatchStepRow({
  organizationId,
  factoryId,
  orderId,
  step,
}: {
  organizationId: string;
  factoryId: string;
  orderId?: string;
  step: WorkOrderTimelineStep;
}) {
  const runHref = getWorkOrderExecutionRunHref(organizationId, factoryId, step.execution, { orderId });
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
    case "statusChanged":
      return { icon: ShieldCheck };
    case "commented":
      return { icon: MessageSquareText };
    case "artifactAdded":
      return { icon: FileText };
    case "closed":
      return { icon: CircleDot };
  }
}
