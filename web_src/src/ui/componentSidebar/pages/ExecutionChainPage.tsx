/* eslint-disable @typescript-eslint/no-explicit-any */
import React, { useState, useEffect } from "react";
import { resolveIcon, flattenObject } from "@/lib/utils";
import { ChainItem, type ChainItemData } from "../../chainItem";
import { SidebarEventItem } from "../SidebarEventItem/SidebarEventItem";
import { SidebarEvent } from "../types";
import {
  WorkflowsWorkflowNodeExecution,
  ComponentsNode,
  ComponentsComponent,
  TriggersTrigger,
  BlueprintsBlueprint
} from "@/api-client";
import { EventState, EventStateMap } from "../../componentBase";
function buildExecutionTabData(
  execution: WorkflowsWorkflowNodeExecution,
  _workflowNode: ComponentsNode,
  _workflowNodes: ComponentsNode[],
): { current?: Record<string, any>; payload?: any } {
  const tabData: { current?: Record<string, any>; payload?: any } = {};

  // Current tab: use outputs if available and non-empty, otherwise use metadata
  const hasOutputs = execution.outputs && Object.keys(execution.outputs).length > 0;
  const dataSource = hasOutputs ? execution.outputs : execution.metadata || {};
  const flattened = flattenObject(dataSource);

  const currentData = {
    ...flattened,
    "Execution ID": execution.id,
    "Execution State": execution.state?.replace("STATE_", "").toLowerCase(),
    "Execution Result": execution.result?.replace("RESULT_", "").toLowerCase(),
    "Execution Started": execution.createdAt ? new Date(execution.createdAt).toLocaleString() : undefined,
  };

  // Filter out undefined and empty values
  tabData.current = Object.fromEntries(
    Object.entries(currentData).filter(([_, value]) => value !== undefined && value !== "" && value !== null),
  );

  // Payload tab: execution inputs and outputs (raw data)
  let payload: Record<string, unknown> = {};

  if (execution.outputs) {
    const outputData: unknown[] = Object.values(execution.outputs)?.find((output) => {
      return Array.isArray(output) && output?.length > 0;
    }) as unknown[];

    if (outputData?.length > 0) {
      payload = outputData?.[0] as Record<string, unknown>;
    }
  }

  tabData.payload = payload;

  return tabData;
}

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
  workflowNodes?: ComponentsNode[]; // Workflow spec nodes for metadata lookup
  components?: ComponentsComponent[]; // Component metadata
  triggers?: TriggersTrigger[]; // Trigger metadata
  blueprints?: BlueprintsBlueprint[]; // Blueprint metadata
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
  workflowNodes = [],
  components = [],
  triggers = [],
  blueprints = [],
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

        const transformedItems: ChainItemData[] = rawExecutions.map((exec: any, index: number) => {
          // Find the workflow node for this execution
          const workflowNode = workflowNodes.find((node) => node.id === exec.nodeId);

          // Get metadata based on node type
          let nodeDisplayName = exec.componentName || exec.nodeId || "Unknown";
          let nodeIconSlug = "box";

          if (workflowNode) {
            nodeDisplayName = workflowNode.name || nodeDisplayName;

            // Get icon based on node type
            if (workflowNode.type === "TYPE_COMPONENT" && workflowNode.component?.name) {
              const componentMeta = components.find((c) => c.name === workflowNode.component!.name);
              nodeIconSlug = componentMeta?.icon || "box";
            } else if (workflowNode.type === "TYPE_TRIGGER" && workflowNode.trigger?.name) {
              const triggerMeta = triggers.find((t) => t.name === workflowNode.trigger!.name);
              nodeIconSlug = triggerMeta?.icon || "play";
            } else if (workflowNode.type === "TYPE_BLUEPRINT" && workflowNode.blueprint?.id) {
              const blueprintMeta = blueprints.find((b) => b.id === workflowNode.blueprint!.id);
              nodeIconSlug = blueprintMeta?.icon || "box";
            }
          }

          return {
            id: exec.id || `execution-${index}`,
            nodeId: exec.nodeId || "",
            componentName: exec.componentName || exec.nodeId || "Unknown",
            nodeName: exec.nodeName,
            nodeDisplayName,
            nodeIcon: exec.nodeIcon || "box",
            nodeIconSlug,
            state: exec.state || "neutral",
            executionId: exec.id,
            originalExecution: exec, // Pass the full execution data
            tabData: buildExecutionTabData(exec, workflowNode || {} as ComponentsNode, workflowNodes),
          };
        });

        setChainItems(transformedItems);
      } catch (err) {
        console.error("Failed to load execution chain:", err);
        setError(err instanceof Error ? err.message : "Failed to load execution chain");
      } finally {
        setLoading(false);
      }
    };

    loadChainData();
  }, [eventId, loadExecutionChain, workflowNodes, components, triggers, blueprints]);

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
