import React from "react";
import { Plus } from "lucide-react";
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
  onTriggerNavigate?: (event: SidebarEvent) => void;
  getTabData?: (event: SidebarEvent) => TabData | undefined;
  onCancelExecution?: (executionId: string) => void;
  onReEmit?: (nodeId: string, eventOrExecutionId: string) => void;
  loadExecutionChain?: (
    eventId: string,
    nodeId?: string,
    currentExecution?: Record<string, unknown>,
    forceReload?: boolean,
  ) => Promise<CanvasesCanvasNodeExecution[]>;
  getExecutionState?: (
    nodeId: string,
    execution: CanvasesCanvasNodeExecution,
  ) => { map: EventStateMap; state: EventState };

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
  onTriggerNavigate,
  getTabData,
  onCancelExecution,
  onReEmit,
  loadExecutionChain,
  getExecutionState,
  hasMoreItems,
  loadingMoreItems,
  showMoreCount,
  onLoadMoreItems,
}) => {
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
                onTriggerNavigate={onTriggerNavigate}
                tabData={getTabData?.(event)}
                onCancelExecution={onCancelExecution}
                onReEmit={onReEmit}
                loadExecutionChain={loadExecutionChain}
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
