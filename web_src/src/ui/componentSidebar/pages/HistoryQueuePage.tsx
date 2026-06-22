import React, { useMemo } from "react";
import { Plus } from "lucide-react";
import { RUNS_SIDEBAR_ROW_CLASS } from "@/components/CanvasToolSidebar/runsSidebarRowLayout";
import { cn } from "@/lib/utils";
import { CompactSidebarEventRow } from "../CompactSidebarEventRow";
import { SidebarEventItem } from "../SidebarEventItem";
import type { SidebarEvent } from "../types";
import type { TabData } from "../SidebarEventItem/SidebarEventItem";
import type { CanvasesCanvasNodeExecution } from "@/api-client";
import type { EventState, EventStateMap } from "../../componentBase";

interface HistoryQueuePageProps {
  page: "history" | "queue";
  events: SidebarEvent[];
  openEventIds: Set<string>;
  onToggleOpen: (eventId: string) => void;
  onEventClick?: (event: SidebarEvent) => void;
  getTabData?: (event: SidebarEvent) => TabData | undefined;
  onCancelQueueItem?: (id: string) => void;
  onCancelExecution?: (executionId: string) => void;
  onReEmit?: (nodeId: string, eventOrExecutionId: string) => void;
  getExecutionState?: (
    nodeId: string,
    execution: CanvasesCanvasNodeExecution,
  ) => { map: EventStateMap; state: EventState };
  compact?: boolean;
  resolveRunId?: (event: SidebarEvent) => string | null;
  fetchRunId?: (event: SidebarEvent) => Promise<string | null>;
  onSelectRun?: (runId: string) => void;

  // Pagination props
  hasMoreItems: boolean;
  loadingMoreItems: boolean;
  showMoreCount: number;
  onLoadMoreItems: () => void;
}

export const HistoryQueuePage: React.FC<HistoryQueuePageProps> = ({
  page,
  events,
  openEventIds,
  onToggleOpen,
  onEventClick,
  getTabData,
  onCancelExecution,
  onReEmit,
  getExecutionState,
  compact = false,
  resolveRunId,
  fetchRunId,
  onSelectRun,
  onCancelQueueItem,
  hasMoreItems,
  loadingMoreItems,
  showMoreCount,
  onLoadMoreItems,
}) => {
  const runIdByEventId = useMemo(() => {
    const map = new Map<string, string | null>();
    if (!compact || !resolveRunId) {
      return map;
    }

    for (const event of events) {
      map.set(event.id, resolveRunId(event));
    }

    return map;
  }, [compact, events, resolveRunId]);

  if (compact) {
    return (
      <div className="flex min-h-0 flex-1 flex-col overflow-y-auto">
        {events.length === 0 ? (
          <div className="px-3 py-4 text-center text-xs text-gray-500">No events found</div>
        ) : (
          <>
            {events.map((event) => (
              <CompactSidebarEventRow
                key={event.id}
                event={event}
                runId={runIdByEventId.get(event.id) ?? null}
                fetchRunId={fetchRunId}
                onSelectRun={onSelectRun}
                onCancelQueueItem={page === "queue" ? onCancelQueueItem : undefined}
                onCancelExecution={onCancelExecution}
                onReEmit={onReEmit}
                getExecutionState={getExecutionState}
              />
            ))}
            {hasMoreItems ? (
              <button
                type="button"
                onClick={onLoadMoreItems}
                disabled={loadingMoreItems}
                className={cn(
                  RUNS_SIDEBAR_ROW_CLASS,
                  "w-full text-xs font-medium text-gray-500 transition-colors hover:bg-gray-50 hover:text-gray-800 disabled:cursor-not-allowed disabled:text-gray-400",
                )}
              >
                {!loadingMoreItems ? <Plus className="h-3.5 w-3.5 shrink-0" /> : null}
                {loadingMoreItems ? "Loading..." : `Show ${showMoreCount > 10 ? "10" : showMoreCount} more`}
              </button>
            ) : null}
          </>
        )}
      </div>
    );
  }

  return (
    <div className="flex-1 overflow-y-auto p-4 min-h-0">
      <div className="flex flex-col gap-3 pb-15">
        {page === "history" && (
          <div>
            <h2 className="text-xs font-semibold uppercase tracking-wider text-gray-500 mb-1">Run History</h2>
          </div>
        )}
        {events.length === 0 ? (
          <div className="text-center py-8 text-gray-500 text-sm">No events found</div>
        ) : (
          <>
            {events.map((event, index) => (
              <SidebarEventItem
                key={event.id}
                event={event}
                index={index}
                variant={page === "history" ? "latest" : "queue"}
                isOpen={page !== "history" ? openEventIds.has(event.id) || event.isOpen : false}
                onToggleOpen={onToggleOpen}
                onEventClick={onEventClick}
                tabData={getTabData?.(event)}
                onCancelExecution={onCancelExecution}
                onReEmit={onReEmit}
                getExecutionState={getExecutionState}
              />
            ))}
            {hasMoreItems && (
              <div className="flex justify-center pt-1">
                <button
                  onClick={onLoadMoreItems}
                  disabled={loadingMoreItems}
                  className="flex items-center gap-1 text-sm font-medium text-gray-500 hover:text-gray-800 disabled:text-gray-400 disabled:cursor-not-allowed rounded-md px-2 py-1.5 border border-border shadow-xs"
                >
                  {loadingMoreItems ? null : <Plus size={16} />}
                  {loadingMoreItems ? "Loading..." : `Show ${showMoreCount > 10 ? "10" : showMoreCount} more`}
                </button>
              </div>
            )}
          </>
        )}
      </div>
    </div>
  );
};
