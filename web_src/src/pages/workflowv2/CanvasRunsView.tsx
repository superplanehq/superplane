import { useMemo, useState } from "react";
import {
  ChevronDown,
  ChevronRight,
  CircleCheck,
  CircleDashed,
  CircleDot,
  CircleX,
  Clock,
  Loader2,
  Play,
  Ban,
  Zap,
  ThumbsUp,
  ThumbsDown,
  FastForward,
  DoorOpen,
} from "lucide-react";
import type {
  CanvasesCanvasEventWithExecutions,
  CanvasesCanvasNodeExecution,
  CanvasesCanvasNodeQueueItem,
  ComponentsNode,
} from "@/api-client";
import { cn } from "@/lib/utils";
import { formatTimeAgo } from "@/utils/date";
import { getState, getTriggerRenderer } from "./mappers";
import { buildEventInfo, buildExecutionInfo } from "./utils";

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

export function computeDuration(execution: CanvasesCanvasNodeExecution): string | null {
  if (execution.state !== "STATE_FINISHED" || !execution.createdAt || !execution.updatedAt) {
    return null;
  }
  const ms = new Date(execution.updatedAt).getTime() - new Date(execution.createdAt).getTime();
  if (ms < 1000) return `${ms}ms`;
  if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`;
  return `${(ms / 60000).toFixed(1)}m`;
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

export function StatusBadge({ status }: { status: string }) {
  const config: Record<string, { label: string; className: string; icon: React.ReactNode }> = {
    completed: {
      label: "Completed",
      className: "bg-emerald-50 text-emerald-700 border-emerald-200",
      icon: <CircleCheck className="h-3 w-3" />,
    },
    success: {
      label: "Success",
      className: "bg-emerald-50 text-emerald-700 border-emerald-200",
      icon: <CircleCheck className="h-3 w-3" />,
    },
    triggered: {
      label: "Triggered",
      className: "bg-violet-50 text-violet-700 border-violet-200",
      icon: <Zap className="h-3 w-3" />,
    },
    finished: {
      label: "Finished",
      className: "bg-emerald-50 text-emerald-700 border-emerald-200",
      icon: <CircleCheck className="h-3 w-3" />,
    },
    waiting: {
      label: "Waiting",
      className: "bg-amber-50 text-amber-700 border-amber-200",
      icon: <Clock className="h-3 w-3" />,
    },
    approved: {
      label: "Approved",
      className: "bg-emerald-50 text-emerald-700 border-emerald-200",
      icon: <ThumbsUp className="h-3 w-3" />,
    },
    rejected: {
      label: "Rejected",
      className: "bg-red-50 text-red-700 border-red-200",
      icon: <ThumbsDown className="h-3 w-3" />,
    },
    "pushed through": {
      label: "Pushed Through",
      className: "bg-blue-50 text-blue-700 border-blue-200",
      icon: <FastForward className="h-3 w-3" />,
    },
    opened: {
      label: "Opened",
      className: "bg-emerald-50 text-emerald-700 border-emerald-200",
      icon: <DoorOpen className="h-3 w-3" />,
    },
    true: {
      label: "True",
      className: "bg-emerald-50 text-emerald-700 border-emerald-200",
      icon: <CircleCheck className="h-3 w-3" />,
    },
    false: {
      label: "False",
      className: "bg-red-50 text-red-700 border-red-200",
      icon: <CircleX className="h-3 w-3" />,
    },
    passed: {
      label: "Passed",
      className: "bg-emerald-50 text-emerald-700 border-emerald-200",
      icon: <CircleCheck className="h-3 w-3" />,
    },
    timeout: {
      label: "Timeout",
      className: "bg-gray-50 text-gray-500 border-gray-200",
      icon: <Clock className="h-3 w-3" />,
    },
    queued: {
      label: "Queued",
      className: "bg-amber-50 text-amber-700 border-amber-200",
      icon: <CircleDashed className="h-3 w-3" />,
    },
    failed: {
      label: "Failed",
      className: "bg-red-50 text-red-700 border-red-200",
      icon: <CircleX className="h-3 w-3" />,
    },
    running: {
      label: "Running",
      className: "bg-blue-50 text-blue-700 border-blue-200",
      icon: <Loader2 className="h-3 w-3 animate-spin" />,
    },
    pending: {
      label: "Pending",
      className: "bg-gray-50 text-gray-600 border-gray-200",
      icon: <Clock className="h-3 w-3" />,
    },
    cancelled: {
      label: "Cancelled",
      className: "bg-gray-50 text-gray-500 border-gray-200",
      icon: <Ban className="h-3 w-3" />,
    },
    error: {
      label: "Error",
      className: "bg-red-50 text-red-700 border-red-200",
      icon: <CircleX className="h-3 w-3" />,
    },
  };

  const c = config[status] || {
    label: status || "Unknown",
    className: "bg-gray-50 text-gray-600 border-gray-200",
    icon: <CircleDot className="h-3 w-3" />,
  };

  return (
    <span
      className={cn(
        "inline-flex items-center gap-1 rounded-full border px-2 py-0.5 text-[11px] font-medium",
        c.className,
      )}
    >
      {c.icon}
      {c.label}
    </span>
  );
}

export function RunRow({
  event,
  nodes,
  queueItems = [],
  isExpanded,
  onToggle,
  onNodeSelect,
  onExecutionSelect,
}: {
  event: CanvasesCanvasEventWithExecutions;
  nodes: ComponentsNode[];
  queueItems?: CanvasesCanvasNodeQueueItem[];
  isExpanded: boolean;
  onToggle: () => void;
  onNodeSelect?: (nodeId: string) => void;
  onExecutionSelect?: (options: { nodeId: string; eventId: string; executionId: string }) => void;
}) {
  const executions = event.executions || [];
  const triggerNode = nodes.find((n) => n.id === event.nodeId);
  const triggerRenderer = getTriggerRenderer(triggerNode?.trigger?.name || "");
  const eventInfo = buildEventInfo(event);
  const { title } = eventInfo ? triggerRenderer.getTitleAndSubtitle({ event: eventInfo }) : { title: "Run" };

  const aggregateStatus = executions.length > 0 ? getAggregateStatus(executions, nodes) : "queued";
  const totalSteps = executions.length + queueItems.length;

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
          <span className="font-mono text-xs text-gray-500">#{event.id?.slice(0, 4)}</span>
          <span className="text-xs font-medium text-gray-900 truncate">{title}</span>
          <StatusBadge status={aggregateStatus} />
          {totalSteps > 0 && (
            <span className="text-xs text-gray-400">
              {totalSteps} {totalSteps === 1 ? "step" : "steps"}
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
            const node = nodes.find((n) => n.id === execution.nodeId);
            const nodeName = node?.name || node?.id || execution.nodeId || "Unknown";
            const status = resolveExecutionDisplayStatus(execution, nodes);
            const duration = computeDuration(execution);
            const hasError = execution.result === "RESULT_FAILED" && execution.resultMessage;

            return (
              <div
                key={execution.id}
                className={cn(
                  "flex items-center gap-3 px-4 py-1.5 pl-11 border-t border-gray-200 min-h-8",
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
                      });
                    } else if (onNodeSelect && execution.nodeId) {
                      onNodeSelect(execution.nodeId);
                    }
                  }
                }}
              >
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
            const node = nodes.find((n) => n.id === item.nodeId);
            const nodeName = node?.name || node?.id || item.nodeId || "Unknown";

            return (
              <div
                key={`q-${item.id}`}
                className={cn(
                  "flex items-center gap-3 px-4 py-1.5 pl-11 border-t border-gray-200 min-h-8",
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
  searchQuery: string;
  nodeQueueItemsMap?: Record<string, CanvasesCanvasNodeQueueItem[]>;
  onNodeSelect?: (nodeId: string) => void;
  onExecutionSelect?: (options: { nodeId: string; eventId: string; executionId: string }) => void;
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
