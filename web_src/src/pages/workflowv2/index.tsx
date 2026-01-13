import { useNodeExecutionStore } from "@/stores/nodeExecutionStore";
import { showErrorToast, showSuccessToast } from "@/utils/toast";
import { QueryClient, useQueryClient } from "@tanstack/react-query";
import debounce from "lodash.debounce";
import { Loader2, Puzzle } from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useNavigate, useParams, useSearchParams } from "react-router-dom";

import {
  BlueprintsBlueprint,
  ComponentsAppInstallationRef,
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
} from "@/api-client";
import { useOrganizationGroups, useOrganizationRoles, useOrganizationUsers } from "@/hooks/useOrganizationData";

import { useBlueprints, useComponents } from "@/hooks/useBlueprintData";
import { useNodeHistory } from "@/hooks/useNodeHistory";
import { usePageTitle } from "@/hooks/usePageTitle";
import { useQueueHistory } from "@/hooks/useQueueHistory";
import { useAvailableApplications, useInstalledApplications } from "@/hooks/useApplications";
import {
  eventExecutionsQueryOptions,
  useTriggers,
  useUpdateWorkflow,
  useWorkflow,
  useWorkflowEvents,
  useWidgets,
  workflowKeys,
} from "@/hooks/useWorkflowData";
import { useWorkflowWebsocket } from "@/hooks/useWorkflowWebsocket";
import { buildBuildingBlockCategories } from "@/ui/buildingBlocks";
import { getActiveNoteId, restoreActiveNoteFocus } from "@/ui/annotationComponent/noteFocus";
import {
  CANVAS_SIDEBAR_STORAGE_KEY,
  CanvasEdge,
  CanvasNode,
  CanvasPage,
  NewNodeData,
  NodeEditData,
  SidebarData,
} from "@/ui/CanvasPage";
import { EventState, EventStateMap } from "@/ui/componentBase";
import { TabData } from "@/ui/componentSidebar/SidebarEventItem/SidebarEventItem";
import { CompositeProps, LastRunState } from "@/ui/composite";
import { getBackgroundColorClass, getColorClass } from "@/utils/colors";
import { filterVisibleConfiguration } from "@/utils/components";
import { withOrganizationHeader } from "@/utils/withOrganizationHeader";
import {
  getComponentAdditionalDataBuilder,
  getComponentBaseMapper,
  getTriggerRenderer,
  getCustomFieldRenderer,
  getState,
  getStateMap,
} from "./mappers";
import { useOnCancelQueueItemHandler } from "./useOnCancelQueueItemHandler";
import { usePushThroughHandler } from "./usePushThroughHandler";
import { useCancelExecutionHandler } from "./useCancelExecutionHandler";
import { useAccount } from "@/contexts/AccountContext";
import {
  buildRunEntryFromEvent,
  buildRunItemFromExecution,
  buildCanvasStatusLogEntry,
  buildTabData,
  generateNodeId,
  getNextInQueueInfo,
  mapCanvasNodesToLogEntries,
  mapExecutionsToSidebarEvents,
  mapQueueItemsToSidebarEvents,
  mapTriggerEventsToSidebarEvents,
  mapWorkflowEventsToRunLogEntries,
  summarizeWorkflowChanges,
} from "./utils";
import { SidebarEvent } from "@/ui/componentSidebar/types";
import { LogEntry, LogRunItem } from "@/ui/CanvasLogSidebar";

const BUNDLE_ICON_SLUG = "component";
const BUNDLE_COLOR = "gray";
const CANVAS_AUTO_SAVE_STORAGE_KEY = "canvas-auto-save-enabled";

type UnsavedChangeKind = "position" | "structural";

export function WorkflowPageV2() {
  const { organizationId, workflowId } = useParams<{
    organizationId: string;
    workflowId: string;
  }>();

  const navigate = useNavigate();
  const [searchParams, setSearchParams] = useSearchParams();
  const queryClient = useQueryClient();
  const { account } = useAccount();
  const updateWorkflowMutation = useUpdateWorkflow(organizationId!, workflowId!);
  const { data: triggers = [], isLoading: triggersLoading } = useTriggers();
  const { data: blueprints = [], isLoading: blueprintsLoading } = useBlueprints(organizationId!);
  const { data: components = [], isLoading: componentsLoading } = useComponents(organizationId!);
  const { data: widgets = [], isLoading: widgetsLoading } = useWidgets();
  const { data: availableApplications = [], isLoading: applicationsLoading } = useAvailableApplications();
  const { data: installedApplications = [] } = useInstalledApplications(organizationId!);
  const { data: workflow, isLoading: workflowLoading } = useWorkflow(organizationId!, workflowId!);
  const { data: workflowEventsResponse } = useWorkflowEvents(workflowId!);

  usePageTitle([workflow?.metadata?.name || "Canvas"]);

  // Warm up org users and roles cache so approval specs can pretty-print
  // user IDs as emails and role names as display names.
  // We don't use the values directly here; loading them populates the
  // react-query cache which prepareApprovalNode reads from.
  useOrganizationUsers(organizationId!);
  useOrganizationRoles(organizationId!);
  useOrganizationGroups(organizationId!);

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

  // Auto-save toggle state
  const [isAutoSaveEnabled, setIsAutoSaveEnabled] = useState(() => {
    if (typeof window !== "undefined") {
      const stored = window.localStorage.getItem(CANVAS_AUTO_SAVE_STORAGE_KEY);
      return stored !== null ? JSON.parse(stored) : true; // Default to enabled
    }
    return true;
  });

  // Revert functionality - track initial workflow snapshot
  const [initialWorkflowSnapshot, setInitialWorkflowSnapshot] = useState<WorkflowsWorkflow | null>(null);
  const lastSavedWorkflowRef = useRef<WorkflowsWorkflow | null>(null);

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

  useEffect(() => {
    if (!workflow) {
      return;
    }

    if (!lastSavedWorkflowRef.current) {
      lastSavedWorkflowRef.current = JSON.parse(JSON.stringify(workflow));
    }
  }, [workflow]);

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

  const handleToggleAutoSave = useCallback(() => {
    const newValue = !isAutoSaveEnabled;
    setIsAutoSaveEnabled(newValue);
    if (typeof window !== "undefined") {
      window.localStorage.setItem(CANVAS_AUTO_SAVE_STORAGE_KEY, JSON.stringify(newValue));
    }
  }, [isAutoSaveEnabled]);

  /**
   * Ref to track pending position updates that need to be auto-saved.
   * Maps node ID to its updated position.
   */
  const pendingPositionUpdatesRef = useRef<Map<string, { x: number; y: number }>>(new Map());
  const pendingAnnotationUpdatesRef = useRef<Map<string, { text?: string; color?: string }>>(new Map());
  const logNodeSelectRef = useRef<(nodeId: string) => void>(() => {});

  /**
   * Debounced auto-save function for node position changes.
   * Waits 1 second after the last position change before saving.
   * Only saves position changes, not structural modifications (deletions, additions, etc).
   * If there are unsaved structural changes, position auto-save is skipped.
   */
  const debouncedAutoSave = useMemo(
    () =>
      debounce(async () => {
        if (!organizationId || !workflowId) return;

        const positionUpdates = new Map(pendingPositionUpdatesRef.current);
        if (positionUpdates.size === 0) return;
        const focusedNoteId = getActiveNoteId();

        try {
          // Check if auto-save is disabled
          if (!isAutoSaveEnabled) {
            return;
          }

          // Check if there are unsaved structural changes
          // If so, skip auto-save to avoid saving those changes accidentally
          if (hasNonPositionalUnsavedChanges) {
            return;
          }

          // Fetch the latest workflow from the cache
          const latestWorkflow = queryClient.getQueryData<WorkflowsWorkflow>(
            workflowKeys.detail(organizationId, workflowId),
          );

          if (!latestWorkflow?.spec?.nodes) return;

          // Apply only position updates to the current state
          const updatedNodes = latestWorkflow.spec.nodes.map((node) => {
            if (!node.id) return node;

            const positionUpdate = positionUpdates.get(node.id);
            if (positionUpdate) {
              return {
                ...node,
                position: positionUpdate,
              };
            }
            return node;
          });

          const updatedWorkflow = {
            ...latestWorkflow,
            spec: {
              ...latestWorkflow.spec,
              nodes: updatedNodes,
            },
          };

          const changeSummary = summarizeWorkflowChanges({
            before: lastSavedWorkflowRef.current,
            after: updatedWorkflow,
            onNodeSelect: (nodeId: string) => logNodeSelectRef.current(nodeId),
          });
          const changeMessage = changeSummary.changeCount
            ? `${changeSummary.changeCount} Canvas changes saved`
            : "Canvas changes saved";

          // Save the workflow with updated positions
          await updateWorkflowMutation.mutateAsync({
            name: latestWorkflow.metadata?.name!,
            description: latestWorkflow.metadata?.description,
            nodes: updatedNodes,
            edges: latestWorkflow.spec?.edges,
          });

          if (changeSummary.detail) {
            setLiveCanvasEntries((prev) => [
              buildCanvasStatusLogEntry({
                id: `canvas-save-${Date.now()}`,
                message: changeMessage,
                type: "success",
                timestamp: new Date().toISOString(),
                detail: changeSummary.detail,
                searchText: changeSummary.searchText,
              }),
              ...prev,
            ]);
          }

          lastSavedWorkflowRef.current = JSON.parse(JSON.stringify(updatedWorkflow));

          // Clear the saved position updates after successful save
          // Keep any new updates that came in during the save
          positionUpdates.forEach((_, nodeId) => {
            if (pendingPositionUpdatesRef.current.get(nodeId) === positionUpdates.get(nodeId)) {
              pendingPositionUpdatesRef.current.delete(nodeId);
            }
          });

          // After save, merge any new pending updates into the cache
          // This prevents the server response from overwriting newer local changes
          const currentWorkflow = queryClient.getQueryData<WorkflowsWorkflow>(
            workflowKeys.detail(organizationId, workflowId),
          );

          if (currentWorkflow?.spec?.nodes && pendingPositionUpdatesRef.current.size > 0) {
            const mergedNodes = currentWorkflow.spec.nodes.map((node) => {
              if (!node.id) return node;

              const pendingUpdate = pendingPositionUpdatesRef.current.get(node.id);
              if (pendingUpdate) {
                return {
                  ...node,
                  position: pendingUpdate,
                };
              }
              return node;
            });

            queryClient.setQueryData(workflowKeys.detail(organizationId, workflowId), {
              ...currentWorkflow,
              spec: {
                ...currentWorkflow.spec,
                nodes: mergedNodes,
              },
            });
          }

          // Auto-save completed silently (no toast or state changes)
        } catch (error: any) {
          console.error("Failed to auto-save canvas changes:", error);
          // Don't show error toast for auto-save failures to avoid being intrusive
        } finally {
          if (focusedNoteId) {
            requestAnimationFrame(() => {
              restoreActiveNoteFocus();
            });
          }
        }
      }, 300),
    [
      organizationId,
      workflowId,
      updateWorkflowMutation,
      queryClient,
      hasNonPositionalUnsavedChanges,
      isAutoSaveEnabled,
    ],
  );

  const handleNodeWebsocketEvent = useCallback(
    (nodeId: string, event: string) => {
      if (event.includes("event_created")) {
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

  // Merge triggers and components from applications into the main arrays
  const allTriggers = useMemo(() => {
    const merged = [...triggers];
    availableApplications.forEach((app) => {
      if (app.triggers) {
        merged.push(...app.triggers);
      }
    });
    return merged;
  }, [triggers, availableApplications]);

  const allComponents = useMemo(() => {
    const merged = [...components];
    availableApplications.forEach((app) => {
      if (app.components) {
        merged.push(...app.components);
      }
    });
    return merged;
  }, [components, availableApplications]);

  const buildingBlocks = useMemo(
    () => buildBuildingBlockCategories(triggers, components, blueprints, availableApplications),
    [triggers, components, blueprints, availableApplications],
  );

  const { nodes, edges } = useMemo(() => {
    // Don't prepare data until everything is loaded
    if (
      !workflow ||
      workflowLoading ||
      triggersLoading ||
      blueprintsLoading ||
      componentsLoading ||
      applicationsLoading
    ) {
      return { nodes: [], edges: [] };
    }

    return prepareData(
      workflow,
      allTriggers,
      blueprints,
      allComponents,
      nodeEventsMap,
      nodeExecutionsMap,
      nodeQueueItemsMap,
      workflowId!,
      queryClient,
      organizationId!,
      account ? { id: account.id, email: account.email } : undefined,
    );
  }, [
    workflow,
    allTriggers,
    blueprints,
    allComponents,
    nodeEventsMap,
    nodeExecutionsMap,
    nodeQueueItemsMap,
    workflowId,
    queryClient,
    workflowLoading,
    triggersLoading,
    blueprintsLoading,
    componentsLoading,
    applicationsLoading,
    organizationId,
    account,
  ]);

  const getSidebarData = useCallback(
    (nodeId: string): SidebarData | null => {
      const node = workflow?.spec?.nodes?.find((n) => n.id === nodeId);
      if (!node) return null;

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
        allComponents,
        allTriggers,
        executionsMap,
        queueItemsMap,
        eventsMapForSidebar,
        totalHistoryCount,
        totalQueueCount,
        workflowId,
        queryClient,
        organizationId,
        account ? { id: account.id, email: account.email } : undefined,
      );

      // Add loading state to sidebar data
      return {
        ...sidebarData,
        isLoading: nodeData.isLoading,
      };
    },
    [
      workflow,
      workflowId,
      blueprints,
      allComponents,
      allTriggers,
      nodeEventsMap,
      getNodeData,
      queryClient,
      organizationId,
      account,
    ],
  );

  // Trigger data loading when sidebar opens for a node
  const loadSidebarData = useCallback(
    (nodeId: string) => {
      const node = workflow?.spec?.nodes?.find((n) => n.id === nodeId);
      if (!node) return;

      // Set current history node for tracking
      setCurrentHistoryNode({ nodeId, nodeType: node?.type || "TYPE_ACTION" });

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
  const [focusRequest, setFocusRequest] = useState<{
    nodeId: string;
    requestId: number;
    tab?: "latest" | "settings" | "execution-chain";
    executionChain?: {
      eventId: string;
      executionId?: string | null;
      triggerEvent?: SidebarEvent | null;
    };
  } | null>(null);
  const [liveRunEntries, setLiveRunEntries] = useState<LogEntry[]>([]);
  const [liveCanvasEntries, setLiveCanvasEntries] = useState<LogEntry[]>([]);
  const handleExecutionChainHandled = useCallback(() => setFocusRequest(null), []);

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

  const handleLogNodeSelect = useCallback(
    (nodeId: string) => {
      handleSidebarChange(true, nodeId);
      setFocusRequest({ nodeId, requestId: Date.now(), tab: "settings" });
    },
    [handleSidebarChange],
  );

  useEffect(() => {
    logNodeSelectRef.current = handleLogNodeSelect;
  }, [handleLogNodeSelect]);

  const handleLogRunNodeSelect = useCallback(
    (nodeId: string) => {
      handleSidebarChange(true, nodeId);
      setFocusRequest({ nodeId, requestId: Date.now(), tab: "latest" });
    },
    [handleSidebarChange],
  );

  const handleLogRunExecutionSelect = useCallback(
    (options: { nodeId: string; eventId: string; executionId: string; triggerEvent?: SidebarEvent }) => {
      handleSidebarChange(true, options.nodeId);
      setFocusRequest({
        nodeId: options.nodeId,
        requestId: Date.now(),
        tab: "execution-chain",
        executionChain: {
          eventId: options.eventId,
          executionId: options.executionId,
          triggerEvent: options.triggerEvent,
        },
      });
    },
    [handleSidebarChange],
  );

  const buildLiveRunItemFromExecution = useCallback(
    (execution: WorkflowsWorkflowNodeExecution): LogRunItem => {
      return buildRunItemFromExecution({
        execution,
        nodes: workflow?.spec?.nodes || [],
        onNodeSelect: handleLogRunNodeSelect,
        onExecutionSelect: handleLogRunExecutionSelect,
        event: execution.rootEvent || undefined,
      });
    },
    [handleLogRunExecutionSelect, handleLogRunNodeSelect, workflow?.spec?.nodes],
  );

  const buildLiveRunEntryFromEvent = useCallback(
    (event: WorkflowsWorkflowEvent, runItems: LogRunItem[] = []): LogEntry => {
      return buildRunEntryFromEvent({
        event,
        nodes: workflow?.spec?.nodes || [],
        runItems,
      });
    },
    [workflow?.spec?.nodes],
  );

  const handleWorkflowEventCreated = useCallback(
    (event: WorkflowsWorkflowEvent) => {
      if (!event.id) {
        return;
      }

      const nodes = workflow?.spec?.nodes || [];
      const node = nodes.find((item) => item.id === event.nodeId);
      if (!node || node.type !== "TYPE_TRIGGER") {
        return;
      }

      setLiveRunEntries((prev) => {
        const entry = buildLiveRunEntryFromEvent(event, []);
        const next = [entry, ...prev.filter((item) => item.id !== entry.id)];
        return next.sort((a, b) => {
          const aTime = Date.parse(a.timestamp || "") || 0;
          const bTime = Date.parse(b.timestamp || "") || 0;
          return bTime - aTime;
        });
      });
    },
    [buildLiveRunEntryFromEvent, workflow?.spec?.nodes],
  );

  const handleExecutionEvent = useCallback(
    (execution: WorkflowsWorkflowNodeExecution) => {
      if (!execution.rootEvent?.id) {
        return;
      }

      setLiveRunEntries((prev) => {
        const runItem = buildLiveRunItemFromExecution(execution);
        const existing = prev.find((item) => item.id === execution.rootEvent?.id);
        const existingRunItems = existing?.runItems || [];
        const runItemsMap = new Map(existingRunItems.map((item) => [item.id, item]));
        runItemsMap.set(runItem.id, runItem);
        const runItems = Array.from(runItemsMap.values());
        const entry = buildLiveRunEntryFromEvent(execution.rootEvent as WorkflowsWorkflowEvent, runItems);
        const next = [entry, ...prev.filter((item) => item.id !== entry.id)];
        return next.sort((a, b) => {
          const aTime = Date.parse(a.timestamp || "") || 0;
          const bTime = Date.parse(b.timestamp || "") || 0;
          return bTime - aTime;
        });
      });
    },
    [buildLiveRunEntryFromEvent, buildLiveRunItemFromExecution],
  );

  useWorkflowWebsocket(
    workflowId!,
    organizationId!,
    handleNodeWebsocketEvent,
    handleWorkflowEventCreated,
    handleExecutionEvent,
  );

  const logEntries = useMemo(() => {
    const nodes = workflow?.spec?.nodes || [];
    const rootEvents = workflowEventsResponse?.events || [];

    const runEntries = mapWorkflowEventsToRunLogEntries({
      events: rootEvents,
      nodes,
      onNodeSelect: handleLogRunNodeSelect,
      onExecutionSelect: handleLogRunExecutionSelect,
    });

    const mergedRunEntries = new Map<string, LogEntry>();
    runEntries.forEach((entry) => mergedRunEntries.set(entry.id, entry));
    liveRunEntries.forEach((entry) => mergedRunEntries.set(entry.id, entry));
    const allRunEntries = Array.from(mergedRunEntries.values());

    const canvasEntries = mapCanvasNodesToLogEntries({
      nodes,
      workflowUpdatedAt: workflow?.metadata?.updatedAt || "",
      onNodeSelect: handleLogNodeSelect,
    });
    const allCanvasEntries = [...liveCanvasEntries, ...canvasEntries];

    return [...allRunEntries, ...allCanvasEntries].sort((a, b) => {
      const aTime = Date.parse(a.timestamp || "") || 0;
      const bTime = Date.parse(b.timestamp || "") || 0;
      return aTime - bTime;
    });
  }, [
    handleLogNodeSelect,
    handleLogRunNodeSelect,
    handleLogRunExecutionSelect,
    liveCanvasEntries,
    liveRunEntries,
    workflow?.metadata?.updatedAt,
    workflow?.spec?.nodes,
    workflowEventsResponse?.events,
  ]);

  const nodeHistoryQuery = useNodeHistory({
    workflowId: workflowId || "",
    nodeId: currentHistoryNode?.nodeId || "",
    nodeType: currentHistoryNode?.nodeType || "TYPE_ACTION",
    allNodes: workflow?.spec?.nodes || [],
    enabled: !!currentHistoryNode && !!workflowId,
    components,
    organizationId: organizationId || "",
    queryClient,
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
      return buildTabData(nodeId, event, {
        workflowNodes: workflow?.spec?.nodes || [],
        nodeEventsMap,
        nodeExecutionsMap,
        nodeQueueItemsMap,
      });
    },
    [workflow, nodeExecutionsMap, nodeEventsMap, nodeQueueItemsMap],
  );

  const handleSaveWorkflow = useCallback(
    async (workflowToSave?: WorkflowsWorkflow, options?: { showToast?: boolean }) => {
      const targetWorkflow = workflowToSave || workflow;
      if (!targetWorkflow || !organizationId || !workflowId) return;
      const shouldRestoreFocus = options?.showToast === false;
      const focusedNoteId = shouldRestoreFocus ? getActiveNoteId() : null;
      const changeSummary = summarizeWorkflowChanges({
        before: lastSavedWorkflowRef.current,
        after: targetWorkflow,
        onNodeSelect: handleLogNodeSelect,
      });
      const changeMessage = changeSummary.changeCount
        ? `${changeSummary.changeCount} Canvas changes saved`
        : "Canvas changes saved";

      try {
        await updateWorkflowMutation.mutateAsync({
          name: targetWorkflow.metadata?.name!,
          description: targetWorkflow.metadata?.description,
          nodes: targetWorkflow.spec?.nodes,
          edges: targetWorkflow.spec?.edges,
        });

        setLiveCanvasEntries((prev) => [
          buildCanvasStatusLogEntry({
            id: `canvas-save-${Date.now()}`,
            message: changeMessage,
            type: "success",
            timestamp: new Date().toISOString(),
            detail: changeSummary.detail,
            searchText: changeSummary.searchText,
          }),
          ...prev,
        ]);
        if (options?.showToast !== false) {
          showSuccessToast("Canvas changes saved");
        }
        setHasUnsavedChanges(false);
        setHasNonPositionalUnsavedChanges(false);

        // Clear the snapshot since changes are now saved
        setInitialWorkflowSnapshot(null);
        lastSavedWorkflowRef.current = JSON.parse(JSON.stringify(targetWorkflow));
      } catch (error: any) {
        console.error("Failed to save changes to the canvas:", error);
        const errorMessage = error?.response?.data?.message || error?.message || "Failed to save changes to the canvas";
        showErrorToast(errorMessage);
        setLiveCanvasEntries((prev) => [
          buildCanvasStatusLogEntry({
            id: `canvas-save-error-${Date.now()}`,
            message: errorMessage,
            type: "error",
            timestamp: new Date().toISOString(),
          }),
          ...prev,
        ]);
      } finally {
        if (focusedNoteId) {
          requestAnimationFrame(() => {
            restoreActiveNoteFocus();
          });
        }
      }
    },
    [workflow, organizationId, workflowId, updateWorkflowMutation],
  );

  const getNodeEditData = useCallback(
    (nodeId: string): NodeEditData | null => {
      const node = workflow?.spec?.nodes?.find((n) => n.id === nodeId);
      if (!node) return null;

      // Get configuration fields from metadata based on node type
      let configurationFields: ComponentsComponent["configuration"] = [];
      let displayLabel: string | undefined = node.name || undefined;
      let appName: string | undefined;

      if (node.type === "TYPE_BLUEPRINT") {
        const blueprintMetadata = blueprints.find((b) => b.id === node.blueprint?.id);
        configurationFields = blueprintMetadata?.configuration || [];
        displayLabel = blueprintMetadata?.name || displayLabel;
      } else if (node.type === "TYPE_COMPONENT") {
        const componentMetadata = allComponents.find((c) => c.name === node.component?.name);
        configurationFields = componentMetadata?.configuration || [];
        displayLabel = componentMetadata?.label || displayLabel;

        // Check if this component is from an application
        const componentApp = availableApplications.find((app) =>
          app.components?.some((c) => c.name === node.component?.name),
        );
        if (componentApp) {
          appName = componentApp.name;
        }
      } else if (node.type === "TYPE_TRIGGER") {
        const triggerMetadata = allTriggers.find((t) => t.name === node.trigger?.name);
        configurationFields = triggerMetadata?.configuration || [];
        displayLabel = triggerMetadata?.label || displayLabel;

        // Check if this trigger is from an application
        const triggerApp = availableApplications.find((app) =>
          app.triggers?.some((t) => t.name === node.trigger?.name),
        );
        if (triggerApp) {
          appName = triggerApp.name;
        }
      } else if (node.type === "TYPE_WIDGET") {
        const widget = widgets.find((w) => w.name === node.widget?.name);
        if (widget) {
          configurationFields = widget.configuration || [];
          displayLabel = widget.label || "Widget";
        }

        return {
          nodeId: node.id!,
          nodeName: node.name!,
          displayLabel,
          configuration: {
            text: node.configuration?.text || "",
            color: node.configuration?.color || "yellow",
          },
          configurationFields,
          appName,
          appInstallationRef: node.appInstallation,
        };
      }

      return {
        nodeId: node.id!,
        nodeName: node.name!,
        displayLabel,
        configuration: node.configuration || {},
        configurationFields,
        appName,
        appInstallationRef: node.appInstallation,
      };
    },
    [workflow, blueprints, allComponents, allTriggers, availableApplications, widgets],
  );

  const handleNodeConfigurationSave = useCallback(
    async (
      nodeId: string,
      updatedConfiguration: Record<string, any>,
      updatedNodeName: string,
      appInstallationRef?: ComponentsAppInstallationRef,
    ) => {
      if (!workflow || !organizationId || !workflowId) return;

      // Save snapshot before making changes
      saveWorkflowSnapshot(workflow);

      // Update the node's configuration, name, and app installation ref in local cache only
      const updatedNodes = workflow?.spec?.nodes?.map((node) => {
        if (node.id === nodeId) {
          // Handle widget nodes like any other node - store in configuration
          if (node.type === "TYPE_WIDGET") {
            return {
              ...node,
              name: updatedNodeName,
              configuration: { ...node.configuration, ...updatedConfiguration },
            };
          }

          return {
            ...node,
            configuration: updatedConfiguration,
            name: updatedNodeName,
            appInstallation: appInstallationRef,
          };
        }
        return node;
      });

      const updatedWorkflow = {
        ...workflow,
        spec: {
          ...workflow.spec,
          nodes: updatedNodes,
        },
      };

      // Update local cache
      queryClient.setQueryData(workflowKeys.detail(organizationId, workflowId), updatedWorkflow);

      if (isAutoSaveEnabled) {
        await handleSaveWorkflow(updatedWorkflow, { showToast: false });
      } else {
        markUnsavedChange("structural");
      }
    },
    [
      workflow,
      organizationId,
      workflowId,
      queryClient,
      saveWorkflowSnapshot,
      handleSaveWorkflow,
      isAutoSaveEnabled,
      markUnsavedChange,
    ],
  );

  const debouncedAnnotationAutoSave = useMemo(
    () =>
      debounce(async () => {
        if (!organizationId || !workflowId) return;

        const annotationUpdates = new Map(pendingAnnotationUpdatesRef.current);
        if (annotationUpdates.size === 0) return;

        if (!isAutoSaveEnabled) {
          return;
        }

        const latestWorkflow = queryClient.getQueryData<WorkflowsWorkflow>(
          workflowKeys.detail(organizationId, workflowId),
        );

        if (!latestWorkflow?.spec?.nodes) return;

        const updatedNodes = latestWorkflow.spec.nodes.map((node) => {
          if (!node.id || node.type !== "TYPE_WIDGET") {
            return node;
          }

          const updates = annotationUpdates.get(node.id);
          if (!updates) {
            return node;
          }

          return {
            ...node,
            configuration: {
              ...node.configuration,
              ...updates,
            },
          };
        });

        const updatedWorkflow = {
          ...latestWorkflow,
          spec: {
            ...latestWorkflow.spec,
            nodes: updatedNodes,
          },
        };

        await handleSaveWorkflow(updatedWorkflow, { showToast: false });

        annotationUpdates.forEach((updates, nodeId) => {
          if (pendingAnnotationUpdatesRef.current.get(nodeId) === updates) {
            pendingAnnotationUpdatesRef.current.delete(nodeId);
          }
        });
      }, 600),
    [organizationId, workflowId, queryClient, handleSaveWorkflow, isAutoSaveEnabled],
  );

  const handleAnnotationUpdate = useCallback(
    (nodeId: string, updates: { text?: string; color?: string }) => {
      if (!workflow || !organizationId || !workflowId) return;
      if (Object.keys(updates).length === 0) return;

      saveWorkflowSnapshot(workflow);

      const latestWorkflow =
        queryClient.getQueryData<WorkflowsWorkflow>(workflowKeys.detail(organizationId, workflowId)) || workflow;

      const updatedNodes = latestWorkflow?.spec?.nodes?.map((node) => {
        if (node.id !== nodeId || node.type !== "TYPE_WIDGET") {
          return node;
        }

        return {
          ...node,
          configuration: {
            ...node.configuration,
            ...updates,
          },
        };
      });

      const updatedWorkflow = {
        ...latestWorkflow,
        spec: {
          ...latestWorkflow.spec,
          nodes: updatedNodes,
        },
      };

      queryClient.setQueryData(workflowKeys.detail(organizationId, workflowId), updatedWorkflow);

      if (isAutoSaveEnabled) {
        const existing = pendingAnnotationUpdatesRef.current.get(nodeId) || {};
        pendingAnnotationUpdatesRef.current.set(nodeId, { ...existing, ...updates });
        debouncedAnnotationAutoSave();
      } else {
        markUnsavedChange("position");
      }
    },
    [
      workflow,
      organizationId,
      workflowId,
      queryClient,
      saveWorkflowSnapshot,
      debouncedAnnotationAutoSave,
      isAutoSaveEnabled,
      markUnsavedChange,
    ],
  );

  const handleNodeAdd = useCallback(
    async (newNodeData: NewNodeData): Promise<string> => {
      if (!workflow || !organizationId || !workflowId) return "";

      // Save snapshot before making changes
      saveWorkflowSnapshot(workflow);

      const { buildingBlock, nodeName, configuration, position, sourceConnection, appInstallationRef } = newNodeData;

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
              : buildingBlock.name === "annotation"
                ? "TYPE_WIDGET"
                : "TYPE_COMPONENT",
        configuration: filteredConfiguration,
        appInstallation: appInstallationRef,
        position: position
          ? {
              x: Math.round(position.x),
              y: Math.round(position.y),
            }
          : {
              x: (workflow?.spec?.nodes?.length || 0) * 250,
              y: 100,
            },
      };

      // Add type-specific reference
      if (buildingBlock.name === "annotation") {
        // Annotation nodes are now widgets
        newNode.widget = { name: "annotation" };
        newNode.configuration = { text: "", color: "yellow" };
      } else if (buildingBlock.type === "component") {
        newNode.component = { name: buildingBlock.name };
      } else if (buildingBlock.type === "trigger") {
        newNode.trigger = { name: buildingBlock.name };
      } else if (buildingBlock.type === "blueprint") {
        newNode.blueprint = { id: buildingBlock.id };
      }

      // Add the new node to the workflow
      const updatedNodes = [...(workflow.spec?.nodes || []), newNode];

      // If there's a source connection, create the edge
      let updatedEdges = workflow.spec?.edges || [];
      if (sourceConnection) {
        const newEdge: ComponentsEdge = {
          sourceId: sourceConnection.nodeId,
          targetId: newNodeId,
          channel: sourceConnection.handleId || "default",
        };
        updatedEdges = [...updatedEdges, newEdge];
      }

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

      if (isAutoSaveEnabled) {
        await handleSaveWorkflow(updatedWorkflow, { showToast: false });
      } else {
        markUnsavedChange("structural");
      }

      // Return the new node ID
      return newNodeId;
    },
    [
      workflow,
      organizationId,
      workflowId,
      queryClient,
      saveWorkflowSnapshot,
      handleSaveWorkflow,
      isAutoSaveEnabled,
      markUnsavedChange,
    ],
  );

  const handlePlaceholderAdd = useCallback(
    async (data: {
      position: { x: number; y: number };
      sourceNodeId: string;
      sourceHandleId: string | null;
    }): Promise<string> => {
      if (!workflow || !organizationId || !workflowId) return "";

      saveWorkflowSnapshot(workflow);

      const placeholderName = "New Component";
      const newNodeId = generateNodeId("component", "node");

      // Create placeholder node - will fail validation but still be saved
      const newNode: ComponentsNode = {
        id: newNodeId,
        name: placeholderName,
        type: "TYPE_COMPONENT",
        // NO component/blueprint/trigger reference - causes validation error
        configuration: {},
        metadata: {},
        position: {
          x: Math.round(data.position.x),
          y: Math.round(data.position.y),
        },
      };

      const newEdge: ComponentsEdge = {
        sourceId: data.sourceNodeId,
        targetId: newNodeId,
        channel: data.sourceHandleId || "default",
      };

      const updatedWorkflow = {
        ...workflow,
        spec: {
          ...workflow.spec,
          nodes: [...(workflow.spec?.nodes || []), newNode],
          edges: [...(workflow.spec?.edges || []), newEdge],
        },
      };

      queryClient.setQueryData(workflowKeys.detail(organizationId, workflowId), updatedWorkflow);

      if (isAutoSaveEnabled) {
        await handleSaveWorkflow(updatedWorkflow, { showToast: false });
      } else {
        markUnsavedChange("structural");
      }

      return newNodeId;
    },
    [
      workflow,
      organizationId,
      workflowId,
      queryClient,
      saveWorkflowSnapshot,
      handleSaveWorkflow,
      isAutoSaveEnabled,
      markUnsavedChange,
    ],
  );

  const handlePlaceholderConfigure = useCallback(
    async (data: {
      placeholderId: string;
      buildingBlock: any;
      nodeName: string;
      configuration: Record<string, any>;
      appName?: string;
    }): Promise<void> => {
      if (!workflow || !organizationId || !workflowId) {
        return;
      }

      saveWorkflowSnapshot(workflow);

      const nodeIndex = workflow.spec?.nodes?.findIndex((n) => n.id === data.placeholderId);
      if (nodeIndex === undefined || nodeIndex === -1) {
        return;
      }

      const filteredConfiguration = filterVisibleConfiguration(
        data.configuration,
        data.buildingBlock.configuration || [],
      );

      // Update placeholder with real component data
      const updatedNode: ComponentsNode = {
        ...workflow.spec!.nodes![nodeIndex],
        name: data.nodeName.trim(),
        type:
          data.buildingBlock.type === "trigger"
            ? "TYPE_TRIGGER"
            : data.buildingBlock.type === "blueprint"
              ? "TYPE_BLUEPRINT"
              : "TYPE_COMPONENT",
        configuration: filteredConfiguration,
      };

      // Add the reference that was missing
      if (data.buildingBlock.type === "component") {
        updatedNode.component = { name: data.buildingBlock.name };
      } else if (data.buildingBlock.type === "trigger") {
        updatedNode.trigger = { name: data.buildingBlock.name };
      } else if (data.buildingBlock.type === "blueprint") {
        updatedNode.blueprint = { id: data.buildingBlock.id };
      }

      const updatedNodes = [...(workflow.spec?.nodes || [])];
      updatedNodes[nodeIndex] = updatedNode;

      // Update outgoing edges from this node to use valid channels
      // Find edges where this node is the source
      const outgoingEdges = workflow.spec?.edges?.filter((edge) => edge.sourceId === data.placeholderId) || [];

      let updatedEdges = [...(workflow.spec?.edges || [])];

      if (outgoingEdges.length > 0) {
        // Get the valid output channels for the new component
        const validChannels = data.buildingBlock.outputChannels?.map((ch: any) => ch.name).filter(Boolean) || [
          "default",
        ];

        // Update each outgoing edge to use a valid channel
        updatedEdges = updatedEdges.map((edge) => {
          if (edge.sourceId === data.placeholderId) {
            // If the current channel is not valid for the new component, use the first valid channel
            const newChannel = validChannels.includes(edge.channel) ? edge.channel : validChannels[0];
            return {
              ...edge,
              channel: newChannel,
            };
          }
          return edge;
        });
      }

      const updatedWorkflow = {
        ...workflow,
        spec: {
          ...workflow.spec,
          nodes: updatedNodes,
          edges: updatedEdges,
        },
      };

      queryClient.setQueryData(workflowKeys.detail(organizationId, workflowId), updatedWorkflow);

      if (isAutoSaveEnabled) {
        await handleSaveWorkflow(updatedWorkflow, { showToast: false });
      } else {
        markUnsavedChange("structural");
      }
    },
    [
      workflow,
      organizationId,
      workflowId,
      queryClient,
      saveWorkflowSnapshot,
      handleSaveWorkflow,
      isAutoSaveEnabled,
      markUnsavedChange,
    ],
  );

  const handleEdgeCreate = useCallback(
    async (sourceId: string, targetId: string, sourceHandle?: string | null) => {
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

      if (isAutoSaveEnabled) {
        await handleSaveWorkflow(updatedWorkflow, { showToast: false });
      } else {
        markUnsavedChange("structural");
      }
    },
    [
      workflow,
      organizationId,
      workflowId,
      queryClient,
      saveWorkflowSnapshot,
      handleSaveWorkflow,
      isAutoSaveEnabled,
      markUnsavedChange,
    ],
  );

  const handleNodeDelete = useCallback(
    async (nodeId: string) => {
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

      if (isAutoSaveEnabled) {
        await handleSaveWorkflow(updatedWorkflow, { showToast: false });
      } else {
        markUnsavedChange("structural");
      }
    },
    [
      workflow,
      organizationId,
      workflowId,
      queryClient,
      saveWorkflowSnapshot,
      handleSaveWorkflow,
      isAutoSaveEnabled,
      markUnsavedChange,
    ],
  );

  const handleEdgeDelete = useCallback(
    async (edgeIds: string[]) => {
      if (!workflow || !organizationId || !workflowId) return;

      // Save snapshot before making changes
      saveWorkflowSnapshot(workflow);

      // Parse edge IDs to extract sourceId, targetId, and channel
      // Edge IDs are formatted as: `${sourceId}--${targetId}--${channel}`
      const edgesToRemove = edgeIds.map((edgeId) => {
        let parts = edgeId?.split("-targets->") || [];
        parts = parts.flatMap((part) => part.split("-using->"));
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

      if (isAutoSaveEnabled) {
        await handleSaveWorkflow(updatedWorkflow, { showToast: false });
      } else {
        markUnsavedChange("structural");
      }
    },
    [
      workflow,
      organizationId,
      workflowId,
      queryClient,
      saveWorkflowSnapshot,
      handleSaveWorkflow,
      isAutoSaveEnabled,
      markUnsavedChange,
    ],
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

      const roundedPosition = {
        x: Math.round(position.x),
        y: Math.round(position.y),
      };

      const updatedNodes = workflow.spec?.nodes?.map((node) =>
        node.id === nodeId
          ? {
              ...node,
              position: roundedPosition,
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

      if (isAutoSaveEnabled) {
        pendingPositionUpdatesRef.current.set(nodeId, roundedPosition);

        debouncedAutoSave();
      } else {
        saveWorkflowSnapshot(workflow);
        markUnsavedChange("position");
      }
    },
    [
      workflow,
      organizationId,
      workflowId,
      queryClient,
      debouncedAutoSave,
      isAutoSaveEnabled,
      saveWorkflowSnapshot,
      markUnsavedChange,
    ],
  );

  const handleNodeCollapseChange = useCallback(
    async (nodeId: string) => {
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

      if (isAutoSaveEnabled) {
        await handleSaveWorkflow(updatedWorkflow, { showToast: false });
      } else {
        markUnsavedChange("structural");
      }
    },
    [
      workflow,
      organizationId,
      workflowId,
      queryClient,
      saveWorkflowSnapshot,
      handleSaveWorkflow,
      isAutoSaveEnabled,
      markUnsavedChange,
    ],
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
    async (nodeId: string) => {
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
      if (isAutoSaveEnabled) {
        await handleSaveWorkflow(updatedWorkflow, { showToast: false });
      } else {
        markUnsavedChange("structural");
      }
    },
    [
      workflow,
      organizationId,
      workflowId,
      blueprints,
      queryClient,
      saveWorkflowSnapshot,
      handleSaveWorkflow,
      isAutoSaveEnabled,
      markUnsavedChange,
    ],
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

      const updatedWorkflow = {
        ...workflow,
        spec: {
          ...workflow.spec,
          nodes: updatedNodes,
        },
      };

      const changeSummary = summarizeWorkflowChanges({
        before: lastSavedWorkflowRef.current,
        after: updatedWorkflow,
        onNodeSelect: handleLogNodeSelect,
      });
      const changeMessage = changeSummary.changeCount
        ? `${changeSummary.changeCount} Canvas changes saved`
        : "Canvas changes saved";

      try {
        await updateWorkflowMutation.mutateAsync({
          name: workflow.metadata?.name!,
          description: workflow.metadata?.description,
          nodes: updatedNodes,
          edges: workflow.spec?.edges,
        });

        setLiveCanvasEntries((prev) => [
          buildCanvasStatusLogEntry({
            id: `canvas-save-${Date.now()}`,
            message: changeMessage,
            type: "success",
            timestamp: new Date().toISOString(),
            detail: changeSummary.detail,
            searchText: changeSummary.searchText,
          }),
          ...prev,
        ]);
        showSuccessToast("Canvas changes saved");
        setHasUnsavedChanges(false);
        setHasNonPositionalUnsavedChanges(false);

        // Clear the snapshot since changes are now saved
        setInitialWorkflowSnapshot(null);
        lastSavedWorkflowRef.current = JSON.parse(JSON.stringify(updatedWorkflow));
      } catch (error: any) {
        console.error("Failed to save changes to the canvas:", error);
        const errorMessage = error?.response?.data?.message || error?.message || "Failed to save changes to the canvas";
        showErrorToast(errorMessage);
      }
    },
    [workflow, organizationId, workflowId, updateWorkflowMutation],
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

  // Provide state function based on component type
  const getExecutionState = useCallback(
    (nodeId: string, execution: WorkflowsWorkflowNodeExecution): { map: EventStateMap; state: EventState } => {
      const node = workflow?.spec?.nodes?.find((n) => n.id === nodeId);
      if (!node) {
        return {
          map: getStateMap("default"),
          state: getState("default")(execution),
        };
      }

      let componentName = "default";
      if (node.type === "TYPE_COMPONENT" && node.component?.name) {
        componentName = node.component.name;
      } else if (node.type === "TYPE_TRIGGER" && node.trigger?.name) {
        componentName = node.trigger.name;
      } else if (node.type === "TYPE_BLUEPRINT" && node.blueprint?.id) {
        componentName = "default";
      }

      return {
        map: getStateMap(componentName),
        state: getState(componentName)(execution),
      };
    },
    [workflow],
  );

  const getCustomField = useCallback(
    (nodeId: string) => {
      const node = workflow?.spec?.nodes?.find((n) => n.id === nodeId);
      if (!node) return null;

      let componentName = "";
      if (node.type === "TYPE_TRIGGER" && node.trigger?.name) {
        componentName = node.trigger.name;
      } else if (node.type === "TYPE_COMPONENT" && node.component?.name) {
        componentName = node.component.name;
      } else if (node.type === "TYPE_BLUEPRINT" && node.blueprint?.id) {
        componentName = "default";
      }

      const renderer = getCustomFieldRenderer(componentName);
      if (!renderer) return null;

      // Return a function that takes the current configuration
      return (configuration: Record<string, unknown>) => {
        return renderer.render(node, configuration);
      };
    },
    [workflow],
  );

  // Show loading indicator while data is being fetched
  if (workflowLoading || triggersLoading || blueprintsLoading || componentsLoading || widgetsLoading) {
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
          <p className="text-sm text-gray-500">Canvas not found</p>
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
      getCustomField={getCustomField}
      onNodeConfigurationSave={handleNodeConfigurationSave}
      onAnnotationUpdate={handleAnnotationUpdate}
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
      onPlaceholderAdd={handlePlaceholderAdd}
      onPlaceholderConfigure={handlePlaceholderConfigure}
      installedApplications={installedApplications}
      hasFitToViewRef={hasFitToViewRef}
      hasUserToggledSidebarRef={hasUserToggledSidebarRef}
      isSidebarOpenRef={isSidebarOpenRef}
      viewportRef={viewportRef}
      unsavedMessage={hasUnsavedChanges ? "You have unsaved changes" : undefined}
      saveIsPrimary={hasUnsavedChanges}
      saveButtonHidden={!hasUnsavedChanges}
      onUndo={handleRevert}
      canUndo={!isAutoSaveEnabled && initialWorkflowSnapshot !== null}
      isAutoSaveEnabled={isAutoSaveEnabled}
      onToggleAutoSave={handleToggleAutoSave}
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
      getExecutionState={getExecutionState}
      workflowNodes={workflow?.spec?.nodes}
      components={components}
      triggers={triggers}
      blueprints={blueprints}
      logEntries={logEntries}
      focusRequest={focusRequest}
      onExecutionChainHandled={handleExecutionChainHandled}
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
  currentUser?: { id?: string; email?: string },
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
          currentUser,
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
        error: node.errorMessage,
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
  const color = BUNDLE_COLOR;
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
        iconSlug: BUNDLE_ICON_SLUG,
        iconColor: getColorClass(color),
        collapsedBackground: getBackgroundColorClass(color),
        collapsed: node.isCollapsed,
        title: displayLabel,
        description: blueprintMetadata?.description,
        isMissing: isMissing,
        error: node.errorMessage,
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
      id: execution.rootEvent?.id,
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
  currentUser?: { id?: string; email?: string },
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
    case "TYPE_WIDGET":
      // support other widgets if necessary
      return prepareAnnotationNode(node);

    default:
      return prepareComponentNode(
        nodes,
        node,
        components,
        nodeExecutionsMap,
        nodeQueueItemsMap,
        workflowId,
        queryClient,
        organizationId,
        currentUser,
      );
  }
}

function prepareAnnotationNode(node: ComponentsNode): CanvasNode {
  return {
    id: node.id!,
    position: { x: node.position?.x!, y: node.position?.y! },
    selectable: false,
    data: {
      type: "annotation",
      label: node.name || "Annotation",
      state: "pending" as const,
      outputChannels: [], // Annotation nodes don't have output channels
      annotation: {
        title: node.name || "Annotation",
        annotationText: node.configuration?.text || "",
        annotationColor: node.configuration?.color || "yellow",
      },
    },
  };
}

function prepareComponentNode(
  nodes: ComponentsNode[],
  node: ComponentsNode,
  components: ComponentsComponent[],
  nodeExecutionsMap: Record<string, WorkflowsWorkflowNodeExecution[]>,
  nodeQueueItemsMap: Record<string, WorkflowsWorkflowNodeQueueItem[]>,
  workflowId: string,
  queryClient: QueryClient,
  organizationId?: string,
  currentUser?: { id?: string; email?: string },
): CanvasNode {
  // Detect placeholder nodes (no component reference, name is "New Component")
  const isPlaceholder = !node.component?.name && node.name === "New Component";

  if (isPlaceholder) {
    // Render placeholder as a ComponentBase with error state styling
    const canvasNode: CanvasNode = {
      id: node.id!,
      position: { x: node.position?.x!, y: node.position?.y! },
      data: {
        type: "component",
        label: "New Component",
        state: "pending" as const,
        outputChannels: ["default"],
        component: {
          iconSlug: "box-dashed",
          iconColor: getColorClass("gray"),
          collapsedBackground: getBackgroundColorClass("gray"),
          collapsed: false,
          title: "New Component",
          includeEmptyState: true,
          emptyStateProps: {
            icon: Puzzle,
            title: "Select a component from the sidebar",
          },
          error: "Select a component from the sidebar",
          parameters: [],
        },
      },
    };
    return canvasNode;
  }

  const componentNameParts = node.component?.name?.split(".") || [];
  const componentName = componentNameParts[0];

  if (componentName == "merge") {
    return prepareMergeNode(nodes, node, components, nodeExecutionsMap, nodeQueueItemsMap);
  }

  return prepareComponentBaseNode(
    nodes,
    node,
    components,
    nodeExecutionsMap,
    nodeQueueItemsMap,
    workflowId,
    queryClient,
    organizationId || "",
    currentUser,
  );
}

function prepareComponentBaseNode(
  nodes: ComponentsNode[],
  node: ComponentsNode,
  components: ComponentsComponent[],
  nodeExecutionsMap: Record<string, WorkflowsWorkflowNodeExecution[]>,
  nodeQueueItemsMap: Record<string, WorkflowsWorkflowNodeQueueItem[]>,
  workflowId: string,
  queryClient: QueryClient,
  organizationId: string,
  currentUser?: { id?: string; email?: string },
): CanvasNode {
  const executions = nodeExecutionsMap[node.id!] || [];
  const metadata = components.find((c) => c.name === node.component?.name);
  const displayLabel = node.name || metadata?.label;
  const componentDef = components.find((c) => c.name === node.component?.name);
  const nodeQueueItems = nodeQueueItemsMap?.[node.id!];

  const additionalData = getComponentAdditionalDataBuilder(node.component?.name || "")?.buildAdditionalData(
    nodes,
    node,
    componentDef!,
    executions,
    workflowId,
    queryClient,
    organizationId,
    currentUser,
  );

  const componentBaseProps = getComponentBaseMapper(node.component?.name || "").props(
    nodes,
    node,
    componentDef!,
    executions,
    nodeQueueItems,
    additionalData,
  );

  // If there's an error and empty state is shown, customize the message
  const hasError = !!node.errorMessage;
  const showingEmptyState = componentBaseProps.includeEmptyState;
  const emptyStateProps =
    hasError && showingEmptyState
      ? {
          ...componentBaseProps.emptyStateProps,
          icon: componentBaseProps.emptyStateProps?.icon || Puzzle,
          title: "Finish configuring this component",
        }
      : componentBaseProps.emptyStateProps;

  return {
    id: node.id!,
    position: { x: node.position?.x || 0, y: node.position?.y || 0 },
    data: {
      type: "component",
      label: displayLabel,
      state: "pending" as const,
      outputChannels: metadata?.outputChannels?.map((channel) => channel.name) || ["default"],
      component: {
        ...componentBaseProps,
        emptyStateProps,
        error: node.errorMessage,
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
      eventState: executionToEventSectionState(execution),
      eventId: execution.rootEvent?.id,
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
        lastEvent: lastEvent,
        nextInQueue: getNextInQueueInfo(nodeQueueItemsMap, node.id!, nodes),
        collapsedBackground: getBackgroundColorClass("white"),
        collapsed: node.isCollapsed,
      },
    },
  };
}

function prepareEdge(edge: ComponentsEdge): CanvasEdge {
  const id = `${edge.sourceId!}-targets->${edge.targetId!}-using->${edge.channel!}`;

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
  workflowId?: string,
  queryClient?: QueryClient,
  organizationId?: string,
  currentUser?: { id?: string; email?: string },
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

  const nodeTitle =
    componentMetadata?.label || blueprintMetadata?.name || triggerMetadata?.label || node.name || "Unknown";
  let iconSlug = "boxes";
  let color = "indigo";

  if (blueprintMetadata) {
    iconSlug = BUNDLE_ICON_SLUG;
    color = BUNDLE_COLOR;
  } else if (componentMetadata) {
    iconSlug = componentMetadata.icon || iconSlug;
    color = componentMetadata.color || color;
  } else if (triggerMetadata) {
    iconSlug = triggerMetadata.icon || iconSlug;
    color = triggerMetadata.color || color;
  }

  const additionalData = getComponentAdditionalDataBuilder(node.component?.name || "")?.buildAdditionalData(
    nodes,
    node,
    componentMetadata!,
    executions,
    workflowId || "",
    queryClient as QueryClient,
    organizationId || "",
    currentUser,
  );

  const latestEvents =
    node.type === "TYPE_TRIGGER"
      ? mapTriggerEventsToSidebarEvents(events, node, 5)
      : mapExecutionsToSidebarEvents(executions, nodes, 5, additionalData);

  // Convert queue items to sidebar events (next in queue)
  const nextInQueueEvents = mapQueueItemsToSidebarEvents(queueItems, nodes, 5);
  const hideQueueEvents = node.type === "TYPE_TRIGGER";

  return {
    latestEvents,
    nextInQueueEvents,
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
