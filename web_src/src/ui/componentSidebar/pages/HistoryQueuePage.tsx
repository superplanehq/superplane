/* eslint-disable @typescript-eslint/no-explicit-any */
import React from "react";
import { Plus } from "lucide-react";
import { SidebarEventItem } from "../SidebarEventItem";
import { SidebarEvent } from "../types";
import { TabData } from "../SidebarEventItem/SidebarEventItem";
import { WorkflowsWorkflowNodeExecution } from "@/api-client";
import { EventState, EventStateMap } from "../../componentBase";

interface HistoryQueuePageProps {
  page: "history" | "queue";
  filteredEvents: SidebarEvent[];
  openEventIds: Set<string>;
  onToggleOpen: (eventId: string) => void;
  onEventClick?: (event: SidebarEvent) => void;
  onTriggerNavigate?: (event: SidebarEvent) => void;
  getTabData?: (event: SidebarEvent) => TabData | undefined;
  onPushThrough?: (executionId: string) => void;
  onCancelExecution?: (executionId: string) => void;
  supportsPushThrough?: boolean;
  onReEmit?: (nodeId: string, eventOrExecutionId: string) => void;
  loadExecutionChain?: (
    eventId: string,
    nodeId?: string,
    currentExecution?: Record<string, unknown>,
    forceReload?: boolean,
  ) => Promise<any[]>;
  getExecutionState?: (
    nodeId: string,
    execution: WorkflowsWorkflowNodeExecution,
  ) => { map: EventStateMap; state: EventState };

  // Pagination props
  hasMoreItems: boolean;
  loadingMoreItems: boolean;
  showMoreCount: number;
  onLoadMoreItems: () => void;

  // Search and filter state
  searchQuery: string;
  statusFilter: string;
}

export const HistoryQueuePage: React.FC<HistoryQueuePageProps> = ({
  page,
  filteredEvents,
  openEventIds,
  onToggleOpen,
  onEventClick,
  onTriggerNavigate,
  getTabData,
  onPushThrough,
  onCancelExecution,
  supportsPushThrough,
  onReEmit,
  loadExecutionChain,
  getExecutionState,
  hasMoreItems,
  loadingMoreItems,
  showMoreCount,
  onLoadMoreItems,
  searchQuery,
  statusFilter,
}) => {
  return (
    <div className="overflow-y-auto px-3" style={{ maxHeight: "70vh" }}>
      <div className="flex flex-col gap-2 pb-15">
        {filteredEvents.length === 0 ? (
          <div className="text-center py-8 text-gray-500 text-sm">
            {searchQuery || statusFilter !== "all" ? "No matching events found" : "No events found"}
          </div>
        ) : (
          <>
            {filteredEvents.map((event, index) => (
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
                onPushThrough={onPushThrough}
                onCancelExecution={onCancelExecution}
                supportsPushThrough={supportsPushThrough}
                onReEmit={onReEmit}
                loadExecutionChain={loadExecutionChain}
                getExecutionState={getExecutionState}
              />
            ))}
            {hasMoreItems && !searchQuery && statusFilter === "all" && (
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
