/* eslint-disable @typescript-eslint/no-explicit-any */
import {
  Background,
  Panel,
  ReactFlow,
  ReactFlowProvider,
  useReactFlow,
  type Edge as ReactFlowEdge,
  type Node as ReactFlowNode,
  type NodeChange,
  type EdgeChange,
} from "@xyflow/react";

import { CircleX, Loader2, ScanLine, ScanText, ScrollText, TriangleAlert, Workflow } from "lucide-react";
import { ZoomSlider } from "@/components/zoom-slider";
import { NodeSearch } from "@/components/node-search";
import { Button } from "@/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";

import {
  ConfigurationField,
  CanvasesCanvasNodeExecution,
  ComponentsNode,
  ComponentsComponent,
  TriggersTrigger,
  BlueprintsBlueprint,
  ComponentsIntegrationRef,
  OrganizationsIntegration,
} from "@/api-client";
import { parseDefaultValues } from "@/utils/components";
import { getActiveNoteId, restoreActiveNoteFocus } from "@/ui/annotationComponent/noteFocus";
import { AiSidebar } from "../ai";
import {
  AiCanvasOperation,
  BuildingBlock,
  BuildingBlockCategory,
  BuildingBlocksSidebar,
} from "../BuildingBlocksSidebar";
import { ComponentSidebar } from "../componentSidebar";
import { TabData } from "../componentSidebar/SidebarEventItem/SidebarEventItem";
import { EmitEventModal } from "../EmitEventModal";
import { EventState, EventStateMap } from "../componentBase";
import { Block, BlockData } from "./Block";
import "./canvas-reset.css";
import { CustomEdge } from "./CustomEdge";
import { Header, type BreadcrumbItem } from "./Header";
import { Simulation } from "./storybooks/useSimulation";
import { CanvasPageState, useCanvasState } from "./useCanvasState";
import { SidebarEvent } from "../componentSidebar/types";
import { CanvasLogSidebar, type LogEntry, type LogScopeFilter, type LogTypeFilter } from "../CanvasLogSidebar";

export interface SidebarData {
  latestEvents: SidebarEvent[];
  nextInQueueEvents: SidebarEvent[];
  title: string;
  iconSrc?: string;
  iconSlug?: string;
  iconColor?: string;
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

interface FocusRequest {
  nodeId: string;
  requestId: number;
  tab?: "latest" | "settings" | "execution-chain";
  executionChain?: {
    eventId: string;
    executionId?: string | null;
    triggerEvent?: SidebarEvent | null;
  };
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
  integrationName?: string;
  blockName?: string;
  integrationRef?: ComponentsIntegrationRef;
}

export interface NewNodeData {
  icon?: string;
  buildingBlock: BuildingBlock;
  nodeName: string;
  displayLabel?: string;
  configuration: Record<string, any>;
  position?: { x: number; y: number };
  integrationName?: string;
  integrationRef?: ComponentsIntegrationRef;
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
  headerBanner?: React.ReactNode;
  organizationId?: string;
  canvasId?: string;
  unsavedMessage?: string;
  saveIsPrimary?: boolean;
  saveButtonHidden?: boolean;
  saveDisabled?: boolean;
  saveDisabledTooltip?: string;
  isAutoSaveEnabled?: boolean;
  onToggleAutoSave?: () => void;
  autoSaveDisabled?: boolean;
  autoSaveDisabledTooltip?: string;
  readOnly?: boolean;
  canReadIntegrations?: boolean;
  canCreateIntegrations?: boolean;
  canUpdateIntegrations?: boolean;
  onExportYamlCopy?: (nodes: CanvasNode[]) => void;
  onExportYamlDownload?: (nodes: CanvasNode[]) => void;
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
  getAutocompleteExampleObj?: (nodeId: string) => Record<string, unknown> | null;
  onNodeConfigurationSave?: (
    nodeId: string,
    configuration: Record<string, any>,
    nodeName: string,
    integrationRef?: ComponentsIntegrationRef,
  ) => void;
  onAnnotationUpdate?: (
    nodeId: string,
    updates: { text?: string; color?: string; width?: number; height?: number; x?: number; y?: number },
  ) => void;
  onAnnotationBlur?: () => void;
  getCustomField?: (
    nodeId: string,
    onRun?: (initialData?: string) => void,
    integration?: OrganizationsIntegration,
  ) => (() => React.ReactNode) | null;
  onSave?: (nodes: CanvasNode[]) => void;
  integrations?: OrganizationsIntegration[];
  onEdgeCreate?: (sourceId: string, targetId: string, sourceHandle?: string | null) => void;
  onNodeDelete?: (nodeId: string) => void;
  onEdgeDelete?: (edgeIds: string[]) => void;
  onResolveExecutionErrors?: (executionIds: string[]) => void;
  onNodePositionChange?: (nodeId: string, position: { x: number; y: number }) => void;
  onNodesPositionChange?: (updates: Array<{ nodeId: string; position: { x: number; y: number } }>) => void;
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
  onTogglePause?: (nodeId: string) => void;
  onToggleView?: (nodeId: string) => void;
  onToggleCollapse?: () => void;
  onAutoLayout?: (selectedNodeIDs: string[]) => void | Promise<void>;
  onReEmit?: (nodeId: string, eventOrExecutionId: string) => void;

  ai?: AiProps;

  // Building blocks for adding new nodes
  buildingBlocks: BuildingBlockCategory[];
  showAiBuilderTab?: boolean;
  onNodeAdd?: (newNodeData: NewNodeData) => Promise<string>;
  onApplyAiOperations?: (operations: AiCanvasOperation[]) => Promise<void>;
  onPlaceholderAdd?: (data: {
    position: { x: number; y: number };
    sourceNodeId: string;
    sourceHandleId: string | null;
  }) => Promise<string>;
  onPlaceholderConfigure?: (data: {
    placeholderId: string;
    buildingBlock: BuildingBlock;
    nodeName: string;
    configuration: Record<string, any>;
    integrationName?: string;
  }) => Promise<void>;

  // Refs to persist state across re-renders
  hasFitToViewRef?: React.MutableRefObject<boolean>;
  hasUserToggledSidebarRef?: React.MutableRefObject<boolean>;
  isSidebarOpenRef?: React.MutableRefObject<boolean | null>;
  viewportRef?: React.MutableRefObject<{ x: number; y: number; zoom: number } | undefined>;

  // Optional: control and observe component sidebar state
  onSidebarChange?: (isOpen: boolean, selectedNodeId: string | null) => void;
  initialSidebar?: { isOpen?: boolean; nodeId?: string | null };
  initialFocusNodeId?: string | null;

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
    execution: CanvasesCanvasNodeExecution,
  ) => { map: EventStateMap; state: EventState };

  // Workflow metadata for ExecutionChainPage
  workflowNodes?: ComponentsNode[];
  components?: ComponentsComponent[];
  triggers?: TriggersTrigger[];
  blueprints?: BlueprintsBlueprint[];

  logEntries?: LogEntry[];
  focusRequest?: FocusRequest | null;
  onExecutionChainHandled?: () => void;
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
        showHeader={callbacks?.showHeader}
        onExpand={callbacks.handleNodeExpand}
        onClick={() => callbacks.handleNodeClick(nodeProps.id)}
        onEdit={() => callbacks.onNodeEdit.current?.(nodeProps.id)}
        onDelete={callbacks.onNodeDelete.current ? () => callbacks.onNodeDelete.current?.(nodeProps.id) : undefined}
        onRun={callbacks.onRun.current ? () => callbacks.onRun.current?.(nodeProps.id) : undefined}
        onDuplicate={callbacks.onDuplicate.current ? () => callbacks.onDuplicate.current?.(nodeProps.id) : undefined}
        onConfigure={callbacks.onConfigure.current ? () => callbacks.onConfigure.current?.(nodeProps.id) : undefined}
        onDeactivate={callbacks.onDeactivate.current ? () => callbacks.onDeactivate.current?.(nodeProps.id) : undefined}
        onTogglePause={
          callbacks.onTogglePause.current ? () => callbacks.onTogglePause.current?.(nodeProps.id) : undefined
        }
        onToggleView={callbacks.onToggleView.current ? () => callbacks.onToggleView.current?.(nodeProps.id) : undefined}
        onToggleCollapse={
          callbacks.onToggleView.current ? () => callbacks.onToggleView.current?.(nodeProps.id) : undefined
        }
        onAnnotationUpdate={
          callbacks.onAnnotationUpdate.current
            ? (nodeId, updates) => callbacks.onAnnotationUpdate.current?.(nodeId, updates)
            : undefined
        }
        onAnnotationBlur={callbacks.onAnnotationBlur.current ? () => callbacks.onAnnotationBlur.current?.() : undefined}
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
  const readOnly = props.readOnly ?? false;
  const [currentTab, setCurrentTab] = useState<"latest" | "settings">("latest");
  const [templateNodeId, setTemplateNodeId] = useState<string | null>(null);
  const [highlightedNodeIds, setHighlightedNodeIds] = useState<Set<string>>(new Set());
  const canvasWrapperRef = useRef<HTMLDivElement | null>(null);

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
    initialData?: string;
  } | null>(null);

  useEffect(() => {
    if (!props.focusRequest?.tab || props.focusRequest.tab === "execution-chain") {
      return;
    }

    setCurrentTab(props.focusRequest.tab);
  }, [props.focusRequest?.requestId, props.focusRequest?.tab]);

  const handleNodeEdit = useCallback(
    (nodeId: string) => {
      // Check if this is a placeholder - if so, open building blocks sidebar instead
      const workflowNode = props.workflowNodes?.find((n) => n.id === nodeId);
      const isPlaceholder = workflowNode?.name === "New Component" && !workflowNode.component?.name;

      if (isPlaceholder) {
        // For placeholders, open building blocks sidebar
        setTemplateNodeId(nodeId);
        setIsBuildingBlocksSidebarOpen(true);
        state.componentSidebar.close();
        return;
      }

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
    [props, state.componentSidebar, setTemplateNodeId, setIsBuildingBlocksSidebarOpen, setCurrentTab],
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
    (nodeId?: string, initialData?: string) => {
      // Hard guard: if running is disabled (e.g., unsaved changes), do nothing
      if (props.runDisabled) return;

      // Check for pending run data from custom field
      // Note: This uses a window property as a workaround to pass nodeId and initialData
      // through the onRun callback chain without breaking existing signatures
      const pendingData = (window as any).__pendingRunData;
      const actualNodeId = nodeId || pendingData?.nodeId;
      const actualInitialData = initialData || pendingData?.initialData;

      if (!actualNodeId) return;

      // Find the node to get its name and channels
      const node = state.nodes.find((n) => n.id === actualNodeId);
      if (!node) return;

      const nodeName = (node.data as any).label || actualNodeId;
      const channels = (node.data as any).outputChannels || ["default"];

      setEmitModalData({
        nodeId: actualNodeId,
        nodeName,
        channels,
        initialData: actualInitialData,
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
    async (position: { x: number; y: number }, sourceConnection: { nodeId: string; handleId: string | null }) => {
      if (readOnly) return;
      if (!sourceConnection || !props.onPlaceholderAdd) return;

      // Save placeholder immediately to backend
      const placeholderId = await props.onPlaceholderAdd({
        position: { x: position.x, y: position.y - 30 },
        sourceNodeId: sourceConnection.nodeId,
        sourceHandleId: sourceConnection.handleId,
      });

      // Set as template node and open building blocks sidebar
      setTemplateNodeId(placeholderId);
      setIsBuildingBlocksSidebarOpen(true);
      state.componentSidebar.close();
    },
    [props, state, setTemplateNodeId, setIsBuildingBlocksSidebarOpen, readOnly],
  );

  const handlePendingConnectionNodeClick = useCallback(
    (nodeId: string) => {
      if (readOnly) return;
      // For both placeholders and legacy pending connections:
      // Set this node as the active template so we can configure it when a building block is selected
      setTemplateNodeId(nodeId);

      // Open the BuildingBlocksSidebar so user can select a component
      setIsBuildingBlocksSidebarOpen(true);

      // Close ComponentSidebar since we're selecting a building block first
      state.componentSidebar.close();
    },
    [setTemplateNodeId, setIsBuildingBlocksSidebarOpen, state.componentSidebar, readOnly],
  );

  const handleBuildingBlockClick = useCallback(
    async (block: BuildingBlock) => {
      if (readOnly) return;
      if (!templateNodeId) {
        return;
      }

      const defaultConfiguration = (() => {
        const defaults = parseDefaultValues(block.configuration || []);
        const filtered = { ...defaults };
        block.configuration?.forEach((field) => {
          if (field.name && field.togglable) {
            delete filtered[field.name];
          }
        });
        return filtered;
      })();

      // Check if templateNodeId is a placeholder (persisted node) or legacy pending connection (local-only)
      const workflowNode = props.workflowNodes?.find((n) => n.id === templateNodeId);
      const isPlaceholder = workflowNode?.name === "New Component" && !workflowNode.component?.name;

      if (isPlaceholder && props.onPlaceholderConfigure) {
        // Handle placeholder node (persisted)
        await props.onPlaceholderConfigure({
          placeholderId: templateNodeId,
          buildingBlock: block,
          nodeName: block.name || "",
          configuration: defaultConfiguration,
          integrationName: block.integrationName,
        });

        setTemplateNodeId(null);
        setIsBuildingBlocksSidebarOpen(false);
        state.componentSidebar.open(templateNodeId);
        setCurrentTab("settings");
        return;
      }

      // Check for local pending connection nodes (legacy)
      const pendingNode = state.nodes.find((n) => n.id === templateNodeId && n.data.isPendingConnection);

      if (pendingNode) {
        // Save immediately with defaults
        if (props.onNodeAdd) {
          const newNodeId = await props.onNodeAdd({
            buildingBlock: block,
            nodeName: block.name || "",
            configuration: defaultConfiguration,
            position: pendingNode.position,
            sourceConnection: pendingNode.data.sourceConnection as
              | { nodeId: string; handleId: string | null }
              | undefined,
            integrationName: block.integrationName,
          });

          // Remove pending node
          state.setNodes((nodes) => nodes.filter((n) => n.id !== templateNodeId));

          // Clear template state
          setTemplateNodeId(null);

          // Close building blocks sidebar
          setIsBuildingBlocksSidebarOpen(false);

          // Open component sidebar for the new node
          state.componentSidebar.open(newNodeId);
          setCurrentTab("settings");
        }
      }
    },
    [templateNodeId, state, props, setCurrentTab, setIsBuildingBlocksSidebarOpen, readOnly],
  );

  const handleAddNote = useCallback(async () => {
    if (readOnly) return;
    if (!props.onNodeAdd) return;

    const viewport = props.viewportRef?.current ?? { x: 0, y: 0, zoom: DEFAULT_CANVAS_ZOOM };
    const canvasRect = canvasWrapperRef.current?.getBoundingClientRect();
    const zoom = viewport.zoom || DEFAULT_CANVAS_ZOOM;
    const visibleWidth = canvasRect?.width ?? window.innerWidth;
    const visibleHeight = canvasRect?.height ?? window.innerHeight;
    const visibleBounds = {
      minX: (0 - viewport.x) / zoom,
      minY: (0 - viewport.y) / zoom,
      maxX: (visibleWidth - viewport.x) / zoom,
      maxY: (visibleHeight - viewport.y) / zoom,
    };

    const noteSize = { width: 320, height: 160 };
    const basePosition = {
      x: (visibleWidth / 2 - viewport.x) / zoom - noteSize.width / 2,
      y: (visibleHeight / 2 - viewport.y) / zoom - noteSize.height / 2,
    };

    const nodes = state.nodes || [];
    const padding = 16;
    const intersects = (pos: { x: number; y: number }) => {
      const bounds = {
        minX: pos.x - padding,
        minY: pos.y - padding,
        maxX: pos.x + noteSize.width + padding,
        maxY: pos.y + noteSize.height + padding,
      };
      return nodes.some((node) => {
        const width = node.width ?? 240;
        const height = node.height ?? 120;
        const nodeBounds = {
          minX: node.position.x,
          minY: node.position.y,
          maxX: node.position.x + width,
          maxY: node.position.y + height,
        };
        return !(
          bounds.maxX < nodeBounds.minX ||
          bounds.minX > nodeBounds.maxX ||
          bounds.maxY < nodeBounds.minY ||
          bounds.minY > nodeBounds.maxY
        );
      });
    };

    const clampToVisible = (pos: { x: number; y: number }) => {
      const minX = visibleBounds.minX + padding;
      const minY = visibleBounds.minY + padding;
      const maxX = visibleBounds.maxX - noteSize.width - padding;
      const maxY = visibleBounds.maxY - noteSize.height - padding;
      return {
        x: Math.min(Math.max(pos.x, minX), maxX),
        y: Math.min(Math.max(pos.y, minY), maxY),
      };
    };

    let position = clampToVisible(basePosition);
    const step = 40;
    const maxRings = 8;
    if (intersects(position)) {
      let found = false;
      for (let ring = 1; ring <= maxRings && !found; ring += 1) {
        for (let dx = -ring; dx <= ring && !found; dx += 1) {
          for (let dy = -ring; dy <= ring && !found; dy += 1) {
            if (Math.abs(dx) !== ring && Math.abs(dy) !== ring) continue;
            const candidate = clampToVisible({
              x: basePosition.x + dx * step,
              y: basePosition.y + dy * step,
            });
            if (!intersects(candidate)) {
              position = candidate;
              found = true;
            }
          }
        }
      }
    }

    const annotationBlock: BuildingBlock = {
      name: "annotation",
      label: "Annotation",
      type: "component",
      isLive: true,
    };

    await props.onNodeAdd({
      buildingBlock: annotationBlock,
      nodeName: "Note",
      configuration: {},
      position,
    });
  }, [props, state.nodes, props.viewportRef, readOnly]);

  const handleBuildingBlockDrop = useCallback(
    async (block: BuildingBlock, position?: { x: number; y: number }) => {
      if (readOnly) return;
      const defaultConfiguration = (() => {
        const defaults = parseDefaultValues(block.configuration || []);
        const filtered = { ...defaults };
        block.configuration?.forEach((field) => {
          if (field.name && field.togglable) {
            delete filtered[field.name];
          }
        });
        return filtered;
      })();

      // Save immediately with defaults
      if (props.onNodeAdd) {
        const newNodeId = await props.onNodeAdd({
          buildingBlock: block,
          nodeName: block.name || "",
          configuration: defaultConfiguration,
          position,
          integrationName: block.integrationName,
        });

        // Close building blocks sidebar
        setIsBuildingBlocksSidebarOpen(false);

        // Open component sidebar for the new node
        state.componentSidebar.open(newNodeId);
        setCurrentTab("settings");
      }
    },
    [state, props, setCurrentTab, setIsBuildingBlocksSidebarOpen, readOnly],
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
    (configuration: Record<string, any>, nodeName: string, integrationRef?: ComponentsIntegrationRef) => {
      if (editingNodeData && props.onNodeConfigurationSave) {
        props.onNodeConfigurationSave(editingNodeData.nodeId, configuration, nodeName, integrationRef);
        // Close the component sidebar after saving
        state.componentSidebar.close();
      }
    },
    [editingNodeData, props, state.componentSidebar],
  );

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
    <div ref={canvasWrapperRef} className="h-[100vh] w-[100vw] overflow-hidden sp-canvas relative flex flex-col">
      {/* Header at the top spanning full width */}
      <div className="relative z-30">
        <CanvasContentHeader
          state={state}
          onSave={props.onSave}
          onUndo={props.onUndo}
          canUndo={props.canUndo}
          organizationId={props.organizationId}
          unsavedMessage={props.unsavedMessage}
          saveIsPrimary={props.saveIsPrimary}
          saveButtonHidden={props.saveButtonHidden}
          saveDisabled={props.saveDisabled}
          saveDisabledTooltip={props.saveDisabledTooltip}
          isAutoSaveEnabled={props.isAutoSaveEnabled}
          onToggleAutoSave={props.onToggleAutoSave}
          autoSaveDisabled={props.autoSaveDisabled}
          autoSaveDisabledTooltip={props.autoSaveDisabledTooltip}
          onExportYamlCopy={props.onExportYamlCopy}
          onExportYamlDownload={props.onExportYamlDownload}
        />
        {props.headerBanner ? <div className="border-b border-black/20">{props.headerBanner}</div> : null}
      </div>

      {/* Main content area with sidebar and canvas */}
      <div className="flex-1 flex relative overflow-hidden">
        <BuildingBlocksSidebar
          isOpen={isBuildingBlocksSidebarOpen}
          onToggle={handleSidebarToggle}
          blocks={props.buildingBlocks || []}
          showAiBuilderTab={props.showAiBuilderTab}
          canvasId={props.canvasId}
          canvasNodes={state.nodes.map((node) => ({
            id: node.id,
            name: String((node.data as { nodeName?: string })?.nodeName || ""),
            label: String((node.data as { label?: string })?.label || ""),
            type: String((node.data as { type?: string })?.type || ""),
          }))}
          onApplyAiOperations={props.onApplyAiOperations}
          integrations={props.integrations}
          canvasZoom={canvasZoom}
          disabled={readOnly}
          disabledMessage="You don't have permission to edit this canvas."
          onBlockClick={handleBuildingBlockClick}
          onAddNote={handleAddNote}
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
              onAutoLayout={props.onAutoLayout}
              onRun={(nodeId) => handleNodeRun(nodeId)}
              onDuplicate={props.onDuplicate}
              onConfigure={props.onConfigure}
              onDeactivate={props.onDeactivate}
              onAnnotationUpdate={props.onAnnotationUpdate}
              onAnnotationBlur={props.onAnnotationBlur}
              onTogglePause={props.onTogglePause}
              runDisabled={props.runDisabled}
              runDisabledTooltip={props.runDisabledTooltip}
              onBuildingBlockDrop={handleBuildingBlockDrop}
              onBuildingBlocksSidebarToggle={handleSidebarToggle}
              onConnectionDropInEmptySpace={handleConnectionDropInEmptySpace}
              onPendingConnectionNodeClick={handlePendingConnectionNodeClick}
              onZoomChange={setCanvasZoom}
              hasFitToViewRef={hasFitToViewRef}
              viewportRefProp={props.viewportRef}
              highlightedNodeIds={highlightedNodeIds}
              workflowNodes={props.workflowNodes}
              setCurrentTab={setCurrentTab}
              onUndo={props.onUndo}
              canUndo={props.canUndo}
              organizationId={props.organizationId}
              unsavedMessage={props.unsavedMessage}
              saveIsPrimary={props.saveIsPrimary}
              saveButtonHidden={props.saveButtonHidden}
              saveDisabled={props.saveDisabled}
              saveDisabledTooltip={props.saveDisabledTooltip}
              isAutoSaveEnabled={props.isAutoSaveEnabled}
              onToggleAutoSave={props.onToggleAutoSave}
              autoSaveDisabled={props.autoSaveDisabled}
              autoSaveDisabledTooltip={props.autoSaveDisabledTooltip}
              readOnly={props.readOnly}
              logEntries={props.logEntries}
              focusRequest={props.focusRequest}
              onExecutionChainHandled={props.onExecutionChainHandled}
              initialFocusNodeId={props.initialFocusNodeId}
              onResolveExecutionErrors={props.onResolveExecutionErrors}
              title={props.title}
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
            getAutocompleteExampleObj={props.getAutocompleteExampleObj}
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
            organizationId={props.organizationId}
            getCustomField={props.getCustomField}
            integrations={props.integrations}
            workflowNodes={props.workflowNodes}
            components={props.components}
            triggers={props.triggers}
            blueprints={props.blueprints}
            onHighlightedNodesChange={setHighlightedNodeIds}
            focusRequest={props.focusRequest}
            onExecutionChainHandled={props.onExecutionChainHandled}
            readOnly={readOnly}
            canReadIntegrations={props.canReadIntegrations}
            canCreateIntegrations={props.canCreateIntegrations}
            canUpdateIntegrations={props.canUpdateIntegrations}
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
          initialData={emitModalData.initialData}
        />
      )}
    </div>
  );
}

function Sidebar({
  state,
  getSidebarData,
  loadSidebarData,
  getTabData,
  getAutocompleteExampleObj,
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
  organizationId,
  getCustomField,
  integrations,
  workflowNodes,
  components,
  triggers,
  blueprints,
  onHighlightedNodesChange,
  focusRequest,
  onExecutionChainHandled,
  readOnly,
  canReadIntegrations,
  canCreateIntegrations,
  canUpdateIntegrations,
}: {
  state: CanvasPageState;
  getSidebarData?: (nodeId: string) => SidebarData | null;
  loadSidebarData?: (nodeId: string) => void;
  getTabData?: (nodeId: string, event: SidebarEvent) => TabData | undefined;
  getAutocompleteExampleObj?: (nodeId: string) => Record<string, unknown> | null;
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
    execution: CanvasesCanvasNodeExecution,
  ) => { map: EventStateMap; state: EventState };
  onSidebarClose?: () => void;
  editingNodeData?: NodeEditData | null;
  onSaveConfiguration?: (configuration: Record<string, any>, nodeName: string) => void;
  onEdit?: (nodeId: string) => void;
  currentTab?: "latest" | "settings";
  onTabChange?: (tab: "latest" | "settings") => void;
  organizationId?: string;
  getCustomField?: (
    nodeId: string,
    onRun?: (initialData?: string) => void,
    integration?: OrganizationsIntegration,
  ) => (() => React.ReactNode) | null;
  integrations?: OrganizationsIntegration[];
  workflowNodes?: ComponentsNode[];
  components?: ComponentsComponent[];
  triggers?: TriggersTrigger[];
  blueprints?: BlueprintsBlueprint[];
  onHighlightedNodesChange?: (nodeIds: Set<string>) => void;
  focusRequest?: FocusRequest | null;
  onExecutionChainHandled?: () => void;
  readOnly?: boolean;
  canReadIntegrations?: boolean;
  canCreateIntegrations?: boolean;
  canUpdateIntegrations?: boolean;
}) {
  const sidebarData = useMemo(() => {
    if (!state.componentSidebar.selectedNodeId || !getSidebarData) {
      return null;
    }
    return getSidebarData(state.componentSidebar.selectedNodeId);
  }, [state.componentSidebar.selectedNodeId, getSidebarData]);

  const isAnnotationNode = useMemo(() => {
    if (!state.componentSidebar.selectedNodeId || !workflowNodes) {
      return false;
    }
    const selectedNode = workflowNodes.find((node) => node.id === state.componentSidebar.selectedNodeId);
    return selectedNode?.type === "TYPE_WIDGET" && selectedNode?.widget?.name === "annotation";
  }, [state.componentSidebar.selectedNodeId, workflowNodes]);

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

  const autocompleteExampleObj = useMemo(() => {
    if (!state.componentSidebar.selectedNodeId || !getAutocompleteExampleObj) {
      return undefined;
    }
    return getAutocompleteExampleObj(state.componentSidebar.selectedNodeId);
  }, [state.componentSidebar.selectedNodeId, getAutocompleteExampleObj]);

  if (!sidebarData) {
    return null;
  }

  // Show loading state when data is being fetched (skip for annotation nodes)
  if (sidebarData.isLoading && currentTab === "latest" && !isAnnotationNode) {
    const saved = localStorage.getItem(COMPONENT_SIDEBAR_WIDTH_STORAGE_KEY);
    const sidebarWidth = saved ? parseInt(saved, 10) : 450;

    return (
      <div
        className="border-l-1 border-border absolute right-0 top-0 h-full z-20 overflow-y-auto overflow-x-hidden bg-white"
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
      iconSlug={isAnnotationNode ? "sticky-note" : sidebarData.iconSlug}
      iconColor={isAnnotationNode ? "text-yellow-600" : sidebarData.iconColor}
      totalInQueueCount={sidebarData.totalInQueueCount}
      totalInHistoryCount={sidebarData.totalInHistoryCount}
      hideQueueEvents={sidebarData.hideQueueEvents}
      getTabData={
        getTabData && state.componentSidebar.selectedNodeId ? (event) => getTabData(event.nodeId!, event) : undefined
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
      blockName={editingNodeData?.blockName}
      nodeConfiguration={editingNodeData?.configuration || {}}
      nodeConfigurationFields={editingNodeData?.configurationFields || []}
      onNodeConfigSave={onSaveConfiguration}
      onNodeConfigCancel={undefined}
      onEdit={onEdit ? () => onEdit(state.componentSidebar.selectedNodeId!) : undefined}
      domainId={organizationId}
      domainType="DOMAIN_TYPE_ORGANIZATION"
      customField={
        getCustomField && state.componentSidebar.selectedNodeId
          ? getCustomField(
              state.componentSidebar.selectedNodeId,
              undefined,
              integrations?.find((i) => i.metadata?.id === editingNodeData?.integrationRef?.id),
            ) || undefined
          : undefined
      }
      integrationName={editingNodeData?.integrationName}
      integrationRef={editingNodeData?.integrationRef}
      integrations={integrations}
      canReadIntegrations={canReadIntegrations}
      canCreateIntegrations={canCreateIntegrations}
      canUpdateIntegrations={canUpdateIntegrations}
      autocompleteExampleObj={autocompleteExampleObj}
      currentTab={isAnnotationNode ? "settings" : currentTab}
      onTabChange={onTabChange}
      workflowNodes={workflowNodes}
      components={components}
      triggers={triggers}
      blueprints={blueprints}
      onHighlightedNodesChange={onHighlightedNodesChange}
      executionChainEventId={focusRequest?.executionChain?.eventId || null}
      executionChainExecutionId={focusRequest?.executionChain?.executionId || null}
      executionChainTriggerEvent={focusRequest?.executionChain?.triggerEvent || null}
      executionChainRequestId={focusRequest?.requestId}
      onExecutionChainHandled={onExecutionChainHandled}
      hideRunsTab={isAnnotationNode}
      hideNodeId={isAnnotationNode}
      readOnly={readOnly}
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
  saveDisabled,
  saveDisabledTooltip,
  isAutoSaveEnabled,
  onToggleAutoSave,
  autoSaveDisabled,
  autoSaveDisabledTooltip,
  onExportYamlCopy,
  onExportYamlDownload,
}: {
  state: CanvasPageState;
  onSave?: (nodes: CanvasNode[]) => void;
  onUndo?: () => void;
  canUndo?: boolean;
  organizationId?: string;
  unsavedMessage?: string;
  saveIsPrimary?: boolean;
  saveButtonHidden?: boolean;
  saveDisabled?: boolean;
  saveDisabledTooltip?: string;
  isAutoSaveEnabled?: boolean;
  onToggleAutoSave?: () => void;
  autoSaveDisabled?: boolean;
  autoSaveDisabledTooltip?: string;
  onExportYamlCopy?: (nodes: CanvasNode[]) => void;
  onExportYamlDownload?: (nodes: CanvasNode[]) => void;
}) {
  const stateRef = useRef(state);
  stateRef.current = state;

  const handleSave = useCallback(() => {
    if (onSave) {
      onSave(stateRef.current.nodes);
    }
  }, [onSave]);

  const handleExportYamlCopy = useCallback(() => {
    if (onExportYamlCopy) {
      onExportYamlCopy(stateRef.current.nodes);
    }
  }, [onExportYamlCopy]);

  const handleExportYamlDownload = useCallback(() => {
    if (onExportYamlDownload) {
      onExportYamlDownload(stateRef.current.nodes);
    }
  }, [onExportYamlDownload]);

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
      saveDisabled={saveDisabled}
      saveDisabledTooltip={saveDisabledTooltip}
      isAutoSaveEnabled={isAutoSaveEnabled}
      onToggleAutoSave={onToggleAutoSave}
      autoSaveDisabled={autoSaveDisabled}
      autoSaveDisabledTooltip={autoSaveDisabledTooltip}
      onExportYamlCopy={onExportYamlCopy ? handleExportYamlCopy : undefined}
      onExportYamlDownload={onExportYamlDownload ? handleExportYamlDownload : undefined}
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
  onTogglePause,
  onToggleView,
  onToggleCollapse,
  onAutoLayout,
  onAnnotationUpdate,
  onAnnotationBlur,
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
  highlightedNodeIds,
  workflowNodes,
  setCurrentTab,
  onUndo,
  canUndo,
  organizationId,
  unsavedMessage,
  saveIsPrimary,
  saveButtonHidden,
  saveDisabled,
  saveDisabledTooltip,
  isAutoSaveEnabled,
  onToggleAutoSave,
  autoSaveDisabled,
  autoSaveDisabledTooltip,
  readOnly,
  logEntries = [],
  focusRequest,
  initialFocusNodeId,
  onResolveExecutionErrors,
  title,
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
  onTogglePause?: (nodeId: string) => void;
  onToggleView?: (nodeId: string) => void;
  onToggleCollapse?: () => void;
  onAutoLayout?: (selectedNodeIDs: string[]) => void | Promise<void>;
  onDelete?: (nodeId: string) => void;
  onAnnotationUpdate?: (
    nodeId: string,
    updates: { text?: string; color?: string; width?: number; height?: number; x?: number; y?: number },
  ) => void;
  onAnnotationBlur?: () => void;
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
  highlightedNodeIds: Set<string>;
  workflowNodes?: ComponentsNode[];
  setCurrentTab?: (tab: "latest" | "settings") => void;
  onUndo?: () => void;
  canUndo?: boolean;
  organizationId?: string;
  unsavedMessage?: string;
  saveIsPrimary?: boolean;
  saveButtonHidden?: boolean;
  saveDisabled?: boolean;
  saveDisabledTooltip?: string;
  isAutoSaveEnabled?: boolean;
  onToggleAutoSave?: () => void;
  autoSaveDisabled?: boolean;
  autoSaveDisabledTooltip?: string;
  readOnly?: boolean;
  logEntries?: LogEntry[];
  focusRequest?: FocusRequest | null;
  onExecutionChainHandled?: () => void;
  initialFocusNodeId?: string | null;
  onResolveExecutionErrors?: (executionIds: string[]) => void;
  title?: string;
}) {
  const { fitView, screenToFlowPosition, getViewport } = useReactFlow();
  const isReadOnly = readOnly ?? false;

  // Determine selection key code to support both Control (Windows/Linux) and Meta (Mac)
  // Similar to existing keyboard shortcuts that check (e.ctrlKey || e.metaKey)
  const selectionKey = useMemo(() => {
    // Check if running on Mac to use Meta (Cmd) key, otherwise use Control (Ctrl) key
    const isMac = navigator.platform.toLowerCase().includes("mac");
    return isMac ? "Meta" : "Control";
  }, []);

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
  const [isLogSidebarOpen, setIsLogSidebarOpen] = useState(false);
  const [logFilter, setLogFilter] = useState<LogTypeFilter>(new Set());
  const [logScope, setLogScope] = useState<LogScopeFilter>("all");
  const [logSearch, setLogSearch] = useState("");
  const [expandedRuns, setExpandedRuns] = useState<Set<string>>(() => new Set());
  const [logSidebarHeight, setLogSidebarHeight] = useState(320);
  const [isSnapToGridEnabled, setIsSnapToGridEnabled] = useState(true);
  const [isAutoLayouting, setIsAutoLayouting] = useState(false);

  useEffect(() => {
    const activeNoteId = getActiveNoteId();
    if (!activeNoteId) return;
    const activeElement = document.activeElement;
    if (activeElement && activeElement !== document.body) return;
    restoreActiveNoteFocus();
  }, [state.nodes]);

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
      const isAnnotationNode = clickedNode?.data?.type === "annotation";

      // Check if this is a placeholder node (persisted, not local-only)
      const workflowNode = workflowNodes?.find((n) => n.id === nodeId);
      const isPlaceholder = workflowNode?.name === "New Component" && !workflowNode.component?.name;

      // Check if this is a template node
      const isTemplateNode = clickedNode?.data?.isTemplate && !clickedNode?.data?.isPendingConnection;

      // Check if the current template is a configured template (not just pending connection)
      const currentTemplateNode = templateNodeId ? stateRef.current.nodes?.find((n) => n.id === templateNodeId) : null;
      const isCurrentTemplateConfigured =
        currentTemplateNode?.data?.isTemplate && !currentTemplateNode?.data?.isPendingConnection;

      // Allow switching to pending connection nodes or other template nodes even if there's a configured template
      // But block switching to other regular/real nodes
      if (
        isCurrentTemplateConfigured &&
        nodeId !== templateNodeId &&
        !isPendingConnection &&
        !isTemplateNode &&
        !isPlaceholder
      ) {
        return;
      }

      if (isAnnotationNode) {
        return;
      }

      if (isPendingConnection && onPendingConnectionNodeClick) {
        // Notify parent that a pending connection node was clicked
        onPendingConnectionNodeClick(nodeId);
      } else if (isPlaceholder && onPendingConnectionNodeClick) {
        // Handle placeholder clicks the same as pending connections
        onPendingConnectionNodeClick(nodeId);
      } else {
        if (isTemplateNode && onTemplateNodeClick) {
          // Notify parent to restore template state
          onTemplateNodeClick(nodeId);
        } else {
          // Regular node click
          stateRef.current.componentSidebar.open(nodeId);

          const nodeData = clickedNode?.data as {
            component?: { error?: string };
            composite?: { error?: string };
            trigger?: { error?: string };
          } | null;
          const hasConfigurationWarning = Boolean(
            nodeData?.component?.error || nodeData?.composite?.error || nodeData?.trigger?.error,
          );

          // Reset to Runs tab when clicking on a regular node
          if (setCurrentTab) {
            setCurrentTab(hasConfigurationWarning ? "settings" : "latest");
          }

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
    [
      templateNodeId,
      workflowNodes,
      onBuildingBlocksSidebarToggle,
      onPendingConnectionNodeClick,
      onTemplateNodeClick,
      setCurrentTab,
    ],
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

  const onTogglePauseRef = useRef(onTogglePause);
  onTogglePauseRef.current = onTogglePause;

  const onToggleViewRef = useRef(onToggleView);
  onToggleViewRef.current = onToggleView;

  const onAnnotationUpdateRef = useRef(onAnnotationUpdate);
  onAnnotationUpdateRef.current = onAnnotationUpdate;
  const onAnnotationBlurRef = useRef(onAnnotationBlur);
  onAnnotationBlurRef.current = onAnnotationBlur;

  const handleSave = useCallback(() => {
    if (onSave) {
      onSave(stateRef.current.nodes);
    }
  }, [onSave]);

  const handleConnect = useCallback(
    (connection: any) => {
      if (isReadOnly) return;
      connectionCompletedRef.current = true;
      if (onEdgeCreate && connection.source && connection.target) {
        onEdgeCreate(connection.source, connection.target, connection.sourceHandle);
      }
    },
    [onEdgeCreate, isReadOnly],
  );

  const handleDragOver = useCallback(
    (event: React.DragEvent) => {
      if (isReadOnly) return;
      event.preventDefault();
      event.dataTransfer.dropEffect = "move";
    },
    [isReadOnly],
  );

  const handleDrop = useCallback(
    (event: React.DragEvent) => {
      if (isReadOnly) return;
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
    [onBuildingBlockDrop, screenToFlowPosition, isReadOnly],
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

  const selectedNodeIDs = useMemo(
    () => state.nodes.filter((node) => node.selected).map((node) => node.id),
    [state.nodes],
  );

  const handleAutoLayout = useCallback(async () => {
    if (!onAutoLayout || isReadOnly || isAutoLayouting || selectedNodeIDs.length === 0) {
      return;
    }

    try {
      setIsAutoLayouting(true);
      await onAutoLayout(selectedNodeIDs);
    } finally {
      setIsAutoLayouting(false);
    }
  }, [onAutoLayout, isReadOnly, isAutoLayouting, selectedNodeIDs]);

  const isAutoLayoutDisabled = isReadOnly || !onAutoLayout || isAutoLayouting || selectedNodeIDs.length === 0;
  const autoLayoutTooltipMessage =
    selectedNodeIDs.length === 0
      ? "Select one or more nodes to auto arrange"
      : "Auto arrange selected components left-to-right";

  useEffect(() => {
    if (!focusRequest) {
      return;
    }

    const targetNode = stateRef.current.nodes?.find((node) => node.id === focusRequest.nodeId);
    if (!targetNode) {
      return;
    }

    stateRef.current.setNodes((nodes) =>
      nodes.map((node) => ({
        ...node,
        selected: node.id === focusRequest.nodeId,
      })),
    );
    fitView({ nodes: [targetNode], duration: 500, maxZoom: 1.2 });
  }, [focusRequest, fitView]);

  // Add keyboard shortcut for toggling collapse/expand
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      // Toggle collapse: Ctrl/Cmd + E
      if ((e.ctrlKey || e.metaKey) && !e.shiftKey && e.key === "e") {
        e.preventDefault();
        handleToggleCollapse();
      }
    };

    document.addEventListener("keydown", handleKeyDown);
    return () => document.removeEventListener("keydown", handleKeyDown);
  }, [handleToggleCollapse]);

  const handlePaneClick = useCallback(() => {
    // Do not close sidebar or reset state while creating a new component
    if (templateNodeId) return;

    // Clear ReactFlow's selection state and close both sidebars
    stateRef.current.setNodes((nodes) =>
      nodes.map((node) => ({
        ...node,
        selected: false,
      })),
    );

    // Close component sidebar
    stateRef.current.componentSidebar.close();

    // Close building blocks sidebar
    if (onBuildingBlocksSidebarToggle) {
      onBuildingBlocksSidebarToggle(false);
    }
  }, [templateNodeId, onBuildingBlocksSidebarToggle]);

  // Handle fit to view on ReactFlow initialization
  const handleInit = useCallback(
    (reactFlowInstance: any) => {
      if (!hasFitToViewRef.current) {
        const hasNodes = (stateRef.current.nodes?.length ?? 0) > 0;

        const focusNodeId = initialFocusNodeId;
        const focusNode = focusNodeId ? stateRef.current.nodes?.find((node) => node.id === focusNodeId) : null;

        if (focusNode) {
          fitView({ nodes: [focusNode], duration: 500, maxZoom: 1.2 });
        } else if (hasNodes) {
          // Fit to view but don't zoom in too much (max zoom of 1.0)
          fitView({ maxZoom: 1.0, padding: 0.5 });
        }

        if (hasNodes) {
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
    [fitView, getViewport, onZoomChange, hasFitToViewRef, viewportRef, initialFocusNodeId],
  );

  const showHeader = !isReadOnly;

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
    onTogglePause: onTogglePauseRef,
    onToggleView: onToggleViewRef,
    onAnnotationUpdate: onAnnotationUpdateRef,
    onAnnotationBlur: onAnnotationBlurRef,
    aiState: state.ai,
    runDisabled,
    runDisabledTooltip,
    showHeader,
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
    onTogglePause: onTogglePauseRef,
    onToggleView: onToggleViewRef,
    onAnnotationUpdate: onAnnotationUpdateRef,
    onAnnotationBlur: onAnnotationBlurRef,
    aiState: state.ai,
    runDisabled,
    runDisabledTooltip,
    showHeader,
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
      if (isReadOnly) return;
      if (params.nodeId) {
        const connectionInfo = { nodeId: params.nodeId, handleId: params.handleId, handleType: params.handleType };
        setConnectingFrom(connectionInfo);
        connectingFromRef.current = connectionInfo;
      }
    },
    [isReadOnly],
  );

  const handleConnectEnd = useCallback(
    (event: MouseEvent | TouchEvent) => {
      if (isReadOnly) return;
      const currentConnectingFrom = connectingFromRef.current;

      if (currentConnectingFrom && !connectionCompletedRef.current) {
        // Only create placeholder for source handles (right side / output)
        // Don't create placeholders for target handles (left side / input)
        if (currentConnectingFrom.handleType === "source") {
          const mouseEvent = event as MouseEvent;
          const canvasPosition = screenToFlowPosition({
            x: mouseEvent.clientX,
            y: mouseEvent.clientY,
          });

          if (onConnectionDropInEmptySpace) {
            onConnectionDropInEmptySpace(canvasPosition, currentConnectingFrom);
          }
        }
      }

      setConnectingFrom(null);
      connectingFromRef.current = null;
      connectionCompletedRef.current = false;
    },
    [screenToFlowPosition, onConnectionDropInEmptySpace, isReadOnly],
  );

  // Find the hovered edge to get its source and target
  const hoveredEdge = useMemo(() => {
    if (!hoveredEdgeId) return null;
    return state.edges?.find((e) => e.id === hoveredEdgeId);
  }, [hoveredEdgeId, state.edges]);

  const nodesWithCallbacks = useMemo(() => {
    const hasHighlightedNodes = highlightedNodeIds.size > 0;
    return state.nodes.map((node) => ({
      ...node,
      data: {
        ...node.data,
        _callbacksRef: callbacksRef,
        _hoveredEdge: hoveredEdge,
        _connectingFrom: connectingFrom,
        _allEdges: state.edges,
        _isHighlighted: highlightedNodeIds.has(node.id),
        _hasHighlightedNodes: hasHighlightedNodes,
      },
    }));
  }, [state.nodes, hoveredEdge, connectingFrom, state.edges, highlightedNodeIds]);

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
        data: {
          ...e.data,
          isHovered: e.id === hoveredEdgeId,
          onDelete: isReadOnly ? undefined : (edgeId: string) => state.onEdgesChange([{ id: edgeId, type: "remove" }]),
        },
        zIndex: e.id === hoveredEdgeId ? 1000 : 0,
      })),
    [state.edges, hoveredEdgeId, state.onEdgesChange, isReadOnly],
  );

  const handleNodesChange = useCallback(
    (changes: NodeChange[]) => {
      if (!isReadOnly) {
        state.onNodesChange(changes);
        return;
      }

      const filteredChanges = changes.filter((change) => change.type === "select" || change.type === "dimensions");
      if (filteredChanges.length > 0) {
        state.onNodesChange(filteredChanges);
      }
    },
    [isReadOnly, state],
  );

  const handleEdgesChange = useCallback(
    (changes: EdgeChange[]) => {
      if (!isReadOnly) {
        state.onEdgesChange(changes);
        return;
      }

      const filteredChanges = changes.filter((change) => change.type === "select");
      if (filteredChanges.length > 0) {
        state.onEdgesChange(filteredChanges);
      }
    },
    [isReadOnly, state],
  );

  const logCounts = useMemo(() => {
    return logEntries.reduce(
      (acc, entry) => {
        acc.total += 1;
        if (entry.type === "error") acc.error += 1;
        if (entry.type === "warning") acc.warning += 1;
        if (entry.type === "success") acc.success += 1;
        if (entry.runItems?.length) {
          acc.total += entry.runItems.length;
          entry.runItems.forEach((item) => {
            if (item.type === "error") acc.error += 1;
            if (item.type === "warning") acc.warning += 1;
            if (item.type === "success") acc.success += 1;
          });
        }
        return acc;
      },
      { total: 0, error: 0, warning: 0, success: 0 },
    );
  }, [logEntries]);

  const filteredLogEntries = useMemo(() => {
    const query = logSearch.trim().toLowerCase();
    const matchesSearch = (value?: string) => !query || (value || "").toLowerCase().includes(query);
    // Show all if filter is empty or contains all three types
    const showAll = logFilter.size === 0 || logFilter.size === 3;

    return logEntries.reduce<LogEntry[]>((acc, entry) => {
      if (logScope !== "all" && entry.source !== logScope) {
        return acc;
      }

      if (entry.type === "run") {
        const runItems = entry.runItems || [];
        const filteredRunItems = runItems.filter((item) => {
          const typeMatch = showAll || (item.type !== "resolved-error" && logFilter.has(item.type));
          const searchMatch =
            matchesSearch(item.searchText) || matchesSearch(typeof item.title === "string" ? item.title : "");
          return typeMatch && searchMatch;
        });

        const entrySearchMatch =
          matchesSearch(entry.searchText) || matchesSearch(typeof entry.title === "string" ? entry.title : "");
        const typeMatch = showAll ? true : filteredRunItems.length > 0;
        const searchMatch = query ? entrySearchMatch || filteredRunItems.length > 0 : true;

        if (typeMatch && searchMatch) {
          acc.push({ ...entry, runItems: filteredRunItems });
        }
        return acc;
      }

      if (!showAll && (entry.type === "resolved-error" || !logFilter.has(entry.type))) {
        return acc;
      }

      const entrySearchMatch =
        matchesSearch(entry.searchText) || matchesSearch(typeof entry.title === "string" ? entry.title : "");
      if (!entrySearchMatch) {
        return acc;
      }

      acc.push(entry);
      return acc;
    }, []);
  }, [logEntries, logFilter, logScope, logSearch]);

  const handleLogButtonClick = useCallback((filterType: "all" | "error" | "warning") => {
    if (filterType === "all") {
      setLogFilter(new Set());
    } else {
      setLogFilter(new Set([filterType]));
    }
    setIsLogSidebarOpen(true);
  }, []);

  const handleRunToggle = useCallback((runId: string) => {
    setExpandedRuns((prev) => {
      const next = new Set(prev);
      if (next.has(runId)) {
        next.delete(runId);
      } else {
        next.add(runId);
      }
      return next;
    });
  }, []);

  return (
    <div className="h-full w-full relative">
      {/* Header */}
      {!hideHeader && (
        <Header
          breadcrumbs={state.breadcrumbs}
          onSave={onSave ? handleSave : undefined}
          onUndo={onUndo}
          canUndo={canUndo}
          organizationId={organizationId}
          unsavedMessage={unsavedMessage}
          saveIsPrimary={saveIsPrimary}
          saveButtonHidden={saveButtonHidden}
          saveDisabled={saveDisabled}
          saveDisabledTooltip={saveDisabledTooltip}
          isAutoSaveEnabled={isAutoSaveEnabled}
          onToggleAutoSave={onToggleAutoSave}
          autoSaveDisabled={autoSaveDisabled}
          autoSaveDisabledTooltip={autoSaveDisabledTooltip}
        />
      )}

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
            selectionOnDrag={true}
            selectionKeyCode={selectionKey}
            multiSelectionKeyCode={selectionKey}
            snapToGrid={isSnapToGridEnabled}
            snapGrid={[48, 48]}
            panOnScrollSpeed={0.8}
            nodesDraggable={!isReadOnly}
            nodesConnectable={!isReadOnly && !!onEdgeCreate}
            elementsSelectable={true}
            onNodesChange={handleNodesChange}
            onEdgesChange={handleEdgesChange}
            onConnect={isReadOnly ? undefined : handleConnect}
            onConnectStart={isReadOnly ? undefined : handleConnectStart}
            onConnectEnd={isReadOnly ? undefined : handleConnectEnd}
            onDragOver={isReadOnly ? undefined : handleDragOver}
            onDrop={isReadOnly ? undefined : handleDrop}
            onMove={handleMove}
            onInit={handleInit}
            deleteKeyCode={null}
            onPaneClick={handlePaneClick}
            onEdgeMouseEnter={handleEdgeMouseEnter}
            onEdgeMouseLeave={handleEdgeMouseLeave}
            defaultViewport={viewport}
            fitView={false}
            style={{ opacity: isInitialized ? 1 : 0 }}
            className="h-full w-full"
          >
            <Background gap={8} size={2} bgColor="#F1F5F9" color="#d9d9d9ff" />
            <ZoomSlider
              position="bottom-left"
              orientation="horizontal"
              screenshotName={title}
              isSnapToGridEnabled={isSnapToGridEnabled}
              onSnapToGridToggle={() => setIsSnapToGridEnabled((prev) => !prev)}
            >
              <Tooltip>
                <TooltipTrigger asChild>
                  <Button variant="ghost" size="icon-sm" onClick={handleToggleCollapse}>
                    {state.isCollapsed ? <ScanText className="h-3 w-3" /> : <ScanLine className="h-3 w-3" />}
                  </Button>
                </TooltipTrigger>
                <TooltipContent>
                  {state.isCollapsed
                    ? "Switch components to Detailed view (Ctrl/Cmd + E)"
                    : "Switch components to Compact view (Ctrl/Cmd + E)"}
                </TooltipContent>
              </Tooltip>
              <Tooltip>
                <TooltipTrigger asChild>
                  <span className="inline-flex">
                    <Button
                      variant="ghost"
                      size="sm"
                      className="h-8 px-2 text-xs font-medium"
                      onClick={handleAutoLayout}
                      disabled={isAutoLayoutDisabled}
                    >
                      {isAutoLayouting ? (
                        <Loader2 className="h-3 w-3 animate-spin" />
                      ) : (
                        <Workflow className="h-3 w-3" />
                      )}
                    </Button>
                  </span>
                </TooltipTrigger>
                <TooltipContent>{autoLayoutTooltipMessage}</TooltipContent>
              </Tooltip>
              <NodeSearch
                onSearch={(searchString) => {
                  const query = searchString.toLowerCase();
                  return state.nodes.filter((node) => {
                    const label = ((node.data?.label as string) || "").toLowerCase();
                    const nodeName = ((node.data as any)?.nodeName || "").toLowerCase();
                    const id = (node.id || "").toLowerCase();
                    return label.includes(query) || nodeName.includes(query) || id.includes(query);
                  });
                }}
                onSelectNode={(node) => {
                  const isAnnotationNode = (node.data as any)?.type === "annotation";
                  if (isAnnotationNode) {
                    return;
                  }
                  state.componentSidebar.open(node.id);
                }}
              />
            </ZoomSlider>
            <Panel
              position="bottom-left"
              className="bg-white text-gray-800 outline-1 outline-slate-950/20 flex items-center gap-1 rounded-md p-0.5 h-8"
              style={{ marginLeft: 370 }}
            >
              <Tooltip>
                <TooltipTrigger asChild>
                  <Button
                    variant="ghost"
                    size="sm"
                    className="h-8 items-center text-xs font-medium"
                    onClick={() => handleLogButtonClick("all")}
                  >
                    <ScrollText className="h-3 w-3" />
                  </Button>
                </TooltipTrigger>
                <TooltipContent>All Logs</TooltipContent>
              </Tooltip>
              <Tooltip>
                <TooltipTrigger asChild>
                  <Button
                    variant="ghost"
                    size="sm"
                    className="h-8 items-center text-xs font-medium"
                    onClick={() => handleLogButtonClick("error")}
                  >
                    <CircleX className={logCounts.error > 0 ? "h-3 w-3 text-red-500" : "h-3 w-3 text-gray-800"} />
                    <span className={logCounts.error > 0 ? "tabular-nums text-red-500" : "tabular-nums text-gray-800"}>
                      {logCounts.error}
                    </span>
                  </Button>
                </TooltipTrigger>
                <TooltipContent>Errors</TooltipContent>
              </Tooltip>
              <Tooltip>
                <TooltipTrigger asChild>
                  <Button
                    variant="ghost"
                    size="sm"
                    className="h-8 items-center text-xs font-medium"
                    onClick={() => handleLogButtonClick("warning")}
                  >
                    <TriangleAlert
                      className={logCounts.warning > 0 ? "h-3 w-3 text-orange-500" : "h-3 w-3 text-gray-800"}
                    />
                    <span
                      className={logCounts.warning > 0 ? "tabular-nums text-orange-500" : "tabular-nums text-gray-800"}
                    >
                      {logCounts.warning}
                    </span>
                  </Button>
                </TooltipTrigger>
                <TooltipContent>Warnings</TooltipContent>
              </Tooltip>
            </Panel>
          </ReactFlow>
        </div>
      </div>
      <CanvasLogSidebar
        isOpen={isLogSidebarOpen}
        onClose={() => setIsLogSidebarOpen(false)}
        filter={logFilter}
        onFilterChange={setLogFilter}
        onResolveErrors={onResolveExecutionErrors}
        height={logSidebarHeight}
        onHeightChange={setLogSidebarHeight}
        scope={logScope}
        onScopeChange={setLogScope}
        searchValue={logSearch}
        onSearchChange={setLogSearch}
        entries={filteredLogEntries}
        counts={logCounts}
        expandedRuns={expandedRuns}
        onToggleRun={handleRunToggle}
      />
    </div>
  );
}

export type { BuildingBlock } from "../BuildingBlocksSidebar";
export { CanvasPage };
