import { Text } from "@/components/Text/text";
import type { FactoriesWorkOrder, FactoriesWorkOrderEvent, FactoriesWorkOrderExecution } from "@/api-client";
import { Avatar } from "@/components/Avatar/avatar";
import { Link } from "@/components/Link/link";
import { Badge } from "@/components/ui/badge";
import { useOrganizationUsers } from "@/hooks/useOrganizationData";
import type { OrgUserDisplay, OrgUserDisplayLookup } from "@/lib/orgUserDisplay";
import { appRunPath } from "@/lib/appPaths";
import { cn } from "@/lib/utils";
import { Fragment, useMemo, useState, type ReactNode } from "react";
import { ChevronDown, ExternalLink, FileText, MessageSquare, Play, Sparkles, UserRound } from "lucide-react";
import {
  buildWorkOrderTimelineView,
  buildWorkOrderUserDisplayLookup,
  buildWorkOrderUserNameLookup,
  formatStepExecutionDuration,
  type WorkOrderTimelineEvent,
  type WorkOrderTimelineEventKind,
  type WorkOrderTimelineStep,
} from "./lib/workOrderTimelineEvents";
import { getWorkOrderExecutionRunHref } from "./lib/workOrderExecutions";
import { OrgUserReference } from "./OrgUserReference";
import { TimelineAutomationActor } from "./timeline";
import { WorkOrderArtifactInline } from "./WorkOrderArtifactInline";

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
  /**
   * Optional trailing element rendered as the final row of the timeline
   * (e.g. the comment composer). Rendered with an icon marker so it hangs
   * off the same vertical rail as the events above it.
   */
  footer?: ReactNode;
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
  footer,
}: WorkOrderTimelineProps) {
  const { data: users = [] } = useOrganizationUsers(organizationId);
  const resolveUserName = useMemo(() => buildWorkOrderUserNameLookup(users, order), [users, order]);
  const resolveUserDisplay = useMemo(() => buildWorkOrderUserDisplayLookup(users, order), [users, order]);
  const pendingView = renderTimelinePendingView({ events, eventsError, isLoading, onRetryEvents });

  if (pendingView) {
    return pendingView;
  }

  const timeline = buildWorkOrderTimelineView(events, resolveUserName);

  if (timeline.events.length === 0 && !footer) {
    return <TimelineActivityEmpty />;
  }

  const latestDispatchIndex = findLatestDispatchIndex(timeline.events);

  return (
    <div>
      {hasMoreEvents ? (
        <div className="mb-3">
          <button
            type="button"
            onClick={onLoadMoreEvents}
            disabled={isLoadingMoreEvents || !onLoadMoreEvents}
            className="text-[13px] font-medium text-violet-600 hover:text-violet-700 disabled:cursor-not-allowed disabled:opacity-60 dark:text-violet-400 dark:hover:text-violet-300"
          >
            {isLoadingMoreEvents ? "Loading…" : "Load more events"}
          </button>
        </div>
      ) : null}

      <ul className="relative space-y-4">
        <span className="pointer-events-none absolute left-[11px] top-3 bottom-3 w-px bg-border" aria-hidden />
        {timeline.events.map((event, index) => (
          <TimelineItem
            key={event.id}
            event={event}
            organizationId={organizationId}
            resolveUserDisplay={resolveUserDisplay}
            isLatestDispatch={index === latestDispatchIndex}
          />
        ))}
      </ul>
      {footer ? (
        <div className="relative mt-4 flex gap-3">
          <TimelineMarker icon={MessageSquare} />
          <div className="min-w-0 flex-1">{footer}</div>
        </div>
      ) : null}
    </div>
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
  return <Text className="text-sm text-muted-foreground">Loading activity…</Text>;
}

function TimelineActivityEmpty() {
  return <p className="text-sm text-muted-foreground">No activity yet.</p>;
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

// Kinds that render an avatar marker (when the actor is resolvable).
const AVATAR_MARKER_KINDS: WorkOrderTimelineEventKind[] = [
  "created",
  "assigned",
  "statusChanged",
  "closed",
  "commented",
  "artifactAdded",
];

function TimelineItem({
  event,
  organizationId,
  resolveUserDisplay,
  isLatestDispatch,
}: {
  event: WorkOrderTimelineEvent;
  organizationId: string;
  resolveUserDisplay: OrgUserDisplayLookup;
  isLatestDispatch: boolean;
}) {
  if (event.kind === "dispatched") {
    return <DispatchTimelineItem event={event} organizationId={organizationId} isLatestDispatch={isLatestDispatch} />;
  }

  const actorDisplay = resolveUserDisplay(event.actorUserId, event.actorName);
  const useAvatar = AVATAR_MARKER_KINDS.includes(event.kind) && Boolean(actorDisplay);

  return (
    <li>
      <div className="flex gap-3 py-0.5">
        <TimelineMarker
          display={useAvatar ? actorDisplay : null}
          icon={useAvatar ? undefined : getFallbackMarkerIcon(event.kind)}
        />
        <div className="min-w-0 flex-1 pt-0.5">
          <TimelineItemBody event={event} organizationId={organizationId} resolveUserDisplay={resolveUserDisplay} />
        </div>
      </div>
    </li>
  );
}

function TimelineItemBody({
  event,
  organizationId,
  resolveUserDisplay,
}: {
  event: WorkOrderTimelineEvent;
  organizationId: string;
  resolveUserDisplay: OrgUserDisplayLookup;
}) {
  const actorDisplay = resolveUserDisplay(event.actorUserId, event.actorName);
  const timeLabel = formatTimelineDate(new Date(event.at));

  if (event.kind === "commented") {
    return <CommentEventBody event={event} actorDisplay={actorDisplay} timeLabel={timeLabel} />;
  }

  if (event.kind === "artifactAdded") {
    return <ArtifactEventBody event={event} actorDisplay={actorDisplay} timeLabel={timeLabel} />;
  }

  return (
    <UserActionEventDescription
      event={event}
      organizationId={organizationId}
      resolveUserDisplay={resolveUserDisplay}
      timeLabel={timeLabel}
    />
  );
}

function CommentEventBody({
  event,
  actorDisplay,
  timeLabel,
}: {
  event: WorkOrderTimelineEvent;
  actorDisplay: OrgUserDisplay | null;
  timeLabel: string;
}) {
  const comment = event.comment;
  if (!comment) return null;
  const isAutomation = (comment.authorKind ?? "").toLowerCase() === "automation";

  return (
    <>
      <p className={inlineParagraphClassName}>
        {isAutomation && (event.actorAutomation || comment.automation) ? (
          <TimelineAutomationActor actor={event.actorAutomation ?? comment.automation!} fallbackLabel="Automation" />
        ) : actorDisplay ? (
          <span className={inlineActorClassName}>{actorDisplay.name}</span>
        ) : (
          <span className={inlineActorClassName}>Someone</span>
        )}{" "}
        commented
        <span className={inlineTimeClassName}>
          {" · "}
          {timeLabel}
        </span>
      </p>
      <p className="mt-1 whitespace-pre-wrap text-[13px] leading-relaxed text-foreground">{comment.body}</p>
    </>
  );
}

function ArtifactEventBody({
  event,
  actorDisplay,
  timeLabel,
}: {
  event: WorkOrderTimelineEvent;
  actorDisplay: OrgUserDisplay | null;
  timeLabel: string;
}) {
  const artifact = event.artifact;
  if (!artifact) return null;

  return (
    <p className={inlineParagraphClassName}>
      {actorDisplay ? (
        <span className={inlineActorClassName}>{actorDisplay.name}</span>
      ) : event.actorAutomation ? (
        <TimelineAutomationActor actor={event.actorAutomation} />
      ) : (
        <span className={inlineActorClassName}>Someone</span>
      )}{" "}
      attached <ArtifactInlineChip artifact={artifact} />
      <span className={inlineTimeClassName}>
        {" · "}
        {timeLabel}
      </span>
    </p>
  );
}

function ArtifactInlineChip({ artifact }: { artifact: NonNullable<WorkOrderTimelineEvent["artifact"]> }) {
  return <WorkOrderArtifactInline artifact={artifact} className="align-baseline" />;
}

const AUTOMATION_ATTRIBUTABLE_KINDS: WorkOrderTimelineEventKind[] = ["created", "statusChanged", "closed"];
const SOMEONE_FALLBACK_KINDS: WorkOrderTimelineEventKind[] = ["created", "statusChanged", "closed"];

function UserActionEventDescription({
  event,
  organizationId,
  resolveUserDisplay,
  timeLabel,
}: {
  event: WorkOrderTimelineEvent;
  organizationId: string;
  resolveUserDisplay: OrgUserDisplayLookup;
  timeLabel: string;
}) {
  const actorDisplay = resolveUserDisplay(event.actorUserId, event.actorName);
  const automationActor =
    !actorDisplay && AUTOMATION_ATTRIBUTABLE_KINDS.includes(event.kind) ? event.actorAutomation : undefined;
  const hasSourceRun = Boolean(event.sourceRunId) && !automationActor;
  const showSomeoneFallback =
    !actorDisplay && !hasSourceRun && !automationActor && SOMEONE_FALLBACK_KINDS.includes(event.kind);

  return (
    <div>
      <p className={cn(inlineParagraphClassName, "flex flex-wrap items-baseline gap-x-1 gap-y-0.5")}>
        {actorDisplay ? (
          <span className={inlineActorClassName}>{actorDisplay.name}</span>
        ) : automationActor ? (
          <TimelineAutomationActor actor={automationActor} />
        ) : showSomeoneFallback ? (
          <span className={inlineActorClassName}>Someone</span>
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
        <span className={inlineTimeClassName}>
          {" · "}
          {timeLabel}
        </span>
      </p>
      {event.kind === "created" && actorDisplay ? (
        <p className="mt-0.5 text-[12px] leading-snug text-muted-foreground">Create work order form</p>
      ) : null}
    </div>
  );
}

function SourceRunAttribution({ event, organizationId }: { event: WorkOrderTimelineEvent; organizationId: string }) {
  const runHref =
    event.sourceAppId && event.sourceRunId ? appRunPath(organizationId, event.sourceAppId, event.sourceRunId) : null;
  const connector = event.kind === "created" ? "from" : "via";
  return (
    <span className="inline-flex flex-wrap items-baseline gap-x-1">
      <span>{event.title}</span>
      <span>{connector}</span>
      {runHref ? (
        <Link href={runHref} className={inlineLinkClassName}>
          run
        </Link>
      ) : (
        <span>run</span>
      )}
    </span>
  );
}

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
  if (assignedDescription) parts.push(<span key="assigned">{assignedDescription}</span>);

  if (unassignedUserIds.length > 0) {
    parts.push(
      <span key="unassigned" className="inline-flex flex-wrap items-baseline gap-x-1">
        unassigned <InlineUserList userIds={unassignedUserIds} resolveUserDisplay={resolveUserDisplay} />
      </span>,
    );
  }

  if (parts.length === 0) return <span>updated assignees</span>;

  return (
    <>
      {parts.map((part, index) => (
        <Fragment key={index}>
          {index > 0 ? <span>and</span> : null}
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
  if (assignedUserIds.length === 0) return null;
  const assignedOthers = actorUserId ? assignedUserIds.filter((userId) => userId !== actorUserId) : assignedUserIds;
  const selfAssigned = Boolean(actorUserId && assignedUserIds.includes(actorUserId));

  if (selfAssigned && assignedOthers.length === 0) return <span>self-assigned</span>;

  if (selfAssigned && assignedOthers.length > 0) {
    return (
      <span className="inline-flex flex-wrap items-baseline gap-x-1">
        self-assigned and assigned <InlineUserList userIds={assignedOthers} resolveUserDisplay={resolveUserDisplay} />
      </span>
    );
  }

  return (
    <span className="inline-flex flex-wrap items-baseline gap-x-1">
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
  if (userIds.length === 0) return null;

  if (userIds.length === 1) {
    return <InlineUserName display={resolveUserDisplay(userIds[0])} />;
  }

  if (userIds.length === 2) {
    return (
      <>
        <InlineUserName display={resolveUserDisplay(userIds[0])} />
        <span>and</span>
        <InlineUserName display={resolveUserDisplay(userIds[1])} />
      </>
    );
  }

  return (
    <>
      {userIds.slice(0, -1).map((userId, index) => (
        <Fragment key={userId}>
          {index > 0 ? <span>,</span> : null}
          <InlineUserName display={resolveUserDisplay(userId)} />
        </Fragment>
      ))}
      <span>, and</span>
      <InlineUserName display={resolveUserDisplay(userIds[userIds.length - 1])} />
    </>
  );
}

function InlineUserName({ display }: { display: OrgUserDisplay | null }) {
  if (!display) return <span className={inlineActorClassName}>Someone</span>;
  return (
    <span className="inline-flex items-baseline gap-1">
      <OrgUserReference display={display} size="xs" showName={false} className="translate-y-[1px]" />
      <span className={inlineActorClassName}>{display.name}</span>
    </span>
  );
}

// ---------- Dispatch run ----------

function DispatchTimelineItem({
  event,
  organizationId,
  isLatestDispatch,
}: {
  event: WorkOrderTimelineEvent;
  organizationId: string;
  isLatestDispatch: boolean;
}) {
  const [open, setOpen] = useState(isLatestDispatch);
  const steps = event.steps ?? [];
  const latestStep = steps.length > 0 ? steps[steps.length - 1] : undefined;
  const overallDurationLabel = formatOverallDuration(steps);
  const usageLabel = latestStep ? formatExecutionUsageLabel(latestStep.execution) : null;
  const statusLabel = latestStep ? runStatusLabel(latestStep.execution) : null;
  const contentId = `line-run-${event.id}`;

  return (
    <li id={event.lineId ? `activity-line-run-${event.lineId}` : undefined} className="scroll-mt-20">
      <div className="min-w-0">
        <div className="flex gap-3">
          <TimelineMarker icon={Sparkles} />
          <button
            type="button"
            aria-expanded={open}
            aria-controls={contentId}
            onClick={() => setOpen((value) => !value)}
            className="flex min-w-0 flex-1 cursor-pointer items-start gap-2 py-0.5 text-left"
          >
            <div className="min-w-0 flex-1 pt-0.5">
              <p className={inlineParagraphClassName}>
                <span className={inlineActorClassName}>{event.lineName ?? "Factory line"}</span>
                {statusLabel ? (
                  <span className={cn("ml-1.5 font-medium", statusLabel.className)}>{statusLabel.label}</span>
                ) : null}
                {isLatestDispatch ? (
                  <Badge
                    variant="outline"
                    className="ml-1.5 inline-flex h-auto rounded-md px-1.5 py-0 align-middle text-[10px] font-medium text-muted-foreground"
                  >
                    Current
                  </Badge>
                ) : null}
                <span className={inlineTimeClassName}>
                  {" · "}Sent {formatTimelineDate(new Date(event.at))}
                  {overallDurationLabel ? ` · ${overallDurationLabel}` : ""}
                  {usageLabel ? ` · ${usageLabel}` : ""}
                </span>
              </p>
            </div>
            <ChevronDown
              className={cn(
                "mt-0.5 size-3.5 shrink-0 text-muted-foreground transition-transform",
                open ? "rotate-180" : "",
              )}
              aria-hidden
            />
          </button>
        </div>

        {open ? (
          <ul id={contentId} className="mt-1 ml-[36px] space-y-0.5">
            {steps.map((step) => (
              <DispatchStepRow key={step.id} organizationId={organizationId} step={step} />
            ))}
          </ul>
        ) : null}
      </div>
    </li>
  );
}

function DispatchStepRow({ organizationId, step }: { organizationId: string; step: WorkOrderTimelineStep }) {
  const runHref = getWorkOrderExecutionRunHref(organizationId, step.execution);
  const durationLabel = formatStepExecutionDuration(step);
  const status = stepStatusLabel(step.execution);
  const nameNode = runHref ? (
    <a
      href={runHref}
      target="_blank"
      rel="noopener noreferrer"
      className="inline-flex items-center gap-1 text-[13px] font-medium tracking-[-0.01em] text-foreground hover:underline"
    >
      {step.stepName}
      <ExternalLink className="size-3 shrink-0 text-muted-foreground" aria-hidden />
    </a>
  ) : (
    <span className="text-[13px] font-medium tracking-[-0.01em] text-foreground">{step.stepName}</span>
  );

  return (
    <li className="min-w-0 py-1.5">
      <div className="flex flex-wrap items-center gap-x-2 gap-y-1">
        {nameNode}
        {status ? (
          <span className={cn("inline-flex items-center gap-1.5 text-[12px]", status.textClassName)}>
            <span className={cn("size-1.5 shrink-0 rounded-full", status.dotClassName)} aria-hidden />
            {status.label}
          </span>
        ) : null}
        {durationLabel ? <span className="text-[12px] tabular-nums text-muted-foreground">{durationLabel}</span> : null}
        {step.artifacts?.map((artifact, index) => (
          <span className="inline-flex min-w-0 max-w-full" key={artifact.id ?? `${artifact.type}-${index}`}>
            <WorkOrderArtifactInline artifact={artifact} />
          </span>
        ))}
      </div>
      {step.note ? (
        <p className="mt-1 text-[12px] leading-relaxed text-muted-foreground" data-testid="work-order-dispatch-note">
          <span className="font-medium text-foreground/80">Note</span>: {step.note}
        </p>
      ) : null}
      {step.comments?.map((comment, index) => (
        <p
          className="mt-1 text-[12px] leading-relaxed text-muted-foreground"
          key={`${comment.label ?? "comment"}-${index}`}
        >
          <span className="font-medium text-foreground/80">{comment.label ?? "Comment"}</span>: {comment.body}
        </p>
      ))}
    </li>
  );
}

interface StatusChip {
  label: string;
  textClassName: string;
  dotClassName: string;
}

function stepStatusLabel(execution: FactoriesWorkOrderExecution): StatusChip | null {
  if (execution.result === "RESULT_PASSED") {
    return {
      label: "Completed",
      textClassName: "text-[color:var(--status-success)]",
      dotClassName: "bg-[var(--status-success-dot)]",
    };
  }
  if (execution.result === "RESULT_FAILED") {
    return {
      label: "Failed",
      textClassName: "text-[color:var(--status-danger)]",
      dotClassName: "bg-[var(--status-danger-dot)]",
    };
  }
  if (execution.result === "RESULT_CANCELLED") {
    return { label: "Cancelled", textClassName: "text-muted-foreground", dotClassName: "bg-muted-foreground/60" };
  }
  const isInFlight =
    execution.state === "STATE_PENDING" ||
    execution.state === "STATE_STARTED" ||
    execution.state === "STATE_CANCELLING";
  if (!isInFlight) return null;
  return {
    label: "Running",
    textClassName: "text-[color:var(--status-running)]",
    dotClassName: "bg-[var(--status-running-dot)]",
  };
}

function runStatusLabel(execution: FactoriesWorkOrderExecution): { label: string; className: string } | null {
  if (execution.result === "RESULT_PASSED")
    return { label: "Complete", className: "text-[color:var(--status-success)]" };
  if (execution.result === "RESULT_FAILED") return { label: "Failed", className: "text-[color:var(--status-danger)]" };
  if (execution.result === "RESULT_CANCELLED") return { label: "Cancelled", className: "text-muted-foreground" };
  const isInFlight =
    execution.state === "STATE_PENDING" ||
    execution.state === "STATE_STARTED" ||
    execution.state === "STATE_CANCELLING";
  if (!isInFlight) return null;
  return { label: "Running", className: "text-[color:var(--status-running)]" };
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

// ---------- Marker + shared classnames ----------

function TimelineMarker({ display, icon: Icon }: { display?: OrgUserDisplay | null; icon?: TimelineMarkerIcon }) {
  if (display) {
    return (
      <span className="relative z-[1]">
        <Avatar src={display.avatarUrl} initials={display.initials} alt={display.name} className="size-6" />
      </span>
    );
  }
  const IconComponent = Icon ?? UserRound;
  return (
    <span className="relative z-[1] flex size-6 shrink-0 items-center justify-center rounded-full border border-border bg-background">
      <IconComponent className="size-3 text-muted-foreground" aria-hidden />
    </span>
  );
}

type TimelineMarkerIcon = typeof UserRound;

function getFallbackMarkerIcon(kind: WorkOrderTimelineEventKind): TimelineMarkerIcon {
  switch (kind) {
    case "dispatched":
      return Sparkles;
    case "artifactAdded":
      return FileText;
    case "statusChanged":
    case "closed":
      return Play;
    default:
      return UserRound;
  }
}

const inlineParagraphClassName = "text-[13px] leading-snug tracking-[-0.01em] text-muted-foreground";
const inlineActorClassName = "font-medium text-foreground";
const inlineTimeClassName = "text-muted-foreground";
const inlineLinkClassName =
  "inline-flex items-center gap-1 rounded-md text-[13px] text-foreground underline decoration-border underline-offset-2 hover:decoration-foreground";

function formatTimelineDate(date: Date): string {
  if (Number.isNaN(date.getTime())) {
    return "";
  }

  return new Intl.DateTimeFormat(undefined, {
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
    hourCycle: "h23",
  })
    .format(date)
    .replace(",", "");
}
