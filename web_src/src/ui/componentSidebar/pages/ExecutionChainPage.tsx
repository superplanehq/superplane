/* eslint-disable @typescript-eslint/no-explicit-any */
import React, { useState, useEffect, useRef, useCallback, useMemo } from "react";
import { resolveIcon, flattenObject, calcRelativeTimeFromDiff } from "@/lib/utils";
import { ChainItem, type ChainItemData } from "../../chainItem";
import { SidebarEvent } from "../types";
import { formatTimeAgo } from "@/utils/date";
import {
  WorkflowsWorkflowNodeExecution,
  ComponentsNode,
  ComponentsComponent,
  TriggersTrigger,
  BlueprintsBlueprint,
} from "@/api-client";
import { EventState, EventStateMap } from "../../componentBase";
import { ChildExecution } from "@/ui/chainItem/ChainItem";
import { getExecutionDetails } from "@/pages/workflowv2/mappers";

function buildExecutionTabData(
  execution: WorkflowsWorkflowNodeExecution,
  workflowNode: ComponentsNode,
  _workflowNodes: ComponentsNode[],
): { current?: Record<string, any>; payload?: any } {
  const tabData: { current?: Record<string, any>; payload?: any } = {};

  let currentData: Record<string, any> = {};

  if (workflowNode?.component?.name) {
    const customDetails = getExecutionDetails(workflowNode.component.name, execution, workflowNode);
    if (customDetails && Object.keys(customDetails).length > 0) {
      currentData = { ...customDetails };
    }
  }

  // If no custom details, fall back to the original logic
  if (Object.keys(currentData).length === 0) {
    const hasOutputs = execution.outputs && Object.keys(execution.outputs).length > 0;
    const dataSource = hasOutputs ? execution.outputs : execution.metadata || {};
    const flattened = flattenObject(dataSource);

    currentData = {
      ...flattened,
    };
  }

  // Only add error if it's not already in custom details
  // Custom details (like from filter mapper) handle error positioning themselves
  if (
    execution.resultMessage &&
    (execution.resultReason === "RESULT_REASON_ERROR" || execution.result === "RESULT_FAILED") &&
    !("Error" in currentData)
  ) {
    currentData["Error"] = execution.resultMessage;
  }

  if (execution.result === "RESULT_CANCELLED" && !("Cancelled by" in currentData)) {
    const cancelledBy = execution.cancelledBy;
    currentData["Cancelled by"] = cancelledBy?.name || cancelledBy?.id || "Unknown";
  }

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

function convertSidebarEventToChainItem(
  triggerEvent: SidebarEvent,
  workflowNodes: ComponentsNode[] = [],
  _components: ComponentsComponent[] = [],
  triggers: TriggersTrigger[] = [],
  getTabData?: (event: SidebarEvent) => any,
): ChainItemData {
  // Find the workflow node for this trigger event
  const workflowNode = workflowNodes.find((node) => node.id === triggerEvent.nodeId);

  // Get metadata based on node type
  let nodeDisplayName = triggerEvent.title || "Trigger Event";
  let nodeIconSlug = "play";

  if (workflowNode) {
    nodeDisplayName = workflowNode.name || nodeDisplayName;

    // Get icon based on node type
    if (workflowNode.type === "TYPE_TRIGGER" && workflowNode.trigger?.name) {
      const triggerMeta = triggers.find((t) => t.name === workflowNode.trigger!.name);
      nodeIconSlug = triggerMeta?.icon || "play";
    }
  }

  return {
    id: triggerEvent.id,
    nodeId: triggerEvent.nodeId || "",
    componentName: triggerEvent.title || "Trigger Event",
    nodeName: triggerEvent.title,
    nodeDisplayName,
    nodeIcon: "play",
    nodeIconSlug,
    state: triggerEvent.state || "neutral",
    executionId: undefined, // Trigger events don't have execution IDs
    originalExecution: undefined,
    originalEvent: triggerEvent.originalEvent,
    childExecutions: undefined,
    workflowNode,
    tabData: getTabData?.(triggerEvent) || {
      current: triggerEvent.values || {},
      payload: triggerEvent.originalEvent || {},
    },
  };
}

interface ExecutionChainPageProps {
  eventId: string | null;
  triggerEvent?: SidebarEvent;
  selectedExecutionId?: string | null;
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
  onHighlightedNodesChange?: (nodeIds: Set<string>) => void;
}

export const ExecutionChainPage: React.FC<ExecutionChainPageProps> = ({
  eventId,
  triggerEvent,
  selectedExecutionId,
  loadExecutionChain,
  openEventIds,
  onToggleOpen,
  getExecutionState,
  getTabData,
  workflowNodes = [],
  components = [],
  triggers = [],
  blueprints = [],
  onHighlightedNodesChange,
}) => {
  const [chainItems, setChainItems] = useState<ChainItemData[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Ref for the scrollable executions container
  const executionsScrollRef = useRef<HTMLDivElement>(null);

  // Calculate summary information for the header
  const summaryInfo = useMemo(() => {
    if (!triggerEvent || chainItems.length === 0) return null;

    const triggerStartTime = triggerEvent.originalEvent?.createdAt || triggerEvent.receivedAt;
    if (!triggerStartTime) return null;

    // Find the latest execution end time
    const lastExecution = chainItems
      .filter((item) => item.originalExecution?.updatedAt)
      .sort(
        (a, b) =>
          new Date(b.originalExecution!.updatedAt!).getTime() - new Date(a.originalExecution!.updatedAt!).getTime(),
      )[0];

    const endTime = lastExecution?.originalExecution?.updatedAt;

    const startDate = new Date(triggerStartTime);
    const timeAgo = formatTimeAgo(startDate);

    let duration = "";
    if (endTime) {
      const endDate = new Date(endTime);
      const durationMs = endDate.getTime() - startDate.getTime();
      duration = calcRelativeTimeFromDiff(durationMs);
    }

    const stepCount = chainItems.length;

    return {
      timeAgo,
      duration,
      stepCount,
    };
  }, [triggerEvent, chainItems]);

  // Load execution chain data function
  const loadChainData = useCallback(async () => {
    if (!eventId || !loadExecutionChain) {
      setLoading(false);
      return;
    }

    try {
      setLoading(true);
      setError(null);

      const rawExecutions = await loadExecutionChain(eventId, undefined, undefined, true);

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

        // Process child executions for composite components
        let childExecutions: ChildExecution[] | undefined = undefined;
        if (exec.childExecutions && exec.childExecutions.length > 0) {
          childExecutions = exec.childExecutions
            .slice()
            .sort((a: any, b: any) => {
              const timeA = a.createdAt ? new Date(a.createdAt).getTime() : 0;
              const timeB = b.createdAt ? new Date(b.createdAt).getTime() : 0;
              return timeA - timeB;
            })
            .map((childExec: any) => {
              const nodeId = childExec?.nodeId?.split(":")?.at(-1);
              let badgeColor = "bg-gray-400";
              let componentName = "Unknown";
              let componentIcon = "box";

              // Find the blueprint node information
              if (workflowNode?.blueprint?.id && nodeId) {
                const blueprint = blueprints.find((b) => b.id === workflowNode.blueprint!.id);
                if (blueprint?.nodes) {
                  const blueprintNode = blueprint.nodes.find((node: any) => node.id === nodeId);
                  if (blueprintNode) {
                    componentName = blueprintNode.name || blueprintNode.component?.name || "Unknown";

                    // Get component icon from components metadata
                    if (blueprintNode.component?.name) {
                      const componentMeta = components.find((c) => c.name === blueprintNode.component!.name);
                      componentIcon = componentMeta?.icon || "box";
                    }
                  }
                }
              }

              if (getExecutionState) {
                const { map, state } = getExecutionState(exec.nodeId, childExec);
                badgeColor = map[state]?.badgeColor || "bg-gray-400";

                return {
                  name: componentName,
                  state: state,
                  nodeId: childExec.nodeId || "",
                  executionId: childExec.id || "",
                  badgeColor,
                  backgroundColor: map[state]?.backgroundColor,
                  componentIcon,
                };
              }

              return {
                name: componentName,
                state: childExec.state?.replace("STATE_", "").toLowerCase() || "unknown",
                nodeId: childExec.nodeId || "",
                executionId: childExec.id || "",
                badgeColor,
                componentIcon,
              };
            });
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
          childExecutions,
          workflowNode,
          tabData: buildExecutionTabData(exec, workflowNode || ({} as ComponentsNode), workflowNodes),
        };
      });

      setChainItems(transformedItems);
    } catch (err) {
      console.error("Failed to load execution chain:", err);
      setError(err instanceof Error ? err.message : "Failed to load execution chain");
    } finally {
      setLoading(false);
    }
  }, [eventId, loadExecutionChain, workflowNodes, components, triggers, blueprints]);

  // Load execution chain data
  useEffect(() => {
    loadChainData();
  }, [loadChainData]);

  // Notify parent of all node IDs in the execution chain for highlighting
  useEffect(() => {
    if (onHighlightedNodesChange && chainItems.length > 0) {
      const nodeIds = new Set<string>();

      // Add trigger event node ID if available
      if (triggerEvent?.nodeId) {
        nodeIds.add(triggerEvent.nodeId);
      }

      // Add all execution node IDs
      chainItems.forEach((item) => {
        if (item.nodeId) {
          nodeIds.add(item.nodeId);
        }
      });

      onHighlightedNodesChange(nodeIds);
    } else if (onHighlightedNodesChange && chainItems.length === 0) {
      // Clear highlights when no items
      onHighlightedNodesChange(new Set());
    }
  }, [chainItems, triggerEvent, onHighlightedNodesChange]);

  // Use ref to track current values without causing re-renders
  const pollingRef = useRef<{
    startedPolling: boolean;
    loadData: (() => void) | null;
  }>({
    startedPolling: false,
    loadData: null,
  });

  pollingRef.current.loadData = loadChainData;

  // Polling effect for in-progress executions
  useEffect(() => {
    const pollInterval = setInterval(() => {
      const { loadData } = pollingRef.current;
      if (loadData) {
        pollingRef.current.startedPolling = true;
        loadData();
      }
    }, 1500);

    return () => {
      clearInterval(pollInterval);
    };
  }, []);

  // Auto-scroll to selected execution
  useEffect(() => {
    if (selectedExecutionId && executionsScrollRef.current && chainItems.length > 0) {
      const selectedElement = executionsScrollRef.current.querySelector(
        `[data-execution-id="${selectedExecutionId}"]`,
      ) as HTMLElement;

      if (selectedElement && !pollingRef.current.startedPolling) {
        const container = executionsScrollRef.current;
        const containerRect = container.getBoundingClientRect();
        const elementRect = selectedElement.getBoundingClientRect();

        // Calculate the scroll position to center the element in the container
        const scrollTop =
          container.scrollTop + elementRect.top - containerRect.top - containerRect.height / 2 + elementRect.height / 2;

        container.scrollTo({
          top: scrollTop,
          behavior: "smooth",
        });
      }
    }
  }, [selectedExecutionId, chainItems.length]);

  if (loading && !pollingRef.current.startedPolling) {
    return (
      <div className="flex items-center justify-center py-8">
        <div className="flex flex-col items-center gap-2">
          <div className="animate-spin rounded-full h-6 w-6 border-b border-blue-600"></div>
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
          <div className="text-sm font-medium text-gray-800">Failed to Load</div>
          <div className="text-xs text-gray-500">{error}</div>
        </div>
      </div>
    );
  }

  return (
    <div className="flex flex-col h-full">
      {/* Scrollable Section with Event and Executions */}
      <div className="flex-1 flex flex-col min-h-0">
        <div ref={executionsScrollRef} className="flex-1 overflow-y-auto p-4">
          <div className="pb-15">
            {triggerEvent && (
              <div className="mb-6 border-b-1 border-border pb-4">
                <h2 className="text-sm font-medium text-gray-800 flex items-center gap-2">
                  {triggerEvent.id && (
                    <span className="text-[13px] text-gray-500 font-mono">#{triggerEvent.id.slice(0, 4)}</span>
                  )}
                  {triggerEvent.title || "Execution Chain"}
                </h2>
                {summaryInfo && (
                  <div className="text-[13px] text-gray-500">
                    {summaryInfo.timeAgo}
                    {summaryInfo.duration && (
                      <>
                        <span className="mx-1">·</span>
                        Duration: {summaryInfo.duration}
                      </>
                    )}
                    <span className="mx-1">·</span>
                    {summaryInfo.stepCount} step{summaryInfo.stepCount !== 1 ? "s" : ""}
                  </div>
                )}
              </div>
            )}
            {/* Event Section (now scrollable) */}
            {triggerEvent && (
              <div className="mb-6">
                <h2 className="text-[13px] font-medium text-gray-500 mb-3">This run was triggered by</h2>
                <ChainItem
                  item={convertSidebarEventToChainItem(triggerEvent, workflowNodes, components, triggers, getTabData)}
                  index={-1}
                  totalItems={undefined}
                  isOpen={openEventIds.has(triggerEvent.id)}
                  isSelected={false}
                  onToggleOpen={onToggleOpen}
                  getExecutionState={getExecutionState}
                />
              </div>
            )}

            {/* Executions Section */}
            <div>
              <h2 className="text-[13px] font-medium text-gray-500 mb-3">
                {chainItems.length} Step{chainItems.length === 1 ? "" : "s"}
              </h2>
              {chainItems.length > 0 ? (
                <div>
                  {chainItems.map((item, index) => (
                    <div key={item.id} data-execution-id={item.executionId}>
                      <ChainItem
                        item={item}
                        index={index}
                        totalItems={chainItems.length}
                        isOpen={openEventIds.has(item.id) || item.executionId === selectedExecutionId}
                        isSelected={item.executionId === selectedExecutionId}
                        onToggleOpen={onToggleOpen}
                        getExecutionState={getExecutionState}
                      />
                    </div>
                  ))}
                </div>
              ) : (
                <div className="flex items-center justify-center py-8">
                  <div className="flex flex-col items-center gap-2 text-center">
                    {React.createElement(resolveIcon("layers"), {
                      size: 24,
                      className: "text-gray-400",
                    })}
                    <div className="text-sm font-medium text-gray-500">No Executions Found</div>
                    <div className="text-xs text-gray-500">
                      This trigger event doesn't have any associated executions yet.
                    </div>
                  </div>
                </div>
              )}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};
