import { useMemo, useRef, type MouseEvent } from "react";
import type { CanvasesCanvasNodeExecution } from "@/api-client";
import { TimeAgo } from "@/components/TimeAgo";
import { RUNS_SIDEBAR_ROW_CLASS } from "@/components/CanvasToolSidebar/runsSidebarRowLayout";
import { appPath } from "@/lib/appPaths";
import { isNormalClick } from "@/lib/linkHelpers";
import { cn } from "@/lib/utils";
import { DEFAULT_EVENT_STATE_MAP, type EventState, type EventStateMap } from "@/ui/componentBase";
import { Link, useParams } from "react-router-dom";
import type { SidebarEvent } from "./types";
import { SidebarEventActionsMenu } from "./SidebarEventItem/SidebarEventActionsMenu";

interface CompactSidebarEventRowProps {
  event: SidebarEvent;
  runId?: string | null;
  fetchRunId?: (event: SidebarEvent) => Promise<string | null>;
  onSelectRun?: (runId: string) => void;
  onCancelQueueItem?: (id: string) => void;
  onCancelExecution?: (executionId: string) => void;
  onReEmit?: (nodeId: string, eventOrExecutionId: string) => void;
  getExecutionState?: (
    nodeId: string,
    execution: CanvasesCanvasNodeExecution,
  ) => { map: EventStateMap; state: EventState };
}

function isRunNavigableEvent(event: SidebarEvent): boolean {
  return event.kind === "trigger" || event.kind === "execution";
}

export function CompactSidebarEventRow({
  event,
  runId,
  fetchRunId,
  onSelectRun,
  onCancelQueueItem,
  onCancelExecution,
  onReEmit,
  getExecutionState,
}: CompactSidebarEventRowProps) {
  const { organizationId, appId } = useParams<{ organizationId: string; appId: string }>();
  const isResolvingRef = useRef(false);

  const eventStateStyle = useMemo(() => {
    if (!getExecutionState) {
      return DEFAULT_EVENT_STATE_MAP.neutral;
    }

    if (event.kind === "queue") {
      return DEFAULT_EVENT_STATE_MAP.queued;
    }

    if (event.kind === "trigger") {
      return DEFAULT_EVENT_STATE_MAP[event.state as EventState] || DEFAULT_EVENT_STATE_MAP.neutral;
    }

    const { map, state } = getExecutionState(
      event.nodeId || "",
      event.originalExecution as CanvasesCanvasNodeExecution,
    );
    return map[state];
  }, [event, getExecutionState]);

  const isSelectable = Boolean(onSelectRun && isRunNavigableEvent(event) && (runId || fetchRunId));
  const runHref = organizationId && appId && runId ? appPath(organizationId, appId, `?run=${runId}`) : null;
  const handleReEmit =
    event.kind === "trigger" && onReEmit
      ? () => {
          onReEmit(event.nodeId || "", event.id);
        }
      : undefined;

  const selectRun = async () => {
    if (!onSelectRun || isResolvingRef.current) {
      return;
    }

    isResolvingRef.current = true;
    try {
      if (runId) {
        onSelectRun(runId);
        return;
      }

      if (!fetchRunId) {
        return;
      }

      const resolvedRunId = await fetchRunId(event);
      if (resolvedRunId) {
        onSelectRun(resolvedRunId);
      }
    } finally {
      isResolvingRef.current = false;
    }
  };

  const handleLinkClick = (clickEvent: MouseEvent<HTMLAnchorElement>) => {
    if (!isNormalClick(clickEvent)) {
      return;
    }

    clickEvent.preventDefault();
    clickEvent.stopPropagation();
    void selectRun();
  };

  return (
    <div
      data-testid="compact-sidebar-event-row"
      data-event-state={event.state || "unknown"}
      data-event-kind={event.kind || "execution"}
      className={cn(
        RUNS_SIDEBAR_ROW_CLASS,
        "group relative w-full transition-colors",
        isSelectable ? "cursor-pointer hover:bg-gray-50" : "cursor-default",
      )}
    >
      {isSelectable ? (
        runHref ? (
          <Link
            to={runHref}
            data-testid="compact-sidebar-event-row-select"
            aria-label={event.title}
            onClick={handleLinkClick}
            className="absolute inset-0 z-10"
          />
        ) : (
          <button
            type="button"
            data-testid="compact-sidebar-event-row-select"
            aria-label={event.title}
            onClick={(clickEvent) => {
              clickEvent.stopPropagation();
              void selectRun();
            }}
            className="absolute inset-0 z-10 cursor-pointer border-0 bg-transparent p-0"
          />
        )
      ) : null}
      <span className="pointer-events-none relative z-0 flex min-w-0 flex-1 items-center gap-1.5">
        <span
          aria-label={eventStateStyle.label || event.state || "neutral"}
          title={eventStateStyle.label || event.state || "neutral"}
          className={cn("inline-block h-2 w-2 shrink-0 rounded-full", eventStateStyle.badgeColor)}
        />
        <span className="min-w-0 flex-1 truncate text-xs font-medium text-gray-800">{event.title}</span>
      </span>
      <div className="relative z-10 shrink-0" onClick={(clickEvent) => clickEvent.stopPropagation()}>
        <SidebarEventActionsMenu
          eventId={event.id}
          executionId={event.executionId}
          onCancelQueueItem={onCancelQueueItem}
          onCancelExecution={onCancelExecution}
          eventState={event.state}
          kind={event.kind || "execution"}
          onReEmit={handleReEmit}
        />
      </div>
      {event.receivedAt ? (
        <span className="pointer-events-none relative shrink-0 text-xs tabular-nums text-gray-500">
          <TimeAgo date={event.receivedAt} includeAgo={false} />
        </span>
      ) : null}
    </div>
  );
}
