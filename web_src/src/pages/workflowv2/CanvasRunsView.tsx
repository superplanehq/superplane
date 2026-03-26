import { useMemo, useState } from "react";
import { ChevronDown, ChevronRight, CircleCheck, CircleDot, CircleX, Clock, Loader2, Play, Ban } from "lucide-react";
import type { CanvasesCanvasEventWithExecutions, CanvasesCanvasNodeExecution, ComponentsNode } from "@/api-client";
import { cn } from "@/lib/utils";
import { formatTimeAgo } from "@/utils/date";
import { getState, getTriggerRenderer } from "./mappers";
import { buildEventInfo, buildExecutionInfo } from "./utils";

export type RunsStatusFilter = "all" | "passed" | "failed" | "running";

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
    return "failed";
  }
  if (states.some((s) => s.result === "RESULT_CANCELLED")) {
    return "cancelled";
  }
  if (states.every((s) => s.result === "RESULT_PASSED")) {
    return "passed";
  }
  return "unknown";
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
  if (execution.state === "STATE_PENDING") return "pending";
  if (execution.state === "STATE_STARTED") return "running";
  if (execution.result === "RESULT_CANCELLED") return "cancelled";
  if (execution.result === "RESULT_FAILED") return "failed";
  if (execution.result === "RESULT_PASSED") return "passed";
  return getExecutionStatus(execution, nodes) || "unknown";
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
    passed: {
      label: "Passed",
      className: "bg-emerald-50 text-emerald-700 border-emerald-200",
      icon: <CircleCheck className="h-3 w-3" />,
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
  isExpanded,
  onToggle,
  onNodeSelect,
  onExecutionSelect,
}: {
  event: CanvasesCanvasEventWithExecutions;
  nodes: ComponentsNode[];
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

  const aggregateStatus = executions.length > 0 ? getAggregateStatus(executions, nodes) : "unknown";

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
          {executions.length > 0 && (
            <span className="text-xs text-gray-400">
              {executions.length} {executions.length === 1 ? "step" : "steps"}
            </span>
          )}
        </div>
        <span className="text-xs text-gray-500 tabular-nums whitespace-nowrap">
          {event.createdAt ? formatRunTimestamp(event.createdAt) : ""}
        </span>
      </button>

      {isExpanded && executions.length > 0 && (
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
      const aggregate = executions.length > 0 ? getAggregateStatus(executions, nodes) : "unknown";
      if (statusFilter === "passed" && aggregate !== "passed") return false;
      if (statusFilter === "failed" && aggregate !== "failed" && aggregate !== "cancelled") return false;
      if (statusFilter === "running" && aggregate !== "running") return false;
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
  let passed = 0;
  let failed = 0;
  let running = 0;
  for (const event of events) {
    const executions = event.executions || [];
    if (executions.length === 0) continue;
    const aggregate = getAggregateStatus(executions, nodes);
    if (aggregate === "passed") passed++;
    else if (aggregate === "failed" || aggregate === "cancelled") failed++;
    else if (aggregate === "running") running++;
  }
  return { passed, failed, running, total: events.length };
}

export function RunsConsoleContent({
  events,
  nodes,
  searchQuery,
  onNodeSelect,
  onExecutionSelect,
}: {
  events: CanvasesCanvasEventWithExecutions[];
  nodes: ComponentsNode[];
  searchQuery: string;
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

  const filteredEvents = useMemo(
    () => filterRunEvents(events, nodes, statusFilter, searchQuery),
    [events, nodes, statusFilter, searchQuery],
  );

  const counts = useMemo(() => computeRunsCounts(events, nodes), [events, nodes]);

  const filterButtons: { key: RunsStatusFilter; label: string; count?: number }[] = [
    { key: "all", label: "All", count: counts.total },
    { key: "passed", label: "Passed", count: counts.passed },
    { key: "failed", label: "Failed", count: counts.failed },
    { key: "running", label: "Running", count: counts.running },
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
        {events.length === 0 ? (
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
                isExpanded={expandedRuns.has(event.id || "")}
                onToggle={() => toggleRun(event.id || "")}
                onNodeSelect={onNodeSelect}
                onExecutionSelect={onExecutionSelect}
              />
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
