import { Text } from "@/components/Text/text";
import type {
  FactoriesWorkOrder,
  FactoriesWorkOrderApprovalStatus,
  FactoriesWorkOrderEvent,
  FactoriesWorkOrderExecution,
} from "@/api-client";
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
import { OrgUserReference } from "./OrgUserReference";
import { ApprovalTimelineBody, ArtifactTimelineBody, CommentTimelineBody, TimelineAutomationActor } from "./timeline";

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
  canResolveApproval?: boolean;
  isResolvingApproval?: boolean;
  onResolveApproval?: (input: {
    approvalId: string;
    status: FactoriesWorkOrderApprovalStatus;
    comment?: string;
  }) => Promise<void>;
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
  canResolveApproval = false,
  isResolvingApproval = false,
  onResolveApproval,
}: WorkOrderTimelineProps) {
  const { data: users = [] } = useOrganizationUsers(organizationId);
  const resolveUserName = useMemo(() => buildWorkOrderUserNameLookup(users, order), [users, order]);
  const resolveUserDisplay = useMemo(() => buildWorkOrderUserDisplayLookup(users, order), [users, order]);
  const pendingApprovalRunIds = useMemo(() => buildPendingApprovalRunIds(order), [order]);
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
      canResolveApproval={canResolveApproval}
      isResolvingApproval={isResolvingApproval}
      onResolveApproval={onResolveApproval}
      pendingApprovalRunIds={pendingApprovalRunIds}
    />
  );
}

// Executions blocked on a pending plan approval. The timeline builds run
// cards from step events whose `execution.id` is the run id, so we resolve
// each approval's `executionId` (an execution row id) to a run id via the
// order's execution list. Callers use this set to distinguish "Running" from
// "Waiting for review" in the run badge.
function buildPendingApprovalRunIds(order: FactoriesWorkOrder): Set<string> {
  const executionRowToRunId = new Map<string, string>();
  for (const execution of order.executions ?? []) {
    if (execution.id && execution.run?.id) {
      executionRowToRunId.set(execution.id, execution.run.id);
    }
  }

  const runIds = new Set<string>();
  for (const approval of order.approvals ?? []) {
    if (approval.status !== "STATUS_PENDING" || !approval.executionId) continue;
    const runId = executionRowToRunId.get(approval.executionId);
    if (runId) {
      runIds.add(runId);
    }
  }
  return runIds;
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
  canResolveApproval,
  isResolvingApproval,
  onResolveApproval,
  pendingApprovalRunIds,
}: {
  timeline: ReturnType<typeof buildWorkOrderTimelineView>;
  organizationId: string;
  resolveUserDisplay: OrgUserDisplayLookup;
  hasMoreEvents: boolean;
  isLoadingMoreEvents: boolean;
  onLoadMoreEvents?: () => void;
  canResolveApproval: boolean;
  isResolvingApproval: boolean;
  onResolveApproval?: (input: {
    approvalId: string;
    status: FactoriesWorkOrderApprovalStatus;
    comment?: string;
  }) => Promise<void>;
  pendingApprovalRunIds: Set<string>;
}) {
  // Latest dispatched run gets a "Current" badge and expands by default so
  // operators can inspect the running run without extra clicks.
  const latestDispatchIndex = findLatestDispatchIndex(timeline.events);

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
          isLatestDispatch={index === latestDispatchIndex}
          canResolveApproval={canResolveApproval}
          isResolvingApproval={isResolvingApproval}
          onResolveApproval={onResolveApproval}
          pendingApprovalRunIds={pendingApprovalRunIds}
        />
      ))}
    </ol>
  );
}

function findLatestDispatchIndex(events: WorkOrderTimelineEvent[]): number {
  let idx = -1;
  events.forEach((event, i) => {
    if (event.kind === "dispatched") {
      idx = i;
    }
  });
  return idx;
}

function TimelineItem({
  event,
  organizationId,
  resolveUserDisplay,
  isLast,
  isLatestDispatch,
  canResolveApproval,
  isResolvingApproval,
  onResolveApproval,
  pendingApprovalRunIds,
}: {
  event: WorkOrderTimelineEvent;
  organizationId: string;
  resolveUserDisplay: OrgUserDisplayLookup;
  isLast: boolean;
  isLatestDispatch: boolean;
  canResolveApproval: boolean;
  isResolvingApproval: boolean;
  onResolveApproval?: (input: {
    approvalId: string;
    status: FactoriesWorkOrderApprovalStatus;
    comment?: string;
  }) => Promise<void>;
  pendingApprovalRunIds: Set<string>;
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
          resolveUserDisplay={resolveUserDisplay}
          isLatestDispatch={isLatestDispatch}
          canResolveApproval={canResolveApproval}
          isResolvingApproval={isResolvingApproval}
          onResolveApproval={onResolveApproval}
          pendingApprovalRunIds={pendingApprovalRunIds}
        />
      </div>
    </li>
  );
}

const USER_ACTION_EVENT_KINDS: WorkOrderTimelineEventKind[] = ["created", "assigned", "statusChanged", "closed"];

function TimelineItemContent({
  event,
  organizationId,
  resolveUserDisplay,
  isLatestDispatch,
  canResolveApproval,
  isResolvingApproval,
  onResolveApproval,
  pendingApprovalRunIds,
}: {
  event: WorkOrderTimelineEvent;
  organizationId: string;
  resolveUserDisplay: OrgUserDisplayLookup;
  isLatestDispatch: boolean;
  canResolveApproval: boolean;
  isResolvingApproval: boolean;
  onResolveApproval?: (input: {
    approvalId: string;
    status: FactoriesWorkOrderApprovalStatus;
    comment?: string;
  }) => Promise<void>;
  pendingApprovalRunIds: Set<string>;
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

  if ((event.kind === "approvalRequested" || event.kind === "approvalResolved") && event.approval) {
    return (
      <ApprovalTimelineBody
        event={event}
        approval={event.approval}
        actorDisplay={actorDisplay}
        resolveUserDisplay={resolveUserDisplay}
        canResolve={canResolveApproval}
        isResolving={isResolvingApproval}
        onResolve={onResolveApproval}
      />
    );
  }

  if (USER_ACTION_EVENT_KINDS.includes(event.kind)) {
    return (
      <>
        <UserActionEventDescription
          event={event}
          organizationId={organizationId}
          resolveUserDisplay={resolveUserDisplay}
        />
        <time className="mt-2 block text-xs text-gray-500 dark:text-gray-400">{formatTimeAgo(new Date(event.at))}</time>
      </>
    );
  }

  if (event.kind === "dispatched") {
    return (
      <DispatchRunCard
        event={event}
        organizationId={organizationId}
        isLatestDispatch={isLatestDispatch}
        pendingApprovalRunIds={pendingApprovalRunIds}
      />
    );
  }

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

function DispatchRunCard({
  event,
  organizationId,
  isLatestDispatch,
  pendingApprovalRunIds,
}: {
  event: WorkOrderTimelineEvent;
  organizationId: string;
  isLatestDispatch: boolean;
  pendingApprovalRunIds: Set<string>;
}) {
  const steps = event.steps ?? [];
  const latestStep = steps.length > 0 ? steps[steps.length - 1] : undefined;
  const overallDurationLabel = formatOverallDuration(steps);
  const usageLabel = latestStep ? formatExecutionUsageLabel(latestStep.execution) : null;
  const latestStepRunId = latestStep?.execution.id;
  const isWaitingForApproval = Boolean(latestStepRunId && pendingApprovalRunIds.has(latestStepRunId));

  return (
    <details
      className="group rounded-lg border border-gray-200 bg-white px-3 py-2 dark:border-gray-700/70 dark:bg-transparent"
      open={isLatestDispatch}
    >
      <summary className="flex cursor-pointer list-none flex-wrap items-center gap-x-2 gap-y-1 marker:hidden">
        <span className="font-medium text-gray-900 dark:text-gray-100">{event.lineName ?? "Dispatched"}</span>
        {latestStep ? (
          <RunResultBadge execution={latestStep.execution} isWaitingForApproval={isWaitingForApproval} />
        ) : null}
        {isLatestDispatch ? (
          <span className="inline-flex items-center rounded border border-violet-200 bg-violet-50 px-1.5 py-0.5 text-[10px] font-medium uppercase tracking-wide text-violet-700 dark:border-violet-500/30 dark:bg-violet-500/10 dark:text-violet-200">
            Current
          </span>
        ) : null}
        <span className="ml-auto flex flex-wrap items-center gap-x-2 gap-y-1 text-xs text-gray-500 dark:text-gray-400">
          <span>Sent {formatTimeAgo(new Date(event.at))}</span>
          {overallDurationLabel ? <span>· {overallDurationLabel}</span> : null}
          {usageLabel ? <span>· {usageLabel}</span> : null}
        </span>
      </summary>

      <div className="mt-3 space-y-2">
        {steps.map((step) => (
          <div key={step.id} className="space-y-1">
            <DispatchStepRow organizationId={organizationId} step={step} />
            {step.note ? (
              <p className="ml-6 text-xs text-gray-500 dark:text-gray-400" data-testid="work-order-dispatch-note">
                {step.note}
              </p>
            ) : null}
          </div>
        ))}
      </div>
    </details>
  );
}

function RunResultBadge({
  execution,
  isWaitingForApproval,
}: {
  execution: FactoriesWorkOrderExecution;
  isWaitingForApproval: boolean;
}) {
  if (execution.result === "RESULT_FAILED") {
    return (
      <span className="inline-flex items-center rounded border border-red-200 bg-red-50 px-1.5 py-0.5 text-[10px] font-medium uppercase tracking-wide text-red-700 dark:border-red-500/30 dark:bg-red-500/10 dark:text-red-200">
        Failed
      </span>
    );
  }
  if (execution.result === "RESULT_PASSED") {
    return (
      <span className="inline-flex items-center rounded border border-emerald-200 bg-emerald-50 px-1.5 py-0.5 text-[10px] font-medium uppercase tracking-wide text-emerald-700 dark:border-emerald-500/30 dark:bg-emerald-500/10 dark:text-emerald-200">
        Complete
      </span>
    );
  }
  const isInFlight =
    execution.state === "STATE_PENDING" ||
    execution.state === "STATE_STARTED" ||
    execution.state === "STATE_CANCELLING";
  if (!isInFlight) {
    return null;
  }
  if (isWaitingForApproval) {
    return (
      <span className="inline-flex items-center rounded border border-amber-200 bg-amber-50 px-1.5 py-0.5 text-[10px] font-medium uppercase tracking-wide text-amber-700 dark:border-amber-500/30 dark:bg-amber-500/10 dark:text-amber-200">
        Waiting for review
      </span>
    );
  }
  return (
    <span className="inline-flex items-center rounded border border-violet-200 bg-violet-50 px-1.5 py-0.5 text-[10px] font-medium uppercase tracking-wide text-violet-700 dark:border-violet-500/30 dark:bg-violet-500/10 dark:text-violet-200">
      Running
    </span>
  );
}

function formatOverallDuration(steps: WorkOrderTimelineStep[]): string | null {
  if (steps.length === 0) return null;
  const started = Date.parse(steps[0]?.startedAt ?? "");
  const lastStep = steps[steps.length - 1];
  const finished = Date.parse(lastStep?.finishedAt ?? lastStep?.startedAt ?? "");
  const duration = finished - started;
  if (!Number.isFinite(duration) || duration <= 0) return null;
  const seconds = Math.round(duration / 1000);
  if (seconds < 60) return `${seconds}s`;
  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) return `${minutes}m`;
  const hours = Math.floor(minutes / 60);
  return `${hours}h ${minutes % 60}m`;
}

function formatExecutionUsageLabel(execution: FactoriesWorkOrderExecution): string | null {
  const tokens = Number(execution.totalTokens ?? 0);
  const cents = Number(execution.costCents ?? 0);
  const parts: string[] = [];
  if (Number.isFinite(tokens) && tokens > 0) {
    parts.push(tokens >= 1000 ? `${(tokens / 1000).toFixed(0)}k tokens` : `${tokens} tokens`);
  }
  if (Number.isFinite(cents) && cents > 0) {
    parts.push(`$${(cents / 100).toFixed(2)}`);
  }
  return parts.length > 0 ? parts.join(" · ") : null;
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
  resolveUserDisplay,
}: {
  event: WorkOrderTimelineEvent;
  organizationId: string;
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
        <SourceRunAttribution event={event} organizationId={organizationId} />
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
function SourceRunAttribution({ event, organizationId }: { event: WorkOrderTimelineEvent; organizationId: string }) {
  const runHref =
    event.sourceAppId && event.sourceRunId ? appRunPath(organizationId, event.sourceAppId, event.sourceRunId) : null;

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
    case "statusChanged":
      return { icon: ShieldCheck };
    case "commented":
      return { icon: MessageSquareText };
    case "artifactAdded":
      return { icon: FileText };
    case "closed":
      return { icon: CircleDot };
    case "approvalRequested":
      return { icon: ShieldCheck };
    case "approvalResolved":
      return { icon: Check };
  }
}
