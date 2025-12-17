/* eslint-disable @typescript-eslint/no-explicit-any */
import {
  Background,
  ReactFlow,
  ReactFlowProvider,
  useReactFlow,
  type Edge as ReactFlowEdge,
  type Node as ReactFlowNode,
} from "@xyflow/react";

import { Loader2, Puzzle, ScanLine, ScanText } from "lucide-react";
import { ZoomSlider } from "@/components/zoom-slider";
import { Button } from "@/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";

import {
  ComponentsAppInstallationRef,
  OrganizationsAppInstallation,
  ConfigurationField,
  WorkflowsWorkflowNodeExecution,
  ComponentsNode,
  ComponentsComponent,
  TriggersTrigger,
  BlueprintsBlueprint,
} from "@/api-client";
import { getCustomFieldRenderer } from "@/pages/workflowv2/mappers";
import { AiSidebar } from "../ai";
import { BuildingBlock, BuildingBlockCategory, BuildingBlocksSidebar } from "../BuildingBlocksSidebar";
import { ComponentSidebar } from "../componentSidebar";
import { TabData } from "../componentSidebar/SidebarEventItem/SidebarEventItem";
import { EmitEventModal } from "../EmitEventModal";
import { ComponentBaseProps, EventState, EventStateMap } from "../componentBase";
import { Block, BlockData } from "./Block";
import "./canvas-reset.css";
import { CustomEdge } from "./CustomEdge";
import { Header, type BreadcrumbItem } from "./Header";
import { Simulation } from "./storybooks/useSimulation";
import { CanvasPageState, useCanvasState } from "./useCanvasState";
import { getBackgroundColorClass } from "@/utils/colors";

export interface SidebarEvent {
  id: string;
  title: string;
  subtitle?: string | React.ReactNode;
  state: string;
  isOpen: boolean;
  receivedAt?: Date;
  values?: Record<string, string>;
  // Optional specific identifiers to avoid overloading `id`
  executionId?: string;
  triggerEventId?: string;
  kind?: "execution" | "trigger" | "queue";
}

export interface SidebarData {
  latestEvents: SidebarEvent[];
  nextInQueueEvents: SidebarEvent[];
  title: string;
  iconSrc?: string;
  iconSlug?: string;
  iconColor?: string;
  iconBackground?: string;
  totalInQueueCount: number;
  totalInHistoryCount: number;
  hideQueueEvents?: boolean;
  isLoading?: boolean;
  isComposite?: boolean;
}

export interface CanvasNode extends ReactFlowNode {
  __simulation?: Simulation;
}

export interface CanvasEdge extends ReactFlowEdge {
  sourceHandle?: string | null;
  targetHandle?: string | null;
}

export interface AiProps {
  enabled: boolean;
  sidebarOpen: boolean;
  setSidebarOpen: (open: boolean) => void;
  showNotifications: boolean;
  notificationMessage?: string;
  suggestions: Record<string, string>;
  onApply: (suggestionId: string) => void;
  onDismiss: (suggestionId: string) => void;
}

export interface NodeEditData {
  nodeId: string;
  nodeName: string;
  displayLabel?: string;
  configuration: Record<string, any>;
  configurationFields: ConfigurationField[];
  appName?: string;
  appInstallationRef?: ComponentsAppInstallationRef;
}

export interface NewNodeData {
  icon?: string;
  buildingBlock: BuildingBlock;
  nodeName: string;
  displayLabel?: string;
  configuration: Record<string, any>;
  position?: { x: number; y: number };
  appName?: string;
  appInstallationRef?: ComponentsAppInstallationRef;
  sourceConnection?: {
    nodeId: string;
    handleId: string | null;
  };
}

export interface CanvasPageProps {
  nodes: CanvasNode[];
  edges: CanvasEdge[];

  startCollapsed?: boolean;
  title?: string;
  breadcrumbs?: BreadcrumbItem[];
  organizationId?: string;
  unsavedMessage?: string;
  saveIsPrimary?: boolean;
  saveButtonHidden?: boolean;
  // Undo functionality
  onUndo?: () => void;
  canUndo?: boolean;
  // Disable running nodes when there are unsaved changes (with tooltip)
  runDisabled?: boolean;
  runDisabledTooltip?: string;

  onNodeExpand?: (nodeId: string, nodeData: unknown) => void;
  getSidebarData?: (nodeId: string) => SidebarData | null;
  loadSidebarData?: (nodeId: string) => void;
  getTabData?: (nodeId: string, event: SidebarEvent) => TabData | undefined;
  getNodeEditData?: (nodeId: string) => NodeEditData | null;
  onNodeConfigurationSave?: (
    nodeId: string,
    configuration: Record<string, any>,
    nodeName: string,
    appInstallationRef?: ComponentsAppInstallationRef,
  ) => void;
  getCustomField?: (nodeId: string) => ((configuration: Record<string, unknown>) => React.ReactNode) | null;
  onSave?: (nodes: CanvasNode[]) => void;
  installedApplications?: OrganizationsAppInstallation[];
  onEdgeCreate?: (sourceId: string, targetId: string, sourceHandle?: string | null) => void;
  onNodeDelete?: (nodeId: string) => void;
  onEdgeDelete?: (edgeIds: string[]) => void;
  onNodePositionChange?: (nodeId: string, position: { x: number; y: number }) => void;
  onCancelQueueItem?: (nodeId: string, queueItemId: string) => void;
  onPushThrough?: (nodeId: string, executionId: string) => void;
  onCancelExecution?: (nodeId: string, executionId: string) => void;
  supportsPushThrough?: (nodeId: string) => boolean;
  onDirty?: () => void;

  onRun?: (nodeId: string, channel: string, data: any) => void | Promise<void>;
  onDuplicate?: (nodeId: string) => void;
  onDocs?: (nodeId: string) => void;
  onEdit?: (nodeId: string) => void;
  onConfigure?: (nodeId: string) => void;
  onDeactivate?: (nodeId: string) => void;
  onToggleView?: (nodeId: string) => void;
  onToggleCollapse?: () => void;
  onReEmit?: (nodeId: string, eventOrExecutionId: string) => void;

  ai?: AiProps;

  // Building blocks for adding new nodes
  buildingBlocks: BuildingBlockCategory[];
  onNodeAdd?: (newNodeData: NewNodeData) => void;

  // Refs to persist state across re-renders
  hasFitToViewRef?: React.MutableRefObject<boolean>;
  hasUserToggledSidebarRef?: React.MutableRefObject<boolean>;
  isSidebarOpenRef?: React.MutableRefObject<boolean | null>;
  viewportRef?: React.MutableRefObject<{ x: number; y: number; zoom: number } | undefined>;

  // Optional: control and observe component sidebar state
  onSidebarChange?: (isOpen: boolean, selectedNodeId: string | null) => void;
  initialSidebar?: { isOpen?: boolean; nodeId?: string | null };

  // Full history functionality
  getAllHistoryEvents?: (nodeId: string) => SidebarEvent[];
  onLoadMoreHistory?: (nodeId: string) => void;
  getHasMoreHistory?: (nodeId: string) => boolean;
  getLoadingMoreHistory?: (nodeId: string) => boolean;

  // Queue functionality
  onLoadMoreQueue?: (nodeId: string) => void;
  getAllQueueEvents?: (nodeId: string) => SidebarEvent[];
  getHasMoreQueue?: (nodeId: string) => boolean;
  getLoadingMoreQueue?: (nodeId: string) => boolean;

  // Execution chain lazy loading
  loadExecutionChain?: (
    eventId: string,
    nodeId?: string,
    currentExecution?: Record<string, unknown>,
    forceReload?: boolean,
  ) => Promise<any[]>;

  // State registry function for determining execution states
  getExecutionState?: (
    nodeId: string,
    execution: WorkflowsWorkflowNodeExecution,
  ) => { map: EventStateMap; state: EventState };

  // Workflow metadata for ExecutionChainPage
  workflowNodes?: ComponentsNode[];
  components?: ComponentsComponent[];
  triggers?: TriggersTrigger[];
  blueprints?: BlueprintsBlueprint[];
}

export const CANVAS_SIDEBAR_STORAGE_KEY = "canvasSidebarOpen";
export const COMPONENT_SIDEBAR_WIDTH_STORAGE_KEY = "componentSidebarWidth";

const EDGE_STYLE = {
  type: "custom",
  style: { stroke: "#C9D5E1", strokeWidth: 3 },
} as const;

const DEFAULT_CANVAS_ZOOM = 0.8;

/*
 * nodeTypes must be defined outside of the component to prevent
 * react-flow from remounting the node types on every render.
 */
const nodeTypes = {
  default: (nodeProps: { data: BlockData & { _callbacksRef?: any }; id: string; selected?: boolean }) => {
    const { _callbacksRef, ...blockData } = nodeProps.data;
    const callbacks = _callbacksRef?.current;

    if (!callbacks) {
      return <Block data={blockData} nodeId={nodeProps.id} selected={nodeProps.selected} />;
    }

    return (
      <Block
        data={blockData}
        nodeId={nodeProps.id}
        selected={nodeProps.selected}
        runDisabled={callbacks?.runDisabled}
        runDisabledTooltip={callbacks?.runDisabledTooltip}
        onExpand={callbacks.handleNodeExpand}
        onClick={() => callbacks.handleNodeClick(nodeProps.id)}
        onEdit={() => callbacks.onNodeEdit.current?.(nodeProps.id)}
        onDelete={callbacks.onNodeDelete.current ? () => callbacks.onNodeDelete.current?.(nodeProps.id) : undefined}
        onRun={callbacks.onRun.current ? () => callbacks.onRun.current?.(nodeProps.id) : undefined}
        onDuplicate={callbacks.onDuplicate.current ? () => callbacks.onDuplicate.current?.(nodeProps.id) : undefined}
        onConfigure={callbacks.onConfigure.current ? () => callbacks.onConfigure.current?.(nodeProps.id) : undefined}
        onDeactivate={callbacks.onDeactivate.current ? () => callbacks.onDeactivate.current?.(nodeProps.id) : undefined}
        onToggleView={callbacks.onToggleView.current ? () => callbacks.onToggleView.current?.(nodeProps.id) : undefined}
        onToggleCollapse={
          callbacks.onToggleView.current ? () => callbacks.onToggleView.current?.(nodeProps.id) : undefined
        }
        ai={{
          show: callbacks.aiState.sidebarOpen,
          suggestion: callbacks.aiState.suggestions[nodeProps.id] || null,
          onApply: () => callbacks.aiState.onApply(nodeProps.id),
          onDismiss: () => callbacks.aiState.onDismiss(nodeProps.id),
        }}
      />
    );
  },
};

function CanvasPage(props: CanvasPageProps) {
  const cancelQueueItemRef = useRef<CanvasPageProps["onCancelQueueItem"]>(props.onCancelQueueItem);
  cancelQueueItemRef.current = props.onCancelQueueItem;
  const state = useCanvasState(props);
  const [newNodeData, setNewNodeData] = useState<NewNodeData | null>(null);
  const [currentTab, setCurrentTab] = useState<"latest" | "settings">("latest");
  const [templateNodeId, setTemplateNodeId] = useState<string | null>(null);

  // Use refs from props if provided, otherwise create local ones
  const hasFitToViewRef = props.hasFitToViewRef || useRef(false);
  const hasUserToggledSidebarRef = props.hasUserToggledSidebarRef || useRef(false);
  const isSidebarOpenRef = props.isSidebarOpenRef || useRef<boolean | null>(null);

  if (isSidebarOpenRef.current === null && typeof window !== "undefined") {
    const storedSidebarState = window.localStorage.getItem(CANVAS_SIDEBAR_STORAGE_KEY);
    if (storedSidebarState !== null) {
      try {
        isSidebarOpenRef.current = JSON.parse(storedSidebarState);
        hasUserToggledSidebarRef.current = true;
      } catch (error) {
        console.warn("Failed to parse canvas sidebar state:", error);
      }
    }
  }

  // Initialize sidebar state from ref if available, otherwise based on whether nodes exist
  const [isBuildingBlocksSidebarOpen, setIsBuildingBlocksSidebarOpen] = useState(() => {
    // If we have a persisted state in the ref, use it
    if (isSidebarOpenRef.current !== null) {
      return isSidebarOpenRef.current;
    }
    // Otherwise, open if no nodes exist
    return props.nodes.length === 0;
  });

  const initialCanvasZoom = props.nodes.length === 0 ? DEFAULT_CANVAS_ZOOM : 1;
  const [canvasZoom, setCanvasZoom] = useState(initialCanvasZoom);
  const [emitModalData, setEmitModalData] = useState<{
    nodeId: string;
    nodeName: string;
    channels: string[];
  } | null>(null);

  const handleNodeEdit = useCallback(
    (nodeId: string) => {
      // Open the sidebar for this node (data will be automatically available via useMemo)
      if (!state.componentSidebar.isOpen || state.componentSidebar.selectedNodeId !== nodeId) {
        state.componentSidebar.open(nodeId);
        // Close building blocks sidebar when component sidebar opens
        setIsBuildingBlocksSidebarOpen(false);
      }

      // Switch to settings tab when edit is called
      setCurrentTab("settings");

      // Fall back to the simple onEdit callback if no getNodeEditData
      if (!props.getNodeEditData && props.onEdit) {
        props.onEdit(nodeId);
      }
    },
    [props.getNodeEditData, props.onEdit, state.componentSidebar],
  );

  // Get editing data for the currently selected node
  const { getNodeEditData } = props;
  const editingNodeData = useMemo(() => {
    if (state.componentSidebar.selectedNodeId && state.componentSidebar.isOpen && getNodeEditData) {
      return getNodeEditData(state.componentSidebar.selectedNodeId);
    }
    return null;
  }, [state.componentSidebar.selectedNodeId, state.componentSidebar.isOpen, getNodeEditData]);

  const handleNodeDelete = useCallback(
    (nodeId: string) => {
      if (props.onNodeDelete) {
        props.onNodeDelete(nodeId);
      }
    },
    [props],
  );

  const handleNodeRun = useCallback(
    (nodeId: string) => {
      // Hard guard: if running is disabled (e.g., unsaved changes), do nothing
      if (props.runDisabled) return;
      // Find the node to get its name and channels
      const node = state.nodes.find((n) => n.id === nodeId);
      if (!node) return;

      const nodeName = (node.data as any).label || nodeId;
      const channels = (node.data as any).outputChannels || ["default"];

      setEmitModalData({
        nodeId,
        nodeName,
        channels,
      });
    },
    [state.nodes, props.runDisabled],
  );

  const handleEmit = useCallback(
    async (channel: string, data: any) => {
      if (!emitModalData || !props.onRun) return;

      // Call the onRun prop with nodeId, channel, and data
      await props.onRun(emitModalData.nodeId, channel, data);
    },
    [emitModalData, props],
  );

  const handleConnectionDropInEmptySpace = useCallback(
    (position: { x: number; y: number }, sourceConnection: { nodeId: string; handleId: string | null }) => {
      if (!sourceConnection) return;

      const pendingNodeId = `pending_connection_${Date.now()}`;

      const placeholderNode: CanvasNode = {
        id: pendingNodeId,
        type: "default",
        position: {
          x: position.x,
          y: position.y - 30,
        },
        data: {
          type: "component",
          label: "New Component",
          state: "neutral",
          component: {
            title: "New Component",
            headerColor: "bg-gray-50",
            iconSlug: "puzzle",
            iconColor: "text-indigo-700",
            collapsedBackground: getBackgroundColorClass("white"),
            hideActionsButton: true,
            includeEmptyState: true,
            emptyStateProps: {
              icon: Puzzle,
              title: "Select the component from sidebar",
            },
          } as ComponentBaseProps,
          isPendingConnection: true,
          sourceConnection: {
            nodeId: sourceConnection.nodeId,
            handleId: sourceConnection.handleId,
          },
        },
      };

      // Add the new node first
      state.setNodes((nodes) => [...nodes, placeholderNode]);

      // Check if current template is a configured template (not just pending connection)
      const currentTemplateNode = templateNodeId ? state.nodes.find((n) => n.id === templateNodeId) : null;
      const isCurrentTemplateConfigured =
        currentTemplateNode?.data?.isTemplate && !currentTemplateNode?.data?.isPendingConnection;

      // Only select and set as template if there isn't a configured template being created
      // Allow switching between pending nodes, but prevent overwriting configured templates
      if (!isCurrentTemplateConfigured) {
        // Then update all nodes to set selection (deselect others, select the new one)
        // This needs to happen in a separate setNodes call to ensure ReactFlow processes the selection
        setTimeout(() => {
          state.setNodes((nodes) =>
            nodes.map((node) => ({
              ...node,
              selected: node.id === pendingNodeId,
            })),
          );
        }, 0);
        setTemplateNodeId(pendingNodeId);
      }

      // Create edge locally (not via parent) - it will be preserved by useCanvasState
      const edgeId = `${sourceConnection.nodeId}--${pendingNodeId}--${sourceConnection.handleId || "default"}`;
      state.setEdges([
        ...state.edges,
        {
          id: edgeId,
          source: sourceConnection.nodeId,
          target: pendingNodeId,
          sourceHandle: sourceConnection.handleId || "default",
        },
      ]);

      // Open component sidebar to track the selected node
      state.componentSidebar.open(pendingNodeId);

      // Open BuildingBlocksSidebar for component selection
      setIsBuildingBlocksSidebarOpen(true);
    },
    [state, setTemplateNodeId, setIsBuildingBlocksSidebarOpen],
  );

  const handlePendingConnectionNodeClick = useCallback(
    (nodeId: string) => {
      // Set this node as the active template
      setTemplateNodeId(nodeId);
      // Open the BuildingBlocksSidebar so user can select a component
      setIsBuildingBlocksSidebarOpen(true);
      // Close ComponentSidebar since pending nodes don't have configuration yet
      state.componentSidebar.close();
    },
    [setTemplateNodeId, setIsBuildingBlocksSidebarOpen, state.componentSidebar],
  );

  const handleTemplateNodeClick = useCallback(
    (nodeId: string) => {
      const templateNode = state.nodes.find((n) => n.id === nodeId);
      if (!templateNode) return;

      const buildingBlock = templateNode.data.buildingBlock as BuildingBlock;

      // Restore template state from the node - including sourceConnection and position
      setTemplateNodeId(nodeId);
      setNewNodeData({
        buildingBlock: buildingBlock,
        nodeName: (templateNode.data.nodeName || buildingBlock?.name || "New Component") as string,
        icon: (templateNode.data.icon || buildingBlock?.icon || "Box") as string,
        configuration: templateNode.data.configuration || {},
        position: templateNode.position,
        sourceConnection: templateNode.data.sourceConnection as { nodeId: string; handleId: string | null } | undefined,
      });
      // Open ComponentSidebar for configuration
      state.componentSidebar.open(nodeId);
    },
    [state, setTemplateNodeId, setNewNodeData],
  );

  const handleBuildingBlockClick = useCallback(
    (block: BuildingBlock) => {
      const pendingNode = state.nodes.find((n) => n.id === templateNodeId && n.data.isPendingConnection);
      if (!pendingNode || !templateNodeId) {
        return;
      }

      // Update the same node (keep ID to preserve edge)
      state.setNodes((nodes) =>
        nodes.map((n) =>
          n.id === templateNodeId
            ? {
                ...n,
                data: {
                  type: "component",
                  label: block.label || block.name || "New Component",
                  state: "neutral",
                  component: {
                    title: block.label || block.name || "New Component",
                    headerColor: "#e5e7eb",
                    iconSlug: block.icon,
                    iconColor: "text-indigo-700",
                    collapsedBackground: getBackgroundColorClass("white"),
                    hideActionsButton: true,
                    includeEmptyState: true,
                  } as ComponentBaseProps,
                  isTemplate: true,
                  isPendingConnection: false, // Remove pending connection flag
                  buildingBlock: block,
                  tempConfiguration: {},
                  tempNodeName: block.name || "",
                  sourceConnection: pendingNode.data.sourceConnection, // Preserve sourceConnection for later
                  configuration: {},
                  nodeName: block.name || "",
                  icon: block.icon,
                },
              }
            : n,
        ),
      );

      setNewNodeData({
        buildingBlock: block,
        nodeName: block.name || "",
        displayLabel: block.label || block.name || "",
        configuration: {},
        position: pendingNode.position,
        sourceConnection: pendingNode.data.sourceConnection as { nodeId: string; handleId: string | null } | undefined,
        appName: block.appName,
      });

      setIsBuildingBlocksSidebarOpen(false);
      state.componentSidebar.open(templateNodeId);
      setCurrentTab("settings");
    },
    [templateNodeId, state, setCurrentTab, setNewNodeData, setIsBuildingBlocksSidebarOpen],
  );

  const handleBuildingBlockDrop = useCallback(
    (block: BuildingBlock, position?: { x: number; y: number }) => {
      if (templateNodeId) {
        return;
      }

      // Generate unique template node ID
      const newTemplateId = `template_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;

      // Create template node data
      const templateNode: CanvasNode = {
        id: newTemplateId,
        type: "default",
        position: position || { x: 0, y: 0 },
        selected: true,
        data: {
          type: "component",
          label: block.label || block.name || "New Component",
          state: "neutral",
          component: {
            title: block.label || block.name || "New Component",
            headerColor: "#e5e7eb",
            iconSlug: block.icon,
            iconColor: "text-indigo-700",
            collapsedBackground: getBackgroundColorClass("white"),
            hideActionsButton: true,
            includeEmptyState: true,
            emptyStateTitle: block.type === "trigger" ? "Waiting for the first event" : undefined,
          } as ComponentBaseProps,
          isTemplate: true,
          buildingBlock: block,
          tempConfiguration: {},
          tempNodeName: block.name || "",
        },
      };

      // Deselect all existing nodes and add the new selected template node
      state.setNodes((nodes) => [...nodes.map((n) => ({ ...n, selected: false })), templateNode]);

      setTemplateNodeId(newTemplateId);
      setNewNodeData({
        icon: block.icon || "circle-off",
        buildingBlock: block,
        nodeName: block.name || "",
        displayLabel: block.label || block.name || "",
        configuration: {},
        position,
        appName: block.appName,
      });

      state.componentSidebar.open(newTemplateId);
      setCurrentTab("settings");
      // Close building blocks sidebar when dropping a new component
      setIsBuildingBlocksSidebarOpen(false);
    },
    [templateNodeId, state, setCurrentTab],
  );

  const handleSidebarToggle = useCallback(
    (open: boolean) => {
      hasUserToggledSidebarRef.current = true;
      isSidebarOpenRef.current = open;
      setIsBuildingBlocksSidebarOpen(open);
      if (typeof window !== "undefined") {
        window.localStorage.setItem(CANVAS_SIDEBAR_STORAGE_KEY, JSON.stringify(open));
      }
    },
    [hasUserToggledSidebarRef, isSidebarOpenRef],
  );

  const handleSaveConfiguration = useCallback(
    (configuration: Record<string, any>, nodeName: string, appInstallationRef?: ComponentsAppInstallationRef) => {
      if (templateNodeId && newNodeData) {
        // Template nodes should always be converted to real nodes
        // Remove the template node first
        state.setNodes((nodes) => nodes.filter((node) => node.id !== templateNodeId));

        // Remove edges connected to the template node (they'll be recreated by parent if needed)
        state.setEdges(state.edges.filter((edge) => edge.source !== templateNodeId && edge.target !== templateNodeId));

        // Create the real node through the normal flow
        if (props.onNodeAdd) {
          props.onNodeAdd({
            buildingBlock: newNodeData.buildingBlock,
            nodeName,
            configuration,
            position: newNodeData.position,
            appName: newNodeData.appName,
            appInstallationRef,
            sourceConnection: newNodeData.sourceConnection, // Will be undefined for dropped components
          });
        }

        // Clear template state
        setTemplateNodeId(null);
        setNewNodeData(null);

        // Close sidebar and reset tab
        state.componentSidebar.close();
        setCurrentTab("latest");
      } else if (editingNodeData && props.onNodeConfigurationSave) {
        props.onNodeConfigurationSave(editingNodeData.nodeId, configuration, nodeName, appInstallationRef);
      }
    },
    [templateNodeId, newNodeData, editingNodeData, props, state, setTemplateNodeId, setNewNodeData, setCurrentTab],
  );

  const handleCancelTemplate = useCallback(() => {
    if (templateNodeId) {
      // Just close the sidebar and clear template state
      // The node remains as a configured template so user can click it again to continue configuring
      setTemplateNodeId(null);
      setNewNodeData(null);

      state.componentSidebar.close();
      setCurrentTab("latest");
    }
  }, [templateNodeId, state]);

  const handleToggleView = useCallback(
    (nodeId: string) => {
      state.toggleNodeCollapse(nodeId);
      props.onToggleView?.(nodeId);
    },
    [state.toggleNodeCollapse, props.onToggleView],
  );

  const handlePushThrough = (executionId: string) => {
    if (state.componentSidebar.selectedNodeId && props.onPushThrough) {
      props.onPushThrough(state.componentSidebar.selectedNodeId, executionId);
    }
  };

  const handleCancelQueueItem = (queueId: string) => {
    if (state.componentSidebar.selectedNodeId && props.onCancelQueueItem) {
      props.onCancelQueueItem!(state.componentSidebar.selectedNodeId!, queueId);
    }
  };

  const handleCancelExecution = (executionId: string) => {
    if (state.componentSidebar.selectedNodeId && props.onCancelExecution) {
      props.onCancelExecution!(state.componentSidebar.selectedNodeId!, executionId);
    }
  };

  const handleSidebarClose = useCallback(() => {
    // Check if the currently open node is a pending connection
    const currentNode = state.nodes.find((n) => n.id === state.componentSidebar.selectedNodeId);
    const isPendingConnection = currentNode?.data?.isPendingConnection;

    state.componentSidebar.close();
    // Reset to latest tab when sidebar closes
    setCurrentTab("latest");

    // Only remove the node if it's a pending connection node (not yet configured)
    if (isPendingConnection && state.componentSidebar.selectedNodeId) {
      const nodeIdToRemove = state.componentSidebar.selectedNodeId;
      state.setNodes((nodes) => nodes.filter((node) => node.id !== nodeIdToRemove));
      state.setEdges(state.edges.filter((edge) => edge.source !== nodeIdToRemove && edge.target !== nodeIdToRemove));

      // Clear template tracking if this was the active template
      if (templateNodeId === nodeIdToRemove) {
        setNewNodeData(null);
        setTemplateNodeId(null);
      }
    }

    // Clear ReactFlow's selection state
    state.setNodes((nodes) =>
      nodes.map((node) => ({
        ...node,
        selected: false,
      })),
    );
  }, [state, templateNodeId]);

  return (
    <div className="h-[100vh] w-[100vw] overflow-hidden sp-canvas relative flex flex-col">
      {/* Header at the top spanning full width */}
      <div className="relative z-20">
        <CanvasContentHeader
          state={state}
          onSave={props.onSave}
          onUndo={props.onUndo}
          canUndo={props.canUndo}
          organizationId={props.organizationId}
          unsavedMessage={props.unsavedMessage}
          saveIsPrimary={props.saveIsPrimary}
          saveButtonHidden={props.saveButtonHidden}
        />
      </div>

      {/* Main content area with sidebar and canvas */}
      <div className="flex-1 flex relative overflow-hidden">
        <BuildingBlocksSidebar
          isOpen={isBuildingBlocksSidebarOpen}
          onToggle={handleSidebarToggle}
          blocks={props.buildingBlocks || []}
          canvasZoom={canvasZoom}
          disabled={!!templateNodeId && !state.nodes.find((n) => n.id === templateNodeId && n.data.isPendingConnection)}
          onBlockClick={handleBuildingBlockClick}
        />

        <div className="flex-1 relative">
          <ReactFlowProvider key="canvas-flow-provider" data-testid="canvas-drop-area">
            <CanvasContent
              state={state}
              onSave={props.onSave}
              onNodeEdit={handleNodeEdit}
              onNodeDelete={handleNodeDelete}
              onEdgeCreate={props.onEdgeCreate}
              hideHeader={true}
              onToggleView={handleToggleView}
              onToggleCollapse={props.onToggleCollapse}
              onRun={handleNodeRun}
              onDuplicate={props.onDuplicate}
              onConfigure={props.onConfigure}
              onDeactivate={props.onDeactivate}
              runDisabled={props.runDisabled}
              runDisabledTooltip={props.runDisabledTooltip}
              onBuildingBlockDrop={handleBuildingBlockDrop}
              onBuildingBlocksSidebarToggle={handleSidebarToggle}
              onConnectionDropInEmptySpace={handleConnectionDropInEmptySpace}
              onPendingConnectionNodeClick={handlePendingConnectionNodeClick}
              onTemplateNodeClick={handleTemplateNodeClick}
              onZoomChange={setCanvasZoom}
              hasFitToViewRef={hasFitToViewRef}
              viewportRefProp={props.viewportRef}
              templateNodeId={templateNodeId}
            />
          </ReactFlowProvider>

          <AiSidebar
            enabled={state.ai.enabled}
            isOpen={state.ai.sidebarOpen}
            setIsOpen={state.ai.setSidebarOpen}
            showNotifications={state.ai.showNotifications}
            notificationMessage={state.ai.notificationMessage}
          />

          <Sidebar
            state={state}
            getSidebarData={props.getSidebarData}
            loadSidebarData={props.loadSidebarData}
            getTabData={props.getTabData}
            onCancelQueueItem={handleCancelQueueItem}
            onPushThrough={handlePushThrough}
            onCancelExecution={handleCancelExecution}
            supportsPushThrough={props.supportsPushThrough}
            onRun={handleNodeRun}
            onDuplicate={props.onDuplicate}
            onDocs={props.onDocs}
            onConfigure={props.onConfigure}
            onDeactivate={props.onDeactivate}
            onToggleView={handleToggleView}
            onDelete={handleNodeDelete}
            runDisabled={props.runDisabled}
            runDisabledTooltip={props.runDisabledTooltip}
            getAllHistoryEvents={props.getAllHistoryEvents}
            onLoadMoreHistory={props.onLoadMoreHistory}
            getHasMoreHistory={props.getHasMoreHistory}
            getLoadingMoreHistory={props.getLoadingMoreHistory}
            onLoadMoreQueue={props.onLoadMoreQueue}
            getAllQueueEvents={props.getAllQueueEvents}
            getHasMoreQueue={props.getHasMoreQueue}
            getLoadingMoreQueue={props.getLoadingMoreQueue}
            onReEmit={props.onReEmit}
            loadExecutionChain={props.loadExecutionChain}
            getExecutionState={props.getExecutionState}
            onSidebarClose={handleSidebarClose}
            editingNodeData={editingNodeData}
            onSaveConfiguration={handleSaveConfiguration}
            onEdit={handleNodeEdit}
            currentTab={currentTab}
            onTabChange={setCurrentTab}
            templateNodeId={templateNodeId}
            onCancelTemplate={handleCancelTemplate}
            newNodeData={newNodeData}
            organizationId={props.organizationId}
            getCustomField={props.getCustomField}
            installedApplications={props.installedApplications}
            workflowNodes={props.workflowNodes}
            components={props.components}
            triggers={props.triggers}
            blueprints={props.blueprints}
          />
        </div>
      </div>

      {/* Edit existing node modal - now handled by settings sidebar */}

      {/* Emit Event Modal */}
      {emitModalData && (
        <EmitEventModal
          isOpen={true}
          onClose={() => setEmitModalData(null)}
          nodeId={emitModalData.nodeId}
          nodeName={emitModalData.nodeName}
          workflowId={props.organizationId || ""}
          organizationId={props.organizationId || ""}
          channels={emitModalData.channels}
          onEmit={handleEmit}
        />
      )}
    </div>
  );
}

/**
 * Create a custom field renderer for template nodes based on building block data
 */
function getTemplateCustomField(
  buildingBlock: BuildingBlock,
): ((configuration: Record<string, unknown>) => React.ReactNode) | null {
  // Determine component name based on building block type and name
  let componentName = "";
  if (buildingBlock.type === "trigger") {
    componentName = buildingBlock.name;
  } else if (buildingBlock.type === "component") {
    componentName = buildingBlock.name;
  } else if (buildingBlock.type === "blueprint") {
    componentName = "default";
  }

  const renderer = getCustomFieldRenderer(componentName);
  if (!renderer) return null;

  // Return a function that takes the current configuration
  return (configuration: Record<string, unknown>) => {
    // Create a mock node for the renderer - it only needs name and the configuration
    const mockNode = {
      name: configuration.nodeName || buildingBlock.label || buildingBlock.name,
      configuration,
    };
    return renderer.render(mockNode as any, configuration);
  };
}

function Sidebar({
  state,
  getSidebarData,
  loadSidebarData,
  getTabData,
  onCancelQueueItem,
  onPushThrough,
  onCancelExecution,
  supportsPushThrough,
  onRun,
  onDuplicate,
  onDocs,
  onConfigure,
  onDeactivate,
  onToggleView,
  onDelete,
  onReEmit,
  runDisabled,
  runDisabledTooltip,
  getAllHistoryEvents,
  onLoadMoreHistory,
  getHasMoreHistory,
  getLoadingMoreHistory,
  onLoadMoreQueue,
  getAllQueueEvents,
  getHasMoreQueue,
  getLoadingMoreQueue,
  loadExecutionChain,
  getExecutionState,
  onSidebarClose,
  editingNodeData,
  onSaveConfiguration,
  onEdit,
  currentTab,
  onTabChange,
  templateNodeId,
  onCancelTemplate,
  newNodeData,
  organizationId,
  getCustomField,
  installedApplications,
  workflowNodes,
  components,
  triggers,
  blueprints,
}: {
  state: CanvasPageState;
  getSidebarData?: (nodeId: string) => SidebarData | null;
  loadSidebarData?: (nodeId: string) => void;
  getTabData?: (nodeId: string, event: SidebarEvent) => TabData | undefined;
  onCancelQueueItem?: (id: string) => void;
  onPushThrough?: (executionId: string) => void;
  onCancelExecution?: (executionId: string) => void;
  supportsPushThrough?: (nodeId: string) => boolean;
  onRun?: (nodeId: string) => void;
  onDuplicate?: (nodeId: string) => void;
  onDocs?: (nodeId: string) => void;
  onConfigure?: (nodeId: string) => void;
  onDeactivate?: (nodeId: string) => void;
  onToggleView?: (nodeId: string) => void;
  onDelete?: (nodeId: string) => void;
  onReEmit?: (nodeId: string, eventOrExecutionId: string) => void;
  runDisabled?: boolean;
  runDisabledTooltip?: string;
  getAllHistoryEvents?: (nodeId: string) => SidebarEvent[];
  onLoadMoreHistory?: (nodeId: string) => void;
  getHasMoreHistory?: (nodeId: string) => boolean;
  getLoadingMoreHistory?: (nodeId: string) => boolean;
  onLoadMoreQueue?: (nodeId: string) => void;
  getAllQueueEvents?: (nodeId: string) => SidebarEvent[];
  getHasMoreQueue?: (nodeId: string) => boolean;
  getLoadingMoreQueue?: (nodeId: string) => boolean;
  loadExecutionChain?: (eventId: string) => Promise<any[]>;
  getExecutionState?: (
    nodeId: string,
    execution: WorkflowsWorkflowNodeExecution,
  ) => { map: EventStateMap; state: EventState };
  onSidebarClose?: () => void;
  editingNodeData?: NodeEditData | null;
  onSaveConfiguration?: (configuration: Record<string, any>, nodeName: string) => void;
  onEdit?: (nodeId: string) => void;
  currentTab?: "latest" | "settings";
  onTabChange?: (tab: "latest" | "settings") => void;
  templateNodeId?: string | null;
  onCancelTemplate?: () => void;
  newNodeData: NewNodeData | null;
  organizationId?: string;
  getCustomField?: (nodeId: string) => ((configuration: Record<string, unknown>) => React.ReactNode) | null;
  installedApplications?: OrganizationsAppInstallation[];
  workflowNodes?: ComponentsNode[];
  components?: ComponentsComponent[];
  triggers?: TriggersTrigger[];
  blueprints?: BlueprintsBlueprint[];
}) {
  const sidebarData = useMemo(() => {
    if (templateNodeId && newNodeData) {
      return {
        title: newNodeData.nodeName,
        iconSlug: newNodeData.icon,
        iconColor: "text-black",
        latestEvents: [],
        nextInQueueEvents: [],
        metadata: [],
        iconBackground: "",
        totalInQueueCount: 0,
        totalInHistoryCount: 0,
        hideQueueEvents: true,
      } as SidebarData;
    }

    if (!state.componentSidebar.selectedNodeId || !getSidebarData) {
      return null;
    }
    return getSidebarData(state.componentSidebar.selectedNodeId);
  }, [state.componentSidebar.selectedNodeId, getSidebarData, templateNodeId, newNodeData]);

  const [latestEvents, setLatestEvents] = useState<SidebarEvent[]>(sidebarData?.latestEvents || []);
  const [nextInQueueEvents, setNextInQueueEvents] = useState<SidebarEvent[]>(sidebarData?.nextInQueueEvents || []);

  // Trigger data loading when sidebar opens for a node
  useEffect(() => {
    if (state.componentSidebar.selectedNodeId && loadSidebarData) {
      loadSidebarData(state.componentSidebar.selectedNodeId);
    }
  }, [state.componentSidebar.selectedNodeId, loadSidebarData]);

  useEffect(() => {
    if (sidebarData?.latestEvents) {
      setLatestEvents(sidebarData.latestEvents);
    }
    if (sidebarData?.nextInQueueEvents) {
      setNextInQueueEvents(sidebarData.nextInQueueEvents);
    }
  }, [sidebarData?.latestEvents, sidebarData?.nextInQueueEvents]);

  if (!sidebarData) {
    return null;
  }

  // Show loading state when data is being fetched
  if (sidebarData.isLoading && currentTab === "latest") {
    const saved = localStorage.getItem(COMPONENT_SIDEBAR_WIDTH_STORAGE_KEY);
    const sidebarWidth = saved ? parseInt(saved, 10) : 450;

    return (
      <div
        className="border-l-1 border-gray-200 border-border absolute right-0 top-0 h-full z-20 overflow-y-auto overflow-x-hidden bg-white shadow-2xl"
        style={{ width: `${sidebarWidth}px`, minWidth: `${sidebarWidth}px`, maxWidth: `${sidebarWidth}px` }}
      >
        <div className="flex items-center justify-center h-full">
          <div className="flex flex-col items-center gap-3">
            <Loader2 className="h-8 w-8 animate-spin text-gray-500" />
            <p className="text-sm text-gray-500">Loading events...</p>
          </div>
        </div>
      </div>
    );
  }

  return (
    <ComponentSidebar
      key={state.componentSidebar.selectedNodeId}
      isOpen={state.componentSidebar.isOpen}
      onClose={onSidebarClose || state.componentSidebar.close}
      latestEvents={latestEvents}
      nextInQueueEvents={nextInQueueEvents}
      nodeId={state.componentSidebar.selectedNodeId || undefined}
      iconSrc={sidebarData.iconSrc}
      iconSlug={sidebarData.iconSlug}
      iconColor={sidebarData.iconColor}
      iconBackground={sidebarData.iconBackground}
      totalInQueueCount={sidebarData.totalInQueueCount}
      totalInHistoryCount={sidebarData.totalInHistoryCount}
      hideQueueEvents={sidebarData.hideQueueEvents}
      getTabData={
        getTabData && state.componentSidebar.selectedNodeId
          ? (event) => getTabData(state.componentSidebar.selectedNodeId!, event)
          : undefined
      }
      onCancelQueueItem={onCancelQueueItem}
      onPushThrough={onPushThrough}
      onCancelExecution={onCancelExecution}
      supportsPushThrough={supportsPushThrough?.(state.componentSidebar.selectedNodeId!)}
      onRun={onRun ? () => onRun(state.componentSidebar.selectedNodeId!) : undefined}
      runDisabled={runDisabled}
      runDisabledTooltip={runDisabledTooltip}
      onDuplicate={onDuplicate ? () => onDuplicate(state.componentSidebar.selectedNodeId!) : undefined}
      onDocs={onDocs ? () => onDocs(state.componentSidebar.selectedNodeId!) : undefined}
      onConfigure={
        onConfigure && sidebarData?.isComposite ? () => onConfigure(state.componentSidebar.selectedNodeId!) : undefined
      }
      onDeactivate={onDeactivate ? () => onDeactivate(state.componentSidebar.selectedNodeId!) : undefined}
      onToggleView={onToggleView ? () => onToggleView(state.componentSidebar.selectedNodeId!) : undefined}
      onDelete={onDelete ? () => onDelete(state.componentSidebar.selectedNodeId!) : undefined}
      getAllHistoryEvents={() => getAllHistoryEvents?.(state.componentSidebar.selectedNodeId!) || []}
      onLoadMoreHistory={() => onLoadMoreHistory?.(state.componentSidebar.selectedNodeId!)}
      getHasMoreHistory={() => getHasMoreHistory?.(state.componentSidebar.selectedNodeId!) || false}
      getLoadingMoreHistory={() => getLoadingMoreHistory?.(state.componentSidebar.selectedNodeId!) || false}
      onLoadMoreQueue={() => onLoadMoreQueue?.(state.componentSidebar.selectedNodeId!)}
      getAllQueueEvents={() => getAllQueueEvents?.(state.componentSidebar.selectedNodeId!) || []}
      getHasMoreQueue={() => getHasMoreQueue?.(state.componentSidebar.selectedNodeId!) || false}
      getLoadingMoreQueue={() => getLoadingMoreQueue?.(state.componentSidebar.selectedNodeId!) || false}
      onReEmit={onReEmit}
      loadExecutionChain={loadExecutionChain}
      getExecutionState={
        getExecutionState ? (nodeId: string, execution: any) => getExecutionState(nodeId, execution) : undefined
      }
      showSettingsTab={true}
      nodeConfigMode="edit"
      nodeName={editingNodeData?.nodeName || ""}
      nodeLabel={editingNodeData?.displayLabel}
      nodeConfiguration={editingNodeData?.configuration || {}}
      nodeConfigurationFields={editingNodeData?.configurationFields || []}
      onNodeConfigSave={onSaveConfiguration}
      onNodeConfigCancel={undefined}
      onEdit={onEdit ? () => onEdit(state.componentSidebar.selectedNodeId!) : undefined}
      domainId={organizationId}
      domainType="DOMAIN_TYPE_ORGANIZATION"
      customField={
        // For template nodes, derive customField from buildingBlock
        templateNodeId && newNodeData
          ? getTemplateCustomField(newNodeData.buildingBlock) || undefined
          : getCustomField && state.componentSidebar.selectedNodeId
            ? getCustomField(state.componentSidebar.selectedNodeId) || undefined
            : undefined
      }
      appName={editingNodeData?.appName}
      appInstallationRef={editingNodeData?.appInstallationRef}
      installedApplications={installedApplications}
      currentTab={currentTab}
      onTabChange={onTabChange}
      templateNodeId={templateNodeId}
      onCancelTemplate={onCancelTemplate}
      newNodeData={newNodeData}
      workflowNodes={workflowNodes}
      components={components}
      triggers={triggers}
      blueprints={blueprints}
    />
  );
}

function CanvasContentHeader({
  state,
  onSave,
  onUndo,
  canUndo,
  organizationId,
  unsavedMessage,
  saveIsPrimary,
  saveButtonHidden,
}: {
  state: CanvasPageState;
  onSave?: (nodes: CanvasNode[]) => void;
  onUndo?: () => void;
  canUndo?: boolean;
  organizationId?: string;
  unsavedMessage?: string;
  saveIsPrimary?: boolean;
  saveButtonHidden?: boolean;
}) {
  const stateRef = useRef(state);
  stateRef.current = state;

  const handleSave = useCallback(() => {
    if (onSave) {
      onSave(stateRef.current.nodes);
    }
  }, [onSave]);

  const handleLogoClick = useCallback(() => {
    if (organizationId) {
      window.location.href = `/${organizationId}`;
    }
  }, [organizationId]);

  return (
    <Header
      breadcrumbs={state.breadcrumbs}
      onSave={onSave ? handleSave : undefined}
      onUndo={onUndo}
      canUndo={canUndo}
      onLogoClick={organizationId ? handleLogoClick : undefined}
      organizationId={organizationId}
      unsavedMessage={unsavedMessage}
      saveIsPrimary={saveIsPrimary}
      saveButtonHidden={saveButtonHidden}
    />
  );
}

function CanvasContent({
  state,
  onSave,
  onNodeEdit,
  onNodeDelete,
  onEdgeCreate,
  hideHeader,
  onRun,
  onDuplicate,
  onConfigure,
  onDeactivate,
  onToggleView,
  onToggleCollapse,
  onBuildingBlockDrop,
  onBuildingBlocksSidebarToggle,
  onConnectionDropInEmptySpace,
  onZoomChange,
  hasFitToViewRef,
  viewportRefProp,
  templateNodeId,
  runDisabled,
  runDisabledTooltip,
  onPendingConnectionNodeClick,
  onTemplateNodeClick,
}: {
  state: CanvasPageState;
  onSave?: (nodes: CanvasNode[]) => void;
  onNodeEdit: (nodeId: string) => void;
  onNodeDelete?: (nodeId: string) => void;
  onEdgeCreate?: (sourceId: string, targetId: string, sourceHandle?: string | null) => void;
  hideHeader?: boolean;
  onRun?: (nodeId: string) => void;
  onDuplicate?: (nodeId: string) => void;
  onConfigure?: (nodeId: string) => void;
  onDeactivate?: (nodeId: string) => void;
  onToggleView?: (nodeId: string) => void;
  onToggleCollapse?: () => void;
  onDelete?: (nodeId: string) => void;
  onBuildingBlockDrop?: (block: BuildingBlock, position?: { x: number; y: number }) => void;
  onBuildingBlocksSidebarToggle?: (open: boolean) => void;
  onConnectionDropInEmptySpace?: (
    position: { x: number; y: number },
    sourceConnection: { nodeId: string; handleId: string | null },
  ) => void;
  onZoomChange?: (zoom: number) => void;
  hasFitToViewRef: React.MutableRefObject<boolean>;
  viewportRefProp?: React.MutableRefObject<{ x: number; y: number; zoom: number } | undefined>;
  templateNodeId?: string | null;
  runDisabled?: boolean;
  runDisabledTooltip?: string;
  onPendingConnectionNodeClick?: (nodeId: string) => void;
  onTemplateNodeClick?: (nodeId: string) => void;
}) {
  const { fitView, screenToFlowPosition, getViewport } = useReactFlow();

  // Use refs to avoid recreating callbacks when state changes
  const stateRef = useRef(state);
  stateRef.current = state;

  // Use viewport ref from props if provided, otherwise create local one
  const viewportRef = viewportRefProp || useRef<{ x: number; y: number; zoom: number } | undefined>(undefined);

  if (!viewportRef.current && (stateRef.current.nodes?.length ?? 0) === 0) {
    viewportRef.current = { x: 0, y: 0, zoom: DEFAULT_CANVAS_ZOOM };
  }

  // Use viewport from ref as the state value
  const viewport = viewportRef.current;

  // Track if we've initialized to prevent flicker
  const [isInitialized, setIsInitialized] = useState(hasFitToViewRef.current);

  const handleNodeExpand = useCallback((nodeId: string) => {
    const node = stateRef.current.nodes?.find((n) => n.id === nodeId);
    if (node && stateRef.current.onNodeExpand) {
      stateRef.current.onNodeExpand(nodeId, node.data);
    }
  }, []);

  const handleNodeClick = useCallback(
    (nodeId: string) => {
      // Check if this is a pending connection node
      const clickedNode = stateRef.current.nodes?.find((n) => n.id === nodeId);
      const isPendingConnection = clickedNode?.data?.isPendingConnection;

      // Check if this is a template node
      const isTemplateNode = clickedNode?.data?.isTemplate && !clickedNode?.data?.isPendingConnection;

      // Check if the current template is a configured template (not just pending connection)
      const currentTemplateNode = templateNodeId ? stateRef.current.nodes?.find((n) => n.id === templateNodeId) : null;
      const isCurrentTemplateConfigured =
        currentTemplateNode?.data?.isTemplate && !currentTemplateNode?.data?.isPendingConnection;

      // Allow switching to pending connection nodes or other template nodes even if there's a configured template
      // But block switching to other regular/real nodes
      if (isCurrentTemplateConfigured && nodeId !== templateNodeId && !isPendingConnection && !isTemplateNode) {
        return;
      }

      if (isPendingConnection && onPendingConnectionNodeClick) {
        // Notify parent that a pending connection node was clicked
        onPendingConnectionNodeClick(nodeId);
      } else {
        if (isTemplateNode && onTemplateNodeClick) {
          // Notify parent to restore template state
          onTemplateNodeClick(nodeId);
        } else {
          // Regular node click
          stateRef.current.componentSidebar.open(nodeId);

          // Close building blocks sidebar when clicking on a regular node
          if (onBuildingBlocksSidebarToggle) {
            onBuildingBlocksSidebarToggle(false);
          }
        }
      }

      stateRef.current.setNodes((nodes) =>
        nodes.map((node) => ({
          ...node,
          selected: node.id === nodeId,
        })),
      );
    },
    [templateNodeId, onBuildingBlocksSidebarToggle, onPendingConnectionNodeClick, onTemplateNodeClick],
  );

  const onRunRef = useRef(onRun);
  onRunRef.current = onRun;

  const onNodeEditRef = useRef(onNodeEdit);
  onNodeEditRef.current = onNodeEdit;

  const onNodeDeleteRef = useRef(onNodeDelete);
  onNodeDeleteRef.current = onNodeDelete;

  const onDuplicateRef = useRef(onDuplicate);
  onDuplicateRef.current = onDuplicate;

  const onConfigureRef = useRef(onConfigure);
  onConfigureRef.current = onConfigure;

  const onDeactivateRef = useRef(onDeactivate);
  onDeactivateRef.current = onDeactivate;

  const onToggleViewRef = useRef(onToggleView);
  onToggleViewRef.current = onToggleView;

  const handleSave = useCallback(() => {
    if (onSave) {
      onSave(stateRef.current.nodes);
    }
  }, [onSave]);

  const handleConnect = useCallback(
    (connection: any) => {
      connectionCompletedRef.current = true;
      if (onEdgeCreate && connection.source && connection.target) {
        onEdgeCreate(connection.source, connection.target, connection.sourceHandle);
      }
    },
    [onEdgeCreate],
  );

  const handleDragOver = useCallback((event: React.DragEvent) => {
    event.preventDefault();
    event.dataTransfer.dropEffect = "move";
  }, []);

  const handleDrop = useCallback(
    (event: React.DragEvent) => {
      event.preventDefault();

      const blockData = event.dataTransfer.getData("application/reactflow");
      if (!blockData || !onBuildingBlockDrop) {
        return;
      }

      try {
        const block: BuildingBlock = JSON.parse(blockData);
        // Get the drop position from the cursor
        const cursorPosition = screenToFlowPosition({
          x: event.clientX,
          y: event.clientY,
        });

        // Adjust position to place node exactly where preview was shown
        // The drag preview has cursor at (width/2, 30px) from top-left
        // So we need to offset by those amounts to get the node's top-left corner
        const nodeWidth = 420; // Matches drag preview width
        const cursorOffsetY = 30; // Y offset used in drag preview
        const position = {
          x: cursorPosition.x - nodeWidth / 2,
          y: cursorPosition.y - cursorOffsetY,
        };

        onBuildingBlockDrop(block, position);
      } catch (error) {
        console.error("Failed to parse building block data:", error);
      }
    },
    [onBuildingBlockDrop, screenToFlowPosition],
  );

  const handleMove = useCallback(
    (_event: any, newViewport: { x: number; y: number; zoom: number }) => {
      // Store the viewport in the ref (which persists across re-renders)
      viewportRef.current = newViewport;

      if (onZoomChange) {
        onZoomChange(newViewport.zoom);
      }
    },
    [onZoomChange, viewportRef],
  );

  const handleToggleCollapse = useCallback(() => {
    state.toggleCollapse();
    onToggleCollapse?.();
  }, [state.toggleCollapse, onToggleCollapse]);

  const handlePaneClick = useCallback(() => {
    // do not close sidebar while we are creating a new component
    if (templateNodeId) return;

    stateRef.current.componentSidebar.close();

    // Also close building blocks sidebar when clicking on canvas
    if (onBuildingBlocksSidebarToggle) {
      onBuildingBlocksSidebarToggle(false);
    }

    // Clear ReactFlow's selection state
    stateRef.current.setNodes((nodes) =>
      nodes.map((node) => ({
        ...node,
        selected: false,
      })),
    );
  }, [templateNodeId, onBuildingBlocksSidebarToggle]);

  // Handle fit to view on ReactFlow initialization
  const handleInit = useCallback(
    (reactFlowInstance: any) => {
      if (!hasFitToViewRef.current) {
        const hasNodes = (stateRef.current.nodes?.length ?? 0) > 0;

        if (hasNodes) {
          // Fit to view but don't zoom in too much (max zoom of 1.0)
          fitView({ maxZoom: 1.0, padding: 0.5 });

          // Store the initial viewport after fit
          const initialViewport = getViewport();
          viewportRef.current = initialViewport;

          if (onZoomChange) {
            onZoomChange(initialViewport.zoom);
          }
        } else {
          const defaultViewport = viewportRef.current ?? { x: 0, y: 0, zoom: DEFAULT_CANVAS_ZOOM };
          viewportRef.current = defaultViewport;
          reactFlowInstance.setViewport(defaultViewport);

          if (onZoomChange) {
            onZoomChange(defaultViewport.zoom);
          }
        }

        hasFitToViewRef.current = true;
        setIsInitialized(true);
      } else {
        // If we've already fit to view once and have a stored viewport, restore it
        if (viewportRef.current) {
          reactFlowInstance.setViewport(viewportRef.current);
        }
        setIsInitialized(true);
      }
    },
    [fitView, getViewport, onZoomChange, hasFitToViewRef, viewportRef],
  );

  // Store callback handlers in a ref so they can be accessed without being in node data
  const callbacksRef = useRef({
    handleNodeExpand,
    handleNodeClick,
    onNodeEdit: onNodeEditRef,
    onNodeDelete: onNodeDeleteRef,
    onRun: onRunRef,
    onDuplicate: onDuplicateRef,
    onConfigure: onConfigureRef,
    onDeactivate: onDeactivateRef,
    onToggleView: onToggleViewRef,
    aiState: state.ai,
    runDisabled,
    runDisabledTooltip,
  });
  callbacksRef.current = {
    handleNodeExpand,
    handleNodeClick,
    onNodeEdit: onNodeEditRef,
    onNodeDelete: onNodeDeleteRef,
    onRun: onRunRef,
    onDuplicate: onDuplicateRef,
    onConfigure: onConfigureRef,
    onDeactivate: onDeactivateRef,
    onToggleView: onToggleViewRef,
    aiState: state.ai,
    runDisabled,
    runDisabledTooltip,
  };

  // Just pass the state nodes directly - callbacks will be added in nodeTypes
  const [hoveredEdgeId, setHoveredEdgeId] = useState<string | null>(null);
  const [connectingFrom, setConnectingFrom] = useState<{
    nodeId: string;
    handleId: string | null;
    handleType: "source" | "target" | null;
  } | null>(null);

  // Track connection completion for empty space drop detection
  const connectionCompletedRef = useRef(false);
  const connectingFromRef = useRef<{
    nodeId: string;
    handleId: string | null;
    handleType: "source" | "target" | null;
  } | null>(null);

  const handleEdgeMouseEnter = useCallback((_event: React.MouseEvent, edge: any) => {
    setHoveredEdgeId(edge.id);
  }, []);

  const handleEdgeMouseLeave = useCallback(() => {
    setHoveredEdgeId(null);
  }, []);

  const handleConnectStart = useCallback(
    (
      _event: any,
      params: { nodeId: string | null; handleId: string | null; handleType: "source" | "target" | null },
    ) => {
      if (params.nodeId) {
        const connectionInfo = { nodeId: params.nodeId, handleId: params.handleId, handleType: params.handleType };
        setConnectingFrom(connectionInfo);
        connectingFromRef.current = connectionInfo;
      }
    },
    [],
  );

  const handleConnectEnd = useCallback(
    (event: MouseEvent | TouchEvent) => {
      const currentConnectingFrom = connectingFromRef.current;

      if (currentConnectingFrom && !connectionCompletedRef.current) {
        const mouseEvent = event as MouseEvent;
        const canvasPosition = screenToFlowPosition({
          x: mouseEvent.clientX,
          y: mouseEvent.clientY,
        });

        if (onConnectionDropInEmptySpace) {
          onConnectionDropInEmptySpace(canvasPosition, currentConnectingFrom);
        }
      }

      setConnectingFrom(null);
      connectingFromRef.current = null;
      connectionCompletedRef.current = false;
    },
    [screenToFlowPosition, onConnectionDropInEmptySpace],
  );

  // Find the hovered edge to get its source and target
  const hoveredEdge = useMemo(() => {
    if (!hoveredEdgeId) return null;
    return state.edges?.find((e) => e.id === hoveredEdgeId);
  }, [hoveredEdgeId, state.edges]);

  const nodesWithCallbacks = useMemo(() => {
    return state.nodes.map((node) => ({
      ...node,
      data: {
        ...node.data,
        _callbacksRef: callbacksRef,
        _hoveredEdge: hoveredEdge,
        _connectingFrom: connectingFrom,
        _allEdges: state.edges,
      },
    }));
  }, [state.nodes, hoveredEdge, connectingFrom, state.edges]);

  const edgeTypes = useMemo(
    () => ({
      custom: CustomEdge,
    }),
    [],
  );
  const styledEdges = useMemo(
    () =>
      state.edges?.map((e) => ({
        ...e,
        ...EDGE_STYLE,
        data: { ...e.data, isHovered: e.id === hoveredEdgeId },
        zIndex: e.id === hoveredEdgeId ? 1000 : 0,
      })),
    [state.edges, hoveredEdgeId],
  );

  return (
    <>
      {/* Header */}
      {!hideHeader && <Header breadcrumbs={state.breadcrumbs} onSave={onSave ? handleSave : undefined} />}

      <div className={hideHeader ? "h-full" : "pt-12 h-full"}>
        <div className="h-full w-full">
          <ReactFlow
            nodes={nodesWithCallbacks}
            edges={styledEdges}
            nodeTypes={nodeTypes}
            edgeTypes={edgeTypes}
            minZoom={0.4}
            maxZoom={1.5}
            zoomOnScroll={true}
            zoomOnPinch={true}
            zoomOnDoubleClick={false}
            panOnScroll={true}
            panOnDrag={true}
            selectionOnDrag={false}
            panOnScrollSpeed={0.8}
            nodesDraggable={true}
            nodesConnectable={!!onEdgeCreate}
            elementsSelectable={true}
            onNodesChange={state.onNodesChange}
            onEdgesChange={state.onEdgesChange}
            onConnect={handleConnect}
            onConnectStart={handleConnectStart}
            onConnectEnd={handleConnectEnd}
            onDragOver={handleDragOver}
            onDrop={handleDrop}
            onMove={handleMove}
            onInit={handleInit}
            onPaneClick={handlePaneClick}
            onEdgeMouseEnter={handleEdgeMouseEnter}
            onEdgeMouseLeave={handleEdgeMouseLeave}
            defaultViewport={viewport}
            fitView={false}
            style={{ opacity: isInitialized ? 1 : 0 }}
          >
            <Background gap={8} size={2} bgColor="#F1F5F9" color="#d9d9d9ff" />
            <ZoomSlider position="bottom-left" orientation="horizontal">
              <Tooltip>
                <TooltipTrigger asChild>
                  <Button
                    variant="ghost"
                    size="icon-sm"
                    onClick={handleToggleCollapse}
                  >
                    {state.isCollapsed ? <ScanText className="h-3 w-3" /> : <ScanLine className="h-3 w-3" />}
                  </Button>
                </TooltipTrigger>
                <TooltipContent>{state.isCollapsed ? "Switch components to Detailed view" : "Switch components to Compact view"}</TooltipContent>
              </Tooltip>
            </ZoomSlider>
          </ReactFlow>
        </div>
      </div>
    </>
  );
}

export type { BuildingBlock } from "../BuildingBlocksSidebar";
export { CanvasPage };
