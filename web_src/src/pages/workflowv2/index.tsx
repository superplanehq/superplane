import { useNodeExecutionStore } from "@/stores/nodeExecutionStore";
import { showErrorToast, showSuccessToast } from "@/utils/toast";
import { QueryClient, useQueryClient } from "@tanstack/react-query";
import { Loader2 } from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useNavigate, useParams, useSearchParams } from "react-router-dom";

import {
  BlueprintsBlueprint,
  ComponentsComponent,
  ComponentsEdge,
  ComponentsNode,
  TriggersTrigger,
  WorkflowsListEventExecutionsResponse,
  WorkflowsWorkflow,
  WorkflowsWorkflowEvent,
  WorkflowsWorkflowNodeExecution,
  WorkflowsWorkflowNodeQueueItem,
  workflowsEmitNodeEvent,
  workflowsInvokeNodeExecutionAction,
} from "@/api-client";
import { organizationKeys, useOrganizationRoles, useOrganizationUsers } from "@/hooks/useOrganizationData";

import { useBlueprints, useComponents } from "@/hooks/useBlueprintData";
import { useNodeHistory } from "@/hooks/useNodeHistory";
import { usePageTitle } from "@/hooks/usePageTitle";
import { useQueueHistory } from "@/hooks/useQueueHistory";
import {
  eventExecutionsQueryOptions,
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
import { EventState } from "@/ui/componentBase";
import { TabData } from "@/ui/componentSidebar/SidebarEventItem/SidebarEventItem";
import { CompositeProps, LastRunState } from "@/ui/composite";
import { getBackgroundColorClass, getColorClass } from "@/utils/colors";
import { filterVisibleConfiguration } from "@/utils/components";
import { withOrganizationHeader } from "@/utils/withOrganizationHeader";
import { getComponentBaseMapper, getTriggerRenderer } from "./mappers";
import { TriggerRenderer } from "./mappers/types";
import { useOnCancelQueueItemHandler } from "./useOnCancelQueueItemHandler";
import { usePushThroughHandler } from "./usePushThroughHandler";
import { useCancelExecutionHandler } from "./useCancelExecutionHandler";
import {
  getNextInQueueInfo,
  mapExecutionsToSidebarEvents,
  mapQueueItemsToSidebarEvents,
  mapTriggerEventsToSidebarEvents,
} from "./utils";

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

  // Execution chain data utilities for lazy loading
  const { loadExecutionChain } = useExecutionChainData(workflowId!, queryClient, workflow);

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

      if (event.startsWith("queue_item")) {
        queryClient.invalidateQueries({
          queryKey: workflowKeys.nodeQueueItemHistory(workflowId!, nodeId),
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
      const queueItemsMap = nodeData.queueItems.length > 0 ? { [nodeId]: nodeData.queueItems.reverse() } : {};
      const eventsMapForSidebar = nodeData.events.length > 0 ? { [nodeId]: nodeData.events } : nodeEventsMap; // Fall back to existing events map for trigger nodes
      const totalHistoryCount = nodeData.totalInHistoryCount;
      const totalQueueCount = nodeData.totalInQueueCount;

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

      loadNodeDataMethod(workflowId!, nodeId, node.type!, queryClient);
    },
    [workflow, workflowId, queryClient, loadNodeDataMethod],
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

  /**
   * Builds a topological path to find all nodes that should execute before the given target node.
   * This follows the directed graph structure of the workflow to determine execution order.
   */

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
        let payload: Record<string, unknown> = {};

        if (triggerEvent.data) {
          payload = triggerEvent.data;
        }

        tabData.payload = payload;

        return Object.keys(tabData).length > 0 ? tabData : undefined;
      }

      if (event.kind === "queue") {
        // Handle queue items - get the queue item data
        const queueItems = nodeQueueItemsMap[nodeId] || [];
        const queueItem = queueItems.find((item: WorkflowsWorkflowNodeQueueItem) => item.id === event.id);

        if (!queueItem) return undefined;

        const tabData: TabData = {};

        if (queueItem.rootEvent) {
          const rootTriggerNode = workflow?.spec?.nodes?.find((n) => n.id === queueItem.rootEvent?.nodeId);
          const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.trigger?.name || "");
          const rootEventValues = rootTriggerRenderer.getRootEventValues(queueItem.rootEvent);

          tabData.root = {
            ...rootEventValues,
            "Event ID": queueItem.rootEvent.id,
            "Node ID": queueItem.rootEvent.nodeId,
            "Created At": queueItem.rootEvent.createdAt
              ? new Date(queueItem.rootEvent.createdAt).toLocaleString()
              : undefined,
          };
        }

        tabData.current = {
          "Queue Item ID": queueItem.id,
          "Node ID": queueItem.nodeId,
          "Created At": queueItem.createdAt ? new Date(queueItem.createdAt).toLocaleString() : undefined,
        };

        tabData.payload = queueItem.input || {};

        return Object.keys(tabData).length > 0 ? tabData : undefined;
      }

      // Handle other components (non-triggers) - get execution for this event
      const executions = nodeExecutionsMap[nodeId] || [];
      const execution = executions.find((exec: WorkflowsWorkflowNodeExecution) => exec.id === event.id);

      if (!execution) return undefined;

      // Extract tab data from execution
      const tabData: TabData = {};

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

      // Execution Chain will be loaded lazily when requested

      return Object.keys(tabData).length > 0 ? tabData : undefined;
    },
    [workflow, nodeExecutionsMap, nodeEventsMap, nodeQueueItemsMap],
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

  const handleReEmit = useCallback(
    async (nodeId: string, eventOrExecutionId: string) => {
      const nodeEvents = nodeEventsMap[nodeId];
      if (!nodeEvents) return;
      const eventToReemit = nodeEvents.find((event) => event.id === eventOrExecutionId);
      if (!eventToReemit) return;
      handleRun(nodeId, eventToReemit.channel || "", eventToReemit.data);
    },
    [handleRun, nodeEventsMap],
  );

  const handleNodeDuplicate = useCallback(
    (nodeId: string) => {
      if (!workflow || !organizationId || !workflowId) return;

      const nodeToDuplicate = workflow.spec?.nodes?.find((node) => node.id === nodeId);
      if (!nodeToDuplicate) return;

      saveWorkflowSnapshot(workflow);

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

      const newNodeId = generateNodeId(blockName, nodeToDuplicate.name || "node");

      const offsetX = 50;
      const offsetY = 50;

      const duplicateNode: ComponentsNode = {
        ...nodeToDuplicate,
        id: newNodeId,
        name: nodeToDuplicate.name || "node",
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

  const handleSidebarChange = useCallback(
    (open: boolean, nodeId: string | null) => {
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
    },
    [searchParams, setSearchParams],
  );

  // Provide pass-through handlers regardless of workflow being loaded to keep hook order stable
  const { onPushThrough, supportsPushThrough } = usePushThroughHandler({
    workflowId: workflowId!,
    organizationId,
    workflow,
  });

  const { onCancelExecution } = useCancelExecutionHandler({
    workflowId: workflowId!,
    workflow,
  });

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
    return (
      <div className="flex items-center justify-center h-screen">
        <div className="flex flex-col items-center gap-4">
          <h1 className="text-4xl font-bold text-gray-700">404</h1>
          <p className="text-lg text-gray-500">Canvas not found</p>
          <p className="text-sm text-gray-400">
            This canvas may have been deleted or you may not have permission to view it.
          </p>
        </div>
      </div>
    );
  }

  const hasRunBlockingChanges = hasUnsavedChanges && hasNonPositionalUnsavedChanges;

  return (
    <CanvasPage
      // Persist right sidebar in query params
      initialSidebar={{
        isOpen: searchParams.get("sidebar") === "1",
        nodeId: searchParams.get("node") || null,
      }}
      onSidebarChange={handleSidebarChange}
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
      onPushThrough={onPushThrough}
      supportsPushThrough={supportsPushThrough}
      onCancelExecution={onCancelExecution}
      getAllHistoryEvents={getAllHistoryEvents}
      onLoadMoreHistory={handleLoadMoreHistory}
      getHasMoreHistory={getHasMoreHistory}
      getLoadingMoreHistory={getLoadingMoreHistory}
      onLoadMoreQueue={onLoadMoreQueue}
      getAllQueueEvents={getAllQueueEvents}
      getHasMoreQueue={getHasMoreQueue}
      getLoadingMoreQueue={getLoadingMoreQueue}
      onReEmit={handleReEmit}
      loadExecutionChain={loadExecutionChain}
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

function useExecutionChainData(workflowId: string, queryClient: QueryClient, workflow?: WorkflowsWorkflow) {
  const loadExecutionChain = useCallback(
    async (
      eventId: string,
      nodeId?: string,
      currentExecution?: Record<string, unknown>,
      forceReload = false,
    ): Promise<WorkflowsWorkflowNodeExecution[]> => {
      const queryOptions = eventExecutionsQueryOptions(workflowId, eventId);

      let allExecutions: WorkflowsWorkflowNodeExecution[] = [];

      if (!forceReload) {
        const cachedData = queryClient.getQueryData(queryOptions.queryKey);
        if (cachedData) {
          allExecutions = (cachedData as WorkflowsListEventExecutionsResponse)?.executions || [];
        }
      }

      if (allExecutions.length === 0) {
        if (forceReload) {
          await queryClient.invalidateQueries({ queryKey: queryOptions.queryKey });
        }
        const data = await queryClient.fetchQuery(queryOptions);
        allExecutions = (data as WorkflowsListEventExecutionsResponse)?.executions || [];
      }

      // Apply topological filtering - the logic you wanted back!
      if (!allExecutions.length || !workflow || !nodeId) return allExecutions;

      const currentExecutionTime = currentExecution?.createdAt
        ? new Date(currentExecution.createdAt as string).getTime()
        : Date.now();
      const nodesBefore = getNodesBeforeTarget(nodeId, workflow);
      nodesBefore.add(nodeId); // Include current node

      const executionsUpToCurrent = allExecutions.filter((exec) => {
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

      return executionsUpToCurrent;
    },
    [workflowId, queryClient, workflow],
  );

  return { loadExecutionChain };
}

// Helper function to build topological path to find all nodes that should execute before the given target node
function getNodesBeforeTarget(targetNodeId: string, workflow: WorkflowsWorkflow): Set<string> {
  const nodesBefore = new Set<string>();
  if (!workflow?.spec?.edges) return nodesBefore;

  // Build adjacency list for the workflow graph
  const adjacencyList: Record<string, string[]> = {};
  workflow.spec.edges.forEach((edge) => {
    if (!edge.sourceId || !edge.targetId) return;
    if (!adjacencyList[edge.sourceId]) {
      adjacencyList[edge.sourceId] = [];
    }
    adjacencyList[edge.sourceId].push(edge.targetId);
  });

  // DFS to find all nodes that can reach the target
  const visited = new Set<string>();
  const canReachTarget = (nodeId: string): boolean => {
    if (visited.has(nodeId)) return false; // Avoid cycles
    if (nodeId === targetNodeId) return true;

    visited.add(nodeId);
    const neighbors = adjacencyList[nodeId] || [];
    const canReach = neighbors.some((neighbor) => canReachTarget(neighbor));
    visited.delete(nodeId); // Allow revisiting in different paths

    return canReach;
  };

  // Check all nodes to see which ones can reach the target
  const allNodeIds = new Set<string>();
  workflow.spec.edges?.forEach((edge) => {
    if (edge.sourceId) allNodeIds.add(edge.sourceId);
    if (edge.targetId) allNodeIds.add(edge.targetId);
  });
  workflow.spec.nodes?.forEach((node) => {
    if (node.id) allNodeIds.add(node.id);
  });

  allNodeIds.forEach((nodeId) => {
    if (canReachTarget(nodeId)) {
      nodesBefore.add(nodeId);
    }
  });

  return nodesBefore;
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

  const nextInQueueInfo = getNextInQueueInfo(nodeQueueItemsMap, node.id!, nodes);
  if (nextInQueueInfo) {
    (canvasNode.data.composite as CompositeProps).nextInQueue = nextInQueueInfo;
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

function executionToEventSectionState(execution: WorkflowsWorkflowNodeExecution): EventState {
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
      return prepareComponentBaseNode(nodes, node, components, nodeExecutionsMap);
    case "noop":
    case "http":
    case "semaphore":
    case "time_gate":
      return prepareComponentBaseNode(nodes, node, components, nodeExecutionsMap, nodeQueueItemsMap);
    case "filter":
      return prepareFilterNode(nodes, node, components, nodeExecutionsMap);
    case "wait":
      return prepareWaitNode(nodes, node, components, nodeExecutionsMap, nodeQueueItemsMap);
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

function prepareComponentBaseNode(
  nodes: ComponentsNode[],
  node: ComponentsNode,
  components: ComponentsComponent[],
  nodeExecutionsMap: Record<string, WorkflowsWorkflowNodeExecution[]>,
  nodeQueueItemsMap?: Record<string, WorkflowsWorkflowNodeQueueItem[]>,
): CanvasNode {
  const executions = nodeExecutionsMap[node.id!] || [];
  const metadata = components.find((c) => c.name === node.component?.name);
  const displayLabel = node.name || metadata?.label;
  const componentDef = components.find((c) => c.name === node.component?.name);
  const nodeQueueItems = nodeQueueItemsMap?.[node.id!];

  return {
    id: node.id!,
    position: { x: node.position?.x || 0, y: node.position?.y || 0 },
    data: {
      type: "component",
      label: displayLabel,
      state: "pending" as const,
      outputChannels: metadata?.outputChannels?.map((channel) => channel.name) || ["default"],
      component: getComponentBaseMapper(node.component?.name || "").props(
        nodes,
        node,
        componentDef!,
        executions,
        nodeQueueItems,
      ),
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
      eventState: executionToEventSectionState(execution),
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
        nextInQueue: getNextInQueueInfo(nodeQueueItemsMap, node.id!, nodes),
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
      eventState: executionToEventSectionState(execution),
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
        nextInQueue: getNextInQueueInfo(nodeQueueItemsMap, node.id!, nodes),
        iconColor: getColorClass(metadata?.color || "yellow"),
        iconBackground: getBackgroundColorClass(metadata?.color || "yellow"),
        headerColor: getBackgroundColorClass(metadata?.color || "yellow"),
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
    isComposite: node.type === "TYPE_BLUEPRINT",
  };
}
