import { useNodeExecutionStore } from "@/stores/nodeExecutionStore";
import { showErrorToast, showSuccessToast } from "@/utils/toast";
import { QueryClient, useQueryClient } from "@tanstack/react-query";
import debounce from "lodash.debounce";
import { Loader2, Puzzle } from "lucide-react";
import * as yaml from "js-yaml";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useNavigate, useParams, useSearchParams } from "react-router-dom";

import {
  BlueprintsBlueprint,
  ComponentsIntegrationRef,
  ComponentsComponent,
  ComponentsEdge,
  ComponentsNode,
  TriggersTrigger,
  CanvasesListEventExecutionsResponse,
  CanvasesCanvas,
  CanvasesCanvasEvent,
  CanvasesCanvasNodeExecution,
  CanvasesCanvasNodeQueueItem,
  canvasesEmitNodeEvent,
  canvasesUpdateNodePause,
  OrganizationsIntegration,
} from "@/api-client";
import {
  useOrganizationAgentSettings,
  useOrganizationGroups,
  useOrganizationRoles,
  useOrganizationUsers,
} from "@/hooks/useOrganizationData";

import { useBlueprints, useComponents } from "@/hooks/useBlueprintData";
import { useNodeHistory } from "@/hooks/useNodeHistory";
import { usePageTitle } from "@/hooks/usePageTitle";
import { useQueueHistory } from "@/hooks/useQueueHistory";
import { useAvailableIntegrations, useConnectedIntegrations } from "@/hooks/useIntegrations";
import {
  eventExecutionsQueryOptions,
  useCreateCanvas,
  useTriggers,
  useUpdateCanvas,
  useCanvas,
  useCanvasEvents,
  useWidgets,
  canvasKeys,
} from "@/hooks/useCanvasData";
import { useCanvasWebsocket } from "@/hooks/useCanvasWebsocket";
import { buildBuildingBlockCategories } from "@/ui/buildingBlocks";
import { AiCanvasOperation } from "@/ui/BuildingBlocksSidebar";
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
import { CreateCanvasModal } from "@/components/CreateCanvasModal";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  getComponentAdditionalDataBuilder,
  getComponentBaseMapper,
  getTriggerRenderer,
  getCustomFieldRenderer,
  getState,
  getStateMap,
} from "./mappers";
import { resolveExecutionErrors } from "./mappers/dash0";
import { getHeaderIconSrc } from "@/ui/componentSidebar/integrationIcons";
import { useOnCancelQueueItemHandler } from "./useOnCancelQueueItemHandler";
import { usePushThroughHandler } from "./usePushThroughHandler";
import { useCancelExecutionHandler } from "./useCancelExecutionHandler";
import { applyAiOperationsToWorkflow } from "./applyAiOperationsToWorkflow";
import { applyHorizontalAutoLayout } from "./autoLayout";
import { useAccount } from "@/contexts/AccountContext";
import { usePermissions } from "@/contexts/PermissionsContext";
import { useApprovalGroupUsersPrefetch } from "@/hooks/useApprovalGroupUsersPrefetch";
import {
  buildRunEntryFromEvent,
  buildRunItemFromExecution,
  buildCanvasStatusLogEntry,
  buildTabData,
  generateNodeId,
  generateUniqueNodeName,
  getNextInQueueInfo,
  mapCanvasNodesToLogEntries,
  mapExecutionsToSidebarEvents,
  mapQueueItemsToSidebarEvents,
  mapTriggerEventsToSidebarEvents,
  mapWorkflowEventsToRunLogEntries,
  summarizeWorkflowChanges,
  buildNodeInfo,
  buildEventInfo,
  buildComponentDefinition,
  buildExecutionInfo,
  buildQueueItemInfo,
} from "./utils";
import { SidebarEvent } from "@/ui/componentSidebar/types";
import { LogEntry, LogRunItem } from "@/ui/CanvasLogSidebar";

const BUNDLE_ICON_SLUG = "component";
const BUNDLE_COLOR = "gray";
const CANVAS_AUTO_SAVE_STORAGE_KEY = "canvas-auto-save-enabled";
const LOCAL_CANVAS_UPDATE_SUPPRESSION_MS = 2000;

type UnsavedChangeKind = "position" | "structural";

export function WorkflowPageV2() {
  const { organizationId, canvasId } = useParams<{
    organizationId: string;
    canvasId: string;
  }>();

  const navigate = useNavigate();
  const [searchParams, setSearchParams] = useSearchParams();
  const queryClient = useQueryClient();
  const { account } = useAccount();
  const { canAct } = usePermissions();
  const updateWorkflowMutation = useUpdateCanvas(organizationId!, canvasId!);
  const { data: triggers = [], isLoading: triggersLoading } = useTriggers();
  const { data: blueprints = [], isLoading: blueprintsLoading } = useBlueprints(organizationId!);
  const { data: components = [], isLoading: componentsLoading } = useComponents(organizationId!);
  const { data: widgets = [], isLoading: widgetsLoading } = useWidgets();
  const { data: availableIntegrations = [], isLoading: integrationsLoading } = useAvailableIntegrations();
  const canReadIntegrations = canAct("integrations", "read");
  const canCreateIntegrations = canAct("integrations", "create");
  const canUpdateIntegrations = canAct("integrations", "update");
  const { data: integrations = [] } = useConnectedIntegrations(organizationId!, { enabled: canReadIntegrations });
  const { data: canvas, isLoading: canvasLoading, error: canvasError } = useCanvas(organizationId!, canvasId!);
  const { data: canvasEventsResponse } = useCanvasEvents(canvasId!);
  const canReadOrg = canAct("org", "read");
  const { data: agentSettings } = useOrganizationAgentSettings(organizationId || "", !!organizationId && canReadOrg);
  const canUpdateCanvas = canAct("canvases", "update");
  const showAiBuilderTab = agentSettings?.agentModeEnabled ?? false;

  usePageTitle([canvas?.metadata?.name || "Canvas"]);

  const isTemplate = canvas?.metadata?.isTemplate ?? false;
  const [canvasDeletedRemotely, setCanvasDeletedRemotely] = useState(false);
  const [remoteCanvasUpdatePending, setRemoteCanvasUpdatePending] = useState(false);
  const isReadOnly = isTemplate || !canUpdateCanvas || canvasDeletedRemotely;
  const isDev = import.meta.env.DEV;
  const [isUseTemplateOpen, setIsUseTemplateOpen] = useState(false);
  const createWorkflowMutation = useCreateCanvas(organizationId!);

  // Warm up org users and roles cache so approval specs can pretty-print
  // user IDs as emails and role names as display names.
  // We don't use the values directly here; loading them populates the
  // react-query cache which prepareApprovalNode reads from.
  const { isLoading: usersLoading } = useOrganizationUsers(organizationId!);
  const { isLoading: rolesLoading } = useOrganizationRoles(organizationId!);
  const { isLoading: groupsLoading } = useOrganizationGroups(organizationId!);

  /**
   * Track if we've already done the initial fit to view.
   * This ref persists across re-renders to prevent viewport changes on save.
   */
  const hasFitToViewRef = useRef(false);

  /**
   * Capture the initial node focus from the URL so we only zoom once.
   */
  const initialFocusNodeIdRef = useRef<string | null>(null);
  if (initialFocusNodeIdRef.current === null) {
    initialFocusNodeIdRef.current = searchParams.get("node") || null;
  }

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
  if (isSidebarOpenRef.current === null && canvas) {
    // Initialize on first render
    isSidebarOpenRef.current = canvas.spec?.nodes?.length === 0;
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
  const canAutoSave = isAutoSaveEnabled && !isTemplate;

  // Revert functionality - track initial workflow snapshot
  const [initialWorkflowSnapshot, setInitialWorkflowSnapshot] = useState<CanvasesCanvas | null>(null);
  const lastSavedWorkflowRef = useRef<CanvasesCanvas | null>(null);
  const canvasRef = useRef<CanvasesCanvas | null>(canvas ?? null);
  const lastLocalCanvasSaveAtRef = useRef<number>(0);
  useEffect(() => {
    canvasRef.current = canvas ?? null;
  }, [canvas]);

  // Use Zustand store for execution data - extract only the methods to avoid recreating callbacks
  // Subscribe to version to ensure React detects all updates
  const storeVersion = useNodeExecutionStore((state) => state.version);
  const getNodeData = useNodeExecutionStore((state) => state.getNodeData);
  const loadNodeDataMethod = useNodeExecutionStore((state) => state.loadNodeData);
  const initializeFromWorkflow = useNodeExecutionStore((state) => state.initializeFromWorkflow);

  // Redirect to home page if workflow is not found (404)
  // Use replace to avoid back button issues and prevent 404 flash
  useEffect(() => {
    if (canvasError && !canvasLoading) {
      // Check if it's a 404 error
      const is404 =
        (canvasError as any)?.status === 404 ||
        (canvasError as any)?.response?.status === 404 ||
        (canvasError as any)?.code === "NOT_FOUND" ||
        (canvasError as any)?.message?.includes("not found") ||
        (canvasError as any)?.message?.includes("404");

      if (is404 && organizationId && !canvasDeletedRemotely) {
        navigate(`/${organizationId}`, { replace: true });
      }
    }
  }, [canvasError, canvasLoading, navigate, organizationId, canvasDeletedRemotely]);

  // Initialize store from workflow.status on workflow load (only once per workflow)
  const hasInitializedStoreRef = useRef<string | null>(null);
  useEffect(() => {
    if (canvas?.metadata?.id && hasInitializedStoreRef.current !== canvas.metadata.id) {
      initializeFromWorkflow(canvas);
      hasInitializedStoreRef.current = canvas.metadata.id;
    }
  }, [canvas, initializeFromWorkflow]);

  useEffect(() => {
    if (!canvas) {
      return;
    }

    if (!lastSavedWorkflowRef.current) {
      lastSavedWorkflowRef.current = JSON.parse(JSON.stringify(canvas));
    }
  }, [canvas]);

  useEffect(() => {
    setHasUnsavedChanges(false);
    setHasNonPositionalUnsavedChanges(false);
    setInitialWorkflowSnapshot(null);
    lastSavedWorkflowRef.current = null;
    lastLocalCanvasSaveAtRef.current = 0;
  }, [canvasId]);

  useEffect(() => {
    if (isTemplate) {
      setHasUnsavedChanges(false);
      setHasNonPositionalUnsavedChanges(false);
    }
  }, [isTemplate, canvasId]);

  useEffect(() => {
    if (!remoteCanvasUpdatePending || hasUnsavedChanges || canvasDeletedRemotely || !organizationId || !canvasId) {
      return;
    }

    queryClient.invalidateQueries({ queryKey: canvasKeys.detail(organizationId, canvasId) });
    queryClient.invalidateQueries({ queryKey: canvasKeys.list(organizationId) });
    setRemoteCanvasUpdatePending(false);
    showSuccessToast("Canvas updated from another session");
  }, [remoteCanvasUpdatePending, hasUnsavedChanges, canvasDeletedRemotely, organizationId, canvasId, queryClient]);

  // Build maps from store for canvas display (using initial data from workflow.status and websocket updates)
  // Rebuild whenever store version changes (indicates data was updated)
  const { nodeExecutionsMap, nodeQueueItemsMap, nodeEventsMap } = useMemo<{
    nodeExecutionsMap: Record<string, CanvasesCanvasNodeExecution[]>;
    nodeQueueItemsMap: Record<string, CanvasesCanvasNodeQueueItem[]>;
    nodeEventsMap: Record<string, CanvasesCanvasEvent[]>;
  }>(() => {
    const executionsMap: Record<string, CanvasesCanvasNodeExecution[]> = {};
    const queueItemsMap: Record<string, CanvasesCanvasNodeQueueItem[]> = {};
    const eventsMap: Record<string, CanvasesCanvasEvent[]> = {};

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

  const approvalGroupNames = useMemo(() => {
    if (!organizationId) return [];

    const groupNames = new Set<string>();
    Object.values(nodeExecutionsMap).forEach((executions) => {
      executions.forEach((execution) => {
        const metadata = execution.metadata as { records?: Array<{ type?: string; group?: string }> } | undefined;
        const records = metadata?.records;
        if (!Array.isArray(records)) return;

        records.forEach((record) => {
          if (record.type === "group" && record.group) {
            groupNames.add(record.group);
          }
        });
      });
    });

    return Array.from(groupNames);
  }, [organizationId, nodeExecutionsMap]);

  const groupUsersUpdatedAt = useApprovalGroupUsersPrefetch({
    organizationId,
    groupNames: approvalGroupNames,
  }).updatedAt;

  // Execution chain data utilities for lazy loading
  const { loadExecutionChain } = useExecutionChainData(canvasId!, queryClient, canvas);

  const saveWorkflowSnapshot = useCallback(
    (currentWorkflow: CanvasesCanvas) => {
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
    if (initialWorkflowSnapshot && organizationId && canvasId) {
      // Restore the initial state
      queryClient.setQueryData(canvasKeys.detail(organizationId, canvasId), initialWorkflowSnapshot);

      // Clear the snapshot since we're back to the initial state
      setInitialWorkflowSnapshot(null);

      // Mark as no unsaved changes since we're back to the saved state
      setHasUnsavedChanges(false);
      setHasNonPositionalUnsavedChanges(false);
    }
  }, [initialWorkflowSnapshot, organizationId, canvasId, queryClient]);

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
   * Waits 100ms after the last position change when auto-save is enabled,
   * or 5s when auto-save is disabled to avoid disrupting editing.
   * Only saves position changes, not structural modifications (deletions, additions, etc).
   * If there are unsaved structural changes, position auto-save is skipped.
   */
  const debouncedAutoSave = useMemo(
    () =>
      debounce(
        async () => {
          if (!organizationId || !canvasId) return;

          const positionUpdates = new Map(pendingPositionUpdatesRef.current);
          if (positionUpdates.size === 0) return;
          const focusedNoteId = getActiveNoteId();

          try {
            if (!canAutoSave) {
              return;
            }

            // Check if there are unsaved structural changes
            // If so, skip auto-save to avoid saving those changes accidentally
            if (hasNonPositionalUnsavedChanges) {
              return;
            }

            if (isTemplate) {
              return;
            }

            // Fetch the latest workflow from the cache
            const latestWorkflow = queryClient.getQueryData<CanvasesCanvas>(
              canvasKeys.detail(organizationId, canvasId),
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
            lastLocalCanvasSaveAtRef.current = Date.now();
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
            const currentWorkflow = queryClient.getQueryData<CanvasesCanvas>(
              canvasKeys.detail(organizationId, canvasId),
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

              queryClient.setQueryData(canvasKeys.detail(organizationId, canvasId), {
                ...currentWorkflow,
                spec: {
                  ...currentWorkflow.spec,
                  nodes: mergedNodes,
                },
              });
            }

            // Auto-save completed silently (no toast or state changes)
          } catch (error) {
            console.error("Failed to auto-save", error);
          } finally {
            if (focusedNoteId) {
              requestAnimationFrame(() => {
                restoreActiveNoteFocus();
              });
            }
          }
        },
        canAutoSave ? 100 : 2000,
      ),
    [
      organizationId,
      canvasId,
      updateWorkflowMutation,
      queryClient,
      hasNonPositionalUnsavedChanges,
      canAutoSave,
      isTemplate,
    ],
  );

  const handleNodeWebsocketEvent = useCallback(
    (nodeId: string, event: string) => {
      if (event.includes("event_created")) {
        queryClient.invalidateQueries({
          queryKey: canvasKeys.nodeEventHistory(canvasId!, nodeId),
        });
      }

      if (event.startsWith("execution")) {
        queryClient.invalidateQueries({
          queryKey: canvasKeys.nodeExecutionHistory(canvasId!, nodeId),
        });
      }

      if (event.startsWith("queue_item")) {
        queryClient.invalidateQueries({
          queryKey: canvasKeys.nodeQueueItemHistory(canvasId!, nodeId),
        });
      }
    },
    [queryClient, canvasId],
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
    availableIntegrations.forEach((integration) => {
      if (integration.triggers) {
        merged.push(...integration.triggers);
      }
    });
    return merged;
  }, [triggers, availableIntegrations]);

  const allComponents = useMemo(() => {
    const merged = [...components];
    availableIntegrations.forEach((integration) => {
      if (integration.components) {
        merged.push(...integration.components);
      }
    });
    return merged;
  }, [components, availableIntegrations]);

  const buildingBlocks = useMemo(
    () => buildBuildingBlockCategories(triggers, components, blueprints, availableIntegrations),
    [triggers, components, blueprints, availableIntegrations],
  );

  const { nodes, edges } = useMemo(() => {
    // Don't prepare data until everything is loaded
    if (!canvas || canvasLoading || triggersLoading || blueprintsLoading || componentsLoading || integrationsLoading) {
      return { nodes: [], edges: [] };
    }

    return prepareData(
      canvas,
      allTriggers,
      blueprints,
      allComponents,
      nodeEventsMap,
      nodeExecutionsMap,
      nodeQueueItemsMap,
      canvasId!,
      queryClient,
      organizationId!,
      account ? { id: account.id, email: account.email } : undefined,
    );
  }, [
    canvas,
    allTriggers,
    blueprints,
    allComponents,
    nodeEventsMap,
    nodeExecutionsMap,
    nodeQueueItemsMap,
    groupUsersUpdatedAt,
    canvasId,
    queryClient,
    canvasLoading,
    triggersLoading,
    blueprintsLoading,
    componentsLoading,
    integrationsLoading,
    organizationId,
    account,
  ]);

  const getSidebarData = useCallback(
    (nodeId: string): SidebarData | null => {
      const node = canvas?.spec?.nodes?.find((n) => n.id === nodeId);
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
        canvas?.spec?.nodes || [],
        blueprints,
        allComponents,
        allTriggers,
        executionsMap,
        queueItemsMap,
        eventsMapForSidebar,
        totalHistoryCount,
        totalQueueCount,
        canvasId,
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
      canvas,
      canvasId,
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
      const node = canvas?.spec?.nodes?.find((n) => n.id === nodeId);
      if (!node) return;

      // Set current history node for tracking
      setCurrentHistoryNode({ nodeId, nodeType: node?.type || "TYPE_ACTION" });

      loadNodeDataMethod(canvasId!, nodeId, node.type!, queryClient);
    },
    [canvas, canvasId, queryClient, loadNodeDataMethod],
  );

  const onCancelQueueItem = useOnCancelQueueItemHandler({
    canvasId: canvasId!,
    organizationId,
    canvas,
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
  const [resolvedExecutionIds, setResolvedExecutionIds] = useState<Set<string>>(new Set());
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
    (execution: CanvasesCanvasNodeExecution): LogRunItem => {
      return buildRunItemFromExecution({
        execution,
        nodes: canvas?.spec?.nodes || [],
        onNodeSelect: handleLogRunNodeSelect,
        onExecutionSelect: handleLogRunExecutionSelect,
        event: execution.rootEvent || undefined,
      });
    },
    [handleLogRunExecutionSelect, handleLogRunNodeSelect, canvas?.spec?.nodes],
  );

  const buildLiveRunEntryFromEvent = useCallback(
    (event: CanvasesCanvasEvent, runItems: LogRunItem[] = []): LogEntry => {
      return buildRunEntryFromEvent({
        event,
        nodes: canvas?.spec?.nodes || [],
        runItems,
      });
    },
    [canvas?.spec?.nodes],
  );

  const handleWorkflowEventCreated = useCallback(
    (event: CanvasesCanvasEvent) => {
      if (!event.id) {
        return;
      }

      const nodes = canvas?.spec?.nodes || [];
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
    [buildLiveRunEntryFromEvent, canvas?.spec?.nodes],
  );

  const handleExecutionEvent = useCallback(
    (execution: CanvasesCanvasNodeExecution) => {
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
        const entry = buildLiveRunEntryFromEvent(execution.rootEvent as CanvasesCanvasEvent, runItems);
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

  const handleCanvasLifecycleEvent = useCallback(
    (_payload: { id?: string; canvasId?: string }, eventName: string) => {
      if (eventName === "canvas_deleted") {
        setCanvasDeletedRemotely(true);
        return;
      }

      const isLocalEcho =
        eventName === "canvas_updated" &&
        Date.now() - lastLocalCanvasSaveAtRef.current < LOCAL_CANVAS_UPDATE_SUPPRESSION_MS;
      if (isLocalEcho) {
        return;
      }

      if (eventName === "canvas_updated" && hasUnsavedChanges) {
        setRemoteCanvasUpdatePending(true);
      } else if (eventName === "canvas_updated") {
        showSuccessToast("Canvas updated from another session");
      }
    },
    [hasUnsavedChanges],
  );

  const shouldApplyCanvasUpdate = useCallback(
    () => !hasUnsavedChanges && !canvasDeletedRemotely,
    [hasUnsavedChanges, canvasDeletedRemotely],
  );

  useCanvasWebsocket(
    canvasId!,
    organizationId!,
    handleNodeWebsocketEvent,
    handleWorkflowEventCreated,
    handleExecutionEvent,
    handleCanvasLifecycleEvent,
    shouldApplyCanvasUpdate,
  );

  const logEntries = useMemo(() => {
    const nodes = canvas?.spec?.nodes || [];
    const rootEvents = canvasEventsResponse?.events || [];

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
      workflowUpdatedAt: canvas?.metadata?.updatedAt || "",
      onNodeSelect: handleLogNodeSelect,
    });

    const allCanvasEntries = [...liveCanvasEntries, ...canvasEntries];

    const resolvedEntries = [...allRunEntries, ...allCanvasEntries].map((entry) => {
      if (!entry.runItems?.length || resolvedExecutionIds.size === 0) {
        return entry;
      }

      const runItems = entry.runItems.map((item) => {
        if (!resolvedExecutionIds.has(item.id)) {
          return item;
        }
        return {
          ...item,
          type: "resolved-error" as const,
        };
      });

      return {
        ...entry,
        runItems,
      };
    });

    return resolvedEntries.sort((a, b) => {
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
    resolvedExecutionIds,
    canvas?.metadata?.updatedAt,
    canvas?.spec?.nodes,
    canvasEventsResponse?.events,
  ]);

  const nodeHistoryQuery = useNodeHistory({
    canvasId: canvasId || "",
    nodeId: currentHistoryNode?.nodeId || "",
    nodeType: currentHistoryNode?.nodeType || "TYPE_ACTION",
    allNodes: canvas?.spec?.nodes || [],
    enabled: !!currentHistoryNode && !!canvasId,
    components,
    organizationId: organizationId || "",
    queryClient,
  });

  const queueHistoryQuery = useQueueHistory({
    canvasId: canvasId || "",
    nodeId: currentHistoryNode?.nodeId || "",
    allNodes: canvas?.spec?.nodes || [],
    enabled: !!currentHistoryNode && !!canvasId,
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
        workflowNodes: canvas?.spec?.nodes || [],
        nodeEventsMap,
        nodeExecutionsMap,
        nodeQueueItemsMap,
      });
    },
    [canvas, nodeExecutionsMap, nodeEventsMap, nodeQueueItemsMap],
  );

  const getAutocompleteExampleObj = useCallback(
    (nodeId: string): Record<string, unknown> | null => {
      const workflowNodes = canvas?.spec?.nodes || [];
      const workflowEdges = canvas?.spec?.edges || [];

      const currentNode = workflowNodes.find((node) => node.id === nodeId);
      const chainNodeIds = new Set<string>();

      if (currentNode?.type === "TYPE_TRIGGER") {
        chainNodeIds.add(nodeId);
      }

      const stack = workflowEdges
        .filter((edge) => edge.targetId === nodeId && edge.sourceId)
        .map((edge) => edge.sourceId as string);

      while (stack.length > 0) {
        const nextId = stack.pop();
        if (!nextId || chainNodeIds.has(nextId)) continue;
        chainNodeIds.add(nextId);
        workflowEdges
          .filter((edge) => edge.targetId === nextId && edge.sourceId)
          .forEach((edge) => {
            if (edge.sourceId) {
              stack.push(edge.sourceId);
            }
          });
      }

      if (chainNodeIds.size === 0) {
        return null;
      }

      const exampleObj: Record<string, unknown> = {};
      const nodeMetadata: Record<string, { name?: string; componentType: string; description?: string }> = {};
      const nodeNamesById: Record<string, string> = {};

      chainNodeIds.forEach((chainNodeId) => {
        const chainNode = workflowNodes.find((node) => node.id === chainNodeId);
        if (!chainNode) return;

        const nodeName = (chainNode.name || "").trim();
        if (nodeName) {
          nodeNamesById[chainNodeId] = nodeName;
        }

        if (chainNode.type === "TYPE_TRIGGER") {
          const triggerMetadata = allTriggers.find((trigger) => trigger.name === chainNode.trigger?.name);

          // Store node metadata with trigger info
          nodeMetadata[chainNodeId] = {
            name: nodeName || undefined,
            componentType: triggerMetadata?.label || "Trigger",
            description: triggerMetadata?.description,
          };

          const latestEvent = nodeEventsMap[chainNodeId]?.[0];
          if (latestEvent?.data) {
            exampleObj[chainNodeId] = { ...(latestEvent.data || {}) } as Record<string, unknown>;
          }
          if (exampleObj[chainNodeId]) {
            return;
          }

          const exampleData = triggerMetadata?.exampleData;
          if (exampleData && typeof exampleData === "object") {
            exampleObj[chainNodeId] = exampleData as Record<string, unknown>;
          }
          return;
        }

        // For components (non-triggers)
        const componentMetadata = allComponents.find((component) => component.name === chainNode.component?.name);

        // Store node metadata with component info
        nodeMetadata[chainNodeId] = {
          name: nodeName || undefined,
          componentType: componentMetadata?.label || "Component",
          description: componentMetadata?.description,
        };

        const latestExecution = nodeExecutionsMap[chainNodeId]?.find(
          (execution) => execution.state === "STATE_FINISHED" && execution.resultReason !== "RESULT_REASON_ERROR",
        );
        if (!latestExecution?.outputs) {
          const exampleOutput = componentMetadata?.exampleOutput;
          if (exampleOutput && typeof exampleOutput === "object") {
            exampleObj[chainNodeId] = exampleOutput as Record<string, unknown>;
          }
          return;
        }

        const outputData: unknown[] = Object.values(latestExecution.outputs)?.find((output) => {
          return Array.isArray(output) && output.length > 0;
        }) as unknown[];

        if (outputData?.length > 0) {
          exampleObj[chainNodeId] = { ...(outputData?.[0] || {}) } as Record<string, unknown>;
          return;
        }

        const exampleOutput = componentMetadata?.exampleOutput;
        if (exampleOutput && typeof exampleOutput === "object" && Object.keys(exampleOutput).length > 0) {
          exampleObj[chainNodeId] = { ...exampleOutput } as Record<string, unknown>;
        }
      });

      if (!exampleObj) {
        return null;
      }

      const getIncomingNodes = (targetId: string): string[] => {
        return workflowEdges
          .filter((edge) => edge.targetId === targetId && edge.sourceId)
          .map((edge) => edge.sourceId as string);
      };

      const previousByDepth: Record<string, unknown> = {};
      let frontier = [nodeId];
      const visited = new Set<string>([nodeId]);
      let depth = 0;

      while (frontier.length > 0) {
        const next: string[] = [];
        frontier.forEach((current) => {
          getIncomingNodes(current).forEach((sourceId) => {
            if (visited.has(sourceId)) return;
            visited.add(sourceId);
            next.push(sourceId);
          });
        });

        if (next.length === 0) {
          break;
        }

        depth += 1;
        const firstAtDepth = next[0];
        if (firstAtDepth && exampleObj[firstAtDepth]) {
          previousByDepth[String(depth)] = exampleObj[firstAtDepth];
        }

        frontier = next;
      }

      const rootNodeId = workflowNodes.find((node) => {
        if (!node.id || !chainNodeIds.has(node.id)) return false;
        return !workflowEdges.some(
          (edge) => edge.targetId === node.id && edge.sourceId && chainNodeIds.has(edge.sourceId as string),
        );
      })?.id;

      if (rootNodeId && exampleObj[rootNodeId]) {
        exampleObj.__root = exampleObj[rootNodeId];
      }

      if (Object.keys(previousByDepth).length > 0) {
        exampleObj.__previousByDepth = previousByDepth;
      }

      // Build name -> nodeId map, keeping the FIRST (closest) node when names are duplicated
      // chainNodeIds is ordered from closest to farthest, so the first occurrence wins
      const nameToNodeId = new Map<string, string>();
      for (const [nId, nodeName] of Object.entries(nodeNamesById)) {
        if (!nodeName || nodeName === "__nodeNames") {
          continue;
        }

        // Only add if we haven't seen this name yet (keep the closest one)
        if (!nameToNodeId.has(nodeName)) {
          nameToNodeId.set(nodeName, nId);
        }
      }

      const namedExampleObj: Record<string, unknown> = {};
      for (const [nodeName, nodeId] of nameToNodeId.entries()) {
        if (nodeName === nodeId || namedExampleObj[nodeName] !== undefined) {
          continue;
        }

        const value = exampleObj[nodeId];
        if (value === undefined) {
          continue;
        }

        namedExampleObj[nodeName] = value;
      }

      if (Object.keys(namedExampleObj).length === 0) {
        return null;
      }

      if (exampleObj.__root) {
        namedExampleObj.__root = exampleObj.__root;
      }

      if (exampleObj.__previousByDepth) {
        namedExampleObj.__previousByDepth = exampleObj.__previousByDepth;
      }

      // Remove the current node from suggestions - you can't reference your own output
      const currentNodeName = currentNode?.name?.trim();
      const currentNodeId = currentNode?.id;
      if (currentNodeName) {
        delete namedExampleObj[currentNodeName];
      }
      if (currentNodeId) {
        delete nodeMetadata[currentNodeId];
      }

      if (Object.keys(nodeMetadata).length > 0) {
        namedExampleObj.__nodeNames = nodeMetadata;
        Object.entries(nodeMetadata).forEach(([, metadata]) => {
          const value = namedExampleObj[metadata.name ?? ""];
          if (value && typeof value === "object" && !Array.isArray(value)) {
            if (metadata.name) {
              (value as Record<string, unknown>).__nodeName = metadata.name;
            }
          }
        });
      }

      return namedExampleObj;
    },
    [canvas, nodeExecutionsMap, nodeEventsMap, allComponents, allTriggers],
  );

  const handleSaveWorkflow = useCallback(
    async (workflowToSave?: CanvasesCanvas, options?: { showToast?: boolean }) => {
      const targetWorkflow = workflowToSave || canvasRef.current;
      if (!targetWorkflow || !organizationId || !canvasId) return;
      if (!canUpdateCanvas) {
        if (options?.showToast !== false) {
          showErrorToast("You don't have permission to update this canvas");
        }
        return;
      }
      if (isTemplate) {
        if (options?.showToast !== false) {
          showErrorToast("Template canvases are read-only");
        }
        return;
      }
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
        lastLocalCanvasSaveAtRef.current = Date.now();
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
        console.error("Failed to save canvas", error);
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
    [organizationId, canvasId, updateWorkflowMutation, isTemplate, canUpdateCanvas],
  );

  const getNodeEditData = useCallback(
    (nodeId: string): NodeEditData | null => {
      const node = canvas?.spec?.nodes?.find((n) => n.id === nodeId);
      if (!node) return null;

      // Get configuration fields from metadata based on node type
      let configurationFields: ComponentsComponent["configuration"] = [];
      let displayLabel: string | undefined = node.name || undefined;
      let integrationName: string | undefined;
      let blockName: string | undefined;

      if (node.type === "TYPE_BLUEPRINT") {
        const blueprintMetadata = blueprints.find((b) => b.id === node.blueprint?.id);
        configurationFields = blueprintMetadata?.configuration || [];
        displayLabel = blueprintMetadata?.name || displayLabel;
      } else if (node.type === "TYPE_COMPONENT") {
        const componentMetadata = allComponents.find((c) => c.name === node.component?.name);
        configurationFields = componentMetadata?.configuration || [];
        displayLabel = componentMetadata?.label || displayLabel;
        blockName = node.component?.name;

        // Check if this component is from an integration
        const componentIntegration = availableIntegrations.find((integration) =>
          integration.components?.some((c: ComponentsComponent) => c.name === node.component?.name),
        );
        if (componentIntegration) {
          integrationName = componentIntegration.name;
        }
      } else if (node.type === "TYPE_TRIGGER") {
        const triggerMetadata = allTriggers.find((t) => t.name === node.trigger?.name);
        configurationFields = triggerMetadata?.configuration || [];
        displayLabel = triggerMetadata?.label || displayLabel;
        blockName = node.trigger?.name;

        // Check if this trigger is from an application
        const triggerIntegration = availableIntegrations.find((integration) =>
          integration.triggers?.some((t: TriggersTrigger) => t.name === node.trigger?.name),
        );
        if (triggerIntegration) {
          integrationName = triggerIntegration.name;
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
          integrationName,
          blockName,
          integrationRef: node.integration,
        };
      }

      return {
        nodeId: node.id!,
        nodeName: node.name!,
        displayLabel,
        configuration: node.configuration || {},
        configurationFields,
        integrationName,
        blockName,
        integrationRef: node.integration,
      };
    },
    [canvas, blueprints, allComponents, allTriggers, availableIntegrations, widgets],
  );

  const handleNodeConfigurationSave = useCallback(
    async (
      nodeId: string,
      updatedConfiguration: Record<string, any>,
      updatedNodeName: string,
      integrationRef?: ComponentsIntegrationRef,
    ) => {
      if (!canvas || !organizationId || !canvasId) return;

      // Save snapshot before making changes
      saveWorkflowSnapshot(canvas);

      // Update the node's configuration, name, and app installation ref in local cache only
      const updatedNodes = canvas?.spec?.nodes?.map((node) => {
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
            integration: integrationRef,
          };
        }
        return node;
      });

      const updatedWorkflow = {
        ...canvas,
        spec: {
          ...canvas.spec,
          nodes: updatedNodes,
        },
      };

      // Update local cache
      queryClient.setQueryData(canvasKeys.detail(organizationId, canvasId), updatedWorkflow);

      if (canAutoSave) {
        await handleSaveWorkflow(updatedWorkflow, { showToast: false });
      } else {
        markUnsavedChange("structural");
      }
    },
    [
      canvas,
      organizationId,
      canvasId,
      queryClient,
      saveWorkflowSnapshot,
      handleSaveWorkflow,
      canAutoSave,
      markUnsavedChange,
    ],
  );

  const debouncedAnnotationAutoSave = useMemo(
    () =>
      debounce(
        async () => {
          if (!organizationId || !canvasId) return;

          const annotationUpdates = new Map(pendingAnnotationUpdatesRef.current);
          if (annotationUpdates.size === 0) return;

          if (!canAutoSave) {
            return;
          }

          if (hasNonPositionalUnsavedChanges) {
            return;
          }

          if (isTemplate) {
            return;
          }

          const latestWorkflow = queryClient.getQueryData<CanvasesCanvas>(canvasKeys.detail(organizationId, canvasId));

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
        },
        canAutoSave ? 100 : 2000,
      ),
    [
      organizationId,
      canvasId,
      queryClient,
      handleSaveWorkflow,
      hasNonPositionalUnsavedChanges,
      canAutoSave,
      isTemplate,
    ],
  );

  const handleAnnotationBlur = useCallback(() => {
    if (!canAutoSave) {
      return;
    }
  }, [canAutoSave]);

  const handleAnnotationUpdate = useCallback(
    (
      nodeId: string,
      updates: { text?: string; color?: string; width?: number; height?: number; x?: number; y?: number },
    ) => {
      if (!canvas || !organizationId || !canvasId) return;
      if (Object.keys(updates).length === 0) return;

      saveWorkflowSnapshot(canvas);

      const latestWorkflow =
        queryClient.getQueryData<CanvasesCanvas>(canvasKeys.detail(organizationId, canvasId)) || canvas;

      // Separate position updates from configuration updates
      const { x, y, ...configurationUpdates } = updates;
      const hasPositionUpdate = x !== undefined || y !== undefined;
      const hasConfigurationUpdate = Object.keys(configurationUpdates).length > 0;
      const hasOnlyTextUpdate =
        !hasPositionUpdate && Object.keys(configurationUpdates).length === 1 && configurationUpdates.text !== undefined;

      const shouldUpdateCache = !canAutoSave || !hasOnlyTextUpdate;
      if (shouldUpdateCache) {
        const updatedNodes = latestWorkflow?.spec?.nodes?.map((node) => {
          if (node.id !== nodeId || node.type !== "TYPE_WIDGET") {
            return node;
          }

          const updatedNode = { ...node };

          // Update position if provided
          if (hasPositionUpdate) {
            updatedNode.position = {
              x: x !== undefined ? x : node.position?.x || 0,
              y: y !== undefined ? y : node.position?.y || 0,
            };
          }

          // Update configuration if provided
          if (hasConfigurationUpdate) {
            updatedNode.configuration = {
              ...node.configuration,
              ...configurationUpdates,
            };
          }

          return updatedNode;
        });

        const updatedWorkflow = {
          ...latestWorkflow,
          spec: {
            ...latestWorkflow.spec,
            nodes: updatedNodes,
          },
        };

        queryClient.setQueryData(canvasKeys.detail(organizationId, canvasId), updatedWorkflow);
      }

      if (hasConfigurationUpdate) {
        if (canAutoSave) {
          const existing = pendingAnnotationUpdatesRef.current.get(nodeId) || {};
          pendingAnnotationUpdatesRef.current.set(nodeId, { ...existing, ...configurationUpdates });
          debouncedAnnotationAutoSave();
        } else {
          markUnsavedChange("structural");
        }
      }

      if (canAutoSave) {
        // Queue position updates for auto-save
        if (hasPositionUpdate) {
          pendingPositionUpdatesRef.current.set(nodeId, {
            x: x !== undefined ? x : latestWorkflow?.spec?.nodes?.find((n) => n.id === nodeId)?.position?.x || 0,
            y: y !== undefined ? y : latestWorkflow?.spec?.nodes?.find((n) => n.id === nodeId)?.position?.y || 0,
          });
          debouncedAutoSave();
        }
      } else if (hasPositionUpdate) {
        markUnsavedChange("position");
      }
    },
    [
      canvas,
      organizationId,
      canvasId,
      queryClient,
      saveWorkflowSnapshot,
      debouncedAnnotationAutoSave,
      debouncedAutoSave,
      canAutoSave,
      markUnsavedChange,
    ],
  );

  const handleNodeAdd = useCallback(
    async (newNodeData: NewNodeData): Promise<string> => {
      if (!canvas || !organizationId || !canvasId) return "";

      // Save snapshot before making changes
      saveWorkflowSnapshot(canvas);

      const { buildingBlock, configuration, position, sourceConnection, integrationRef } = newNodeData;

      // Filter configuration to only include visible fields
      const filteredConfiguration = filterVisibleConfiguration(configuration, buildingBlock.configuration || []);

      // Get existing node names for unique name generation
      const existingNodeNames = (canvas.spec?.nodes || []).map((n) => n.name || "").filter(Boolean);

      // Generate unique node name based on component name + ordinal
      const uniqueNodeName = generateUniqueNodeName(buildingBlock.name || "node", existingNodeNames);

      // Generate a unique node ID
      const newNodeId = generateNodeId(buildingBlock.name || "node", uniqueNodeName);

      // Create the new node
      const newNode: ComponentsNode = {
        id: newNodeId,
        name: uniqueNodeName,
        type:
          buildingBlock.type === "trigger"
            ? "TYPE_TRIGGER"
            : buildingBlock.type === "blueprint"
              ? "TYPE_BLUEPRINT"
              : buildingBlock.name === "annotation"
                ? "TYPE_WIDGET"
                : "TYPE_COMPONENT",
        configuration: filteredConfiguration,
        integration: integrationRef,
        position: position
          ? {
              x: Math.round(position.x),
              y: Math.round(position.y),
            }
          : {
              x: (canvas?.spec?.nodes?.length || 0) * 250,
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
      const updatedNodes = [...(canvas.spec?.nodes || []), newNode];

      // If there's a source connection, create the edge
      let updatedEdges = canvas.spec?.edges || [];
      if (sourceConnection) {
        const newEdge: ComponentsEdge = {
          sourceId: sourceConnection.nodeId,
          targetId: newNodeId,
          channel: sourceConnection.handleId || "default",
        };
        updatedEdges = [...updatedEdges, newEdge];
      }

      const updatedWorkflow = {
        ...canvas,
        spec: {
          ...canvas.spec,
          nodes: updatedNodes,
          edges: updatedEdges,
        },
      };

      // Update local cache
      queryClient.setQueryData(canvasKeys.detail(organizationId, canvasId), updatedWorkflow);

      if (canAutoSave) {
        await handleSaveWorkflow(updatedWorkflow, { showToast: false });
      } else {
        markUnsavedChange("structural");
      }

      // Return the new node ID
      return newNodeId;
    },
    [
      canvas,
      organizationId,
      canvasId,
      queryClient,
      saveWorkflowSnapshot,
      handleSaveWorkflow,
      canAutoSave,
      markUnsavedChange,
    ],
  );

  const handleApplyAiOperations = useCallback(
    async (operations: AiCanvasOperation[]) => {
      if (!operations.length || !organizationId || !canvasId) {
        return;
      }

      const latestWorkflow =
        queryClient.getQueryData<CanvasesCanvas>(canvasKeys.detail(organizationId, canvasId)) || canvas;
      if (!latestWorkflow) {
        throw new Error("Canvas not found.");
      }

      saveWorkflowSnapshot(latestWorkflow);
      const updatedWorkflow = applyAiOperationsToWorkflow({
        workflow: latestWorkflow,
        operations,
        buildingBlocks,
      });

      queryClient.setQueryData(canvasKeys.detail(organizationId, canvasId), updatedWorkflow);

      if (canAutoSave) {
        await handleSaveWorkflow(updatedWorkflow, { showToast: false });
      } else {
        markUnsavedChange("structural");
      }
    },
    [
      buildingBlocks,
      canAutoSave,
      canvas,
      canvasId,
      handleSaveWorkflow,
      markUnsavedChange,
      organizationId,
      queryClient,
      saveWorkflowSnapshot,
    ],
  );

  const handlePlaceholderAdd = useCallback(
    async (data: {
      position: { x: number; y: number };
      sourceNodeId: string;
      sourceHandleId: string | null;
    }): Promise<string> => {
      if (!canvas || !organizationId || !canvasId) return "";

      saveWorkflowSnapshot(canvas);

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
        ...canvas,
        spec: {
          ...canvas.spec,
          nodes: [...(canvas.spec?.nodes || []), newNode],
          edges: [...(canvas.spec?.edges || []), newEdge],
        },
      };

      queryClient.setQueryData(canvasKeys.detail(organizationId, canvasId), updatedWorkflow);

      if (canAutoSave) {
        await handleSaveWorkflow(updatedWorkflow, { showToast: false });
      } else {
        markUnsavedChange("structural");
      }

      return newNodeId;
    },
    [
      canvas,
      organizationId,
      canvasId,
      queryClient,
      saveWorkflowSnapshot,
      handleSaveWorkflow,
      canAutoSave,
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
      if (!canvas || !organizationId || !canvasId) {
        return;
      }

      saveWorkflowSnapshot(canvas);

      const nodeIndex = canvas.spec?.nodes?.findIndex((n) => n.id === data.placeholderId);
      if (nodeIndex === undefined || nodeIndex === -1) {
        return;
      }

      const filteredConfiguration = filterVisibleConfiguration(
        data.configuration,
        data.buildingBlock.configuration || [],
      );

      // Get existing node names for unique name generation (exclude the placeholder being configured)
      const existingNodeNames = (canvas.spec?.nodes || [])
        .filter((n) => n.id !== data.placeholderId)
        .map((n) => n.name || "")
        .filter(Boolean);

      // Generate unique node name based on component name + ordinal
      const uniqueNodeName = generateUniqueNodeName(data.buildingBlock.name || "node", existingNodeNames);

      // Update placeholder with real component data
      const updatedNode: ComponentsNode = {
        ...canvas.spec!.nodes![nodeIndex],
        name: uniqueNodeName,
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

      const updatedNodes = [...(canvas.spec?.nodes || [])];
      updatedNodes[nodeIndex] = updatedNode;

      // Update outgoing edges from this node to use valid channels
      // Find edges where this node is the source
      const outgoingEdges = canvas.spec?.edges?.filter((edge) => edge.sourceId === data.placeholderId) || [];

      let updatedEdges = [...(canvas.spec?.edges || [])];

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
        ...canvas,
        spec: {
          ...canvas.spec,
          nodes: updatedNodes,
          edges: updatedEdges,
        },
      };

      queryClient.setQueryData(canvasKeys.detail(organizationId, canvasId), updatedWorkflow);

      if (canAutoSave) {
        await handleSaveWorkflow(updatedWorkflow, { showToast: false });
      } else {
        markUnsavedChange("structural");
      }
    },
    [
      canvas,
      organizationId,
      canvasId,
      queryClient,
      saveWorkflowSnapshot,
      handleSaveWorkflow,
      canAutoSave,
      markUnsavedChange,
    ],
  );

  const handleEdgeCreate = useCallback(
    async (sourceId: string, targetId: string, sourceHandle?: string | null) => {
      if (!canvas || !organizationId || !canvasId) return;

      // Save snapshot before making changes
      saveWorkflowSnapshot(canvas);

      // Create the new edge
      const newEdge: ComponentsEdge = {
        sourceId,
        targetId,
        channel: sourceHandle || "default",
      };

      // Add the new edge to the workflow
      const updatedEdges = [...(canvas.spec?.edges || []), newEdge];

      const updatedWorkflow = {
        ...canvas,
        spec: {
          ...canvas.spec,
          edges: updatedEdges,
        },
      };

      // Update local cache
      queryClient.setQueryData(canvasKeys.detail(organizationId, canvasId), updatedWorkflow);

      if (canAutoSave) {
        await handleSaveWorkflow(updatedWorkflow, { showToast: false });
      } else {
        markUnsavedChange("structural");
      }
    },
    [
      canvas,
      organizationId,
      canvasId,
      queryClient,
      saveWorkflowSnapshot,
      handleSaveWorkflow,
      canAutoSave,
      markUnsavedChange,
    ],
  );

  const handleNodeDelete = useCallback(
    async (nodeId: string) => {
      if (!canvas || !organizationId || !canvasId) return;

      // Save snapshot before making changes
      saveWorkflowSnapshot(canvas);

      // Remove the node from the workflow
      const updatedNodes = canvas.spec?.nodes?.filter((node) => node.id !== nodeId);

      // Remove any edges connected to this node
      const updatedEdges = canvas.spec?.edges?.filter((edge) => edge.sourceId !== nodeId && edge.targetId !== nodeId);

      const updatedWorkflow = {
        ...canvas,
        spec: {
          ...canvas.spec,
          nodes: updatedNodes,
          edges: updatedEdges,
        },
      };

      // Update local cache
      queryClient.setQueryData(canvasKeys.detail(organizationId, canvasId), updatedWorkflow);

      if (canAutoSave) {
        await handleSaveWorkflow(updatedWorkflow, { showToast: false });
      } else {
        markUnsavedChange("structural");
      }
    },
    [
      canvas,
      organizationId,
      canvasId,
      queryClient,
      saveWorkflowSnapshot,
      handleSaveWorkflow,
      canAutoSave,
      markUnsavedChange,
    ],
  );

  const handleEdgeDelete = useCallback(
    async (edgeIds: string[]) => {
      if (!canvas || !organizationId || !canvasId) return;

      // Save snapshot before making changes
      saveWorkflowSnapshot(canvas);

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
      const updatedEdges = canvas.spec?.edges?.filter((edge) => {
        return !edgesToRemove.some(
          (toRemove) =>
            edge.sourceId === toRemove.sourceId &&
            edge.targetId === toRemove.targetId &&
            edge.channel === toRemove.channel,
        );
      });

      const updatedWorkflow = {
        ...canvas,
        spec: {
          ...canvas.spec,
          edges: updatedEdges,
        },
      };

      // Update local cache
      queryClient.setQueryData(canvasKeys.detail(organizationId, canvasId), updatedWorkflow);

      if (canAutoSave) {
        await handleSaveWorkflow(updatedWorkflow, { showToast: false });
      } else {
        markUnsavedChange("structural");
      }
    },
    [
      canvas,
      organizationId,
      canvasId,
      queryClient,
      saveWorkflowSnapshot,
      handleSaveWorkflow,
      canAutoSave,
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
      if (!canvas || !organizationId || !canvasId) return;

      const roundedPosition = {
        x: Math.round(position.x),
        y: Math.round(position.y),
      };

      const updatedNodes = canvas.spec?.nodes?.map((node) =>
        node.id === nodeId
          ? {
              ...node,
              position: roundedPosition,
            }
          : node,
      );

      const updatedWorkflow = {
        ...canvas,
        spec: {
          ...canvas.spec,
          nodes: updatedNodes,
        },
      };

      queryClient.setQueryData(canvasKeys.detail(organizationId, canvasId), updatedWorkflow);

      if (canAutoSave) {
        pendingPositionUpdatesRef.current.set(nodeId, roundedPosition);

        debouncedAutoSave();
      } else {
        saveWorkflowSnapshot(canvas);
        markUnsavedChange("position");
      }
    },
    [
      canvas,
      organizationId,
      canvasId,
      queryClient,
      debouncedAutoSave,
      canAutoSave,
      saveWorkflowSnapshot,
      markUnsavedChange,
    ],
  );

  const handleNodesPositionChange = useCallback(
    (updates: Array<{ nodeId: string; position: { x: number; y: number } }>) => {
      if (!canvas || !organizationId || !canvasId || updates.length === 0) return;

      // Create a map of nodeId -> rounded position for efficient lookup
      const positionMap = new Map(
        updates.map((update) => [
          update.nodeId,
          {
            x: Math.round(update.position.x),
            y: Math.round(update.position.y),
          },
        ]),
      );

      // Update all nodes in a single operation
      const updatedNodes = canvas.spec?.nodes?.map((node) =>
        node.id && positionMap.has(node.id)
          ? {
              ...node,
              position: positionMap.get(node.id)!,
            }
          : node,
      );

      const updatedWorkflow = {
        ...canvas,
        spec: {
          ...canvas.spec,
          nodes: updatedNodes,
        },
      };

      queryClient.setQueryData(canvasKeys.detail(organizationId, canvasId), updatedWorkflow);

      if (canAutoSave) {
        // Add all position updates to pending updates
        positionMap.forEach((position, nodeId) => {
          pendingPositionUpdatesRef.current.set(nodeId, position);
        });

        debouncedAutoSave();
      } else {
        saveWorkflowSnapshot(canvas);
        markUnsavedChange("position");
      }
    },
    [
      canvas,
      organizationId,
      canvasId,
      queryClient,
      debouncedAutoSave,
      canAutoSave,
      saveWorkflowSnapshot,
      markUnsavedChange,
    ],
  );

  const handleAutoLayout = useCallback(
    async (selectedNodeIDs: string[] = []) => {
      if (!canvas || !organizationId || !canvasId) return;

      const latestWorkflow =
        queryClient.getQueryData<CanvasesCanvas>(canvasKeys.detail(organizationId, canvasId)) || canvas;

      try {
        if (!canAutoSave) {
          saveWorkflowSnapshot(latestWorkflow);
        }

        const updatedWorkflow = await applyHorizontalAutoLayout(latestWorkflow, {
          nodeIds: selectedNodeIDs,
        });
        queryClient.setQueryData(canvasKeys.detail(organizationId, canvasId), updatedWorkflow);

        if (canAutoSave) {
          await handleSaveWorkflow(updatedWorkflow, { showToast: false });
        } else {
          markUnsavedChange("position");
        }
      } catch (error) {
        console.error("Failed to auto layout canvas", error);
        showErrorToast("Failed to auto layout canvas");
      }
    },
    [
      canvas,
      organizationId,
      canvasId,
      queryClient,
      canAutoSave,
      saveWorkflowSnapshot,
      handleSaveWorkflow,
      markUnsavedChange,
    ],
  );

  const handleNodeCollapseChange = useCallback(
    async (nodeId: string) => {
      if (!canvas || !organizationId || !canvasId) return;

      // Save snapshot before making changes
      saveWorkflowSnapshot(canvas);

      // Find the current node to determine its collapsed state
      const currentNode = canvas.spec?.nodes?.find((node) => node.id === nodeId);
      if (!currentNode) return;

      // Toggle the collapsed state
      const newIsCollapsed = !currentNode.isCollapsed;

      const updatedNodes = canvas.spec?.nodes?.map((node) =>
        node.id === nodeId
          ? {
              ...node,
              isCollapsed: newIsCollapsed,
            }
          : node,
      );

      const updatedWorkflow = {
        ...canvas,
        spec: {
          ...canvas.spec,
          nodes: updatedNodes,
        },
      };

      queryClient.setQueryData(canvasKeys.detail(organizationId, canvasId), updatedWorkflow);

      if (canAutoSave) {
        await handleSaveWorkflow(updatedWorkflow, { showToast: false });
      } else {
        markUnsavedChange("structural");
      }
    },
    [
      canvas,
      organizationId,
      canvasId,
      queryClient,
      saveWorkflowSnapshot,
      handleSaveWorkflow,
      canAutoSave,
      markUnsavedChange,
    ],
  );

  const handleConfigure = useCallback(
    (nodeId: string) => {
      const node = canvas?.spec?.nodes?.find((n) => n.id === nodeId);
      if (!node) return;
      if (node.type === "TYPE_BLUEPRINT" && node.blueprint?.id && organizationId && canvas) {
        // Pass workflow info as URL parameters
        const params = new URLSearchParams({
          fromWorkflow: canvasId!,
          workflowName: canvas.metadata?.name || "Canvas",
        });
        navigate(`/${organizationId}/custom-components/${node.blueprint.id}?${params.toString()}`);
      }
    },
    [canvas, organizationId, canvasId, navigate],
  );

  const handleRun = useCallback(
    async (nodeId: string, channel: string, data: any) => {
      if (!canvasId) return;

      try {
        await canvasesEmitNodeEvent(
          withOrganizationHeader({
            path: {
              canvasId: canvasId,
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
        showErrorToast("Failed to emit event");
        throw error; // Re-throw to let EmitEventModal handle it
      }
    },
    [canvasId],
  );

  const handleTogglePause = useCallback(
    async (nodeId: string) => {
      if (!canvasId || !organizationId || !canvas) return;

      const node = canvas.spec?.nodes?.find((n) => n.id === nodeId);
      if (!node) return;

      if (node.type === "TYPE_TRIGGER") {
        showErrorToast("Triggers cannot be paused");
        return;
      }

      const nextPaused = !node.paused;

      try {
        const result = await canvasesUpdateNodePause(
          withOrganizationHeader({
            path: {
              canvasId: canvasId,
              nodeId: nodeId,
            },
            body: {
              paused: nextPaused,
            },
          }),
        );

        const updatedPaused = result.data?.node?.paused ?? nextPaused;
        const updatedNodes = (canvas.spec?.nodes || []).map((item) =>
          item.id === nodeId ? { ...item, paused: updatedPaused } : item,
        );

        const updatedWorkflow = {
          ...canvas,
          spec: {
            ...canvas.spec,
            nodes: updatedNodes,
          },
        };

        queryClient.setQueryData(canvasKeys.detail(organizationId, canvasId), updatedWorkflow);
        showSuccessToast(updatedPaused ? "Component paused" : "Component resumed");
      } catch (error) {
        let parsedError = error as { message: string };
        if (parsedError?.message) {
          showErrorToast(parsedError.message);
        } else {
          console.error("Failed to update node pause state:", error);
        }
      }
    },
    [canvasId, organizationId, canvas, queryClient],
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
      if (!canvas || !organizationId || !canvasId) return;

      const nodeToDuplicate = canvas.spec?.nodes?.find((node) => node.id === nodeId);
      if (!nodeToDuplicate) return;

      saveWorkflowSnapshot(canvas);

      const existingNodeNames = (canvas.spec?.nodes || []).map((n) => n.name || "").filter(Boolean);

      let baseName = nodeToDuplicate.name?.trim() || "";
      if (!baseName) {
        if (nodeToDuplicate.type === "TYPE_TRIGGER" && nodeToDuplicate.trigger?.name) {
          baseName = nodeToDuplicate.trigger.name;
        } else if (nodeToDuplicate.type === "TYPE_COMPONENT" && nodeToDuplicate.component?.name) {
          baseName = nodeToDuplicate.component.name;
        } else if (nodeToDuplicate.type === "TYPE_BLUEPRINT" && nodeToDuplicate.blueprint?.id) {
          // For blueprints, we need to find the blueprint metadata to get the name
          const blueprintMetadata = blueprints.find((b) => b.id === nodeToDuplicate.blueprint?.id);
          baseName = blueprintMetadata?.name || "blueprint";
        } else {
          baseName = "node";
        }
      }

      // Generate unique node name based on the existing node name + ordinal
      const uniqueNodeName = generateUniqueNodeName(baseName, existingNodeNames);

      const newNodeId = generateNodeId(baseName, uniqueNodeName);

      const offsetX = 50;
      const offsetY = 50;

      const duplicateNode: ComponentsNode = {
        ...nodeToDuplicate,
        id: newNodeId,
        name: uniqueNodeName,
        position: {
          x: (nodeToDuplicate.position?.x || 0) + offsetX,
          y: (nodeToDuplicate.position?.y || 0) + offsetY,
        },
        // Reset collapsed state for the duplicate
        isCollapsed: false,
      };

      // Add the duplicate node to the workflow
      const updatedNodes = [...(canvas.spec?.nodes || []), duplicateNode];

      const updatedWorkflow = {
        ...canvas,
        spec: {
          ...canvas.spec,
          nodes: updatedNodes,
        },
      };

      // Update local cache
      queryClient.setQueryData(canvasKeys.detail(organizationId, canvasId), updatedWorkflow);
      if (canAutoSave) {
        await handleSaveWorkflow(updatedWorkflow, { showToast: false });
      } else {
        markUnsavedChange("structural");
      }
    },
    [
      canvas,
      organizationId,
      canvasId,
      blueprints,
      queryClient,
      saveWorkflowSnapshot,
      handleSaveWorkflow,
      canAutoSave,
      markUnsavedChange,
    ],
  );

  const handleSave = useCallback(
    async (canvasNodes: CanvasNode[]) => {
      if (!canvas || !organizationId || !canvasId) return;
      if (isTemplate) {
        showErrorToast("Template canvases are read-only");
        return;
      }

      // Map canvas nodes back to ComponentsNode format with updated positions
      const updatedNodes = canvas.spec?.nodes?.map((node) => {
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
        ...canvas,
        spec: {
          ...canvas.spec,
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
        lastLocalCanvasSaveAtRef.current = Date.now();
        await updateWorkflowMutation.mutateAsync({
          name: canvas.metadata?.name!,
          description: canvas.metadata?.description,
          nodes: updatedNodes,
          edges: canvas.spec?.edges,
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
      } catch (error) {
        console.error("Failed to save canvas", error);
        const errorMessage =
          (error as { response?: { data?: { message: string } } })?.response?.data?.message ||
          (error as { message: string })?.message ||
          "Failed to save changes to the canvas";
        showErrorToast(errorMessage);
      }
    },
    [canvas, organizationId, canvasId, updateWorkflowMutation, isTemplate],
  );

  const getYamlExportPayload = useCallback(
    (canvasNodes: CanvasNode[]) => {
      if (!canvas) return null;

      const updatedNodes =
        canvas.spec?.nodes?.map((node) => {
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
        }) || [];

      const exportWorkflow = {
        metadata: {
          name: canvas.metadata?.name || "Canvas",
          description: canvas.metadata?.description || "",
          isTemplate: canvas.metadata?.isTemplate ?? false,
        },
        spec: {
          nodes: updatedNodes,
          edges: canvas.spec?.edges || [],
        },
      };

      const yamlText = yaml.dump(exportWorkflow, {
        forceQuotes: true,
        quotingType: '"',
        lineWidth: 0,
      });

      const safeName = (canvas.metadata?.name || "canvas")
        .toLowerCase()
        .replace(/[^a-z0-9]+/g, "-")
        .replace(/(^-|-$)/g, "");
      const filename = `${safeName || "canvas"}.yaml`;

      return { yamlText, filename };
    },
    [canvas],
  );

  const handleExportYamlDownload = useCallback(
    (canvasNodes: CanvasNode[]) => {
      const payload = getYamlExportPayload(canvasNodes);
      if (!payload) return;

      const blob = new Blob([payload.yamlText], { type: "text/yaml;charset=utf-8" });
      const url = URL.createObjectURL(blob);

      const link = document.createElement("a");
      link.href = url;
      link.download = payload.filename;
      document.body.appendChild(link);
      link.click();
      link.remove();
      URL.revokeObjectURL(url);

      showSuccessToast("Canvas exported as YAML");
    },
    [getYamlExportPayload],
  );

  const handleExportYamlCopy = useCallback(
    async (canvasNodes: CanvasNode[]) => {
      const payload = getYamlExportPayload(canvasNodes);
      if (!payload) return;

      try {
        await navigator.clipboard.writeText(payload.yamlText);
        showSuccessToast("YAML copied to clipboard");
      } catch (_error) {
        showErrorToast("Failed to copy YAML to clipboard");
      }
    },
    [getYamlExportPayload],
  );

  const handleUseTemplateSubmit = useCallback(
    async (data: { name: string; description?: string; templateId?: string }) => {
      if (!canvas || !organizationId) return;

      const latestWorkflow =
        queryClient.getQueryData<CanvasesCanvas>(canvasKeys.detail(organizationId, canvasId!)) || canvas;

      const result = await createWorkflowMutation.mutateAsync({
        name: data.name,
        description: data.description,
        nodes: latestWorkflow.spec?.nodes,
        edges: latestWorkflow.spec?.edges,
      });

      if (result?.data?.canvas?.metadata?.id) {
        setIsUseTemplateOpen(false);
        navigate(`/${organizationId}/canvases/${result.data.canvas.metadata.id}`);
      }
    },
    [canvas, organizationId, createWorkflowMutation, navigate, queryClient, canvasId],
  );

  // Provide pass-through handlers regardless of workflow being loaded to keep hook order stable
  const { onPushThrough, supportsPushThrough } = usePushThroughHandler({
    canvasId: canvasId!,
    organizationId,
    canvas,
  });

  const { onCancelExecution } = useCancelExecutionHandler({
    canvasId: canvasId!,
    canvas,
  });

  const [isResolvingErrors, setIsResolvingErrors] = useState(false);

  const handleResolveExecutionErrors = useCallback(
    async (executionIds: string[]) => {
      if (!canvasId || executionIds.length === 0 || isResolvingErrors) {
        return;
      }

      setIsResolvingErrors(true);
      try {
        await resolveExecutionErrors(canvasId, executionIds);
        setResolvedExecutionIds((prev) => {
          const next = new Set(prev);
          executionIds.forEach((id) => next.add(id));
          return next;
        });
        await queryClient.invalidateQueries({ queryKey: canvasKeys.eventList(canvasId, 50) });
        await queryClient.invalidateQueries({ queryKey: canvasKeys.nodeExecutions() });
        showSuccessToast("Execution errors resolved");
      } catch (_error) {
        showErrorToast("Failed to resolve execution errors");
      } finally {
        setIsResolvingErrors(false);
      }
    },
    [canvasId, isResolvingErrors, queryClient],
  );

  // Provide state function based on component type
  const getExecutionState = useCallback(
    (nodeId: string, execution: CanvasesCanvasNodeExecution): { map: EventStateMap; state: EventState } => {
      const node = canvas?.spec?.nodes?.find((n) => n.id === nodeId);
      if (!node) {
        return {
          map: getStateMap("default"),
          state: getState("default")(buildExecutionInfo(execution)),
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
        state: getState(componentName)(buildExecutionInfo(execution)),
      };
    },
    [canvas],
  );

  const getCustomField = useCallback(
    (nodeId: string, onRun?: (initialData?: string) => void, integration?: OrganizationsIntegration) => {
      const node = canvas?.spec?.nodes?.find((n) => n.id === nodeId);
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

      const context: {
        onRun?: (initialData?: string) => void;
        integration?: OrganizationsIntegration;
      } = onRun ? { onRun } : {};
      if (integration) context.integration = integration;

      // Return a function that takes the current configuration
      return (configuration?: Record<string, unknown>) => {
        const nodeWithConfiguration = {
          ...node,
          configuration: configuration ?? node.configuration,
        };

        return renderer.render(
          buildNodeInfo(nodeWithConfiguration),
          Object.keys(context).length > 0 ? context : undefined,
        );
      };
    },
    [canvas],
  );

  // Show loading indicator while data is being fetched
  if (
    canvasLoading ||
    triggersLoading ||
    blueprintsLoading ||
    componentsLoading ||
    widgetsLoading ||
    usersLoading ||
    rolesLoading ||
    groupsLoading
  ) {
    return (
      <div className="flex items-center justify-center h-screen">
        <div className="flex flex-col items-center gap-3">
          <Loader2 className="h-8 w-8 animate-spin text-gray-500" />
          <p className="text-sm text-gray-500">Loading canvas...</p>
        </div>
      </div>
    );
  }

  if (!canvas && !canvasLoading) {
    // Workflow not found after loading - could be deleted or doesn't exist
    // Show a brief message then redirect (handled by the error useEffect above)
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

  const handleReloadRemoteCanvas = async () => {
    if (!organizationId || !canvasId) {
      return;
    }

    pendingPositionUpdatesRef.current.clear();
    pendingAnnotationUpdatesRef.current.clear();
    setHasUnsavedChanges(false);
    setHasNonPositionalUnsavedChanges(false);
    setInitialWorkflowSnapshot(null);
    setRemoteCanvasUpdatePending(false);
    lastSavedWorkflowRef.current = null;

    await queryClient.invalidateQueries({ queryKey: canvasKeys.detail(organizationId, canvasId) });
    await queryClient.invalidateQueries({ queryKey: canvasKeys.list(organizationId) });
  };

  const hasRunBlockingChanges = hasUnsavedChanges && hasNonPositionalUnsavedChanges;
  const remoteUpdateBanner = remoteCanvasUpdatePending ? (
    <div className="bg-amber-100 px-4 py-2.5 flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
      <div>
        <p className="text-sm font-medium text-gray-900">Canvas updated elsewhere</p>
        <p className="text-[13px] text-black/60">
          A newer canvas version is available. Reloading will discard your unsaved local changes.
        </p>
      </div>
      <div className="flex gap-2">
        <Button size="sm" onClick={handleReloadRemoteCanvas}>
          Reload remote
        </Button>
      </div>
    </div>
  ) : null;
  const templateBanner = isTemplate ? (
    <div className="bg-orange-100 px-4 py-2.5 flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
      <div>
        <p className="text-sm font-medium text-gray-900">Template preview</p>
        <p className="text-[13px] text-black/60">Read-only template. Save your edits to a new canvas.</p>
      </div>
      <Button size="sm" onClick={() => setIsUseTemplateOpen(true)}>
        {hasUnsavedChanges ? "Save changes to new canvas" : "Use template"}
      </Button>
    </div>
  ) : null;
  const headerBanner =
    remoteUpdateBanner && templateBanner ? (
      <div className="flex flex-col">
        {remoteUpdateBanner}
        {templateBanner}
      </div>
    ) : (
      remoteUpdateBanner || templateBanner
    );
  const saveDisabled = !canUpdateCanvas;
  const saveDisabledTooltip = saveDisabled ? "You don't have permission to edit this canvas." : undefined;
  const autoSaveDisabled = !canUpdateCanvas;
  const autoSaveDisabledTooltip = autoSaveDisabled ? "You don't have permission to edit this canvas." : undefined;
  const saveButtonHidden = isTemplate || !canUpdateCanvas || !hasUnsavedChanges;
  const saveIsPrimary = hasUnsavedChanges && !isTemplate && canUpdateCanvas;
  const canUndo = !isTemplate && canUpdateCanvas && !isAutoSaveEnabled && initialWorkflowSnapshot !== null;
  const runDisabled = hasRunBlockingChanges || isTemplate || !canUpdateCanvas || canvasDeletedRemotely;
  const runDisabledTooltip = canvasDeletedRemotely
    ? "This canvas was deleted in another session."
    : !canUpdateCanvas
      ? "You don't have permission to emit events on this canvas."
      : isTemplate
        ? "Templates are read-only"
        : hasRunBlockingChanges
          ? "Save canvas changes before running"
          : undefined;

  return (
    <>
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
            navigate(`/${organizationId}/canvases/${canvasId}/nodes/${nodeId}/${executionId}`);
          }
        }}
        title={canvas?.metadata?.name || "Canvas"}
        headerBanner={headerBanner}
        nodes={nodes}
        edges={edges}
        organizationId={organizationId}
        canvasId={canvasId}
        onDirty={!isReadOnly ? () => markUnsavedChange("structural") : undefined}
        getSidebarData={getSidebarData}
        loadSidebarData={loadSidebarData}
        getTabData={getTabData}
        getNodeEditData={getNodeEditData}
        getAutocompleteExampleObj={getAutocompleteExampleObj}
        getCustomField={getCustomField}
        onNodeConfigurationSave={!isReadOnly ? handleNodeConfigurationSave : undefined}
        onAnnotationUpdate={!isReadOnly ? handleAnnotationUpdate : undefined}
        onAnnotationBlur={!isReadOnly ? handleAnnotationBlur : undefined}
        onSave={isTemplate ? undefined : handleSave}
        onEdgeCreate={!isReadOnly ? handleEdgeCreate : undefined}
        onNodeDelete={!isReadOnly ? handleNodeDelete : undefined}
        onEdgeDelete={!isReadOnly ? handleEdgeDelete : undefined}
        onAutoLayout={!isReadOnly ? handleAutoLayout : undefined}
        onNodePositionChange={!isReadOnly ? handleNodePositionChange : undefined}
        onNodesPositionChange={!isReadOnly ? handleNodesPositionChange : undefined}
        onToggleView={!isReadOnly ? handleNodeCollapseChange : undefined}
        onToggleCollapse={!isReadOnly ? () => markUnsavedChange("structural") : undefined}
        onRun={handleRun}
        onTogglePause={!isReadOnly ? handleTogglePause : undefined}
        onDuplicate={!isReadOnly ? handleNodeDuplicate : undefined}
        onConfigure={!isReadOnly ? handleConfigure : undefined}
        buildingBlocks={buildingBlocks}
        showAiBuilderTab={showAiBuilderTab}
        onNodeAdd={!isReadOnly ? handleNodeAdd : undefined}
        onApplyAiOperations={!isReadOnly ? handleApplyAiOperations : undefined}
        onPlaceholderAdd={!isReadOnly ? handlePlaceholderAdd : undefined}
        onPlaceholderConfigure={!isReadOnly ? handlePlaceholderConfigure : undefined}
        integrations={canReadIntegrations ? integrations : []}
        canReadIntegrations={canReadIntegrations}
        canCreateIntegrations={canCreateIntegrations}
        canUpdateIntegrations={canUpdateIntegrations}
        readOnly={isReadOnly}
        hasFitToViewRef={hasFitToViewRef}
        hasUserToggledSidebarRef={hasUserToggledSidebarRef}
        isSidebarOpenRef={isSidebarOpenRef}
        viewportRef={viewportRef}
        initialFocusNodeId={initialFocusNodeIdRef.current}
        unsavedMessage={hasUnsavedChanges ? "You have unsaved changes" : undefined}
        saveIsPrimary={saveIsPrimary}
        saveButtonHidden={saveButtonHidden}
        saveDisabled={saveDisabled}
        saveDisabledTooltip={saveDisabledTooltip}
        onUndo={!isReadOnly ? handleRevert : undefined}
        canUndo={canUndo}
        isAutoSaveEnabled={isAutoSaveEnabled && !isTemplate}
        onToggleAutoSave={isTemplate ? undefined : handleToggleAutoSave}
        autoSaveDisabled={autoSaveDisabled}
        autoSaveDisabledTooltip={autoSaveDisabledTooltip}
        onExportYamlCopy={isDev ? handleExportYamlCopy : undefined}
        onExportYamlDownload={isDev ? handleExportYamlDownload : undefined}
        runDisabled={runDisabled}
        runDisabledTooltip={runDisabledTooltip}
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
        onReEmit={canUpdateCanvas ? handleReEmit : undefined}
        loadExecutionChain={loadExecutionChain}
        getExecutionState={getExecutionState}
        workflowNodes={canvas?.spec?.nodes}
        components={components}
        triggers={triggers}
        blueprints={blueprints}
        logEntries={logEntries}
        onResolveExecutionErrors={canUpdateCanvas ? handleResolveExecutionErrors : undefined}
        focusRequest={focusRequest}
        onExecutionChainHandled={handleExecutionChainHandled}
        breadcrumbs={[
          {
            label: isTemplate ? "Templates" : "Canvases",
            href: `/${organizationId}`,
          },
          {
            label: canvas?.metadata?.name || (isTemplate ? "Template" : "Canvas"),
          },
        ]}
      />
      {canvas ? (
        <CreateCanvasModal
          isOpen={isUseTemplateOpen}
          onClose={() => setIsUseTemplateOpen(false)}
          onSubmit={handleUseTemplateSubmit}
          isLoading={createWorkflowMutation.isPending}
          templates={[
            {
              id: canvas.metadata?.id || "",
              name: canvas.metadata?.name || "Untitled template",
              description: canvas.metadata?.description,
            },
          ]}
          defaultTemplateId={canvas.metadata?.id || ""}
          mode="create"
          fromTemplate
        />
      ) : null}
      <Dialog open={canvasDeletedRemotely} onOpenChange={() => {}}>
        <DialogContent showCloseButton={false}>
          <DialogHeader>
            <DialogTitle>Canvas deleted</DialogTitle>
            <DialogDescription>
              This canvas was deleted from another session. You can no longer edit or run it.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button
              onClick={() => {
                if (organizationId) {
                  navigate(`/${organizationId}`, { replace: true });
                }
              }}
            >
              Go to canvases
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}

function useExecutionChainData(workflowId: string, queryClient: QueryClient, workflow?: CanvasesCanvas) {
  const loadExecutionChain = useCallback(
    async (
      eventId: string,
      nodeId?: string,
      currentExecution?: Record<string, unknown>,
      forceReload = false,
    ): Promise<CanvasesCanvasNodeExecution[]> => {
      const queryOptions = eventExecutionsQueryOptions(workflowId, eventId);

      let allExecutions: CanvasesCanvasNodeExecution[] = [];

      if (!forceReload) {
        const cachedData = queryClient.getQueryData(queryOptions.queryKey);
        if (cachedData) {
          allExecutions = (cachedData as CanvasesListEventExecutionsResponse)?.executions || [];
        }
      }

      if (allExecutions.length === 0) {
        if (forceReload) {
          await queryClient.invalidateQueries({ queryKey: queryOptions.queryKey });
        }
        const data = await queryClient.fetchQuery(queryOptions);
        allExecutions = (data as CanvasesListEventExecutionsResponse)?.executions || [];
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
function getNodesBeforeTarget(targetNodeId: string, workflow: CanvasesCanvas): Set<string> {
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
  workflow: CanvasesCanvas,
  triggers: TriggersTrigger[],
  blueprints: BlueprintsBlueprint[],
  components: ComponentsComponent[],
  nodeEventsMap: Record<string, CanvasesCanvasEvent[]>,
  nodeExecutionsMap: Record<string, CanvasesCanvasNodeExecution[]>,
  nodeQueueItemsMap: Record<string, CanvasesCanvasNodeQueueItem[]>,
  workflowId: string,
  queryClient: QueryClient,
  organizationId: string,
  currentUser?: { id?: string; email?: string },
): {
  nodes: CanvasNode[];
  edges: CanvasEdge[];
} {
  const edges = workflow?.spec?.edges?.map(prepareEdge) || [];
  const workflowEdges = workflow?.spec?.edges || [];
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
          workflowEdges,
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
  nodeEventsMap: Record<string, CanvasesCanvasEvent[]>,
): CanvasNode {
  const triggerMetadata = triggers.find((t) => t.name === node.trigger?.name);
  const renderer = getTriggerRenderer(node.trigger?.name || "");
  const lastEvent = nodeEventsMap[node.id!]?.[0];
  const triggerProps = renderer.getTriggerProps({
    node: buildNodeInfo(node),
    definition: buildComponentDefinition(triggerMetadata),
    lastEvent: buildEventInfo(lastEvent),
  });

  // Use node name if available, otherwise fall back to trigger label (from metadata)
  const displayLabel = node.name || triggerMetadata?.label || node.trigger?.name || "Trigger";

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
        warning: node.warningMessage,
      },
    },
  };
}

function prepareCompositeNode(
  nodes: ComponentsNode[],
  node: ComponentsNode,
  blueprints: BlueprintsBlueprint[],
  nodeExecutionsMap: Record<string, CanvasesCanvasNodeExecution[]>,
  nodeQueueItemsMap: Record<string, CanvasesCanvasNodeQueueItem[]>,
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
        warning: node.warningMessage,
        paused: !!node.paused,
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
    const eventInfo = buildEventInfo(execution.rootEvent!);
    const { title, subtitle } = rootTriggerRenderer.getTitleAndSubtitle({ event: eventInfo });
    (canvasNode.data.composite as CompositeProps).lastRunItem = {
      title: title,
      subtitle: subtitle,
      id: execution.rootEvent?.id,
      receivedAt: new Date(execution.createdAt!),
      state: getRunItemState(execution),
      values: rootTriggerRenderer.getRootEventValues({ event: eventInfo }),
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

function getRunItemState(execution: CanvasesCanvasNodeExecution): LastRunState {
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
  nodeEventsMap: Record<string, CanvasesCanvasEvent[]>,
  nodeExecutionsMap: Record<string, CanvasesCanvasNodeExecution[]>,
  nodeQueueItemsMap: Record<string, CanvasesCanvasNodeQueueItem[]>,
  workflowId: string,
  queryClient: any,
  organizationId: string,
  currentUser?: { id?: string; email?: string },
  edges?: ComponentsEdge[],
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
        edges,
      );
  }
}

function prepareAnnotationNode(node: ComponentsNode): CanvasNode {
  const width = (node.configuration?.width as number) || 320;
  const height = (node.configuration?.height as number) || 200;
  return {
    id: node.id!,
    position: { x: node.position?.x!, y: node.position?.y! },
    selectable: true,
    style: { width, height },
    data: {
      type: "annotation",
      label: node.name || "Annotation",
      state: "pending" as const,
      outputChannels: [], // Annotation nodes don't have output channels
      annotation: {
        title: node.name || "Annotation",
        annotationText: node.configuration?.text || "",
        annotationColor: node.configuration?.color || "yellow",
        width,
        height,
      },
    },
  };
}

function prepareComponentNode(
  nodes: ComponentsNode[],
  node: ComponentsNode,
  components: ComponentsComponent[],
  nodeExecutionsMap: Record<string, CanvasesCanvasNodeExecution[]>,
  nodeQueueItemsMap: Record<string, CanvasesCanvasNodeQueueItem[]>,
  workflowId: string,
  queryClient: QueryClient,
  organizationId?: string,
  currentUser?: { id?: string; email?: string },
  edges?: ComponentsEdge[],
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
    return prepareMergeNode(nodes, node, components, nodeExecutionsMap, nodeQueueItemsMap, edges);
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
  nodeExecutionsMap: Record<string, CanvasesCanvasNodeExecution[]>,
  nodeQueueItemsMap: Record<string, CanvasesCanvasNodeQueueItem[]>,
  workflowId: string,
  queryClient: QueryClient,
  organizationId: string,
  currentUser?: { id?: string; email?: string },
): CanvasNode {
  const executions = nodeExecutionsMap[node.id!] || [];
  const metadata = components.find((c) => c.name === node.component?.name);
  const displayLabel = node.name || metadata?.label;
  const componentDef = components.find((c) => c.name === node.component?.name);
  const fallbackComponentDef = componentDef || {
    name: node.component?.name,
    label: node.name,
  };
  const nodeQueueItems = nodeQueueItemsMap?.[node.id!];

  const additionalData = componentDef
    ? getComponentAdditionalDataBuilder(node.component?.name || "")?.buildAdditionalData({
        nodes: nodes.map((n) => buildNodeInfo(n)),
        node: buildNodeInfo(node),
        componentDefinition: buildComponentDefinition(componentDef!),
        lastExecutions: executions.map((e) => buildExecutionInfo(e)),
        canvasId: workflowId,
        queryClient: queryClient,
        organizationId: organizationId,
        currentUser: currentUser,
      })
    : undefined;

  const componentBaseProps = getComponentBaseMapper(node.component?.name || "").props({
    nodes: nodes.map((n) => buildNodeInfo(n)),
    node: buildNodeInfo(node),
    componentDefinition: buildComponentDefinition(fallbackComponentDef),
    lastExecutions: executions.map((e) => buildExecutionInfo(e)),
    nodeQueueItems: nodeQueueItems?.map((q) => buildQueueItemInfo(q)),
    additionalData: additionalData,
  });

  // If the mapper didn't provide a custom icon, resolve from the app logo map
  if (!componentBaseProps.iconSrc) {
    const resolvedIconSrc = getHeaderIconSrc(node.component?.name);
    if (resolvedIconSrc) {
      componentBaseProps.iconSrc = resolvedIconSrc;
    }
  }

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
        warning: node.warningMessage,
        paused: !!node.paused,
      },
    },
  };
}

function prepareMergeNode(
  nodes: ComponentsNode[],
  node: ComponentsNode,
  components: ComponentsComponent[],
  nodeExecutionsMap: Record<string, CanvasesCanvasNodeExecution[]>,
  nodeQueueItemsMap?: Record<string, CanvasesCanvasNodeQueueItem[]>,
  edges?: ComponentsEdge[],
): CanvasNode {
  const executions = nodeExecutionsMap[node.id!] || [];
  const execution = executions.length > 0 ? executions[0] : null;
  const componentDef = components.find((c) => c.name === "merge");

  // Calculate incoming sources count from edges
  const incomingSourcesCount = edges?.filter((edge) => edge.targetId === node.id).length || 0;
  const additionalData = { incomingSourcesCount };

  // Use the merge state function and mapper for consistent state handling
  const mergeStateResolver = getState("merge");
  const mergeStateMap = getStateMap("merge");
  const mergeMapper = getComponentBaseMapper("merge");

  let lastEvent;
  if (execution) {
    const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
    const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.trigger?.name || "");

    const executionInfo = buildExecutionInfo(execution);
    const { title } = rootTriggerRenderer.getTitleAndSubtitle({ event: buildEventInfo(execution.rootEvent!) });

    // Get subtitle from the merge mapper with incoming sources count
    const eventSubtitle = mergeMapper.subtitle({ node: buildNodeInfo(node), execution: executionInfo, additionalData });

    lastEvent = {
      receivedAt: new Date(execution.createdAt!),
      eventTitle: title,
      eventSubtitle: eventSubtitle,
      eventState: mergeStateResolver(executionInfo),
      eventId: execution.rootEvent?.id,
    };
  }

  const displayLabel = node.name || componentDef?.label || "Merge";

  return {
    id: node.id!,
    position: { x: node.position?.x || 0, y: node.position?.y || 0 },
    data: {
      type: "merge",
      label: displayLabel,
      state: "pending" as const,
      outputChannels: componentDef?.outputChannels?.map((channel) => channel.name!) || ["default"],
      merge: {
        title: displayLabel,
        lastEvent: lastEvent,
        nextInQueue: getNextInQueueInfo(nodeQueueItemsMap, node.id!, nodes),
        collapsedBackground: getBackgroundColorClass("white"),
        collapsed: node.isCollapsed,
        eventStateMap: mergeStateMap,
        error: node.errorMessage,
        warning: node.warningMessage,
        paused: !!node.paused,
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
  nodeExecutionsMap: Record<string, CanvasesCanvasNodeExecution[]>,
  nodeQueueItemsMap: Record<string, CanvasesCanvasNodeQueueItem[]>,
  nodeEventsMap: Record<string, CanvasesCanvasEvent[]>,
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

  const additionalData = getComponentAdditionalDataBuilder(node.component?.name || "")?.buildAdditionalData({
    nodes: nodes.map((n) => buildNodeInfo(n)),
    node: buildNodeInfo(node),
    componentDefinition: buildComponentDefinition(componentMetadata!),
    lastExecutions: executions.map((e) => buildExecutionInfo(e)),
    canvasId: workflowId || "",
    queryClient: queryClient as QueryClient,
    organizationId: organizationId || "",
    currentUser: currentUser,
  });

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
    totalInHistoryCount: totalHistoryCount ? totalHistoryCount : 0,
    totalInQueueCount: totalQueueCount ? totalQueueCount : 0,
    hideQueueEvents,
    isComposite: node.type === "TYPE_BLUEPRINT",
  };
}
