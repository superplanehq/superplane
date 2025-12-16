/* eslint-disable @typescript-eslint/no-explicit-any */
import React, { useState, useEffect } from "react";
import { resolveIcon } from "@/lib/utils";
import { ChainItem, type ChainItemData } from "../../chainItem";
import { SidebarEventItem } from "../SidebarEventItem/SidebarEventItem";
import { SidebarEvent } from "../types";
import { WorkflowsWorkflowNodeExecution } from "@/api-client";
import { EventState, EventStateMap } from "../../componentBase";

interface ExecutionChainPageProps {
  eventId: string | null;
  triggerEvent?: SidebarEvent;
  loadExecutionChain?: (
    eventId: string,
    nodeId?: string,
    currentExecution?: Record<string, unknown>,
    forceReload?: boolean,
  ) => Promise<any[]>;
  openEventIds: Set<string>;
  onToggleOpen: (itemId: string) => void;
  getExecutionState?: (
    nodeId: string,
    execution: WorkflowsWorkflowNodeExecution,
  ) => { map: EventStateMap; state: EventState };
  getTabData?: (event: SidebarEvent) => any;
  onEventClick?: (event: SidebarEvent) => void;
}

export const ExecutionChainPage: React.FC<ExecutionChainPageProps> = ({
  eventId,
  triggerEvent,
  loadExecutionChain,
  openEventIds,
  onToggleOpen,
  getExecutionState,
  getTabData,
  onEventClick,
}) => {
  const [chainItems, setChainItems] = useState<ChainItemData[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Load execution chain data
  useEffect(() => {
    const loadChainData = async () => {
      if (!eventId || !loadExecutionChain) {
        setLoading(false);
        return;
      }

      try {
        setLoading(true);
        setError(null);

        const rawExecutions = await loadExecutionChain(eventId);

        const transformedItems: ChainItemData[] = rawExecutions.map((exec: any, index: number) => ({
          id: exec.id || `execution-${index}`,
          nodeId: exec.nodeId || "",
          componentName: exec.componentName || exec.nodeId || "Unknown",
          nodeName: exec.nodeName,
          nodeIcon: exec.nodeIcon || "box",
          state: exec.state || "neutral",
          executionId: exec.id,
          originalExecution: exec, // Pass the full execution data
          tabData: {
            current: exec.current || exec.metadata || exec.details,
            payload: exec.payload || exec.outputs || exec.data,
          },
        }));

        setChainItems(transformedItems);
      } catch (err) {
        console.error("Failed to load execution chain:", err);
        setError(err instanceof Error ? err.message : "Failed to load execution chain");
      } finally {
        setLoading(false);
      }
    };

    loadChainData();
  }, [eventId, loadExecutionChain]);

  if (loading) {
    return (
      <div className="flex items-center justify-center py-8">
        <div className="flex flex-col items-center gap-2">
          <div className="animate-spin rounded-full h-6 w-6 border-b-2 border-blue-600"></div>
          <div className="text-xs text-gray-500">Loading execution chain...</div>
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="flex items-center justify-center py-8">
        <div className="flex flex-col items-center gap-2 text-center">
          {React.createElement(resolveIcon("alert-circle"), {
            size: 24,
            className: "text-red-500",
          })}
          <div className="text-sm font-medium text-gray-900">Failed to Load</div>
          <div className="text-xs text-gray-500">{error}</div>
        </div>
      </div>
    );
  }

  if (chainItems.length === 0) {
    return (
      <div className="flex items-center justify-center py-8">
        <div className="flex flex-col items-center gap-2 text-center">
          {React.createElement(resolveIcon("layers"), {
            size: 24,
            className: "text-gray-400",
          })}
          <div className="text-sm font-medium text-gray-600">No Executions Found</div>
          <div className="text-xs text-gray-500">This trigger event doesn't have any associated executions yet.</div>
        </div>
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-0">
      {/* Event Section */}
      {triggerEvent && (
        <div className="mb-6 mt-2">
          <h2 className="text-xs font-semibold uppercase text-gray-500 mb-2 px-1">Event</h2>
          <SidebarEventItem
            event={triggerEvent}
            index={0}
            isOpen={openEventIds.has(triggerEvent.id)}
            onToggleOpen={onToggleOpen}
            onEventClick={onEventClick}
            tabData={getTabData?.(triggerEvent)}
            getExecutionState={getExecutionState}
            variant="latest"
          />
        </div>
      )}

      {/* Executions Section */}
      <div>
        <h2 className="text-xs font-semibold uppercase text-gray-500 mb-2 px-1">
          {chainItems.length} Execution{chainItems.length === 1 ? "" : "s"}
        </h2>
        {chainItems.map((item, index) => (
          <ChainItem
            key={item.id}
            item={item}
            index={index}
            totalItems={chainItems.length}
            isOpen={openEventIds.has(item.id)}
            onToggleOpen={onToggleOpen}
            getExecutionState={getExecutionState}
          />
        ))}
      </div>
    </div>
  );
};
