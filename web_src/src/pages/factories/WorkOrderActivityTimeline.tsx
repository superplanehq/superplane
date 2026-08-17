import { Text } from "@/components/Text/text";
import type { FactoriesWorkOrder, FactoriesWorkOrderArtifact, FactoriesWorkOrderEvent } from "@/api-client";
import { Link } from "@/components/Link/link";
import { Button } from "@/components/ui/button";
import { useOrganizationUsers } from "@/hooks/useOrganizationData";
import type { OrgUserDisplayLookup } from "@/lib/orgUserDisplay";
import { cn } from "@/lib/utils";
import { useMemo, type ReactNode } from "react";
import { FileText, MessageSquare, Play, UserRound, type LucideIcon } from "lucide-react";
import {
  buildLatestArtifactDataById,
  toWorkOrderArtifactLikes,
  type WorkOrderArtifactLike,
} from "./lib/workOrderArtifact";
import {
  buildWorkOrderTimelineView,
  buildWorkOrderUserDisplayLookup,
  buildWorkOrderUserNameLookup,
  findLatestDispatchIndex,
  type WorkOrderTimelineEvent,
  type WorkOrderTimelineEventKind,
} from "./lib/workOrderTimelineEvents";
import { formatWorkOrderDateTime as formatTimelineDate } from "./lib/workOrderDateTime";
import { getWorkOrderRunHref } from "./lib/workOrderExecutions";
import { ArtifactEventBody } from "./timeline/ArtifactEventBody";
import { CommentEventBody } from "./timeline/CommentEventBody";
import { DispatchTimelineItem } from "./timeline/DispatchTimelineItem";
import { TimelineAutomationActor } from "./timeline";
import { TimelineMarker } from "./timeline/TimelineMarker";
import {
  timelineActorClassName as inlineActorClassName,
  timelineLinkClassName as inlineLinkClassName,
  timelineParagraphClassName as inlineParagraphClassName,
  timelineTimeClassName as inlineTimeClassName,
} from "./timeline/timelineStyles";
import { AssigneeChangeDescription } from "./workOrderTimelineAssignee";

interface WorkOrderTimelineProps {
  organizationId: string;
  factoryKey: string;
  order: FactoriesWorkOrder;
  events?: FactoriesWorkOrderEvent[];
  eventsError?: Error | null;
  isLoading?: boolean;
  hasMoreEvents?: boolean;
  isLoadingMoreEvents?: boolean;
  onLoadMoreEvents?: () => void;
  onRetryEvents?: () => void;
  /**
   * Current artifacts list (from useWorkOrderArtifacts), used to overlay
   * live data — e.g. a PR's current `state` — on top of each
   * `artifactAdded` event's data snapshot, so the "attached" chip in the
   * timeline doesn't go stale the moment the PR changes state. Falls
   * back to the event's own snapshot when an artifact isn't found here
   * (very old events beyond the artifacts list, or a deleted artifact).
   */
  artifacts?: FactoriesWorkOrderArtifact[];
  /** Optional trailing timeline content, such as the comment composer. */
  footer?: ReactNode;
}

export function WorkOrderActivityTimeline({
  organizationId,
  factoryKey,
  order,
  events,
  eventsError = null,
  isLoading = false,
  hasMoreEvents = false,
  isLoadingMoreEvents = false,
  onLoadMoreEvents,
  onRetryEvents,
  artifacts,
  footer,
}: WorkOrderTimelineProps) {
  const { data: users = [] } = useOrganizationUsers(organizationId);
  const resolveUserName = useMemo(() => buildWorkOrderUserNameLookup(users, order), [users, order]);
  const resolveUserDisplay = useMemo(() => buildWorkOrderUserDisplayLookup(users, order), [users, order]);
  const latestArtifactDataById = useMemo(() => buildLatestArtifactDataById(artifacts), [artifacts]);
  // Work-order-wide sibling list. A branch chip in a dispatch step can
  // look up the PR attached in a *different* step to build its tree URL,
  // so pass the whole list through instead of only that step's artifacts.
  const relatedArtifacts = useMemo(() => toWorkOrderArtifactLikes(artifacts), [artifacts]);
  const pendingView = renderTimelinePendingView({ events, eventsError, isLoading, onRetryEvents });
  const timeline = pendingView
    ? { events: [] as WorkOrderTimelineEvent[] }
    : buildWorkOrderTimelineView(events, resolveUserName, order.executions);

  // Without a footer, keep the historical "single message" layout: the
  // pending/empty state occupies the whole slot on its own.
  if (!footer && pendingView) {
    return pendingView;
  }
  if (!footer && timeline.events.length === 0) {
    return <TimelineActivityEmpty />;
  }

  return (
    <div>
      {pendingView ? (
        <div className="mb-4">{pendingView}</div>
      ) : (
        <TimelineEventsList
          organizationId={organizationId}
          factoryKey={factoryKey}
          orderNumber={order.number}
          events={timeline.events}
          resolveUserDisplay={resolveUserDisplay}
          latestArtifactDataById={latestArtifactDataById}
          relatedArtifacts={relatedArtifacts}
          hasMoreEvents={hasMoreEvents}
          isLoadingMoreEvents={isLoadingMoreEvents}
          onLoadMoreEvents={onLoadMoreEvents}
        />
      )}
      {footer ? (
        <div className="relative mt-4 flex gap-3">
          <TimelineMarker icon={MessageSquare} />
          <div className="min-w-0 flex-1">{footer}</div>
        </div>
      ) : null}
    </div>
  );
}

interface TimelineEventsListProps {
  organizationId: string;
  factoryKey: string;
  orderNumber?: string;
  events: WorkOrderTimelineEvent[];
  resolveUserDisplay: OrgUserDisplayLookup;
  latestArtifactDataById: Map<string, Record<string, unknown>>;
  relatedArtifacts: readonly WorkOrderArtifactLike[];
  hasMoreEvents: boolean;
  isLoadingMoreEvents: boolean;
  onLoadMoreEvents?: () => void;
}

function TimelineEventsList({
  organizationId,
  factoryKey,
  orderNumber,
  events,
  resolveUserDisplay,
  latestArtifactDataById,
  relatedArtifacts,
  hasMoreEvents,
  isLoadingMoreEvents,
  onLoadMoreEvents,
}: TimelineEventsListProps) {
  const latestDispatchIndex = findLatestDispatchIndex(events);
  return (
    <>
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

      {events.length > 0 ? (
        <ul className="relative space-y-4">
          <span className="pointer-events-none absolute left-[11px] top-3 bottom-3 w-px bg-border" aria-hidden />
          {events.map((event, index) => (
            <TimelineItem
              key={event.id}
              event={event}
              organizationId={organizationId}
              factoryKey={factoryKey}
              orderNumber={orderNumber}
              resolveUserDisplay={resolveUserDisplay}
              latestArtifactDataById={latestArtifactDataById}
              relatedArtifacts={relatedArtifacts}
              isLatestDispatch={index === latestDispatchIndex}
            />
          ))}
        </ul>
      ) : null}
    </>
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
  factoryKey,
  orderNumber,
  resolveUserDisplay,
  latestArtifactDataById,
  relatedArtifacts,
  isLatestDispatch,
}: {
  event: WorkOrderTimelineEvent;
  organizationId: string;
  factoryKey: string;
  orderNumber?: string;
  resolveUserDisplay: OrgUserDisplayLookup;
  latestArtifactDataById: Map<string, Record<string, unknown>>;
  relatedArtifacts: readonly WorkOrderArtifactLike[];
  isLatestDispatch: boolean;
}) {
  if (event.kind === "dispatched") {
    return (
      <DispatchTimelineItem
        event={event}
        organizationId={organizationId}
        factoryKey={factoryKey}
        orderNumber={orderNumber}
        isLatestDispatch={isLatestDispatch}
        relatedArtifacts={relatedArtifacts}
      />
    );
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
          <TimelineItemBody
            event={event}
            organizationId={organizationId}
            factoryKey={factoryKey}
            orderNumber={orderNumber}
            resolveUserDisplay={resolveUserDisplay}
            latestArtifactDataById={latestArtifactDataById}
            relatedArtifacts={relatedArtifacts}
          />
        </div>
      </div>
    </li>
  );
}

function TimelineItemBody({
  event,
  organizationId,
  factoryKey,
  orderNumber,
  resolveUserDisplay,
  latestArtifactDataById,
  relatedArtifacts,
}: {
  event: WorkOrderTimelineEvent;
  organizationId: string;
  factoryKey: string;
  orderNumber?: string;
  resolveUserDisplay: OrgUserDisplayLookup;
  latestArtifactDataById: Map<string, Record<string, unknown>>;
  relatedArtifacts: readonly WorkOrderArtifactLike[];
}) {
  const actorDisplay = resolveUserDisplay(event.actorUserId, event.actorName);
  const timeLabel = formatTimelineDate(new Date(event.at));

  if (event.kind === "commented") {
    return (
      <CommentEventBody
        event={event}
        actorDisplay={actorDisplay}
        timeLabel={timeLabel}
        organizationId={organizationId}
        factoryKey={factoryKey}
        orderNumber={orderNumber}
      />
    );
  }

  if (event.kind === "artifactAdded") {
    return (
      <ArtifactEventBody
        event={event}
        actorDisplay={actorDisplay}
        timeLabel={timeLabel}
        latestArtifactDataById={latestArtifactDataById}
        relatedArtifacts={relatedArtifacts}
      />
    );
  }

  return (
    <UserActionEventDescription
      event={event}
      organizationId={organizationId}
      factoryKey={factoryKey}
      orderNumber={orderNumber}
      resolveUserDisplay={resolveUserDisplay}
      timeLabel={timeLabel}
    />
  );
}

const AUTOMATION_ATTRIBUTABLE_KINDS: WorkOrderTimelineEventKind[] = ["created", "statusChanged", "closed"];
const SOMEONE_FALLBACK_KINDS: WorkOrderTimelineEventKind[] = ["created", "statusChanged", "closed"];

function UserActionEventDescription({
  event,
  organizationId,
  factoryKey,
  orderNumber,
  resolveUserDisplay,
  timeLabel,
}: {
  event: WorkOrderTimelineEvent;
  organizationId: string;
  factoryKey: string;
  orderNumber?: string;
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
          <SourceRunAttribution
            event={event}
            organizationId={organizationId}
            factoryKey={factoryKey}
            orderNumber={orderNumber}
          />
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

function SourceRunAttribution({
  event,
  organizationId,
  factoryKey,
  orderNumber,
}: {
  event: WorkOrderTimelineEvent;
  organizationId: string;
  factoryKey: string;
  orderNumber?: string;
}) {
  const runHref = getWorkOrderRunHref(organizationId, factoryKey, event.sourceAppId, event.sourceRunId, {
    orderNumber,
  });
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
