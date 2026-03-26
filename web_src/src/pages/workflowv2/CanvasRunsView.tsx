import React, { useMemo, useState } from "react";
import { ChevronDown, ChevronRight, Loader2, Play } from "lucide-react";
import type {
  CanvasesCanvasEventWithExecutions,
  CanvasesCanvasNodeExecution,
  CanvasesCanvasNodeQueueItem,
  ComponentsNode,
} from "@/api-client";
import { cn, resolveIcon } from "@/lib/utils";
import { formatTimeAgo } from "@/utils/date";
import { DEFAULT_EVENT_STATE_MAP, type EventState } from "@/ui/componentBase";
import { getHeaderIconSrc } from "@/ui/componentSidebar/integrationIcons";
import type { SidebarEvent } from "@/ui/componentSidebar/types";
import { getState, getTriggerRenderer } from "./mappers";
import { buildEventInfo, buildExecutionInfo, mapTriggerEventToSidebarEvent } from "./utils";

function resolveNodeIconSlug(
  node: ComponentsNode | undefined,
  componentIconMap: Record<string, string>,
): string | undefined {
  const name = node?.component?.name || node?.trigger?.name;
  if (!name) return undefined;
  return componentIconMap[name];
}

export type RunsStatusFilter = "all" | "completed" | "errors" | "running" | "queued";

export function getExecutionStatus(execution: CanvasesCanvasNodeExecution, nodes: ComponentsNode[]) {
  const node = nodes.find((n) => n.id === execution.nodeId);
  const componentName = node?.component?.name || "";
  const stateResolver = getState(componentName);
  return stateResolver(buildExecutionInfo(execution));
}

export function getAggregateStatus(executions: CanvasesCanvasNodeExecution[], nodes: ComponentsNode[]) {
  const states = executions.map((e) => ({
    state: e.state,
    result: e.result,
    resolved: getExecutionStatus(e, nodes),
  }));

  if (states.some((s) => s.state === "STATE_STARTED" || s.state === "STATE_PENDING")) {
    return "running";
  }
  if (states.some((s) => s.result === "RESULT_FAILED")) {
    return "error";
  }
  if (states.some((s) => s.result === "RESULT_CANCELLED")) {
    return "cancelled";
  }
  if (states.every((s) => s.result === "RESULT_PASSED")) {
    return "completed";
  }
  if (states.every((s) => s.state === "STATE_FINISHED")) {
    return "completed";
  }
  return "queued";
}

function formatDurationMs(ms: number): string {
  if (ms < 1000) return `${ms}ms`;
  if (ms < 60000) {
    const s = ms / 1000;
    return s % 1 === 0 ? `${s}s` : `${s.toFixed(1)}s`;
  }
  const m = ms / 60000;
  if (m % 1 === 0) return `${m}m`;
  if (m < 10) return `${m.toFixed(1)}m`;
  return `${Math.round(m)}m`;
}

export function computeDuration(execution: CanvasesCanvasNodeExecution): string | null {
  if (execution.state !== "STATE_FINISHED" || !execution.createdAt || !execution.updatedAt) {
    return null;
  }
  const ms = new Date(execution.updatedAt).getTime() - new Date(execution.createdAt).getTime();
  return formatDurationMs(ms);
}

export function resolveExecutionDisplayStatus(execution: CanvasesCanvasNodeExecution, nodes: ComponentsNode[]): string {
  const componentState = getExecutionStatus(execution, nodes);
  if (componentState && componentState !== "neutral") return componentState;

  if (execution.state === "STATE_PENDING") return "pending";
  if (execution.state === "STATE_STARTED") return "running";
  if (execution.result === "RESULT_CANCELLED") return "cancelled";
  if (execution.result === "RESULT_FAILED") return "failed";
  if (execution.result === "RESULT_PASSED") return "success";
  return "unknown";
}

export function formatRunTimestamp(value: string): string {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;

  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffHours = diffMs / (1000 * 60 * 60);

  if (diffHours < 24) {
    return formatTimeAgo(date);
  }

  const weekdays = ["Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"];
  const months = ["Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"];
  return `${weekdays[date.getDay()]} ${months[date.getMonth()]} ${date.getDate()}, ${String(date.getHours()).padStart(2, "0")}:${String(date.getMinutes()).padStart(2, "0")}`;
}

const STATUS_TO_EVENT_STATE: Record<string, EventState> = {
  completed: "success",
  success: "success",
  triggered: "triggered",
  finished: "success",
  passed: "success",
  approved: "success",
  opened: "success",
  true: "success",
  waiting: "queued",
  queued: "queued",
  pending: "queued",
  running: "running",
  failed: "failed",
  error: "error",
  rejected: "error",
  false: "error",
  timeout: "failed",
  cancelled: "cancelled",
  "pushed through": "running",
};

const STATUS_LABELS: Record<string, string> = {
  completed: "Completed",
  success: "Success",
  triggered: "Triggered",
  finished: "Finished",
  waiting: "Waiting",
  approved: "Approved",
  rejected: "Rejected",
  "pushed through": "Pushed Through",
  opened: "Opened",
  true: "True",
  false: "False",
  passed: "Passed",
  timeout: "Timeout",
  queued: "Queued",
  failed: "Failed",
  running: "Running",
  pending: "Pending",
  cancelled: "Cancelled",
  error: "Error",
};

export function StatusBadge({ status }: { status: string }) {
  const eventState = STATUS_TO_EVENT_STATE[status] || "neutral";
  const style = DEFAULT_EVENT_STATE_MAP[eventState];
  const label = STATUS_LABELS[status] || status || "Unknown";

  return (
    <div
      className={`uppercase text-[11px] py-[1.5px] px-[5px] font-semibold rounded flex items-center tracking-wide justify-center text-white ${style.badgeColor}`}
    >
      <span>{label}</span>
    </div>
  );
}

export function RunRow({
  event,
  nodes,
  componentIconMap = {},
  queueItems = [],
  isExpanded,
  onToggle,
  onNodeSelect,
  onExecutionSelect,
}: {
  event: CanvasesCanvasEventWithExecutions;
  nodes: ComponentsNode[];
  componentIconMap?: Record<string, string>;
  queueItems?: CanvasesCanvasNodeQueueItem[];
  isExpanded: boolean;
  onToggle: () => void;
  onNodeSelect?: (nodeId: string) => void;
  onExecutionSelect?: (options: {
    nodeId: string;
    eventId: string;
    executionId: string;
    triggerEvent?: SidebarEvent;
  }) => void;
}) {
  const executions = event.executions || [];
  const triggerNode = nodes.find((n) => n.id === event.nodeId);
  const triggerRenderer = getTriggerRenderer(triggerNode?.trigger?.name || "");
  const eventInfo = buildEventInfo(event);
  const { title } = eventInfo ? triggerRenderer.getTitleAndSubtitle({ event: eventInfo }) : { title: "Run" };

  const triggerSidebarEvent = useMemo(() => {
    if (!triggerNode || !event.id) return undefined;
    return mapTriggerEventToSidebarEvent(
      {
        id: event.id,
        canvasId: event.canvasId,
        nodeId: event.nodeId,
        channel: event.channel,
        data: event.data,
        createdAt: event.createdAt,
        customName: event.customName,
      },
      triggerNode,
    );
  }, [event, triggerNode]);

  const aggregateStatus = executions.length > 0 ? getAggregateStatus(executions, nodes) : "queued";
  const totalSteps = executions.length + queueItems.length;

  const totalDuration = useMemo(() => {
    if (!event.createdAt || executions.length === 0) return null;
    const isAllFinished = executions.every((e) => e.state === "STATE_FINISHED");
    if (!isAllFinished && queueItems.length === 0) return null;
    const startMs = new Date(event.createdAt).getTime();
    let latestEndMs = startMs;
    for (const exec of executions) {
      if (exec.updatedAt) {
        const endMs = new Date(exec.updatedAt).getTime();
        if (endMs > latestEndMs) latestEndMs = endMs;
      }
    }
    const diffMs = latestEndMs - startMs;
    if (diffMs <= 0) return null;
    return computeDuration({
      createdAt: event.createdAt,
      updatedAt: new Date(latestEndMs).toISOString(),
      state: "STATE_FINISHED",
    } as CanvasesCanvasNodeExecution);
  }, [event.createdAt, executions, queueItems.length]);

  const triggerIconSrc = getHeaderIconSrc(triggerNode?.trigger?.name);
  const triggerIconSlug = resolveNodeIconSlug(triggerNode, componentIconMap);
  const triggerName = triggerNode?.name || triggerNode?.trigger?.name || "Trigger";

  return (
    <div className="border-b border-gray-200 last:border-b-0">
      <button
        type="button"
        onClick={onToggle}
        className="flex w-full items-center gap-3 px-4 py-1.5 text-left hover:bg-gray-50 transition-colors min-h-8"
        aria-expanded={isExpanded}
      >
        <div className="text-gray-400">
          {isExpanded ? <ChevronDown className="h-4 w-4" /> : <ChevronRight className="h-4 w-4" />}
        </div>
        <div className="flex flex-1 items-center gap-2 min-w-0">
          <div className="flex-shrink-0 w-4 h-4 flex items-center justify-center">
            {triggerIconSrc ? (
              <img src={triggerIconSrc} alt={triggerName} className="h-4 w-4 object-contain" />
            ) : (
              React.createElement(resolveIcon(triggerIconSlug || "bolt"), {
                size: 14,
                className: "text-gray-500",
              })
            )}
          </div>
          <span className="text-xs text-gray-600 truncate flex-shrink-0">{triggerName}</span>
          <span className="text-gray-300">·</span>
          <span className="font-mono text-[10px] text-gray-400">#{event.id?.slice(0, 4)}</span>
          <span className="text-xs font-medium text-gray-900 truncate">{title}</span>
          <StatusBadge status={aggregateStatus} />
          {totalSteps > 0 && (
            <span className="text-xs text-gray-400 whitespace-nowrap">
              {totalSteps} {totalSteps === 1 ? "step" : "steps"}
              {totalDuration && <span className="ml-1">· {totalDuration}</span>}
            </span>
          )}
        </div>
        <span className="text-xs text-gray-500 tabular-nums whitespace-nowrap">
          {event.createdAt ? formatRunTimestamp(event.createdAt) : ""}
        </span>
      </button>

      {isExpanded && (executions.length > 0 || queueItems.length > 0) && (
        <div className="bg-gray-50">
          {executions.map((execution) => {
            const baseNodeId = execution.nodeId?.includes(":") ? execution.nodeId.split(":")[0] : execution.nodeId;
            const node = nodes.find((n) => n.id === execution.nodeId) || nodes.find((n) => n.id === baseNodeId);
            const nodeName = node?.name || execution.nodeId || "Unknown";
            const componentIconSrc = getHeaderIconSrc(node?.component?.name || node?.trigger?.name);
            const componentIconSlug = resolveNodeIconSlug(node, componentIconMap);
            const status = resolveExecutionDisplayStatus(execution, nodes);
            const duration = computeDuration(execution);
            const hasError = execution.result === "RESULT_FAILED" && execution.resultMessage;

            return (
              <div
                key={execution.id}
                className={cn(
                  "flex items-center gap-2 px-4 py-1.5 pl-11 border-t border-gray-200 min-h-8",
                  (onNodeSelect || onExecutionSelect) && "cursor-pointer hover:bg-gray-100 transition-colors",
                )}
                role={onNodeSelect || onExecutionSelect ? "button" : undefined}
                tabIndex={onNodeSelect || onExecutionSelect ? 0 : undefined}
                onClick={() => {
                  if (onExecutionSelect && execution.id && event.id && execution.nodeId) {
                    onExecutionSelect({
                      nodeId: execution.nodeId,
                      eventId: event.id,
                      executionId: execution.id,
                      triggerEvent: triggerSidebarEvent,
                    });
                  } else if (onNodeSelect && execution.nodeId) {
                    onNodeSelect(execution.nodeId);
                  }
                }}
                onKeyDown={(e) => {
                  if (e.key === "Enter" || e.key === " ") {
                    e.preventDefault();
                    if (onExecutionSelect && execution.id && event.id && execution.nodeId) {
                      onExecutionSelect({
                        nodeId: execution.nodeId,
                        eventId: event.id,
                        executionId: execution.id,
                        triggerEvent: triggerSidebarEvent,
                      });
                    } else if (onNodeSelect && execution.nodeId) {
                      onNodeSelect(execution.nodeId);
                    }
                  }
                }}
              >
                <div className="flex-shrink-0 w-3.5 h-3.5 flex items-center justify-center">
                  {componentIconSrc ? (
                    <img src={componentIconSrc} alt={nodeName} className="h-3.5 w-3.5 object-contain" />
                  ) : (
                    React.createElement(resolveIcon(componentIconSlug || "box"), {
                      size: 13,
                      className: "text-gray-400",
                    })
                  )}
                </div>
                <div className="flex flex-1 items-center gap-2 min-w-0">
                  <span className="text-xs text-gray-700 truncate">{nodeName}</span>
                  <StatusBadge status={status} />
                  {duration && <span className="text-xs text-gray-500 tabular-nums">{duration}</span>}
                </div>
                {hasError && (
                  <span className="text-xs text-red-600 truncate max-w-[300px]">{execution.resultMessage}</span>
                )}
                <span className="text-xs text-gray-400 tabular-nums whitespace-nowrap">
                  {execution.createdAt ? formatRunTimestamp(execution.createdAt) : ""}
                </span>
              </div>
            );
          })}
          {queueItems.map((item) => {
            const qBaseNodeId = item.nodeId?.includes(":") ? item.nodeId.split(":")[0] : item.nodeId;
            const node = nodes.find((n) => n.id === item.nodeId) || nodes.find((n) => n.id === qBaseNodeId);
            const nodeName = node?.name || item.nodeId || "Unknown";
            const queueIconSrc = getHeaderIconSrc(node?.component?.name || node?.trigger?.name);
            const queueIconSlug = resolveNodeIconSlug(node, componentIconMap);

            return (
              <div
                key={`q-${item.id}`}
                className={cn(
                  "flex items-center gap-2 px-4 py-1.5 pl-11 border-t border-gray-200 min-h-8",
                  onNodeSelect && "cursor-pointer hover:bg-gray-100 transition-colors",
                )}
                role={onNodeSelect ? "button" : undefined}
                tabIndex={onNodeSelect ? 0 : undefined}
                onClick={() => {
                  if (onNodeSelect && item.nodeId) {
                    onNodeSelect(item.nodeId);
                  }
                }}
                onKeyDown={(e) => {
                  if ((e.key === "Enter" || e.key === " ") && onNodeSelect && item.nodeId) {
                    e.preventDefault();
                    onNodeSelect(item.nodeId);
                  }
                }}
              >
                <div className="flex-shrink-0 w-3.5 h-3.5 flex items-center justify-center">
                  {queueIconSrc ? (
                    <img src={queueIconSrc} alt={nodeName} className="h-3.5 w-3.5 object-contain" />
                  ) : (
                    React.createElement(resolveIcon(queueIconSlug || "box"), {
                      size: 13,
                      className: "text-gray-400",
                    })
                  )}
                </div>
                <div className="flex flex-1 items-center gap-2 min-w-0">
                  <span className="text-xs text-gray-700 truncate">{nodeName}</span>
                  <StatusBadge status="queued" />
                </div>
                <span className="text-xs text-gray-400 tabular-nums whitespace-nowrap">
                  {item.createdAt ? formatRunTimestamp(item.createdAt) : ""}
                </span>
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}

export function filterRunEvents(
  events: CanvasesCanvasEventWithExecutions[],
  nodes: ComponentsNode[],
  statusFilter: RunsStatusFilter,
  searchQuery: string,
) {
  return events.filter((event) => {
    const executions = event.executions || [];

    if (statusFilter !== "all") {
      const aggregate = executions.length > 0 ? getAggregateStatus(executions, nodes) : "queued";
      if (statusFilter === "completed" && aggregate !== "completed") return false;
      if (statusFilter === "errors" && aggregate !== "error" && aggregate !== "cancelled") return false;
      if (statusFilter === "running" && aggregate !== "running") return false;
      if (statusFilter === "queued" && aggregate !== "queued") return false;
    }

    if (searchQuery.trim()) {
      const query = searchQuery.trim().toLowerCase();
      const triggerNode = nodes.find((n) => n.id === event.nodeId);
      const triggerRenderer = getTriggerRenderer(triggerNode?.trigger?.name || "");
      const eventInfo = buildEventInfo(event);
      const { title } = eventInfo ? triggerRenderer.getTitleAndSubtitle({ event: eventInfo }) : { title: "" };

      const searchableText = [
        event.id,
        title,
        triggerNode?.name,
        ...executions.map((e) => {
          const node = nodes.find((n) => n.id === e.nodeId);
          return [node?.name, node?.id, e.resultMessage].filter(Boolean).join(" ");
        }),
      ]
        .filter(Boolean)
        .join(" ")
        .toLowerCase();

      if (!searchableText.includes(query)) return false;
    }

    return true;
  });
}

export function computeRunsCounts(events: CanvasesCanvasEventWithExecutions[], nodes: ComponentsNode[]) {
  let completed = 0;
  let errors = 0;
  let running = 0;
  let queued = 0;
  for (const event of events) {
    const executions = event.executions || [];
    if (executions.length === 0) {
      queued++;
      continue;
    }
    const aggregate = getAggregateStatus(executions, nodes);
    if (aggregate === "completed") completed++;
    else if (aggregate === "error" || aggregate === "cancelled") errors++;
    else if (aggregate === "running") running++;
    else if (aggregate === "queued") queued++;
  }
  return { completed, errors, running, queued, total: events.length };
}

export function RunsConsoleContent({
  events,
  totalCount,
  hasNextPage,
  isFetchingNextPage,
  onLoadMore,
  nodes,
  componentIconMap = {},
  searchQuery,
  nodeQueueItemsMap = {},
  onNodeSelect,
  onExecutionSelect,
}: {
  events: CanvasesCanvasEventWithExecutions[];
  totalCount?: number;
  hasNextPage?: boolean;
  isFetchingNextPage?: boolean;
  onLoadMore?: () => void;
  nodes: ComponentsNode[];
  componentIconMap?: Record<string, string>;
  searchQuery: string;
  nodeQueueItemsMap?: Record<string, CanvasesCanvasNodeQueueItem[]>;
  onNodeSelect?: (nodeId: string) => void;
  onExecutionSelect?: (options: {
    nodeId: string;
    eventId: string;
    executionId: string;
    triggerEvent?: SidebarEvent;
  }) => void;
}) {
  const [statusFilter, setStatusFilter] = useState<RunsStatusFilter>("all");
  const [expandedRuns, setExpandedRuns] = useState<Set<string>>(new Set());

  const toggleRun = (runId: string) => {
    setExpandedRuns((prev) => {
      const next = new Set(prev);
      if (next.has(runId)) {
        next.delete(runId);
      } else {
        next.add(runId);
      }
      return next;
    });
  };

  const { queueItemsByEventId, orphanQueueEvents } = useMemo(() => {
    const map: Record<string, CanvasesCanvasNodeQueueItem[]> = {};
    const orphansByEvent: Record<
      string,
      { event: CanvasesCanvasNodeQueueItem["rootEvent"]; items: CanvasesCanvasNodeQueueItem[] }
    > = {};
    const eventIds = new Set(events.map((e) => e.id));

    for (const items of Object.values(nodeQueueItemsMap)) {
      for (const item of items) {
        const eventId = item.rootEvent?.id;
        if (!eventId) continue;

        if (eventIds.has(eventId)) {
          if (!map[eventId]) map[eventId] = [];
          map[eventId].push(item);
        } else {
          if (!orphansByEvent[eventId]) {
            orphansByEvent[eventId] = { event: item.rootEvent, items: [] };
          }
          orphansByEvent[eventId].items.push(item);
        }
      }
    }

    const orphanEvents: CanvasesCanvasEventWithExecutions[] = Object.entries(orphansByEvent).map(
      ([eventId, { event: rootEvent, items }]) => ({
        id: eventId,
        canvasId: items[0]?.canvasId,
        nodeId: rootEvent?.nodeId,
        channel: rootEvent?.channel,
        data: rootEvent?.data as Record<string, unknown> | undefined,
        createdAt: rootEvent?.createdAt,
        executions: [],
      }),
    );

    return { queueItemsByEventId: map, orphanQueueEvents: orphanEvents };
  }, [nodeQueueItemsMap, events]);

  const allEvents = useMemo(() => {
    if (orphanQueueEvents.length === 0) return events;
    const merged = [...orphanQueueEvents, ...events];
    merged.sort((a, b) => {
      const ta = a.createdAt ? new Date(a.createdAt).getTime() : 0;
      const tb = b.createdAt ? new Date(b.createdAt).getTime() : 0;
      return tb - ta;
    });
    return merged;
  }, [events, orphanQueueEvents]);

  const filteredEvents = useMemo(
    () => filterRunEvents(allEvents, nodes, statusFilter, searchQuery),
    [allEvents, nodes, statusFilter, searchQuery],
  );

  const counts = useMemo(() => computeRunsCounts(allEvents, nodes), [allEvents, nodes]);

  const allCount = totalCount != null && totalCount > 0 ? totalCount : counts.total;

  const filterButtons: { key: RunsStatusFilter; label: string; count?: number }[] = [
    { key: "all", label: "All", count: allCount },
    { key: "completed", label: "Completed", count: counts.completed },
    { key: "errors", label: "Errors", count: counts.errors },
    { key: "running", label: "Running", count: counts.running },
    { key: "queued", label: "Queued", count: counts.queued },
  ];

  return (
    <div className="flex flex-col flex-1 min-h-0">
      <div className="flex items-center gap-1 px-4 py-1.5 border-b border-gray-200">
        {filterButtons.map((btn) => (
          <button
            key={btn.key}
            type="button"
            onClick={() => setStatusFilter(btn.key)}
            className={cn(
              "rounded-md px-2 py-0.5 text-[11px] font-medium transition-colors",
              statusFilter === btn.key ? "bg-slate-900 text-white" : "text-gray-600 hover:bg-gray-100",
            )}
          >
            {btn.label}
            {btn.count !== undefined && btn.count > 0 && (
              <span className={cn("ml-1 tabular-nums", statusFilter === btn.key ? "text-white/70" : "text-gray-400")}>
                {btn.count}
              </span>
            )}
          </button>
        ))}
      </div>
      <div className="flex-1 overflow-auto">
        {allEvents.length === 0 ? (
          <div className="flex flex-col items-center justify-center px-4 py-10 text-center">
            <Play className="h-6 w-6 text-gray-300 mb-2" />
            <p className="text-[13px] font-medium text-gray-600">No runs yet</p>
            <p className="mt-0.5 text-xs text-gray-400">Trigger your canvas to see run history here.</p>
          </div>
        ) : filteredEvents.length === 0 ? (
          <div className="px-4 py-6 text-center">
            <p className="text-[13px] text-gray-500">No runs match the current filters.</p>
          </div>
        ) : (
          <div className="divide-y divide-gray-200">
            {filteredEvents.map((event) => (
              <RunRow
                key={event.id}
                event={event}
                nodes={nodes}
                componentIconMap={componentIconMap}
                queueItems={queueItemsByEventId[event.id || ""] || []}
                isExpanded={expandedRuns.has(event.id || "")}
                onToggle={() => toggleRun(event.id || "")}
                onNodeSelect={onNodeSelect}
                onExecutionSelect={onExecutionSelect}
              />
            ))}
            {hasNextPage && statusFilter === "all" && !searchQuery.trim() && (
              <div className="px-4 py-2 text-center border-t border-gray-200">
                <button
                  type="button"
                  onClick={onLoadMore}
                  disabled={isFetchingNextPage}
                  className="text-xs font-medium text-blue-600 hover:text-blue-700 disabled:text-gray-400 transition-colors"
                >
                  {isFetchingNextPage ? (
                    <span className="inline-flex items-center gap-1">
                      <Loader2 className="h-3 w-3 animate-spin" />
                      Loading...
                    </span>
                  ) : (
                    `Load more (${allEvents.length} of ${allCount})`
                  )}
                </button>
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  );
}
