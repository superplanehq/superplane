import { useCallback, useMemo, useState } from "react";
import type { CanvasesCanvasEventWithExecutions, CanvasesCanvasNodeQueueItem, ComponentsNode } from "@/api-client";
import type { SidebarEvent } from "@/ui/componentSidebar/types";
import { LoadMoreButton } from "./LoadMoreButton";
import { RunRow } from "./RunRow";
import { FilterBar } from "./FilterBar";
import {
  type RunsStatusFilter,
  computeRunsCounts,
  filterRunEvents,
  mergeQueueItemsWithEvents,
} from "@/pages/workflowv2/canvasRunsUtils";
import { Play } from "lucide-react";

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
      <FilterBar
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
