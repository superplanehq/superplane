import React, { useCallback, useMemo, useState } from "react";
import { ChevronDown, ChevronRight, Play } from "lucide-react";
import type {
  CanvasesCanvasEventWithExecutions,
  CanvasesCanvasNodeExecution,
  CanvasesCanvasNodeQueueItem,
  ComponentsNode,
} from "@/api-client";
import { cn, resolveIcon } from "@/lib/utils";
import { getHeaderIconSrc } from "@/ui/componentSidebar/integrationIcons";
import type { SidebarEvent } from "@/ui/componentSidebar/types";
import { LoadMoreButton, RunsFilterBar } from "./CanvasRunsComponents";
import {
  type RunsStatusFilter,
  computeDuration,
  computeRunsCounts,
  filterRunEvents,
  findNode,
  formatRunTimestamp,
  getAggregateStatus,
  getStatusBadgeProps,
  mergeQueueItemsWithEvents,
  resolveExecutionDisplayStatus,
  resolveNodeIconSlug,
} from "./canvasRunsUtils";
import { getTriggerRenderer } from "./mappers";
import { buildEventInfo, buildTriggerSidebarEvent } from "./utils";

function StatusBadge({ status }: { status: string }) {
  const { badgeColor, label } = getStatusBadgeProps(status);
  return (
    <div
      className={`uppercase text-[11px] py-[1.5px] px-[5px] font-semibold rounded flex items-center tracking-wide justify-center text-white ${badgeColor}`}
    >
      <span>{label}</span>
    </div>
  );
}

function NodeIcon({
  iconSrc,
  iconSlug,
  alt,
  size = 14,
  className = "text-gray-500",
}: {
  iconSrc: string | undefined;
  iconSlug: string | undefined;
  alt: string;
  size?: number;
  className?: string;
}) {
  if (iconSrc) {
    return <img src={iconSrc} alt={alt} style={{ width: size, height: size }} className="object-contain" />;
  }
  return React.createElement(resolveIcon(iconSlug || "box"), { size, className });
}

function handleKeyboardActivation(e: React.KeyboardEvent, handler: () => void) {
  if (e.key === "Enter" || e.key === " ") {
    e.preventDefault();
    handler();
  }
}

function resolveExecutionRowData(
  execution: CanvasesCanvasNodeExecution,
  node: ComponentsNode | undefined,
  componentIconMap: Record<string, string>,
  nodes: ComponentsNode[],
) {
  return {
    nodeName: node?.name || execution.nodeId || "Unknown",
    iconSrc: getHeaderIconSrc(node?.component?.name || node?.trigger?.name),
    iconSlug: resolveNodeIconSlug(node, componentIconMap),
    status: resolveExecutionDisplayStatus(execution, nodes),
    duration: computeDuration(execution),
    errorMessage: execution.result === "RESULT_FAILED" ? execution.resultMessage : undefined,
    timestamp: execution.createdAt ? formatRunTimestamp(execution.createdAt) : undefined,
  };
}

function ExecutionRow({
  execution,
  node,
  componentIconMap,
  nodes,
  onSelect,
}: {
  execution: CanvasesCanvasNodeExecution;
  node: ComponentsNode | undefined;
  componentIconMap: Record<string, string>;
  nodes: ComponentsNode[];
  onSelect?: () => void;
}) {
  const { nodeName, iconSrc, iconSlug, status, duration, errorMessage, timestamp } = resolveExecutionRowData(
    execution,
    node,
    componentIconMap,
    nodes,
  );

  return (
    <div
      className={cn(
        "flex items-center gap-2 px-4 py-1.5 pl-11 border-t border-gray-200 min-h-8",
        onSelect && "cursor-pointer hover:bg-gray-100 transition-colors",
      )}
      role={onSelect ? "button" : undefined}
      tabIndex={onSelect ? 0 : undefined}
      onClick={onSelect}
      onKeyDown={onSelect ? (e) => handleKeyboardActivation(e, onSelect) : undefined}
    >
      <div className="flex-shrink-0 w-3.5 h-3.5 flex items-center justify-center">
        <NodeIcon iconSrc={iconSrc} iconSlug={iconSlug} alt={nodeName} size={13} className="text-gray-400" />
      </div>
      <div className="flex flex-1 items-center gap-2 min-w-0">
        <span className="text-xs text-gray-700 truncate">{nodeName}</span>
        <StatusBadge status={status} />
        {duration && <span className="text-xs text-gray-500 tabular-nums">{duration}</span>}
      </div>
      {errorMessage && <span className="text-xs text-red-600 truncate max-w-[300px]">{errorMessage}</span>}
      {timestamp && <span className="text-xs text-gray-400 tabular-nums whitespace-nowrap">{timestamp}</span>}
    </div>
  );
}

function QueueItemRow({
  item,
  node,
  componentIconMap,
  onNodeSelect,
}: {
  item: CanvasesCanvasNodeQueueItem;
  node: ComponentsNode | undefined;
  componentIconMap: Record<string, string>;
  onNodeSelect?: (nodeId: string) => void;
}) {
  const nodeName = node?.name || item.nodeId || "Unknown";
  const iconSrc = getHeaderIconSrc(node?.component?.name || node?.trigger?.name);
  const iconSlug = resolveNodeIconSlug(node, componentIconMap);

  return (
    <div
      className={cn(
        "flex items-center gap-2 px-4 py-1.5 pl-11 border-t border-gray-200 min-h-8",
        onNodeSelect && "cursor-pointer hover:bg-gray-100 transition-colors",
      )}
      role={onNodeSelect ? "button" : undefined}
      tabIndex={onNodeSelect ? 0 : undefined}
      onClick={() => {
        if (onNodeSelect && item.nodeId) onNodeSelect(item.nodeId);
      }}
      onKeyDown={(e) => {
        if (onNodeSelect && item.nodeId) handleKeyboardActivation(e, () => onNodeSelect(item.nodeId!));
      }}
    >
      <div className="flex-shrink-0 w-3.5 h-3.5 flex items-center justify-center">
        <NodeIcon iconSrc={iconSrc} iconSlug={iconSlug} alt={nodeName} size={13} className="text-gray-400" />
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
}

function RunRowHeader({
  event,
  triggerNode,
  componentIconMap,
  executions,
  queueItemCount,
  isExpanded,
  onToggle,
}: {
  event: CanvasesCanvasEventWithExecutions;
  triggerNode: ComponentsNode | undefined;
  componentIconMap: Record<string, string>;
  executions: CanvasesCanvasNodeExecution[];
  queueItemCount: number;
  isExpanded: boolean;
  onToggle: () => void;
}) {
  const triggerRenderer = getTriggerRenderer(triggerNode?.trigger?.name || "");
  const eventInfo = buildEventInfo(event);
  const { title } = eventInfo ? triggerRenderer.getTitleAndSubtitle({ event: eventInfo }) : { title: "Run" };
  const aggregateStatus = executions.length > 0 ? getAggregateStatus(executions, []) : "queued";
  const totalSteps = executions.length + queueItemCount;

  const errorAckLabel = useMemo(() => {
    const failedExecs = executions.filter((e) => e.result === "RESULT_FAILED");
    if (failedExecs.length === 0) return null;
    return failedExecs.every((e) => e.resultReason === "RESULT_REASON_ERROR_RESOLVED") ? "acknowledged" : "new";
  }, [executions]);

  const totalDuration = useMemo(() => {
    if (!event.createdAt || executions.length === 0) return null;
    if (!executions.every((e) => e.state === "STATE_FINISHED")) return null;
    const startMs = new Date(event.createdAt).getTime();
    let latestEndMs = startMs;
    for (const exec of executions) {
      if (exec.updatedAt) {
        const endMs = new Date(exec.updatedAt).getTime();
        if (endMs > latestEndMs) latestEndMs = endMs;
      }
    }
    if (latestEndMs <= startMs) return null;
    return computeDuration({
      createdAt: event.createdAt,
      updatedAt: new Date(latestEndMs).toISOString(),
      state: "STATE_FINISHED",
    } as CanvasesCanvasNodeExecution);
  }, [event.createdAt, executions]);

  const triggerIconSrc = getHeaderIconSrc(triggerNode?.trigger?.name);
  const triggerIconSlug = resolveNodeIconSlug(triggerNode, componentIconMap);
  const triggerName = triggerNode?.name || triggerNode?.trigger?.name || "Trigger";

  return (
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
          <NodeIcon iconSrc={triggerIconSrc} iconSlug={triggerIconSlug || "bolt"} alt={triggerName} />
        </div>
        <span className="text-xs text-gray-600 truncate flex-shrink-0">{triggerName}</span>
        <span className="text-gray-300">·</span>
        <span className="font-mono text-[10px] text-gray-400">#{event.id?.slice(0, 4)}</span>
        <span className="text-xs font-medium text-gray-900 truncate">{title}</span>
        <StatusBadge status={aggregateStatus} />
        {errorAckLabel && (
          <span
            className={cn(
              "text-[10px] font-medium rounded px-1.5 py-0.5",
              errorAckLabel === "new" ? "bg-red-50 text-red-600" : "bg-gray-100 text-gray-500",
            )}
          >
            {errorAckLabel === "new" ? "New" : "Acknowledged"}
          </span>
        )}
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
  const executions = useMemo(() => event.executions || [], [event.executions]);
  const triggerNode = useMemo(() => nodes.find((n) => n.id === event.nodeId), [nodes, event.nodeId]);
  const triggerSidebarEvent = useMemo(() => buildTriggerSidebarEvent(event, triggerNode), [event, triggerNode]);

  const makeExecutionSelectHandler = useCallback(
    (execution: CanvasesCanvasNodeExecution) => {
      if (onExecutionSelect && execution.id && event.id && execution.nodeId) {
        return () =>
          onExecutionSelect({
            nodeId: execution.nodeId!,
            eventId: event.id!,
            executionId: execution.id!,
            triggerEvent: triggerSidebarEvent,
          });
      }
      if (onNodeSelect && execution.nodeId) {
        return () => onNodeSelect(execution.nodeId!);
      }
      return undefined;
    },
    [onExecutionSelect, onNodeSelect, event.id, triggerSidebarEvent],
  );

  return (
    <div className="border-b border-gray-200 last:border-b-0">
      <RunRowHeader
        event={event}
        triggerNode={triggerNode}
        componentIconMap={componentIconMap}
        executions={executions}
        queueItemCount={queueItems.length}
        isExpanded={isExpanded}
        onToggle={onToggle}
      />
      {isExpanded && (executions.length > 0 || queueItems.length > 0) && (
        <div className="bg-gray-50">
          {executions.map((execution) => (
            <ExecutionRow
              key={execution.id}
              execution={execution}
              node={findNode(nodes, execution.nodeId)}
              componentIconMap={componentIconMap}
              nodes={nodes}
              onSelect={makeExecutionSelectHandler(execution)}
            />
          ))}
          {queueItems.map((item) => (
            <QueueItemRow
              key={`q-${item.id}`}
              item={item}
              node={findNode(nodes, item.nodeId)}
              componentIconMap={componentIconMap}
              onNodeSelect={onNodeSelect}
            />
          ))}
        </div>
      )}
    </div>
  );
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

  const toggleRun = useCallback((runId: string) => {
    setExpandedRuns((prev) => {
      const next = new Set(prev);
      if (next.has(runId)) next.delete(runId);
      else next.add(runId);
      return next;
    });
  }, []);

  const { queueItemsByEventId, allEvents } = useMemo(
    () => mergeQueueItemsWithEvents(events, nodeQueueItemsMap),
    [events, nodeQueueItemsMap],
  );

  const filteredEvents = useMemo(
    () => filterRunEvents(allEvents, nodes, statusFilter, searchQuery),
    [allEvents, nodes, statusFilter, searchQuery],
  );

  const counts = useMemo(() => computeRunsCounts(allEvents, nodes), [allEvents, nodes]);
  const allCount = totalCount != null && totalCount > 0 ? totalCount : counts.total;

  return (
    <div className="flex flex-col flex-1 min-h-0">
      <RunsFilterBar
        statusFilter={statusFilter}
        onFilterChange={setStatusFilter}
        counts={{
          all: allCount,
          completed: counts.completed,
          errors: counts.errors,
          running: counts.running,
          queued: counts.queued,
        }}
      />
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
              <LoadMoreButton
                isFetchingNextPage={isFetchingNextPage}
                onLoadMore={onLoadMore}
                loadedCount={allEvents.length}
                totalCount={allCount}
              />
            )}
          </div>
        )}
      </div>
    </div>
  );
}
