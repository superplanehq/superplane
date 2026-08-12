import { Text } from "@/components/Text/text";
import type { FactoriesWorkOrder, FactoriesWorkOrderEvent } from "@/api-client";
import { Link } from "@/components/Link/link";
import { Button } from "@/components/ui/button";
import { useOrganizationUsers } from "@/hooks/useOrganizationData";
import type { OrgUserDisplay, OrgUserDisplayLookup } from "@/lib/orgUserDisplay";
import { appRunPath } from "@/lib/appPaths";
import { cn } from "@/lib/utils";
import { Fragment, useMemo, type ReactNode } from "react";
import { FileText, MessageSquare, Play, UserRound, type LucideIcon } from "lucide-react";
import {
  buildWorkOrderTimelineView,
  buildWorkOrderUserDisplayLookup,
  buildWorkOrderUserNameLookup,
  type WorkOrderTimelineEvent,
  type WorkOrderTimelineEventKind,
} from "./lib/workOrderTimelineEvents";
import { formatWorkOrderDateTime as formatTimelineDate } from "./lib/workOrderDateTime";
import { OrgUserReference } from "./OrgUserReference";
import { DispatchTimelineItem } from "./timeline/DispatchTimelineItem";
import { TimelineAutomationActor } from "./timeline";
import { TimelineMarker } from "./timeline/TimelineMarker";
import {
  timelineActorClassName as inlineActorClassName,
  timelineLinkClassName as inlineLinkClassName,
  timelineParagraphClassName as inlineParagraphClassName,
  timelineTimeClassName as inlineTimeClassName,
} from "./timeline/timelineStyles";
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
  /** Optional trailing timeline content, such as the comment composer. */
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

  const timeline = buildWorkOrderTimelineView(events, resolveUserName, order.executions);

  if (timeline.events.length === 0 && !footer) {
    return <TimelineActivityEmpty />;
  }

  const latestDispatchIndex = findLatestDispatchIndex(timeline.events);

  return (
    <div>
      {hasMoreEvents ? (
        <div className="mb-3">
          <Button
            type="button"
            variant="link"
            onClick={onLoadMoreEvents}
            disabled={isLoadingMoreEvents || !onLoadMoreEvents}
            className="h-auto p-0 text-[13px] text-foreground"
          >
            {isLoadingMoreEvents ? "Loading…" : "Load more events"}
          </Button>
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
    <div role="alert" className="rounded-lg border border-destructive/30 px-4 py-3">
      <Text className="text-destructive">Failed to load activity.</Text>
      {onRetryEvents ? (
        <Button
          type="button"
          variant="link"
          onClick={onRetryEvents}
          className="mt-2 h-auto p-0 text-sm text-foreground"
        >
          Try again
        </Button>
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

function getFallbackMarkerIcon(kind: WorkOrderTimelineEventKind): LucideIcon {
  switch (kind) {
    case "artifactAdded":
      return FileText;
    case "statusChanged":
    case "closed":
      return Play;
    default:
      return UserRound;
  }
}
