import SemaphoreLogo from "@/assets/semaphore-logo-sign-black.svg";
import { useNodeExecutionStore } from "@/stores/nodeExecutionStore";
import { showErrorToast, showSuccessToast } from "@/utils/toast";
import { QueryClient, useQueries, useQueryClient } from "@tanstack/react-query";
import { Loader2 } from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useNavigate, useParams, useSearchParams } from "react-router-dom";

import {
  BlueprintsBlueprint,
  ComponentsComponent,
  ComponentsEdge,
  ComponentsNode,
  TriggersTrigger,
  WorkflowsListNodeEventsResponse,
  WorkflowsListNodeExecutionsResponse,
  WorkflowsListNodeQueueItemsResponse,
  WorkflowsWorkflow,
  WorkflowsWorkflowEvent,
  WorkflowsWorkflowNodeExecution,
  WorkflowsWorkflowNodeQueueItem,
  workflowsEmitNodeEvent,
  workflowsInvokeNodeExecutionAction,
} from "@/api-client";
import { organizationKeys, useOrganizationRoles, useOrganizationUsers } from "@/hooks/useOrganizationData";

import { useBlueprints, useComponents } from "@/hooks/useBlueprintData";
import { usePageTitle } from "@/hooks/usePageTitle";
import {
  eventExecutionsQueryOptions,
  nodeEventsQueryOptions,
  nodeExecutionsQueryOptions,
  nodeQueueItemsQueryOptions,
  useTriggers,
  useUpdateWorkflow,
  useWorkflow,
  workflowKeys,
} from "@/hooks/useWorkflowData";
import { useWorkflowWebsocket } from "@/hooks/useWorkflowWebsocket";
import { flattenObject } from "@/lib/utils";
import { buildBuildingBlockCategories } from "@/ui/buildingBlocks";
import {
  CANVAS_SIDEBAR_STORAGE_KEY,
  CanvasEdge,
  CanvasNode,
  CanvasPage,
  NewNodeData,
  NodeEditData,
  SidebarData,
  SidebarEvent,
} from "@/ui/CanvasPage";
import { ChainExecutionState, TabData } from "@/ui/componentSidebar/SidebarEventItem/SidebarEventItem";
import { CompositeProps, LastRunState } from "@/ui/composite";
import { getBackgroundColorClass, getColorClass } from "@/utils/colors";
import { filterVisibleConfiguration } from "@/utils/components";
import { formatTimeAgo } from "@/utils/date";
import { withOrganizationHeader } from "@/utils/withOrganizationHeader";
import { getTriggerRenderer } from "./renderers";
import { TriggerRenderer } from "./renderers/types";
import { useOnCancelQueueItemHandler } from "./useOnCancelQueueItemHandler";
import { useNodeHistory } from "@/hooks/useNodeHistory";
import { useQueueHistory } from "@/hooks/useQueueHistory";
import { mapExecutionsToSidebarEvents, mapQueueItemsToSidebarEvents, mapTriggerEventsToSidebarEvents } from "./utils";

type UnsavedChangeKind = "position" | "structural";

export function WorkflowPageV2() {
  const { organizationId, workflowId } = useParams<{
    organizationId: string;
    workflowId: string;
  }>();

  const navigate = useNavigate();
  const [searchParams, setSearchParams] = useSearchParams();
  const queryClient = useQueryClient();
  const updateWorkflowMutation = useUpdateWorkflow(organizationId!, workflowId!);
  const { data: triggers = [], isLoading: triggersLoading } = useTriggers();
  const { data: blueprints = [], isLoading: blueprintsLoading } = useBlueprints(organizationId!);
  const { data: components = [], isLoading: componentsLoading } = useComponents(organizationId!);
  const { data: workflow, isLoading: workflowLoading } = useWorkflow(organizationId!, workflowId!);

  usePageTitle([workflow?.metadata?.name || "Canvas"]);

  // Warm up org users and roles cache so approval specs can pretty-print
  // user IDs as emails and role names as display names.
  // We don't use the values directly here; loading them populates the
  // react-query cache which prepareApprovalNode reads from.
  useOrganizationUsers(organizationId!);
  useOrganizationRoles(organizationId!);

  /**
   * Track if we've already done the initial fit to view.
   * This ref persists across re-renders to prevent viewport changes on save.
   */
  const hasFitToViewRef = useRef(false);

  /**
   * Track if the user has manually toggled the building blocks sidebar.
   * This ref persists across re-renders to preserve user preference.
   */
  const hasUserToggledSidebarRef = useRef(false);

  /**
   * Track the building blocks sidebar state.
   * Initialize based on whether nodes exist (open if no nodes).
   * This ref persists across re-renders to preserve sidebar state.
   */
  const isSidebarOpenRef = useRef<boolean | null>(null);
  if (isSidebarOpenRef.current === null && typeof window !== "undefined") {
    const storedSidebarState = window.localStorage.getItem(CANVAS_SIDEBAR_STORAGE_KEY);
    if (storedSidebarState !== null) {
      try {
        isSidebarOpenRef.current = JSON.parse(storedSidebarState);
        hasUserToggledSidebarRef.current = true;
      } catch (error) {
        console.warn("Failed to parse sidebar state from local storage:", error);
      }
    }
  }
  if (isSidebarOpenRef.current === null && workflow) {
    // Initialize on first render
    isSidebarOpenRef.current = workflow.spec?.nodes?.length === 0;
  }

  /**
   * Track the canvas viewport state.
   * This ref persists across re-renders to preserve viewport position and zoom.
   */
  const viewportRef = useRef<{ x: number; y: number; zoom: number } | undefined>(undefined);

  // Track unsaved changes on the canvas
  const [hasUnsavedChanges, setHasUnsavedChanges] = useState(false);
  const [hasNonPositionalUnsavedChanges, setHasNonPositionalUnsavedChanges] = useState(false);

  // Revert functionality - track initial workflow snapshot
  const [initialWorkflowSnapshot, setInitialWorkflowSnapshot] = useState<WorkflowsWorkflow | null>(null);

  // Use Zustand store for execution data - extract only the methods to avoid recreating callbacks
  // Subscribe to version to ensure React detects all updates
  const storeVersion = useNodeExecutionStore((state) => state.version);
  const getNodeData = useNodeExecutionStore((state) => state.getNodeData);
  const loadNodeDataMethod = useNodeExecutionStore((state) => state.loadNodeData);
  const initializeFromWorkflow = useNodeExecutionStore((state) => state.initializeFromWorkflow);

  // Initialize store from workflow.status on workflow load (only once per workflow)
  const hasInitializedStoreRef = useRef<string | null>(null);
  useEffect(() => {
    if (workflow?.metadata?.id && hasInitializedStoreRef.current !== workflow.metadata.id) {
      initializeFromWorkflow(workflow);
      hasInitializedStoreRef.current = workflow.metadata.id;
    }
  }, [workflow, initializeFromWorkflow]);

  // Build maps from store for canvas display (using initial data from workflow.status and websocket updates)
  // Rebuild whenever store version changes (indicates data was updated)
  const { nodeExecutionsMap, nodeQueueItemsMap, nodeEventsMap } = useMemo<{
    nodeExecutionsMap: Record<string, WorkflowsWorkflowNodeExecution[]>;
    nodeQueueItemsMap: Record<string, WorkflowsWorkflowNodeQueueItem[]>;
    nodeEventsMap: Record<string, WorkflowsWorkflowEvent[]>;
  }>(() => {
    const executionsMap: Record<string, WorkflowsWorkflowNodeExecution[]> = {};
    const queueItemsMap: Record<string, WorkflowsWorkflowNodeQueueItem[]> = {};
    const eventsMap: Record<string, WorkflowsWorkflowEvent[]> = {};

    // Get current store data
    const storeData = useNodeExecutionStore.getState().data;

    storeData.forEach((data, nodeId) => {
      if (data.executions.length > 0) {
        executionsMap[nodeId] = data.executions;
      }
      if (data.queueItems.length > 0) {
        queueItemsMap[nodeId] = data.queueItems;
      }
      if (data.events.length > 0) {
        eventsMap[nodeId] = data.events;
      }
    });

    return { nodeExecutionsMap: executionsMap, nodeQueueItemsMap: queueItemsMap, nodeEventsMap: eventsMap };
  }, [storeVersion]);

  // Execution chain data based on node executions from store
  const { executionChainMap } = useExecutionChainData(workflowId!, nodeExecutionsMap);

  const saveWorkflowSnapshot = useCallback(
    (currentWorkflow: WorkflowsWorkflow) => {
      if (!initialWorkflowSnapshot) {
        setInitialWorkflowSnapshot(JSON.parse(JSON.stringify(currentWorkflow)));
      }
    },
    [initialWorkflowSnapshot],
  );

  // Revert to initial state
  const markUnsavedChange = useCallback((kind: UnsavedChangeKind) => {
    setHasUnsavedChanges(true);
    if (kind === "structural") {
      setHasNonPositionalUnsavedChanges(true);
    }
  }, []);

  const handleRevert = useCallback(() => {
    if (initialWorkflowSnapshot && organizationId && workflowId) {
      // Restore the initial state
      queryClient.setQueryData(workflowKeys.detail(organizationId, workflowId), initialWorkflowSnapshot);

      // Clear the snapshot since we're back to the initial state
      setInitialWorkflowSnapshot(null);

      // Mark as no unsaved changes since we're back to the saved state
      setHasUnsavedChanges(false);
      setHasNonPositionalUnsavedChanges(false);
    }
  }, [initialWorkflowSnapshot, organizationId, workflowId, queryClient]);

  const handleNodeWebsocketEvent = useCallback(
    (nodeId: string, event: string) => {
      if (event.startsWith("event_created")) {
        queryClient.invalidateQueries({
          queryKey: workflowKeys.nodeEventHistory(workflowId!, nodeId),
        });
      }

      if (event.startsWith("execution")) {
        queryClient.invalidateQueries({
          queryKey: workflowKeys.nodeExecutionHistory(workflowId!, nodeId),
        });
      }
    },
    [queryClient, workflowId],
  );

  useWorkflowWebsocket(workflowId!, organizationId!, handleNodeWebsocketEvent);

  // Warn user before leaving page with unsaved changes
  useEffect(() => {
    const handleBeforeUnload = (e: BeforeUnloadEvent) => {
      if (hasUnsavedChanges) {
        e.preventDefault();
        e.returnValue = "Your work isn't saved, unsaved changes will be lost. Are you sure you want to leave?";
      }
    };

    window.addEventListener("beforeunload", handleBeforeUnload);
    return () => window.removeEventListener("beforeunload", handleBeforeUnload);
  }, [hasUnsavedChanges]);

  const buildingBlocks = useMemo(
    () => buildBuildingBlockCategories(triggers, components, blueprints),
    [triggers, components, blueprints],
  );

  const { nodes, edges } = useMemo(() => {
    // Don't prepare data until everything is loaded
    if (!workflow || workflowLoading || triggersLoading || blueprintsLoading || componentsLoading) {
      return { nodes: [], edges: [] };
    }
    return prepareData(
      workflow,
      triggers,
      blueprints,
      components,
      nodeEventsMap,
      nodeExecutionsMap,
      nodeQueueItemsMap,
      workflowId!,
      queryClient,
      organizationId!,
    );
  }, [
    workflow,
    triggers,
    blueprints,
    components,
    nodeEventsMap,
    nodeExecutionsMap,
    nodeQueueItemsMap,
    workflowId,
    queryClient,
    workflowLoading,
    triggersLoading,
    blueprintsLoading,
    componentsLoading,
    organizationId,
  ]);

  const getSidebarData = useCallback(
    (nodeId: string): SidebarData | null => {
      const node = workflow?.spec?.nodes?.find((n) => n.id === nodeId);
      if (!node) return null;
      setCurrentHistoryNode({ nodeId, nodeType: node?.type || "TYPE_ACTION" });

      // Get current data from store (don't trigger load here - that's done in useEffect)
      const nodeData = getNodeData(nodeId);

      // Build maps with current node data for sidebar
      const executionsMap = nodeData.executions.length > 0 ? { [nodeId]: nodeData.executions } : {};
      const queueItemsMap = nodeData.queueItems.length > 0 ? { [nodeId]: nodeData.queueItems } : {};
      const eventsMapForSidebar = nodeData.events.length > 0 ? { [nodeId]: nodeData.events } : nodeEventsMap; // Fall back to existing events map for trigger nodes

      // Try to get total count from API cache if available
      let totalHistoryCount: number | undefined;
      if (workflowId) {
        if (node.type === "TYPE_TRIGGER") {
          const eventsCacheData = queryClient.getQueryData(
            nodeEventsQueryOptions(workflowId, nodeId, { limit: 10 }).queryKey,
          ) as WorkflowsListNodeEventsResponse;
          totalHistoryCount = eventsCacheData?.totalCount;
        } else {
          const executionsCacheData = queryClient.getQueryData(
            nodeExecutionsQueryOptions(workflowId, nodeId, { limit: 10 }).queryKey,
          ) as WorkflowsListNodeExecutionsResponse;
          totalHistoryCount = executionsCacheData?.totalCount;
        }
      }

      let totalQueueCount: number | undefined;
      if (workflowId) {
        const queueItemsCacheData = queryClient.getQueryData(
          nodeQueueItemsQueryOptions(workflowId, nodeId).queryKey,
        ) as WorkflowsListNodeQueueItemsResponse;
        totalQueueCount = queueItemsCacheData?.totalCount;
      }

      const sidebarData = prepareSidebarData(
        node,
        workflow?.spec?.nodes || [],
        blueprints,
        components,
        triggers,
        executionsMap,
        queueItemsMap,
        eventsMapForSidebar,
        totalHistoryCount,
        totalQueueCount,
      );

      // Add loading state to sidebar data
      return {
        ...sidebarData,
        isLoading: nodeData.isLoading,
      };
    },
    [workflow, workflowId, blueprints, components, triggers, nodeEventsMap, getNodeData, queryClient],
  );

  // Trigger data loading when sidebar opens for a node
  const loadSidebarData = useCallback(
    (nodeId: string) => {
      const node = workflow?.spec?.nodes?.find((n) => n.id === nodeId);
      if (!node) return;

      const nodeData = getNodeData(nodeId);

      // Trigger load if not already loaded or loading
      if (!nodeData.isLoaded && !nodeData.isLoading) {
        loadNodeDataMethod(workflowId!, nodeId, node.type!, queryClient);
      }
    },
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [workflow?.spec?.nodes?.map((n) => n.id), workflowId, queryClient, getNodeData, loadNodeDataMethod],
  );

  const onCancelQueueItem = useOnCancelQueueItemHandler({
    workflowId: workflowId!,
    organizationId,
    workflow,
    loadSidebarData,
  });

  const [currentHistoryNode, setCurrentHistoryNode] = useState<{ nodeId: string; nodeType: string } | null>(null);

  const nodeHistoryQuery = useNodeHistory({
    workflowId: workflowId || "",
    nodeId: currentHistoryNode?.nodeId || "",
    nodeType: currentHistoryNode?.nodeType || "TYPE_ACTION",
    allNodes: workflow?.spec?.nodes || [],
    enabled: !!currentHistoryNode && !!workflowId,
  });

  const queueHistoryQuery = useQueueHistory({
    workflowId: workflowId || "",
    nodeId: currentHistoryNode?.nodeId || "",
    allNodes: workflow?.spec?.nodes || [],
    enabled: !!currentHistoryNode && !!workflowId,
  });

  const getAllHistoryEvents = useCallback(
    (nodeId: string): SidebarEvent[] => {
      if (currentHistoryNode?.nodeId === nodeId) {
        return nodeHistoryQuery.getAllHistoryEvents();
      }

      return [];
    },
    [currentHistoryNode, nodeHistoryQuery],
  );
  // Load more history for a specific node
  const handleLoadMoreHistory = useCallback(
    (nodeId: string) => {
      if (!currentHistoryNode || currentHistoryNode.nodeId !== nodeId) {
        setCurrentHistoryNode({ nodeId, nodeType: currentHistoryNode?.nodeType || "TYPE_ACTION" });
      } else {
        nodeHistoryQuery.handleLoadMore();
      }
    },
    [currentHistoryNode, nodeHistoryQuery],
  );

  const getHasMoreHistory = useCallback(
    (nodeId: string): boolean => {
      if (currentHistoryNode?.nodeId === nodeId) {
        return nodeHistoryQuery.hasMoreHistory;
      }
      return false;
    },
    [currentHistoryNode, nodeHistoryQuery.hasMoreHistory],
  );

  const getLoadingMoreHistory = useCallback(
    (nodeId: string): boolean => {
      if (currentHistoryNode?.nodeId === nodeId) {
        return nodeHistoryQuery.isLoadingMore;
      }
      return false;
    },
    [currentHistoryNode, nodeHistoryQuery.isLoadingMore],
  );

  const onLoadMoreQueue = useCallback(
    (nodeId: string) => {
      if (!currentHistoryNode || currentHistoryNode.nodeId !== nodeId) {
        setCurrentHistoryNode({ nodeId, nodeType: currentHistoryNode?.nodeType || "TYPE_ACTION" });
      } else {
        queueHistoryQuery.handleLoadMore();
      }
    },
    [currentHistoryNode, queueHistoryQuery],
  );

  const getAllQueueEvents = useCallback(
    (nodeId: string): SidebarEvent[] => {
      if (currentHistoryNode?.nodeId === nodeId) {
        return queueHistoryQuery.getAllHistoryEvents();
      }

      return [];
    },
    [currentHistoryNode, queueHistoryQuery],
  );

  const getHasMoreQueue = useCallback(
    (nodeId: string): boolean => {
      if (currentHistoryNode?.nodeId === nodeId) {
        return queueHistoryQuery.hasMoreHistory;
      }
      return false;
    },
    [currentHistoryNode, queueHistoryQuery.hasMoreHistory],
  );

  const getLoadingMoreQueue = useCallback(
    (nodeId: string): boolean => {
      if (currentHistoryNode?.nodeId === nodeId) {
        return queueHistoryQuery.isLoadingMore;
      }
      return false;
    },
    [currentHistoryNode, queueHistoryQuery.isLoadingMore],
  );

  const workflowEdges = useMemo(() => workflow?.spec?.edges || [], [workflow?.spec?.edges]);

  /**
   * Builds a topological path to find all nodes that should execute before the given target node.
   * This follows the directed graph structure of the workflow to determine execution order.
   */
  const getNodesBeforeTarget = useCallback(
    (targetNodeId: string): Set<string> => {
      if (workflowEdges.length === 0) {
        return new Set();
      }

      const nodesBefore = new Set<string>();

      const incomingEdges = new Map<string, string[]>();
      workflowEdges.forEach((edge) => {
        if (!incomingEdges.has(edge.targetId!)) {
          incomingEdges.set(edge.targetId!, []);
        }
        incomingEdges.get(edge.targetId!)!.push(edge.sourceId!);
      });

      const visited = new Set<string>();
      const dfs = (nodeId: string) => {
        if (visited.has(nodeId)) return;
        visited.add(nodeId);

        const incomingNodes = incomingEdges.get(nodeId) || [];
        incomingNodes.forEach((sourceNodeId) => {
          nodesBefore.add(sourceNodeId);
          dfs(sourceNodeId);
        });
      };

      dfs(targetNodeId);
      return nodesBefore;
    },
    [workflowEdges],
  );

  const getTabData = useCallback(
    (nodeId: string, event: SidebarEvent): TabData | undefined => {
      const node = workflow?.spec?.nodes?.find((n) => n.id === nodeId);
      if (!node) return undefined;

      if (node.type === "TYPE_TRIGGER") {
        const events = nodeEventsMap[nodeId] || [];
        const triggerEvent = events.find((evt) => evt.id === event.id);

        if (!triggerEvent) return undefined;

        const tabData: TabData = {};
        const triggerRenderer = getTriggerRenderer(node.trigger?.name || "");

        const eventValues = triggerRenderer.getRootEventValues(triggerEvent);

        tabData.current = {
          ...eventValues,
          "Event ID": triggerEvent.id,
          "Node ID": triggerEvent.nodeId,
          "Created At": triggerEvent.createdAt ? new Date(triggerEvent.createdAt).toLocaleString() : undefined,
        };

        // Payload tab: raw event data
        const payload: Record<string, unknown> = {};

        if (triggerEvent.data) {
          payload.data = triggerEvent.data;
        }

        if (Object.keys(payload).length > 0) {
          tabData.payload = payload;
        }

        return Object.keys(tabData).length > 0 ? tabData : undefined;
      }

      // Handle other components (non-triggers) - get execution for this event
      const executions = nodeExecutionsMap[nodeId] || [];
      const execution = executions.find((exec: WorkflowsWorkflowNodeExecution) => exec.id === event.id);

      if (!execution) return undefined;

      // Extract tab data from execution
      const tabData: TabData = {};

      // Current tab: flatten execution outputs for easy viewing
      if (execution.outputs) {
        const flattened = flattenObject(execution.outputs);
        if (Object.keys(flattened).length > 0) {
          tabData.current = {
            ...flattened,
            "Execution ID": execution.id,
            "Execution State": execution.state?.replace("STATE_", "").toLowerCase(),
            "Execution Result": execution.result?.replace("RESULT_", "").toLowerCase(),
            "Execution Started": execution.createdAt ? new Date(execution.createdAt).toLocaleString() : undefined,
          };
        }
      } else {
        // Fallback to basic execution data if no outputs
        tabData.current = {
          "Execution ID": execution.id,
          "Execution State": execution.state,
          "Execution Result": execution.result,
          "Execution Started": execution.createdAt ? new Date(execution.createdAt).toLocaleString() : undefined,
        };
      }

      // Root tab: root event data
      if (execution.rootEvent) {
        const rootTriggerNode = workflow?.spec?.nodes?.find((n) => n.id === execution.rootEvent?.nodeId);
        const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.trigger?.name || "");
        const rootEventValues = rootTriggerRenderer.getRootEventValues(execution.rootEvent);

        tabData.root = {
          ...rootEventValues,
          "Event ID": execution.rootEvent.id,
          "Node ID": execution.rootEvent.nodeId,
          "Created At": execution.rootEvent.createdAt
            ? new Date(execution.rootEvent.createdAt).toLocaleString()
            : undefined,
        };
      }

      // Payload tab: execution inputs and outputs (raw data)
      const payload: Record<string, unknown> = {};

      if (execution.input) {
        payload.input = execution.input;
      }

      if (execution.outputs) {
        payload.outputs = execution.outputs;
      }

      if (execution.metadata) {
        payload.metadata = execution.metadata;
      }

      if (Object.keys(payload).length > 0) {
        tabData.payload = payload;
      }

      // Execution Chain tab: get execution chain for the root event
      if (execution.rootEvent?.id) {
        const executionChain = executionChainMap[execution.rootEvent.id];
        if (executionChain && executionChain.length > 0) {
          const currentExecutionTime = execution.createdAt ? new Date(execution.createdAt).getTime() : Date.now();

          const nodesBefore = getNodesBeforeTarget(nodeId);
          nodesBefore.add(nodeId);

          const executionsUpToCurrent = executionChain.filter((exec) => {
            const execTime = exec.createdAt ? new Date(exec.createdAt).getTime() : 0;
            const isNodeBefore = nodesBefore.has(exec.nodeId || "");
            const isBeforeCurrentTime = execTime <= currentExecutionTime;
            return isNodeBefore && isBeforeCurrentTime;
          });

          // Sort the filtered executions by creation time to get chronological order
          executionsUpToCurrent.sort((a, b) => {
            const timeA = a.createdAt ? new Date(a.createdAt).getTime() : 0;
            const timeB = b.createdAt ? new Date(b.createdAt).getTime() : 0;
            return timeA - timeB;
          });

          // Group executions by node to create hierarchy
          const nodeExecutions: Record<string, WorkflowsWorkflowNodeExecution[]> = {};
          executionsUpToCurrent.forEach((exec) => {
            const execNodeId = exec.nodeId || "unknown";
            if (!nodeExecutions[execNodeId]) {
              nodeExecutions[execNodeId] = [];
            }
            nodeExecutions[execNodeId].push(exec);
          });

          const sortedNodeEntries = Object.values(nodeExecutions)
            .flatMap((execs) => execs)
            .sort((execsA, execsB) => {
              const timeA = execsA.createdAt ? new Date(execsA.createdAt).getTime() : 0;
              const timeB = execsB.createdAt ? new Date(execsB.createdAt).getTime() : 0;
              return timeA - timeB;
            });

          const nodesById = workflow?.spec?.nodes?.reduce(
            (acc, node) => {
              if (!node?.id) return acc;
              acc[node.id] = node;
              return acc;
            },
            {} as Record<string, ComponentsNode>,
          );

          const chainData = sortedNodeEntries.map((exec) => {
            const nodeInfo = nodesById?.[exec.nodeId || ""];

            const getSidebarEventItemState = (exec: WorkflowsWorkflowNodeExecution) => {
              if (exec.state === "STATE_FINISHED") {
                if (exec.result === "RESULT_PASSED") {
                  return ChainExecutionState.COMPLETED;
                }
                return ChainExecutionState.FAILED;
              }

              if (exec.state === "STATE_STARTED") {
                return ChainExecutionState.RUNNING;
              }

              return ChainExecutionState.FAILED;
            };

            const mainItem = {
              name: nodeInfo?.name || exec.nodeId || "Unknown",
              state: getSidebarEventItemState(exec),
              children:
                exec?.childExecutions && exec.childExecutions.length > 0
                  ? exec.childExecutions
                      .slice()
                      .sort((a, b) => {
                        const timeA = a.createdAt ? new Date(a.createdAt).getTime() : 0;
                        const timeB = b.createdAt ? new Date(b.createdAt).getTime() : 0;
                        return timeA - timeB;
                      })
                      .map((childExec) => {
                        const childNodeId = childExec?.nodeId?.split(":")?.at(-1);

                        return {
                          name: childNodeId || "Unknown",
                          state: getSidebarEventItemState(childExec),
                        };
                      })
                  : undefined,
            };

            return mainItem;
          });

          if (chainData.length > 0) {
            tabData.executionChain = chainData;
          }
        }
      }

      return Object.keys(tabData).length > 0 ? tabData : undefined;
    },
    [workflow, nodeExecutionsMap, nodeEventsMap, executionChainMap, getNodesBeforeTarget],
  );

  const getNodeEditData = useCallback(
    (nodeId: string): NodeEditData | null => {
      const node = workflow?.spec?.nodes?.find((n) => n.id === nodeId);
      if (!node) return null;

      // Get configuration fields from metadata based on node type
      let configurationFields: ComponentsComponent["configuration"] = [];
      let displayLabel: string | undefined = node.name || undefined;

      if (node.type === "TYPE_BLUEPRINT") {
        const blueprintMetadata = blueprints.find((b) => b.id === node.blueprint?.id);
        configurationFields = blueprintMetadata?.configuration || [];
        displayLabel = blueprintMetadata?.name || displayLabel;
      } else if (node.type === "TYPE_COMPONENT") {
        const componentMetadata = components.find((c) => c.name === node.component?.name);
        configurationFields = componentMetadata?.configuration || [];
        displayLabel = componentMetadata?.label || displayLabel;
      } else if (node.type === "TYPE_TRIGGER") {
        const triggerMetadata = triggers.find((t) => t.name === node.trigger?.name);
        configurationFields = triggerMetadata?.configuration || [];
        displayLabel = triggerMetadata?.label || displayLabel;
      }

      return {
        nodeId: node.id!,
        nodeName: node.name!,
        displayLabel,
        configuration: node.configuration || {},
        configurationFields,
      };
    },
    [workflow, blueprints, components, triggers],
  );

  const handleNodeConfigurationSave = useCallback(
    (nodeId: string, updatedConfiguration: Record<string, any>, updatedNodeName: string) => {
      if (!workflow || !organizationId || !workflowId) return;

      // Save snapshot before making changes
      saveWorkflowSnapshot(workflow);

      // Update the node's configuration and name in local cache only
      const updatedNodes = workflow?.spec?.nodes?.map((node) =>
        node.id === nodeId
          ? {
              ...node,
              configuration: updatedConfiguration,
              name: updatedNodeName,
            }
          : node,
      );

      const updatedWorkflow = {
        ...workflow,
        spec: {
          ...workflow.spec,
          nodes: updatedNodes,
        },
      };

      // Update local cache without triggering API call
      queryClient.setQueryData(workflowKeys.detail(organizationId, workflowId), updatedWorkflow);
      markUnsavedChange("structural");
    },
    [workflow, organizationId, workflowId, queryClient, saveWorkflowSnapshot, markUnsavedChange],
  );

  const generateNodeId = (blockName: string, nodeName: string) => {
    const randomChars = Math.random().toString(36).substring(2, 8);
    const sanitizedBlock = blockName.toLowerCase().replace(/[^a-z0-9]/g, "-");
    const sanitizedName = nodeName.toLowerCase().replace(/[^a-z0-9]/g, "-");
    return `${sanitizedBlock}-${sanitizedName}-${randomChars}`;
  };

  const handleNodeAdd = useCallback(
    (newNodeData: NewNodeData) => {
      if (!workflow || !organizationId || !workflowId) return;

      // Save snapshot before making changes
      saveWorkflowSnapshot(workflow);

      const { buildingBlock, nodeName, configuration, position } = newNodeData;

      // Filter configuration to only include visible fields
      const filteredConfiguration = filterVisibleConfiguration(configuration, buildingBlock.configuration || []);

      // Generate a unique node ID
      const newNodeId = generateNodeId(buildingBlock.name || "node", nodeName.trim());

      // Create the new node
      const newNode: ComponentsNode = {
        id: newNodeId,
        name: nodeName.trim(),
        type:
          buildingBlock.type === "trigger"
            ? "TYPE_TRIGGER"
            : buildingBlock.type === "blueprint"
              ? "TYPE_BLUEPRINT"
              : "TYPE_COMPONENT",
        configuration: filteredConfiguration,
        position: position || {
          x: (workflow?.spec?.nodes?.length || 0) * 250,
          y: 100,
        },
      };

      // Add type-specific reference
      if (buildingBlock.type === "component") {
        newNode.component = { name: buildingBlock.name };
      } else if (buildingBlock.type === "trigger") {
        newNode.trigger = { name: buildingBlock.name };
      } else if (buildingBlock.type === "blueprint") {
        newNode.blueprint = { id: buildingBlock.id };
      }

      // Add the new node to the workflow
      const updatedNodes = [...(workflow.spec?.nodes || []), newNode];

      const updatedWorkflow = {
        ...workflow,
        spec: {
          ...workflow.spec,
          nodes: updatedNodes,
        },
      };

      // Update local cache
      queryClient.setQueryData(workflowKeys.detail(organizationId, workflowId), updatedWorkflow);
      markUnsavedChange("structural");
    },
    [workflow, organizationId, workflowId, queryClient, saveWorkflowSnapshot, markUnsavedChange],
  );

  const handleEdgeCreate = useCallback(
    (sourceId: string, targetId: string, sourceHandle?: string | null) => {
      if (!workflow || !organizationId || !workflowId) return;

      // Save snapshot before making changes
      saveWorkflowSnapshot(workflow);

      // Create the new edge
      const newEdge: ComponentsEdge = {
        sourceId,
        targetId,
        channel: sourceHandle || "default",
      };

      // Add the new edge to the workflow
      const updatedEdges = [...(workflow.spec?.edges || []), newEdge];

      const updatedWorkflow = {
        ...workflow,
        spec: {
          ...workflow.spec,
          edges: updatedEdges,
        },
      };

      // Update local cache
      queryClient.setQueryData(workflowKeys.detail(organizationId, workflowId), updatedWorkflow);
      markUnsavedChange("structural");
    },
    [workflow, organizationId, workflowId, queryClient, saveWorkflowSnapshot, markUnsavedChange],
  );

  const handleNodeDelete = useCallback(
    (nodeId: string) => {
      if (!workflow || !organizationId || !workflowId) return;

      // Save snapshot before making changes
      saveWorkflowSnapshot(workflow);

      // Remove the node from the workflow
      const updatedNodes = workflow.spec?.nodes?.filter((node) => node.id !== nodeId);

      // Remove any edges connected to this node
      const updatedEdges = workflow.spec?.edges?.filter((edge) => edge.sourceId !== nodeId && edge.targetId !== nodeId);

      const updatedWorkflow = {
        ...workflow,
        spec: {
          ...workflow.spec,
          nodes: updatedNodes,
          edges: updatedEdges,
        },
      };

      // Update local cache
      queryClient.setQueryData(workflowKeys.detail(organizationId, workflowId), updatedWorkflow);
      markUnsavedChange("structural");
    },
    [workflow, organizationId, workflowId, queryClient, saveWorkflowSnapshot, markUnsavedChange],
  );

  const handleEdgeDelete = useCallback(
    (edgeIds: string[]) => {
      if (!workflow || !organizationId || !workflowId) return;

      // Parse edge IDs to extract sourceId, targetId, and channel
      // Edge IDs are formatted as: `${sourceId}--${targetId}--${channel}`
      const edgesToRemove = edgeIds.map((edgeId) => {
        const parts = edgeId.split("--");
        return {
          sourceId: parts[0],
          targetId: parts[1],
          channel: parts[2],
        };
      });

      // Remove the edges from the workflow
      const updatedEdges = workflow.spec?.edges?.filter((edge) => {
        return !edgesToRemove.some(
          (toRemove) =>
            edge.sourceId === toRemove.sourceId &&
            edge.targetId === toRemove.targetId &&
            edge.channel === toRemove.channel,
        );
      });

      const updatedWorkflow = {
        ...workflow,
        spec: {
          ...workflow.spec,
          edges: updatedEdges,
        },
      };

      // Update local cache
      queryClient.setQueryData(workflowKeys.detail(organizationId, workflowId), updatedWorkflow);
      markUnsavedChange("structural");
    },
    [workflow, organizationId, workflowId, queryClient, markUnsavedChange],
  );

  /**
   * Updates the position of a node in the local cache.
   * Called when a node is dragged in the CanvasPage.
   *
   * @param nodeId - The ID of the node to update.
   * @param position - The new position of the node.
   */
  const handleNodePositionChange = useCallback(
    (nodeId: string, position: { x: number; y: number }) => {
      if (!workflow || !organizationId || !workflowId) return;

      // Save snapshot before making changes
      saveWorkflowSnapshot(workflow);

      const updatedNodes = workflow.spec?.nodes?.map((node) =>
        node.id === nodeId
          ? {
              ...node,
              position: {
                x: Math.round(position.x),
                y: Math.round(position.y),
              },
            }
          : node,
      );

      const updatedWorkflow = {
        ...workflow,
        spec: {
          ...workflow.spec,
          nodes: updatedNodes,
        },
      };

      queryClient.setQueryData(workflowKeys.detail(organizationId, workflowId), updatedWorkflow);
      markUnsavedChange("position");
    },
    [workflow, organizationId, workflowId, queryClient, saveWorkflowSnapshot, markUnsavedChange],
  );

  const handleNodeCollapseChange = useCallback(
    (nodeId: string) => {
      if (!workflow || !organizationId || !workflowId) return;

      // Save snapshot before making changes
      saveWorkflowSnapshot(workflow);

      // Find the current node to determine its collapsed state
      const currentNode = workflow.spec?.nodes?.find((node) => node.id === nodeId);
      if (!currentNode) return;

      // Toggle the collapsed state
      const newIsCollapsed = !currentNode.isCollapsed;

      const updatedNodes = workflow.spec?.nodes?.map((node) =>
        node.id === nodeId
          ? {
              ...node,
              isCollapsed: newIsCollapsed,
            }
          : node,
      );

      const updatedWorkflow = {
        ...workflow,
        spec: {
          ...workflow.spec,
          nodes: updatedNodes,
        },
      };

      queryClient.setQueryData(workflowKeys.detail(organizationId, workflowId), updatedWorkflow);
      markUnsavedChange("structural");
    },
    [workflow, organizationId, workflowId, queryClient, saveWorkflowSnapshot, markUnsavedChange],
  );

  const handleConfigure = useCallback(
    (nodeId: string) => {
      const node = workflow?.spec?.nodes?.find((n) => n.id === nodeId);
      if (!node) return;
      if (node.type === "TYPE_BLUEPRINT" && node.blueprint?.id && organizationId && workflow) {
        // Pass workflow info as URL parameters
        const params = new URLSearchParams({
          fromWorkflow: workflowId!,
          workflowName: workflow.metadata?.name || "Canvas",
        });
        navigate(`/${organizationId}/custom-components/${node.blueprint.id}?${params.toString()}`);
      }
    },
    [workflow, organizationId, workflowId, navigate],
  );

  const handleRun = useCallback(
    async (nodeId: string, channel: string, data: any) => {
      if (!workflowId) return;

      try {
        await workflowsEmitNodeEvent(
          withOrganizationHeader({
            path: {
              workflowId: workflowId,
              nodeId: nodeId,
            },
            body: {
              channel,
              data,
            },
          }),
        );
        // Note: Success toast is shown by EmitEventModal
      } catch (error) {
        console.error("Failed to emit event:", error);
        showErrorToast("Failed to emit event");
        throw error; // Re-throw to let EmitEventModal handle it
      }
    },
    [workflowId],
  );

  const handleNodeDuplicate = useCallback(
    (nodeId: string) => {
      if (!workflow || !organizationId || !workflowId) return;

      const nodeToDuplicate = workflow.spec?.nodes?.find((node) => node.id === nodeId);
      if (!nodeToDuplicate) return;

      saveWorkflowSnapshot(workflow);

      const originalName = nodeToDuplicate.name || "node";
      const duplicateName = `${originalName} copy`;

      let blockName = "node";
      if (nodeToDuplicate.type === "TYPE_TRIGGER" && nodeToDuplicate.trigger?.name) {
        blockName = nodeToDuplicate.trigger.name;
      } else if (nodeToDuplicate.type === "TYPE_COMPONENT" && nodeToDuplicate.component?.name) {
        blockName = nodeToDuplicate.component.name;
      } else if (nodeToDuplicate.type === "TYPE_BLUEPRINT" && nodeToDuplicate.blueprint?.id) {
        // For blueprints, we need to find the blueprint metadata to get the name
        const blueprintMetadata = blueprints.find((b) => b.id === nodeToDuplicate.blueprint?.id);
        blockName = blueprintMetadata?.name || "blueprint";
      }

      const newNodeId = generateNodeId(blockName, duplicateName);

      const offsetX = 50;
      const offsetY = 50;

      const duplicateNode: ComponentsNode = {
        ...nodeToDuplicate,
        id: newNodeId,
        name: duplicateName,
        position: {
          x: (nodeToDuplicate.position?.x || 0) + offsetX,
          y: (nodeToDuplicate.position?.y || 0) + offsetY,
        },
        // Reset collapsed state for the duplicate
        isCollapsed: false,
      };

      // Add the duplicate node to the workflow
      const updatedNodes = [...(workflow.spec?.nodes || []), duplicateNode];

      const updatedWorkflow = {
        ...workflow,
        spec: {
          ...workflow.spec,
          nodes: updatedNodes,
        },
      };

      // Update local cache
      queryClient.setQueryData(workflowKeys.detail(organizationId, workflowId), updatedWorkflow);
      markUnsavedChange("structural");
    },
    [workflow, organizationId, workflowId, blueprints, queryClient, saveWorkflowSnapshot, markUnsavedChange],
  );

  const handleSave = useCallback(
    async (canvasNodes: CanvasNode[]) => {
      if (!workflow || !organizationId || !workflowId) return;

      // Map canvas nodes back to ComponentsNode format with updated positions
      const updatedNodes = workflow.spec?.nodes?.map((node) => {
        const canvasNode = canvasNodes.find((cn) => cn.id === node.id);
        const componentType = (canvasNode?.data?.type as string) || "";
        if (canvasNode) {
          return {
            ...node,
            position: {
              x: Math.round(canvasNode.position.x),
              y: Math.round(canvasNode.position.y),
            },
            isCollapsed: (canvasNode.data[componentType] as { collapsed: boolean })?.collapsed || false,
          };
        }
        return node;
      });

      try {
        await updateWorkflowMutation.mutateAsync({
          name: workflow.metadata?.name!,
          description: workflow.metadata?.description,
          nodes: updatedNodes,
          edges: workflow.spec?.edges,
        });

        showSuccessToast("Canvas changes saved");
        setHasUnsavedChanges(false);
        setHasNonPositionalUnsavedChanges(false);

        // Clear the snapshot since changes are now saved
        setInitialWorkflowSnapshot(null);
      } catch (error: any) {
        console.error("Failed to save changes to the canvas:", error);
        const errorMessage = error?.response?.data?.message || error?.message || "Failed to save changes to the canvas";
        showErrorToast(errorMessage);
      }
    },
    [workflow, organizationId, workflowId, updateWorkflowMutation],
  );

  // Show loading indicator while data is being fetched
  if (workflowLoading || triggersLoading || blueprintsLoading || componentsLoading) {
    return (
      <div className="flex items-center justify-center h-screen">
        <div className="flex flex-col items-center gap-3">
          <Loader2 className="h-8 w-8 animate-spin text-gray-500" />
          <p className="text-sm text-gray-500">Loading canvas...</p>
        </div>
      </div>
    );
  }

  if (!workflow) {
    return null;
  }

  const hasRunBlockingChanges = hasUnsavedChanges && hasNonPositionalUnsavedChanges;

  return (
    <CanvasPage
      // Persist right sidebar in query params
      initialSidebar={{
        isOpen: searchParams.get("sidebar") === "1",
        nodeId: searchParams.get("node") || undefined,
      }}
      onSidebarChange={(open, nodeId) => {
        const next = new URLSearchParams(searchParams);
        if (open) {
          next.set("sidebar", "1");
          if (nodeId) {
            next.set("node", nodeId);
          } else {
            next.delete("node");
          }
        } else {
          next.delete("sidebar");
          next.delete("node");
        }
        setSearchParams(next, { replace: true });
      }}
      onNodeExpand={(nodeId) => {
        const latestExecution = nodeExecutionsMap[nodeId]?.[0];
        const executionId = latestExecution?.id;
        if (executionId) {
          navigate(`/${organizationId}/workflows/${workflowId}/nodes/${nodeId}/${executionId}`);
        }
      }}
      title={workflow.metadata?.name!}
      nodes={nodes}
      edges={edges}
      organizationId={organizationId}
      onDirty={() => markUnsavedChange("structural")}
      getSidebarData={getSidebarData}
      loadSidebarData={loadSidebarData}
      getTabData={getTabData}
      getNodeEditData={getNodeEditData}
      onNodeConfigurationSave={handleNodeConfigurationSave}
      onSave={handleSave}
      onEdgeCreate={handleEdgeCreate}
      onNodeDelete={handleNodeDelete}
      onEdgeDelete={handleEdgeDelete}
      onNodePositionChange={handleNodePositionChange}
      onToggleView={handleNodeCollapseChange}
      onToggleCollapse={() => markUnsavedChange("structural")}
      onRun={handleRun}
      onDuplicate={handleNodeDuplicate}
      onConfigure={handleConfigure}
      buildingBlocks={buildingBlocks}
      onNodeAdd={handleNodeAdd}
      hasFitToViewRef={hasFitToViewRef}
      hasUserToggledSidebarRef={hasUserToggledSidebarRef}
      isSidebarOpenRef={isSidebarOpenRef}
      viewportRef={viewportRef}
      unsavedMessage={hasUnsavedChanges ? "You have unsaved changes" : undefined}
      saveIsPrimary={hasUnsavedChanges}
      saveButtonHidden={!hasUnsavedChanges}
      onUndo={handleRevert}
      canUndo={initialWorkflowSnapshot !== null}
      runDisabled={hasRunBlockingChanges}
      runDisabledTooltip={hasRunBlockingChanges ? "Save canvas changes before running" : undefined}
      onCancelQueueItem={onCancelQueueItem}
      getAllHistoryEvents={getAllHistoryEvents}
      onLoadMoreHistory={handleLoadMoreHistory}
      getHasMoreHistory={getHasMoreHistory}
      getLoadingMoreHistory={getLoadingMoreHistory}
      onLoadMoreQueue={onLoadMoreQueue}
      getAllQueueEvents={getAllQueueEvents}
      getHasMoreQueue={getHasMoreQueue}
      getLoadingMoreQueue={getLoadingMoreQueue}
      breadcrumbs={[
        {
          label: "Canvases",
          href: `/${organizationId}`,
        },
        {
          label: workflow.metadata?.name!,
        },
      ]}
    />
  );
}

function useExecutionChainData(
  workflowId: string,
  nodeExecutionsMap: Record<string, WorkflowsWorkflowNodeExecution[]>,
) {
  // Get all unique root event IDs from executions
  const rootEventIds = useMemo(() => {
    const eventIds = new Set<string>();
    Object.values(nodeExecutionsMap).forEach((executions) => {
      executions.forEach((execution) => {
        if (execution.rootEvent?.id) {
          eventIds.add(execution.rootEvent.id);
        }
      });
    });
    return Array.from(eventIds);
  }, [nodeExecutionsMap]);

  // Fetch execution chains for each unique root event
  const executionChainResults = useQueries({
    queries: rootEventIds.map((eventId) => eventExecutionsQueryOptions(workflowId, eventId)),
  });

  // Check if any queries are still loading
  const isLoading = executionChainResults.some((result) => result.isLoading);

  // Build map of eventId -> execution chain
  const executionChainMap = useMemo(() => {
    const map: Record<string, WorkflowsWorkflowNodeExecution[]> = {};
    rootEventIds.forEach((eventId, index) => {
      const result = executionChainResults[index];
      if (result.data?.executions && result.data.executions.length > 0) {
        map[eventId] = result.data.executions;
      }
    });
    return map;
  }, [executionChainResults, rootEventIds]);

  return { executionChainMap, isLoading };
}

function prepareData(
  workflow: WorkflowsWorkflow,
  triggers: TriggersTrigger[],
  blueprints: BlueprintsBlueprint[],
  components: ComponentsComponent[],
  nodeEventsMap: Record<string, WorkflowsWorkflowEvent[]>,
  nodeExecutionsMap: Record<string, WorkflowsWorkflowNodeExecution[]>,
  nodeQueueItemsMap: Record<string, WorkflowsWorkflowNodeQueueItem[]>,
  workflowId: string,
  queryClient: QueryClient,
  organizationId: string,
): {
  nodes: CanvasNode[];
  edges: CanvasEdge[];
} {
  const edges = workflow?.spec?.edges?.map(prepareEdge) || [];
  const nodes =
    workflow?.spec?.nodes
      ?.map((node) => {
        return prepareNode(
          workflow?.spec?.nodes!,
          node,
          triggers,
          blueprints,
          components,
          nodeEventsMap,
          nodeExecutionsMap,
          nodeQueueItemsMap,
          workflowId,
          queryClient,
          organizationId,
        );
      })
      .map((node) => ({
        ...node,
        dragHandle: ".canvas-node-drag-handle",
      })) || [];

  return { nodes, edges };
}

function prepareTriggerNode(
  node: ComponentsNode,
  triggers: TriggersTrigger[],
  nodeEventsMap: Record<string, WorkflowsWorkflowEvent[]>,
): CanvasNode {
  const triggerMetadata = triggers.find((t) => t.name === node.trigger?.name);
  const renderer = getTriggerRenderer(node.trigger?.name || "");
  const lastEvent = nodeEventsMap[node.id!]?.[0];
  const triggerProps = renderer.getTriggerProps(node, triggerMetadata!, lastEvent);

  // Use node name if available, otherwise fall back to trigger label (from metadata)
  const displayLabel = node.name || triggerMetadata?.label!;

  return {
    id: node.id!,
    position: { x: node.position?.x!, y: node.position?.y! },
    data: {
      type: "trigger",
      label: displayLabel,
      state: "pending" as const,
      outputChannels: ["default"],
      trigger: {
        ...triggerProps,
        collapsed: node.isCollapsed,
      },
    },
  };
}

function prepareCompositeNode(
  nodes: ComponentsNode[],
  node: ComponentsNode,
  blueprints: BlueprintsBlueprint[],
  nodeExecutionsMap: Record<string, WorkflowsWorkflowNodeExecution[]>,
  nodeQueueItemsMap: Record<string, WorkflowsWorkflowNodeQueueItem[]>,
): CanvasNode {
  const blueprintMetadata = blueprints.find((b) => b.id === node.blueprint?.id);
  const isMissing = !blueprintMetadata;
  const color = blueprintMetadata?.color || "gray";
  const executions = nodeExecutionsMap[node.id!] || [];
  const queueItems = nodeQueueItemsMap[node.id!] || [];

  // Use node name if available, otherwise fall back to blueprint name (from metadata)
  const displayLabel = node.name || blueprintMetadata?.name!;

  const configurationFields = blueprintMetadata?.configuration || [];
  const fieldLabelMap = configurationFields.reduce<Record<string, string>>((acc, field) => {
    if (field.name) {
      acc[field.name] = field.label || field.name;
    }
    return acc;
  }, {});

  const canvasNode: CanvasNode = {
    id: node.id!,
    position: { x: node.position?.x!, y: node.position?.y! },
    data: {
      type: "composite",
      label: displayLabel,
      state: "pending" as const,
      outputChannels: blueprintMetadata?.outputChannels?.map((c) => c.name!) || ["default"],
      composite: {
        iconSlug: blueprintMetadata?.icon || "box-x",
        iconColor: getColorClass(color),
        iconBackground: getBackgroundColorClass(color),
        headerColor: getBackgroundColorClass(color),
        collapsedBackground: getBackgroundColorClass(color),
        collapsed: node.isCollapsed,
        title: displayLabel,
        description: blueprintMetadata?.description,
        isMissing: isMissing,
        parameters:
          Object.keys(node.configuration!).length > 0
            ? [
                {
                  icon: "cog",
                  items: Object.keys(node.configuration!).reduce(
                    (acc, key) => {
                      const displayKey = fieldLabelMap[key] || key;
                      acc[displayKey] = `${node.configuration![key]}`;
                      return acc;
                    },
                    {} as Record<string, string>,
                  ),
                },
              ]
            : [],
      },
    },
  };

  if (executions.length > 0) {
    const execution = executions[0];
    const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
    const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.trigger?.name || "");

    const { title, subtitle } = rootTriggerRenderer.getTitleAndSubtitle(execution.rootEvent!);
    (canvasNode.data.composite as CompositeProps).lastRunItem = {
      title: title,
      subtitle: subtitle,
      receivedAt: new Date(execution.createdAt!),
      state: getRunItemState(execution),
      values: rootTriggerRenderer.getRootEventValues(execution.rootEvent!),
      childEventsInfo: {
        count: execution.childExecutions?.length || 0,
        waitingInfos: [],
      },
    };
  }

  if (queueItems.length > 0) {
    const next = queueItems[0] as any;
    let inferredTitle =
      next?.name || next?.input?.title || next?.input?.name || next?.input?.eventTitle || next?.id || "Queued";

    // Heuristic: if the workflow has a single trigger and it is a schedule,
    // show a friendly title consistent with executions.
    const onlyTrigger = nodes.filter((n) => n.type === "TYPE_TRIGGER");
    if (inferredTitle === next?.id || inferredTitle === "Queued") {
      if (onlyTrigger.length === 1 && onlyTrigger[0]?.trigger?.name === "schedule") {
        inferredTitle = "Event emitted by schedule";
      }
    }

    const inferredSubtitle: string =
      (typeof next?.input?.subtitle === "string" && next?.input?.subtitle) ||
      (next?.createdAt ? formatTimeAgo(new Date(next.createdAt)).replace(" ago", "") : "");

    (canvasNode.data.composite as CompositeProps).nextInQueue = {
      title: inferredTitle,
      subtitle: inferredSubtitle,
      receivedAt: next?.createdAt ? new Date(next.createdAt) : new Date(),
    };
  }

  return canvasNode;
}

function getRunItemState(execution: WorkflowsWorkflowNodeExecution): LastRunState {
  if (execution.state == "STATE_PENDING" || execution.state == "STATE_STARTED") {
    return "running";
  }

  if (execution.state == "STATE_FINISHED" && execution.result == "RESULT_PASSED") {
    return "success";
  }

  return "failed";
}

function prepareNode(
  nodes: ComponentsNode[],
  node: ComponentsNode,
  triggers: TriggersTrigger[],
  blueprints: BlueprintsBlueprint[],
  components: ComponentsComponent[],
  nodeEventsMap: Record<string, WorkflowsWorkflowEvent[]>,
  nodeExecutionsMap: Record<string, WorkflowsWorkflowNodeExecution[]>,
  nodeQueueItemsMap: Record<string, WorkflowsWorkflowNodeQueueItem[]>,
  workflowId: string,
  queryClient: any,
  organizationId: string,
): CanvasNode {
  switch (node.type) {
    case "TYPE_TRIGGER":
      return prepareTriggerNode(node, triggers, nodeEventsMap);
    case "TYPE_BLUEPRINT":
      const componentMetadata = components.find((c) => c.name === node.component?.name);
      const compositeNode = prepareCompositeNode(nodes, node, blueprints, nodeExecutionsMap, nodeQueueItemsMap);

      // Override outputChannels with component metadata if available
      if (componentMetadata?.outputChannels) {
        return {
          ...compositeNode,
          data: {
            ...compositeNode.data,
            outputChannels: componentMetadata.outputChannels.map((c) => c.name!),
          },
        };
      }

      return compositeNode;
    default:
      return prepareComponentNode(
        nodes,
        node,
        blueprints,
        components,
        nodeExecutionsMap,
        nodeQueueItemsMap,
        workflowId,
        queryClient,
        organizationId,
      );
  }
}

function prepareComponentNode(
  nodes: ComponentsNode[],
  node: ComponentsNode,
  blueprints: BlueprintsBlueprint[],
  components: ComponentsComponent[],
  nodeExecutionsMap: Record<string, WorkflowsWorkflowNodeExecution[]>,
  nodeQueueItemsMap: Record<string, WorkflowsWorkflowNodeQueueItem[]>,
  workflowId: string,
  queryClient: any,
  organizationId?: string,
): CanvasNode {
  switch (node.component?.name) {
    case "approval":
      return prepareApprovalNode(nodes, node, components, nodeExecutionsMap, workflowId, queryClient, organizationId);
    case "if":
      return prepareIfNode(nodes, node, nodeExecutionsMap);
    case "noop":
      return prepareNoopNode(nodes, node, components, nodeExecutionsMap);
    case "filter":
      return prepareFilterNode(nodes, node, components, nodeExecutionsMap);
    case "http":
      return prepareHttpNode(node, components, nodeExecutionsMap);
    case "semaphore":
      return prepareSemaphoreNode(nodes, node, components, nodeExecutionsMap, nodeQueueItemsMap);
    case "wait":
      return prepareWaitNode(nodes, node, components, nodeExecutionsMap, nodeQueueItemsMap);
    case "time_gate":
      return prepareTimeGateNode(nodes, node, components, nodeExecutionsMap, nodeQueueItemsMap);
    case "merge":
      return prepareMergeNode(nodes, node, components, nodeExecutionsMap, nodeQueueItemsMap);
  }

  //
  // TODO: render other component-type nodes as composites for now
  // For generic components, we need to get outputChannels from component metadata
  //
  const componentMetadata = components.find((c) => c.name === node.component?.name);
  const compositeNode = prepareCompositeNode(nodes, node, blueprints, nodeExecutionsMap, nodeQueueItemsMap);

  // Override outputChannels with component metadata if available
  if (componentMetadata?.outputChannels) {
    return {
      ...compositeNode,
      data: {
        ...compositeNode.data,
        outputChannels: componentMetadata.outputChannels.map((c) => c.name!),
      },
    };
  }

  return compositeNode;
}

function prepareApprovalNode(
  nodes: ComponentsNode[],
  node: ComponentsNode,
  components: ComponentsComponent[],
  nodeExecutionsMap: Record<string, WorkflowsWorkflowNodeExecution[]>,
  workflowId: string,
  queryClient: any,
  organizationId?: string,
): CanvasNode {
  const metadata = components.find((c) => c.name === "approval");
  const executions = nodeExecutionsMap[node.id!] || [];
  const execution = executions.length > 0 ? executions[0] : null;
  const executionMetadata = execution?.metadata as any;
  const configuration = (node.configuration || {}) as any;
  const items: any[] = Array.isArray(configuration.items) ? configuration.items : [];

  // Try to enrich display values from cached org users/roles
  let usersById: Record<string, { email?: string; name?: string }> = {};
  let rolesByName: Record<string, string> = {};
  if (organizationId) {
    const usersResp: any = queryClient.getQueryData(organizationKeys.users(organizationId));
    if (Array.isArray(usersResp)) {
      usersResp.forEach((u: any) => {
        const id = u.metadata?.id;
        const email = u.metadata?.email;
        const name = u.spec?.displayName;
        if (id) usersById[id] = { email, name };
      });
    }

    const rolesResp: any = queryClient.getQueryData(organizationKeys.roles(organizationId));
    if (Array.isArray(rolesResp)) {
      rolesResp.forEach((r: any) => {
        const name = r.metadata?.name;
        const display = r.spec?.displayName;
        if (name) rolesByName[name] = display || name;
      });
    }
  }

  let rootTriggerRenderer: TriggerRenderer | null = null;
  if (execution) {
    const rootTriggerNode = nodes.find((n) => n.id === execution!.rootEvent?.nodeId);
    rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.trigger?.name || "");
  }

  // Map backend records to approval items
  const approvals = (executionMetadata?.records || []).map((record: any) => {
    const isPending = record.state === "pending";
    const isExecutionActive = execution?.state === "STATE_STARTED";

    const approvalComment = record.approval?.comment;
    const hasApprovalArtifacts = record.state === "approved" && approvalComment;

    return {
      id: `${record.index}`,
      title:
        record.type === "user" && record.user
          ? record.user.name || record.user.email
          : record.type === "role" && record.role
            ? record.role
            : record.type === "group" && record.group
              ? record.group
              : "Unknown",
      approved: record.state === "approved",
      rejected: record.state === "rejected",
      approverName: record.user?.name,
      approverAvatar: record.user?.avatarUrl,
      rejectionComment: record.rejection?.reason,
      interactive: isPending && isExecutionActive,
      requireArtifacts:
        isPending && isExecutionActive
          ? [
              {
                label: "comment",
                optional: true,
              },
            ]
          : undefined,
      artifacts: hasApprovalArtifacts
        ? {
            Comment: approvalComment,
          }
        : undefined,
      artifactCount: hasApprovalArtifacts ? 1 : undefined,
      onApprove: async (artifacts?: Record<string, string>) => {
        if (!execution?.id) return;

        try {
          await workflowsInvokeNodeExecutionAction(
            withOrganizationHeader({
              path: {
                workflowId: workflowId,
                executionId: execution.id,
                actionName: "approve",
              },
              body: {
                parameters: {
                  index: record.index,
                  comment: artifacts?.comment,
                },
              },
            }),
          );

          queryClient.invalidateQueries({
            queryKey: workflowKeys.nodeExecution(workflowId, node.id!),
          });
        } catch (error: any) {
          console.error("Failed to approve:", error);
        }
      },
      onReject: async (comment?: string) => {
        if (!execution?.id) return;

        try {
          await workflowsInvokeNodeExecutionAction(
            withOrganizationHeader({
              path: {
                workflowId: workflowId,
                executionId: execution.id,
                actionName: "reject",
              },
              body: {
                parameters: {
                  index: record.index,
                  reason: comment,
                },
              },
            }),
          );

          queryClient.invalidateQueries({
            queryKey: workflowKeys.nodeExecution(workflowId, node.id!),
          });
        } catch (error: any) {
          console.error("Failed to reject:", error);
        }
      },
    };
  });

  // Use node name if available, otherwise fall back to component label (from metadata)
  const displayLabel = node.name || metadata?.label!;

  return {
    id: node.id!,
    position: { x: node.position?.x!, y: node.position?.y! },
    data: {
      type: "approval",
      label: displayLabel,
      state: "pending" as const,
      outputChannels: metadata?.outputChannels?.map((c) => c.name!) || ["default"],
      approval: {
        iconSlug: metadata?.icon || "hand",
        iconColor: getColorClass(metadata?.color || "orange"),
        iconBackground: getBackgroundColorClass(metadata?.color || "orange"),
        headerColor: getBackgroundColorClass(metadata?.color || "orange"),
        collapsedBackground: getBackgroundColorClass(metadata?.color || "orange"),
        collapsed: node.isCollapsed,
        title: displayLabel,
        description: metadata?.description,
        receivedAt: execution ? new Date(execution.createdAt!) : undefined,
        approvals,
        // Display Approval settings similar to IF component specs
        spec:
          items.length > 0
            ? {
                title: "approvals required",
                tooltipTitle: "approvals required",
                values: items.map((item) => {
                  const type = (item.type || "").toString();
                  let value =
                    type === "user"
                      ? item.user || ""
                      : type === "role"
                        ? item.role || ""
                        : type === "group"
                          ? item.group || ""
                          : "";
                  const label = type ? `${type[0].toUpperCase()}${type.slice(1)}` : "Item";

                  // Pretty-print values
                  if (type === "user" && value && usersById[value]) {
                    value = usersById[value].email || usersById[value].name || value;
                  }
                  if (type === "role" && value) {
                    value = rolesByName[value] || value.replace(/^(org_|canvas_)/i, "");
                    // Fallback to simple suffix mapping when not found
                    const suffix = (item.role || "").split("_").pop();
                    if (!rolesByName[item.role || ""] && suffix) {
                      const map: any = { viewer: "Viewer", admin: "Admin", owner: "Owner" };
                      value = map[suffix] || value;
                    }
                  }
                  return {
                    badges: [
                      { label: `${label}:`, bgColor: "bg-gray-100", textColor: "text-gray-700" },
                      { label: value || "—", bgColor: "bg-emerald-100", textColor: "text-emerald-800" },
                    ],
                  };
                }),
              }
            : undefined,
        awaitingEvent:
          execution?.state === "STATE_STARTED" && rootTriggerRenderer
            ? rootTriggerRenderer.getTitleAndSubtitle(execution.rootEvent!)
            : undefined,
        lastRunData:
          execution && rootTriggerRenderer
            ? {
                title: rootTriggerRenderer.getTitleAndSubtitle(execution.rootEvent!).title,
                subtitle: rootTriggerRenderer.getTitleAndSubtitle(execution.rootEvent!).subtitle,
                receivedAt: new Date(execution.createdAt!),
                state:
                  getRunItemState(execution) === "success"
                    ? ("processed" as const)
                    : getRunItemState(execution) === "running"
                      ? ("running" as const)
                      : ("discarded" as const),
              }
            : undefined,
      },
    },
  };
}

function prepareIfNode(
  nodes: ComponentsNode[],
  node: ComponentsNode,
  nodeExecutionsMap: Record<string, WorkflowsWorkflowNodeExecution[]>,
): CanvasNode {
  const executions = nodeExecutionsMap[node.id!] || [];
  const execution = executions.length > 0 ? executions[0] : null;

  // Parse conditions from node configuration
  const expression = node.configuration?.expression;

  // Get last execution for event data
  let trueEvent, falseEvent;
  if (execution) {
    const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
    const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.trigger?.name || "");

    const { title } = rootTriggerRenderer.getTitleAndSubtitle(execution.rootEvent!);

    const eventData = {
      receivedAt: new Date(execution.createdAt!),
      eventTitle: title,
      eventState: getRunItemState(execution) === "success" ? ("success" as const) : ("failed" as const),
    };

    if (execution.outputs?.["true"]) {
      trueEvent = eventData;
    } else if (execution.outputs?.["false"]) {
      falseEvent = eventData;
    }
  }

  return {
    id: node.id!,
    position: { x: node.position?.x || 0, y: node.position?.y || 0 },
    data: {
      type: "if",
      label: node.name!,
      state: "pending" as const,
      if: {
        title: node.name!,
        expression,
        trueEvent: trueEvent || {
          eventTitle: "No events received yet",
          eventState: "neutral" as const,
        },
        falseEvent: falseEvent || {
          eventTitle: "No events received yet",
          eventState: "neutral" as const,
        },
        trueSectionLabel: "TRUE",
        falseSectionLabel: "FALSE",
        collapsedBackground: getBackgroundColorClass("white"),
        collapsed: node.isCollapsed,
      },
    },
  };
}

function prepareNoopNode(
  nodes: ComponentsNode[],
  node: ComponentsNode,
  components: ComponentsComponent[],
  nodeExecutionsMap: Record<string, WorkflowsWorkflowNodeExecution[]>,
): CanvasNode {
  const executions = nodeExecutionsMap[node.id!] || [];
  const execution = executions.length > 0 ? executions[0] : null;
  const metadata = components.find((c) => c.name === "noop");

  // Get last event data
  let lastEvent;
  if (execution) {
    const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
    const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.trigger?.name || "");

    const { title } = rootTriggerRenderer.getTitleAndSubtitle(execution.rootEvent!);

    lastEvent = {
      receivedAt: new Date(execution.createdAt!),
      eventTitle: title,
      eventState: getRunItemState(execution) === "success" ? ("success" as const) : ("failed" as const),
    };
  }

  const displayLabel = node.name || metadata?.label!;

  return {
    id: node.id!,
    position: { x: node.position?.x || 0, y: node.position?.y || 0 },
    data: {
      type: "noop",
      label: displayLabel,
      state: "pending" as const,
      noop: {
        title: displayLabel,
        lastEvent: lastEvent || {
          eventTitle: "No events received yet",
          eventState: "neutral" as const,
        },
        collapsedBackground: getBackgroundColorClass("white"),
        collapsed: node.isCollapsed,
      },
    },
  };
}

function prepareMergeNode(
  nodes: ComponentsNode[],
  node: ComponentsNode,
  components: ComponentsComponent[],
  nodeExecutionsMap: Record<string, WorkflowsWorkflowNodeExecution[]>,
  nodeQueueItemsMap?: Record<string, WorkflowsWorkflowNodeQueueItem[]>,
): CanvasNode {
  const executions = nodeExecutionsMap[node.id!] || [];
  const execution = executions.length > 0 ? executions[0] : null;
  const metadata = components.find((c) => c.name === "noop");

  let lastEvent;
  if (execution) {
    const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
    const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.trigger?.name || "");

    const { title } = rootTriggerRenderer.getTitleAndSubtitle(execution.rootEvent!);

    lastEvent = {
      receivedAt: new Date(execution.createdAt!),
      eventTitle: title,
      eventState: getRunItemState(execution),
    };
  }

  const displayLabel = node.name || metadata?.label!;

  return {
    id: node.id!,
    position: { x: node.position?.x || 0, y: node.position?.y || 0 },
    data: {
      type: "merge",
      label: displayLabel,
      state: "pending" as const,
      merge: {
        title: displayLabel,
        lastEvent: lastEvent || {
          eventTitle: "No events received yet",
          eventState: "neutral" as const,
        },
        nextInQueue:
          nodeQueueItemsMap && (nodeQueueItemsMap[node.id!] || []).length > 0
            ? (() => {
                const item: any = (nodeQueueItemsMap[node.id!] || [])[0] as any;
                const title =
                  item?.name ||
                  item?.input?.title ||
                  item?.input?.name ||
                  item?.input?.eventTitle ||
                  item?.id ||
                  "Queued";
                const subtitle = typeof item?.input?.subtitle === "string" ? item.input.subtitle : undefined;
                return { title, subtitle };
              })()
            : undefined,
        collapsedBackground: getBackgroundColorClass("white"),
        collapsed: node.isCollapsed,
      },
    },
  };
}

function prepareFilterNode(
  nodes: ComponentsNode[],
  node: ComponentsNode,
  components: ComponentsComponent[],
  nodeExecutionsMap: Record<string, WorkflowsWorkflowNodeExecution[]>,
): CanvasNode {
  const executions = nodeExecutionsMap[node.id!] || [];
  const execution = executions.length > 0 ? executions[0] : null;
  const metadata = components.find((c) => c.name === "filter");

  // Parse filters from node configuration
  const expression = node.configuration?.expression as string;

  let lastEvent;
  if (execution) {
    const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
    const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.trigger?.name || "");

    const { title } = rootTriggerRenderer.getTitleAndSubtitle(execution.rootEvent!);

    lastEvent = {
      receivedAt: new Date(execution.createdAt!),
      eventTitle: title,
      eventState: getRunItemState(execution) === "success" ? ("success" as const) : ("failed" as const),
    };
  }

  // Use node name if available, otherwise fall back to component label (from metadata)
  const displayLabel = node.name || metadata?.label!;

  return {
    id: node.id!,
    position: { x: node.position?.x || 0, y: node.position?.y || 0 },
    data: {
      type: "filter",
      label: displayLabel,
      state: "pending" as const,
      filter: {
        title: displayLabel,
        expression,
        lastEvent: lastEvent || {
          eventTitle: "No events received yet",
          eventState: "neutral" as const,
        },
        collapsedBackground: getBackgroundColorClass("white"),
        collapsed: node.isCollapsed,
      },
    },
  };
}

function prepareHttpNode(
  node: ComponentsNode,
  components: ComponentsComponent[],
  nodeExecutionsMap: Record<string, WorkflowsWorkflowNodeExecution[]>,
): CanvasNode {
  const metadata = components.find((c) => c.name === "http");
  const executions = nodeExecutionsMap[node.id!] || [];
  const execution = executions.length > 0 ? executions[0] : null;

  // Configuration always comes from the node, not the execution
  const configuration = node.configuration as any;

  let lastExecution;
  if (execution) {
    const outputs = execution.outputs as any;
    const response = outputs?.default?.[0];

    lastExecution = {
      statusCode: response?.status,
      receivedAt: new Date(execution.createdAt!),
      state:
        getRunItemState(execution) === "success"
          ? ("success" as const)
          : getRunItemState(execution) === "running"
            ? ("running" as const)
            : ("failed" as const),
    };
  }

  // Use node name if available, otherwise fall back to component label (from metadata)
  const displayLabel = node.name || metadata?.label!;

  return {
    id: node.id!,
    position: { x: node.position?.x || 0, y: node.position?.y || 0 },
    data: {
      type: "http",
      label: displayLabel,
      state: "pending" as const,
      outputChannels: metadata?.outputChannels?.map((c) => c.name!) || ["default"],
      http: {
        iconSlug: metadata?.icon || "globe",
        iconColor: getColorClass(metadata?.color || "gray"),
        iconBackground: getBackgroundColorClass(metadata?.color || "gray"),
        headerColor: getBackgroundColorClass(metadata?.color || "gray"),
        title: displayLabel,
        method: configuration?.method,
        url: configuration?.url,
        payload: configuration?.payload,
        headers: configuration?.headers,
        lastExecution,
        collapsedBackground: getBackgroundColorClass("white"),
        collapsed: node.isCollapsed,
      },
    },
  };
}

interface ExecutionMetadata {
  workflow?: {
    id: string;
    url: string;
    state: string;
    result: string;
  };
}

function prepareSemaphoreNode(
  nodes: ComponentsNode[],
  node: ComponentsNode,
  components: ComponentsComponent[],
  nodeExecutionsMap: Record<string, WorkflowsWorkflowNodeExecution[]>,
  nodeQueueItemsMap?: Record<string, WorkflowsWorkflowNodeQueueItem[]>,
): CanvasNode {
  const metadata = components.find((c) => c.name === "semaphore");
  const executions = nodeExecutionsMap[node.id!] || [];
  const execution = executions.length > 0 ? executions[0] : null;

  // Configuration always comes from the node, not the execution
  const configuration = node.configuration as any;
  const nodeMetadata = node.metadata as any;

  let lastExecution;
  if (execution) {
    const metadata = execution.metadata as ExecutionMetadata;
    const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
    const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.trigger?.name || "");

    const { title } = rootTriggerRenderer.getTitleAndSubtitle(execution.rootEvent!);

    // Determine state based on workflow result for finished executions
    let state: "success" | "failed" | "running";
    if (metadata.workflow?.state === "finished") {
      // Use workflow result to determine color/icon when finished
      state = metadata.workflow?.result === "passed" ? "success" : "failed";
    } else {
      // Use execution state for running/pending states
      state = getRunItemState(execution) === "running" ? "running" : "failed";
    }

    // Calculate duration for finished executions
    let duration: number | undefined;
    if (state !== "running" && execution.updatedAt && execution.createdAt) {
      duration = new Date(execution.updatedAt).getTime() - new Date(execution.createdAt).getTime();
    }

    lastExecution = {
      title: title,
      receivedAt: new Date(execution.createdAt!),
      completedAt: execution.updatedAt ? new Date(execution.updatedAt) : undefined,
      state: state,
      values: rootTriggerRenderer.getRootEventValues(execution.rootEvent!),
      duration: duration,
    };
  }

  // Use node name if available, otherwise fall back to component label (from metadata)
  const displayLabel = node.name || metadata?.label!;

  // Build metadata array
  const metadataItems = [];
  if (nodeMetadata?.project?.name) {
    metadataItems.push({ icon: "folder", label: nodeMetadata.project.name });
  } else if (configuration.project) {
    metadataItems.push({ icon: "folder", label: configuration.project });
  }

  if (configuration?.ref) {
    metadataItems.push({ icon: "git-branch", label: configuration.ref });
  }
  if (configuration?.pipelineFile) {
    metadataItems.push({ icon: "file-code", label: configuration.pipelineFile });
  }

  return {
    id: node.id!,
    position: { x: node.position?.x || 0, y: node.position?.y || 0 },
    data: {
      type: "semaphore",
      label: displayLabel,
      state: "pending" as const,
      outputChannels: metadata?.outputChannels?.map((c) => c.name!) || ["default"],
      semaphore: {
        iconSrc: SemaphoreLogo,
        iconSlug: metadata?.icon || "workflow",
        iconColor: getColorClass(metadata?.color || "gray"),
        iconBackground: getBackgroundColorClass(metadata?.color || "gray"),
        headerColor: getBackgroundColorClass(metadata?.color || "gray"),
        title: displayLabel,
        metadata: metadataItems,
        parameters: configuration?.parameters,
        lastExecution,
        nextInQueue:
          nodeQueueItemsMap && (nodeQueueItemsMap[node.id!] || []).length > 0
            ? (() => {
                const item: any = (nodeQueueItemsMap[node.id!] || [])[0] as any;
                const title =
                  item?.name ||
                  item?.input?.title ||
                  item?.input?.name ||
                  item?.input?.eventTitle ||
                  item?.id ||
                  "Queued";
                const subtitle = typeof item?.input?.subtitle === "string" ? item.input.subtitle : undefined;
                return { title, subtitle };
              })()
            : undefined,
        collapsedBackground: getBackgroundColorClass("white"),
        collapsed: node.isCollapsed,
      },
    },
  };
}

function prepareWaitNode(
  nodes: ComponentsNode[],
  node: ComponentsNode,
  components: ComponentsComponent[],
  nodeExecutionsMap: Record<string, WorkflowsWorkflowNodeExecution[]>,
  nodeQueueItemsMap?: Record<string, WorkflowsWorkflowNodeQueueItem[]>,
): CanvasNode {
  const metadata = components.find((c) => c.name === "wait");
  const configuration = node.configuration as any;
  const executions = nodeExecutionsMap[node.id!] || [];
  const execution = executions.length > 0 ? executions[0] : null;

  let lastExecution;
  if (execution) {
    const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
    const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.trigger?.name || "");

    const { title } = rootTriggerRenderer.getTitleAndSubtitle(execution.rootEvent!);

    // Calculate expected duration from configuration
    let expectedDuration: number | undefined;
    if (configuration?.duration) {
      const { value, unit } = configuration.duration;
      const multipliers = { seconds: 1000, minutes: 60000, hours: 3600000 };
      expectedDuration = value * (multipliers[unit as keyof typeof multipliers] || 1000);
    }

    lastExecution = {
      title: title,
      receivedAt: new Date(execution.createdAt!),
      completedAt: execution.updatedAt ? new Date(execution.updatedAt) : undefined,
      state:
        getRunItemState(execution) === "success"
          ? ("success" as const)
          : getRunItemState(execution) === "running"
            ? ("running" as const)
            : ("failed" as const),
      values: rootTriggerRenderer.getRootEventValues(execution.rootEvent!),
      expectedDuration: expectedDuration,
    };
  }

  // Use node name if available, otherwise fall back to component label (from metadata)
  const displayLabel = node.name || metadata?.label!;

  return {
    id: node.id!,
    position: { x: node.position?.x || 0, y: node.position?.y || 0 },
    data: {
      type: "wait",
      label: displayLabel,
      state: "pending" as const,
      outputChannels: metadata?.outputChannels?.map((c) => c.name!) || ["default"],
      wait: {
        title: displayLabel,
        duration: configuration?.duration,
        lastExecution,
        nextInQueue:
          nodeQueueItemsMap && (nodeQueueItemsMap[node.id!] || []).length > 0
            ? (() => {
                const item: any = (nodeQueueItemsMap[node.id!] || [])[0] as any;
                const title =
                  item?.name ||
                  item?.input?.title ||
                  item?.input?.name ||
                  item?.input?.eventTitle ||
                  item?.id ||
                  "Queued";
                const subtitle = typeof item?.input?.subtitle === "string" ? item.input.subtitle : undefined;
                return { title, subtitle };
              })()
            : undefined,
        iconColor: getColorClass(metadata?.color || "yellow"),
        iconBackground: getBackgroundColorClass(metadata?.color || "yellow"),
        headerColor: getBackgroundColorClass(metadata?.color || "yellow"),
        collapsedBackground: getBackgroundColorClass("white"),
        collapsed: node.isCollapsed,
      },
    },
  };
}

function prepareTimeGateNode(
  nodes: ComponentsNode[],
  node: ComponentsNode,
  components: ComponentsComponent[],
  nodeExecutionsMap: Record<string, WorkflowsWorkflowNodeExecution[]>,
  nodeQueueItemsMap: Record<string, WorkflowsWorkflowNodeQueueItem[]>,
): CanvasNode {
  const metadata = components.find((c) => c.name === "time_gate");
  const configuration = node.configuration as any;

  // Format time gate configuration for display
  const mode = configuration?.mode || "include_range";
  const days = configuration?.days || [];
  const daysDisplay = days.length > 0 ? days.join(", ") : "";

  // Get timezone information
  const timezone = configuration?.timezone || "0";
  const getTimezoneDisplay = (timezoneOffset: string) => {
    const offset = parseFloat(timezoneOffset);
    if (offset === 0) return "GMT+0 (UTC)";
    if (offset > 0) return `GMT+${offset}`;
    return `GMT${offset}`; // Already has the minus sign
  };
  const timezoneDisplay = getTimezoneDisplay(timezone);

  // Handle different time window formats based on mode
  let startTime = "00:00";
  let endTime = "23:59";

  if (mode === "include_specific" || mode === "exclude_specific") {
    startTime = `${configuration.startDayInYear} ${configuration.startTime}`;
    endTime = `${configuration.endDayInYear} ${configuration.endTime}`;
  } else {
    startTime = `${configuration.startTime}`;
    endTime = `${configuration.endTime}`;
  }

  const timeWindow = `${startTime} - ${endTime}`;

  const executions = nodeExecutionsMap[node.id!] || [];
  const execution = executions.length > 0 ? executions[0] : null;

  let lastExecution:
    | {
        title: string;
        receivedAt: Date;
        state: "success" | "failed" | "running";
        values?: Record<string, string>;
        nextRunTime?: Date;
      }
    | undefined;

  if (execution) {
    const executionState = getRunItemState(execution);
    const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
    const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.trigger?.name || "");

    const { title } = rootTriggerRenderer.getTitleAndSubtitle(execution.rootEvent!);

    lastExecution = {
      title: title,
      receivedAt: new Date(execution.createdAt!),
      state:
        executionState === "success"
          ? ("success" as const)
          : executionState === "failed"
            ? ("failed" as const)
            : ("running" as const),
      values: rootTriggerRenderer.getRootEventValues(execution.rootEvent!),
    };

    if (executionState === "running") {
      // Get next run time from execution metadata
      const executionMetadata = execution.metadata as { nextValidTime?: string };
      if (executionMetadata?.nextValidTime) {
        lastExecution.nextRunTime = new Date(executionMetadata.nextValidTime);
      }
    }
  }

  // Use node name if available, otherwise fall back to component label (from metadata)
  const displayLabel = node.name || metadata?.label || "Time Gate";

  return {
    id: node.id!,
    position: { x: node.position?.x || 0, y: node.position?.y || 0 },
    data: {
      type: "time_gate",
      label: displayLabel,
      state: "pending" as const,
      outputChannels: metadata?.outputChannels?.map((c) => c.name!) || ["default"],
      time_gate: {
        title: displayLabel,
        mode,
        timeWindow,
        days: daysDisplay,
        timezone: timezoneDisplay,
        lastExecution,
        nextInQueue: nodeQueueItemsMap[node.id!]?.[0] ? { title: nodeQueueItemsMap[node.id!]?.[0].id } : undefined,
        iconColor: getColorClass(metadata?.color || "blue"),
        iconBackground: getBackgroundColorClass(metadata?.color || "blue"),
        headerColor: getBackgroundColorClass(metadata?.color || "blue"),
        collapsedBackground: getBackgroundColorClass("white"),
        collapsed: node.isCollapsed,
      },
    },
  };
}

function prepareEdge(edge: ComponentsEdge): CanvasEdge {
  const id = `${edge.sourceId!}--${edge.targetId!}--${edge.channel!}`;

  return {
    id: id,
    source: edge.sourceId!,
    target: edge.targetId!,
    sourceHandle: edge.channel!,
  };
}

function prepareSidebarData(
  node: ComponentsNode,
  nodes: ComponentsNode[],
  blueprints: BlueprintsBlueprint[],
  components: ComponentsComponent[],
  triggers: TriggersTrigger[],
  nodeExecutionsMap: Record<string, WorkflowsWorkflowNodeExecution[]>,
  nodeQueueItemsMap: Record<string, WorkflowsWorkflowNodeQueueItem[]>,
  nodeEventsMap: Record<string, WorkflowsWorkflowEvent[]>,
  totalHistoryCount?: number,
  totalQueueCount?: number,
): SidebarData {
  const executions = nodeExecutionsMap[node.id!] || [];
  const queueItems = nodeQueueItemsMap[node.id!] || [];
  const events = nodeEventsMap[node.id!] || [];

  // Get metadata based on node type
  const blueprintMetadata =
    node.type === "TYPE_BLUEPRINT" ? blueprints.find((b) => b.id === node.blueprint?.id) : undefined;
  const componentMetadata =
    node.type === "TYPE_COMPONENT" ? components.find((c) => c.name === node.component?.name) : undefined;
  const triggerMetadata =
    node.type === "TYPE_TRIGGER" ? triggers.find((t) => t.name === node.trigger?.name) : undefined;

  const configurationFields =
    blueprintMetadata?.configuration || componentMetadata?.configuration || triggerMetadata?.configuration || [];

  const fieldLabelMap = configurationFields.reduce<Record<string, string>>((acc, field) => {
    if (field.name) {
      acc[field.name] = field.label || field.name;
    }
    return acc;
  }, {});

  const nodeTitle =
    componentMetadata?.label || blueprintMetadata?.name || triggerMetadata?.label || node.name || "Unknown";
  let iconSlug = "boxes";
  let color = "indigo";

  if (blueprintMetadata) {
    iconSlug = blueprintMetadata.icon || iconSlug;
    color = blueprintMetadata.color || color;
  } else if (componentMetadata) {
    iconSlug = componentMetadata.icon || iconSlug;
    color = componentMetadata.color || color;
  } else if (triggerMetadata) {
    iconSlug = triggerMetadata.icon || iconSlug;
    color = triggerMetadata.color || color;
  }

  const latestEvents =
    node.type === "TYPE_TRIGGER"
      ? mapTriggerEventsToSidebarEvents(events, node, 5)
      : mapExecutionsToSidebarEvents(executions, nodes, 5);

  // Convert queue items to sidebar events (next in queue)
  const nextInQueueEvents = mapQueueItemsToSidebarEvents(queueItems, nodes, 5);

  // Build metadata from node configuration
  const metadataItems = [
    {
      icon: "cog",
      label: `Node ID: ${node.id}`,
    },
  ];

  const hideQueueEvents = node.type === "TYPE_TRIGGER";

  // Add configuration fields to metadata (only simple types)
  if (node.configuration) {
    Object.entries(node.configuration).forEach(([key, value]) => {
      // Only include simple types (string, number, boolean)
      // Exclude objects, arrays, null, undefined
      const valueType = typeof value;
      const isSimpleType = valueType === "string" || valueType === "number" || valueType === "boolean";

      if (isSimpleType) {
        const displayKey = fieldLabelMap[key] || key;
        metadataItems.push({
          icon: "settings",
          label: `${displayKey}: ${value}`,
        });
      }
    });
  }

  return {
    latestEvents,
    nextInQueueEvents,
    metadata: metadataItems,
    title: nodeTitle,
    iconSlug,
    iconColor: getColorClass(color),
    iconBackground: getBackgroundColorClass(color),
    totalInHistoryCount: totalHistoryCount ? totalHistoryCount : 0,
    totalInQueueCount: totalQueueCount ? totalQueueCount : 0,
    hideQueueEvents,
  };
}
