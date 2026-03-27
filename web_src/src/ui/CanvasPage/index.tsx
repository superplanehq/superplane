/* eslint-disable @typescript-eslint/no-explicit-any */
import {
  Background,
  Panel,
  ReactFlow,
  ReactFlowProvider,
  ViewportPortal,
  useOnSelectionChange,
  useReactFlow,
  useViewport,
  type Edge as ReactFlowEdge,
  type Node as ReactFlowNode,
  type NodeChange,
  type EdgeChange,
} from "@xyflow/react";

import {
  CircleX,
  GitBranch,
  Group,
  Loader2,
  Map as MapIcon,
  Play,
  ScanLine,
  ScanText,
  Copy,
  LayoutGrid,
  Trash2,
  TriangleAlert,
  Workflow,
} from "lucide-react";
import { ZoomSlider } from "@/components/zoom-slider";
import { NodeSearch } from "@/components/node-search";
import { Button } from "@/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { cn } from "@/lib/utils";
import { useCallback, useEffect, useMemo, useRef, useState, type SyntheticEvent } from "react";

import {
  ConfigurationField,
  CanvasesCanvasEventWithExecutions,
  CanvasesCanvasNodeExecution,
  CanvasesCanvasNodeQueueItem,
  ComponentsNode,
  ComponentsComponent,
  TriggersTrigger,
  BlueprintsBlueprint,
  ComponentsIntegrationRef,
  OrganizationsIntegration,
} from "@/api-client";
import { buildSidebarComponentDocsPayload } from "@/utils/componentDocsUrl";
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
import { GROUP_CHILD_EDGE_PADDING, GROUP_CHILD_MIN_Y_OFFSET } from "../groupNode/constants";
import { GroupNode } from "../groupNode";
import { CanvasMiniMap } from "./CanvasMiniMap";
import "./canvas-reset.css";
import { CustomEdge } from "./CustomEdge";
import { Header, type BreadcrumbItem } from "./Header";
import { Simulation } from "./storybooks/useSimulation";
import { CanvasPageState, useCanvasState } from "./useCanvasState";
import { useMinimapVisibility } from "./useMinimapVisibility";
import { SidebarEvent } from "../componentSidebar/types";
import { CanvasLogSidebar, type ConsoleTab, type LogEntry } from "../CanvasLogSidebar";
import { IntegrationStatusIndicator, type MissingIntegration } from "../IntegrationStatusIndicator";
import { countUnacknowledgedErrors } from "@/pages/workflowv2/canvasRunsUtils";

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

function clampGroupChildNodePositionChanges(changes: NodeChange[], nodes: CanvasNode[]): NodeChange[] {
  const nodesById = new Map(nodes.map((n) => [n.id, n]));

  return changes.map((change) => {
    if (change.type !== "position") return change;
    const posChange = change as { id: string; type: "position"; position?: { x: number; y: number } };
    if (!posChange.position) return change;
    const node = nodesById.get(posChange.id);
    if (!node?.parentId) return change;
    const parent = nodesById.get(node.parentId);
    if (!parent || (parent.data as { type?: string })?.type !== "group") return change;

    const x = Math.max(posChange.position.x, GROUP_CHILD_EDGE_PADDING);
    const y = Math.max(posChange.position.y, GROUP_CHILD_MIN_Y_OFFSET);

    if (x === posChange.position.x && y === posChange.position.y) return change;
    return {
      ...posChange,
      position: { ...posChange.position, x, y },
    };
  });
}

const DEFAULT_GROUP_MIN_WIDTH = 480;
const DEFAULT_GROUP_MIN_HEIGHT = 320;
const GROUP_RESIZE_PADDING = 30;

function computeGroupSizeFromChildren(groupId: string, nodes: CanvasNode[]): { width: number; height: number } | null {
  const children = nodes.filter((n) => n.parentId === groupId);
  if (children.length === 0) return null;

  let maxRight = 0;
  let maxBottom = 0;

  for (const child of children) {
    const cx = child.position?.x ?? 0;
    const cy = child.position?.y ?? 0;
    const cw = child.measured?.width ?? child.width ?? 240;
    const ch = child.measured?.height ?? child.height ?? 80;
    maxRight = Math.max(maxRight, cx + cw);
    maxBottom = Math.max(maxBottom, cy + ch);
  }

  return {
    width: Math.max(DEFAULT_GROUP_MIN_WIDTH, Math.round(maxRight + GROUP_RESIZE_PADDING)),
    height: Math.max(DEFAULT_GROUP_MIN_HEIGHT, Math.round(maxBottom + GROUP_RESIZE_PADDING)),
  };
}

function resizeGroupsAfterChildChanges(
  changes: NodeChange[],
  nodes: CanvasNode[],
  setNodes: (updater: (nodes: CanvasNode[]) => CanvasNode[]) => void,
) {
  const childChangedIds = new Set(
    changes.filter((c) => c.type === "dimensions" || c.type === "position").map((c) => c.id),
  );
  if (childChangedIds.size === 0) return;

  const affectedGroupIds = new Set<string>();
  for (const node of nodes) {
    if (node.parentId && childChangedIds.has(node.id)) {
      affectedGroupIds.add(node.parentId);
    }
  }
  if (affectedGroupIds.size === 0) return;

  setNodes((currentNodes) => {
    let changed = false;
    const updated = currentNodes.map((node) => {
      if (!affectedGroupIds.has(node.id)) return node;
      const size = computeGroupSizeFromChildren(node.id, currentNodes);
      if (!size) return node;

      const currentW = node.width ?? 0;
      const currentH = node.height ?? 0;
      if (Math.abs(currentW - size.width) < 1 && Math.abs(currentH - size.height) < 1) return node;

      changed = true;
      return {
        ...node,
        width: size.width,
        height: size.height,
        style: { ...node.style, width: size.width, height: size.height, zIndex: -1 },
      };
    });
    return changed ? updated : currentNodes;
  });
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
  /** Integration catalog label; used to resolve docs.superplane.com path for integration components. */
  integrationLabel?: string;
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
  versionLabel?: string;
  onCreateVersion?: () => void;
  onPublishVersion?: () => void;
  onDiscardVersion?: () => void;
  createVersionDisabled?: boolean;
  createVersionDisabledTooltip?: string;
  publishVersionDisabled?: boolean;
  publishVersionDisabledTooltip?: string;
  discardVersionDisabled?: boolean;
  discardVersionDisabledTooltip?: string;
  headerMode?: "default" | "version-live" | "version-edit" | "versioning-disabled";
  saveState?: "saved" | "saving" | "unsaved" | "error";
  lastSavedAt?: Date | string | null;
  saveErrorMessage?: string | null;
  /** Node settings sidebar: canvas uses debounced autosave without closing the panel after each save. */
  configurationSaveMode?: "manual" | "auto";
  onEnterEditMode?: () => void;
  enterEditModeDisabled?: boolean;
  enterEditModeDisabledTooltip?: string;
  onExitEditMode?: () => void;
  exitEditModeDisabled?: boolean;
  exitEditModeDisabledTooltip?: string;
  unpublishedDraftChangeCount?: number;
  isAutoLayoutOnUpdateEnabled?: boolean;
  onToggleAutoLayoutOnUpdate?: () => void;
  autoLayoutOnUpdateDisabled?: boolean;
  autoLayoutOnUpdateDisabledTooltip?: string;
  topViewMode?: "canvas" | "yaml" | "memory" | "settings";
  onTopViewModeChange?: (mode: "canvas" | "yaml" | "memory" | "settings") => void;
  canvasStateMode?: "default" | "editing" | "previewing-previous-version" | "awaiting-approval";
  memoryItemCount?: number;
  onExportYamlCopy?: (nodes: CanvasNode[]) => void;
  onExportYamlDownload?: (nodes: CanvasNode[]) => void;
  dataViewContent?: React.ReactNode;
  versionControlSidebar?: React.ReactNode;
  isVersionControlOpen?: boolean;
  onOpenVersionControl?: () => void;
  versionControlButtonLabel?: string;
  versionControlButtonTooltip?: string;
  versionControlNotificationCount?: number;
  showBottomStatusControls?: boolean;
  readOnly?: boolean;
  hideAddControls?: boolean;
  canReadIntegrations?: boolean;
  canCreateIntegrations?: boolean;
  canUpdateIntegrations?: boolean;
  missingIntegrations?: MissingIntegration[];
  onConnectIntegration?: (integrationName: string) => void;
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
  ) => void | Promise<void>;
  onAnnotationUpdate?: (
    nodeId: string,
    updates: { text?: string; color?: string; width?: number; height?: number; x?: number; y?: number },
  ) => void;
  onAnnotationBlur?: () => void;
  onGroupUpdate?: (nodeId: string, updates: { label?: string; description?: string; color?: string }) => void;
  onGroupNodes?: (
    bounds: { x: number; y: number; width: number; height: number },
    nodePositions: Array<{ id: string; x: number; y: number }>,
  ) => void;
  onUngroupNodes?: (groupNodeId: string) => void;
  getCustomField?: (
    nodeId: string,
    onRun?: (initialData?: string) => void,
    integration?: OrganizationsIntegration,
  ) => (() => React.ReactNode) | null;
  onSave?: (nodes: CanvasNode[]) => void;
  integrations?: OrganizationsIntegration[];
  onEdgeCreate?: (sourceId: string, targetId: string, sourceHandle?: string | null) => void;
  onNodeDelete?: (nodeId: string) => void;
  onNodesDelete?: (nodeIds: string[]) => void;
  onDuplicateNodes?: (nodeIds: string[]) => void;
  onAutoLayoutNodes?: (nodeIds: string[]) => void;
  onEdgeDelete?: (edgeIds: string[]) => void;
  runsEvents?: CanvasesCanvasEventWithExecutions[];
  runsTotalCount?: number;
  runsHasNextPage?: boolean;
  runsIsFetchingNextPage?: boolean;
  onRunsLoadMore?: () => void;
  runsNodes?: ComponentsNode[];
  runsComponentIconMap?: Record<string, string>;
  runsNodeQueueItemsMap?: Record<string, CanvasesCanvasNodeQueueItem[]>;
  onRunNodeSelect?: (nodeId: string) => void;
  onRunExecutionSelect?: (options: {
    nodeId: string;
    eventId: string;
    executionId: string;
    triggerEvent?: SidebarEvent;
  }) => void;
  onAcknowledgeErrors?: (executionIds: string[]) => void;
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

  /** Opens the version node diff modal when using "View details" on a non-live published preview (same as sidebar compare). */
  onPreviewPreviousVersionViewDetails?: () => void;
  /** Change request being previewed while awaiting approval (floating bar + versioning sidebar). */
  awaitingApprovalBanner?: {
    title: string;
    description?: string;
    onApprove: () => void | Promise<void>;
    onReject: () => void | Promise<void>;
    onPublish: () => void | Promise<void>;
    onOpenVersioningTab?: () => void;
    /** Opens the same version node diff dialog as the version sidebar compare control. */
    onViewNodeDiff?: () => void;
    canAct: boolean;
    actionPending: boolean;
    /** Label + colors for open vs ready-to-publish (matches version sidebar + diff dialog). */
    reviewUi: {
      label: string;
      floatingBarBgClassName: string;
      dotClassName: string;
      titleClassName: string;
    };
  };
}

export const CANVAS_SIDEBAR_STORAGE_KEY = "canvasSidebarOpen";
export const COMPONENT_SIDEBAR_WIDTH_STORAGE_KEY = "componentSidebarWidth";
export const CONSOLE_OPEN_STORAGE_KEY = "consoleOpen";
export const CONSOLE_HEIGHT_STORAGE_KEY = "consoleHeight";

const EDGE_STYLE = {
  type: "custom",
  style: { stroke: "#C9D5E1", strokeWidth: 3 },
} as const;

const DEFAULT_CANVAS_ZOOM = 0.8;
const MIN_CANVAS_ZOOM = 0.1;

/*
 * nodeTypes must be defined outside of the component to prevent
 * react-flow from remounting the node types on every render.
 */
function DefaultNodeRenderer(nodeProps: { data: BlockData & { _callbacksRef?: any }; id: string; selected?: boolean }) {
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
      showHeader={callbacks?.showHeader && !callbacks?.hasMultiSelection}
      onExpand={callbacks.handleNodeExpand}
      onClick={(e) => callbacks.handleNodeClick(nodeProps.id, e)}
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
          ? (nodeId: string, updates: any) => callbacks.onAnnotationUpdate.current?.(nodeId, updates)
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
}

function GroupNodeRenderer(nodeProps: {
  data: BlockData & { _callbacksRef?: any };
  id: string;
  selected?: boolean;
  width?: number;
  height?: number;
}) {
  const { _callbacksRef, ...blockData } = nodeProps.data;
  const callbacks = _callbacksRef?.current;
  const groupData = blockData.group || {};

  const handleGroupUpdate = callbacks?.onGroupUpdate?.current
    ? (updates: any) => callbacks.onGroupUpdate.current?.(nodeProps.id, updates)
    : undefined;

  const handleUngroup = callbacks?.onUngroupNodes?.current
    ? () => callbacks.onUngroupNodes.current?.(nodeProps.id)
    : undefined;

  const handleDelete = callbacks?.onNodeDelete?.current
    ? () => callbacks.onNodeDelete.current?.(nodeProps.id)
    : undefined;

  return (
    <div data-testid="canvas-group-node" style={{ width: nodeProps.width, height: nodeProps.height }}>
      <GroupNode
        {...groupData}
        selected={nodeProps.selected}
        onGroupUpdate={handleGroupUpdate}
        onUngroup={handleUngroup}
        onDelete={handleDelete}
      />
    </div>
  );
}

const nodeTypes = {
  default: DefaultNodeRenderer,
  group: GroupNodeRenderer,
};

function CanvasPage(props: CanvasPageProps) {
  const cancelQueueItemRef = useRef<CanvasPageProps["onCancelQueueItem"]>(props.onCancelQueueItem);
  cancelQueueItemRef.current = props.onCancelQueueItem;
  const state = useCanvasState(props);
  const readOnly = props.readOnly ?? false;
  const [currentTab, setCurrentTab] = useState<"latest" | "settings" | "docs">("latest");
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
  const canvasNodesForAiContext = useMemo(
    () =>
      (props.workflowNodes || []).map((node) => ({
        id: node.id || "",
        name: node.name || "",
        label: node.name || "",
        type: node.type || "",
      })),
    [props.workflowNodes],
  );

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
      if (!editingNodeData || !props.onNodeConfigurationSave) {
        return;
      }
      const result = props.onNodeConfigurationSave(editingNodeData.nodeId, configuration, nodeName, integrationRef);
      if (props.configurationSaveMode !== "auto") {
        state.componentSidebar.close();
      }
      return result;
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

  const canvasStateMode = props.canvasStateMode || "default";
  const showPreviewFloatingBar =
    canvasStateMode === "previewing-previous-version" && !!props.onPreviewPreviousVersionViewDetails;
  const showAwaitingFloatingBar = canvasStateMode === "awaiting-approval" && !!props.awaitingApprovalBanner;

  const canvasStateBorderClass =
    canvasStateMode === "editing"
      ? "border-3 border-amber-500"
      : canvasStateMode === "previewing-previous-version"
        ? "border-3 border-sky-500"
        : "";
  const canvasStateBadgeClass =
    canvasStateMode === "editing"
      ? "bg-amber-500"
      : canvasStateMode === "previewing-previous-version"
        ? "bg-sky-500"
        : "";
  const canvasStateLabel =
    canvasStateMode === "editing"
      ? "Edit Mode"
      : canvasStateMode === "previewing-previous-version"
        ? "Previewing Previous Version"
        : "";

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
          versionLabel={props.versionLabel}
          onCreateVersion={props.onCreateVersion}
          onPublishVersion={props.onPublishVersion}
          onDiscardVersion={props.onDiscardVersion}
          createVersionDisabled={props.createVersionDisabled}
          createVersionDisabledTooltip={props.createVersionDisabledTooltip}
          publishVersionDisabled={props.publishVersionDisabled}
          publishVersionDisabledTooltip={props.publishVersionDisabledTooltip}
          discardVersionDisabled={props.discardVersionDisabled}
          discardVersionDisabledTooltip={props.discardVersionDisabledTooltip}
          headerMode={props.headerMode}
          saveState={props.saveState}
          lastSavedAt={props.lastSavedAt}
          saveErrorMessage={props.saveErrorMessage}
          onEnterEditMode={props.onEnterEditMode}
          enterEditModeDisabled={props.enterEditModeDisabled}
          enterEditModeDisabledTooltip={props.enterEditModeDisabledTooltip}
          onExitEditMode={props.onExitEditMode}
          exitEditModeDisabled={props.exitEditModeDisabled}
          exitEditModeDisabledTooltip={props.exitEditModeDisabledTooltip}
          unpublishedDraftChangeCount={props.unpublishedDraftChangeCount}
          topViewMode={props.topViewMode}
          onTopViewModeChange={props.onTopViewModeChange}
          memoryItemCount={props.memoryItemCount}
          onExportYamlCopy={props.onExportYamlCopy}
          onExportYamlDownload={props.onExportYamlDownload}
          canvasId={props.canvasId}
        />
        {props.headerBanner ? <div className="border-b border-black/20">{props.headerBanner}</div> : null}
      </div>

      {/* Main content area with sidebar and canvas/memory/settings views */}
      {props.topViewMode && props.topViewMode !== "canvas" ? (
        <div className="flex-1 flex relative overflow-hidden">
          {props.versionControlSidebar}
          <div className="flex-1 overflow-auto bg-slate-50">{props.dataViewContent}</div>
        </div>
      ) : (
        <div className="flex-1 flex relative overflow-hidden">
          {props.versionControlSidebar}
          {props.hideAddControls ? null : (
            <BuildingBlocksSidebar
              isOpen={isBuildingBlocksSidebarOpen}
              onToggle={handleSidebarToggle}
              blocks={props.buildingBlocks || []}
              showAiBuilderTab={props.showAiBuilderTab}
              canvasId={props.canvasId}
              organizationId={props.organizationId}
              canvasNodes={canvasNodesForAiContext}
              onApplyAiOperations={props.onApplyAiOperations}
              integrations={props.integrations}
              canvasZoom={canvasZoom}
              disabled={readOnly}
              disabledMessage="You don't have permission to edit this canvas."
              onBlockClick={handleBuildingBlockClick}
              onAddNote={handleAddNote}
            />
          )}

          <div className={`flex-1 relative ${canvasStateBorderClass}`}>
            {showPreviewFloatingBar || showAwaitingFloatingBar ? (
              <div className="pointer-events-none absolute inset-x-0 top-0 z-[19] flex justify-center pt-3">
                <div
                  className={cn(
                    "pointer-events-auto flex max-w-[min(100vw-2rem,42rem)] items-center gap-2 rounded-md pl-3 pr-1.5 py-1.5 shadow-md backdrop-blur-sm outline outline-1 outline-offset-0 outline-black/10",
                    showAwaitingFloatingBar
                      ? props.awaitingApprovalBanner?.reviewUi.floatingBarBgClassName
                      : "bg-sky-50",
                  )}
                >
                  <span
                    className={cn(
                      "flex min-w-0 max-w-full items-center gap-1 text-sm",
                      showAwaitingFloatingBar ? undefined : "shrink-0 truncate font-medium text-sky-700",
                    )}
                  >
                    {showAwaitingFloatingBar ? (
                      <>
                        <span className={props.awaitingApprovalBanner?.reviewUi.dotClassName}>{"\u25cf"}</span>
                        <span className={props.awaitingApprovalBanner?.reviewUi.titleClassName}>
                          {props.awaitingApprovalBanner?.reviewUi.label}
                        </span>
                      </>
                    ) : (
                      "Previewing previous version"
                    )}
                  </span>
                  <Button
                    type="button"
                    variant="outline"
                    size="sm"
                    className="shrink-0"
                    onClick={() => {
                      if (showAwaitingFloatingBar) {
                        props.awaitingApprovalBanner?.onViewNodeDiff?.();
                      } else {
                        props.onPreviewPreviousVersionViewDetails?.();
                      }
                    }}
                  >
                    View details
                  </Button>
                </div>
              </div>
            ) : null}
            {canvasStateLabel ? (
              <div
                className={`uppercase absolute bottom-0 right-0 z-20 px-3 py-1 text-xs font-semibold text-white ${canvasStateBadgeClass}`}
              >
                {canvasStateLabel}
              </div>
            ) : null}
            <ReactFlowProvider key="canvas-flow-provider" data-testid="canvas-drop-area">
              <CanvasContent
                state={state}
                onSave={props.onSave}
                onNodeEdit={handleNodeEdit}
                onNodeDelete={handleNodeDelete}
                onNodesDelete={props.onNodesDelete}
                onDuplicateNodes={props.onDuplicateNodes}
                onAutoLayoutNodes={props.onAutoLayoutNodes}
                onEdgeCreate={props.onEdgeCreate}
                hideHeader={true}
                onToggleView={handleToggleView}
                onToggleCollapse={props.onToggleCollapse}
                onRun={(nodeId) => handleNodeRun(nodeId)}
                onDuplicate={props.onDuplicate}
                onConfigure={props.onConfigure}
                onDeactivate={props.onDeactivate}
                onAnnotationUpdate={props.onAnnotationUpdate}
                onAnnotationBlur={props.onAnnotationBlur}
                onGroupUpdate={props.onGroupUpdate}
                onGroupNodes={props.onGroupNodes}
                onUngroupNodes={props.onUngroupNodes}
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
                versionLabel={props.versionLabel}
                onCreateVersion={props.onCreateVersion}
                onPublishVersion={props.onPublishVersion}
                onDiscardVersion={props.onDiscardVersion}
                createVersionDisabled={props.createVersionDisabled}
                createVersionDisabledTooltip={props.createVersionDisabledTooltip}
                publishVersionDisabled={props.publishVersionDisabled}
                publishVersionDisabledTooltip={props.publishVersionDisabledTooltip}
                discardVersionDisabled={props.discardVersionDisabled}
                discardVersionDisabledTooltip={props.discardVersionDisabledTooltip}
                headerMode={props.headerMode}
                saveState={props.saveState}
                lastSavedAt={props.lastSavedAt}
                saveErrorMessage={props.saveErrorMessage}
                onEnterEditMode={props.onEnterEditMode}
                enterEditModeDisabled={props.enterEditModeDisabled}
                enterEditModeDisabledTooltip={props.enterEditModeDisabledTooltip}
                onExitEditMode={props.onExitEditMode}
                exitEditModeDisabled={props.exitEditModeDisabled}
                exitEditModeDisabledTooltip={props.exitEditModeDisabledTooltip}
                unpublishedDraftChangeCount={props.unpublishedDraftChangeCount}
                isVersionControlOpen={props.isVersionControlOpen}
                onOpenVersionControl={props.onOpenVersionControl}
                versionControlButtonTooltip={props.versionControlButtonTooltip}
                versionControlNotificationCount={props.versionControlNotificationCount}
                showBottomStatusControls={props.showBottomStatusControls}
                isAutoLayoutOnUpdateEnabled={props.isAutoLayoutOnUpdateEnabled}
                onToggleAutoLayoutOnUpdate={props.onToggleAutoLayoutOnUpdate}
                autoLayoutOnUpdateDisabled={props.autoLayoutOnUpdateDisabled}
                autoLayoutOnUpdateDisabledTooltip={props.autoLayoutOnUpdateDisabledTooltip}
                readOnly={props.readOnly}
                logEntries={props.logEntries}
                focusRequest={props.focusRequest}
                onExecutionChainHandled={props.onExecutionChainHandled}
                initialFocusNodeId={props.initialFocusNodeId}
                runsEvents={props.runsEvents}
                runsTotalCount={props.runsTotalCount}
                runsHasNextPage={props.runsHasNextPage}
                runsIsFetchingNextPage={props.runsIsFetchingNextPage}
                onRunsLoadMore={props.onRunsLoadMore}
                runsNodes={props.runsNodes}
                runsComponentIconMap={props.runsComponentIconMap}
                runsNodeQueueItemsMap={props.runsNodeQueueItemsMap}
                onRunNodeSelect={props.onRunNodeSelect}
                onRunExecutionSelect={props.onRunExecutionSelect}
                onAcknowledgeErrors={props.onAcknowledgeErrors}
                title={props.title}
                missingIntegrations={props.missingIntegrations}
                onConnectIntegration={props.onConnectIntegration}
                canCreateIntegrations={props.canCreateIntegrations}
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
              configurationSaveMode={props.configurationSaveMode}
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
      )}

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
  configurationSaveMode = "manual",
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
  onSaveConfiguration?: (
    configuration: Record<string, any>,
    nodeName: string,
    integrationRef?: ComponentsIntegrationRef,
  ) => void | Promise<void>;
  configurationSaveMode?: "manual" | "auto";
  onEdit?: (nodeId: string) => void;
  currentTab?: "latest" | "settings" | "docs";
  onTabChange?: (tab: "latest" | "settings" | "docs") => void;
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

  const componentDocsData = useMemo(() => {
    const blockName = editingNodeData?.blockName;
    if (!blockName) return null;

    const matchedComponent = components?.find((c) => c.name === blockName);
    if (matchedComponent) {
      return buildSidebarComponentDocsPayload(blockName, editingNodeData, {
        label: matchedComponent.label,
        description: matchedComponent.description,
        examplePayload: matchedComponent.exampleOutput,
        payloadLabel: "Example Output",
      });
    }

    const matchedTrigger = triggers?.find((t) => t.name === blockName);
    if (matchedTrigger) {
      return buildSidebarComponentDocsPayload(blockName, editingNodeData, {
        label: matchedTrigger.label,
        description: matchedTrigger.description,
        examplePayload: matchedTrigger.exampleData,
        payloadLabel: "Example Data",
      });
    }

    return null;
  }, [
    editingNodeData?.blockName,
    editingNodeData?.displayLabel,
    editingNodeData?.integrationName,
    editingNodeData?.integrationLabel,
    components,
    triggers,
  ]);

  if (!sidebarData) {
    return null;
  }

  // Show loading state when data is being fetched (skip for annotation nodes)
  if (sidebarData.isLoading && currentTab === "latest" && !isAnnotationNode) {
    const saved = localStorage.getItem(COMPONENT_SIDEBAR_WIDTH_STORAGE_KEY);
    const sidebarWidth = saved ? parseInt(saved, 10) : 450;

    return (
      <div
        className="border-l-1 border-border absolute right-0 top-0 h-full z-21 overflow-y-auto overflow-x-hidden bg-white"
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
      configurationSaveMode={configurationSaveMode}
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
      componentDescription={componentDocsData?.description}
      componentExamplePayload={componentDocsData?.examplePayload}
      componentPayloadLabel={componentDocsData?.payloadLabel}
      componentDocumentationUrl={componentDocsData?.documentationUrl}
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
      hideDocsTab={isAnnotationNode}
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
  versionLabel,
  onCreateVersion,
  onPublishVersion,
  onDiscardVersion,
  createVersionDisabled,
  createVersionDisabledTooltip,
  publishVersionDisabled,
  publishVersionDisabledTooltip,
  discardVersionDisabled,
  discardVersionDisabledTooltip,
  headerMode,
  saveState,
  lastSavedAt,
  saveErrorMessage,
  onEnterEditMode,
  enterEditModeDisabled,
  enterEditModeDisabledTooltip,
  onExitEditMode,
  exitEditModeDisabled,
  exitEditModeDisabledTooltip,
  unpublishedDraftChangeCount,
  topViewMode,
  onTopViewModeChange,
  memoryItemCount,
  onExportYamlCopy,
  onExportYamlDownload,
  canvasId,
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
  versionLabel?: string;
  onCreateVersion?: () => void;
  onPublishVersion?: () => void;
  onDiscardVersion?: () => void;
  createVersionDisabled?: boolean;
  createVersionDisabledTooltip?: string;
  publishVersionDisabled?: boolean;
  publishVersionDisabledTooltip?: string;
  discardVersionDisabled?: boolean;
  discardVersionDisabledTooltip?: string;
  headerMode?: "default" | "version-live" | "version-edit" | "versioning-disabled";
  saveState?: "saved" | "saving" | "unsaved" | "error";
  lastSavedAt?: Date | string | null;
  saveErrorMessage?: string | null;
  onEnterEditMode?: () => void;
  enterEditModeDisabled?: boolean;
  enterEditModeDisabledTooltip?: string;
  onExitEditMode?: () => void;
  exitEditModeDisabled?: boolean;
  exitEditModeDisabledTooltip?: string;
  unpublishedDraftChangeCount?: number;
  topViewMode?: "canvas" | "yaml" | "memory" | "settings";
  onTopViewModeChange?: (mode: "canvas" | "yaml" | "memory" | "settings") => void;
  memoryItemCount?: number;
  onExportYamlCopy?: (nodes: CanvasNode[]) => void;
  onExportYamlDownload?: (nodes: CanvasNode[]) => void;
  canvasId?: string;
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
      versionLabel={versionLabel}
      onCreateVersion={onCreateVersion}
      onPublishVersion={onPublishVersion}
      onDiscardVersion={onDiscardVersion}
      createVersionDisabled={createVersionDisabled}
      createVersionDisabledTooltip={createVersionDisabledTooltip}
      publishVersionDisabled={publishVersionDisabled}
      publishVersionDisabledTooltip={publishVersionDisabledTooltip}
      discardVersionDisabled={discardVersionDisabled}
      discardVersionDisabledTooltip={discardVersionDisabledTooltip}
      mode={headerMode}
      saveState={saveState}
      lastSavedAt={lastSavedAt}
      saveErrorMessage={saveErrorMessage}
      onEnterEditMode={onEnterEditMode}
      enterEditModeDisabled={enterEditModeDisabled}
      enterEditModeDisabledTooltip={enterEditModeDisabledTooltip}
      onExitEditMode={onExitEditMode}
      exitEditModeDisabled={exitEditModeDisabled}
      exitEditModeDisabledTooltip={exitEditModeDisabledTooltip}
      unpublishedDraftChangeCount={unpublishedDraftChangeCount}
      topViewMode={topViewMode}
      onTopViewModeChange={onTopViewModeChange}
      memoryItemCount={memoryItemCount}
      onExportYamlCopy={onExportYamlCopy ? handleExportYamlCopy : undefined}
      onExportYamlDownload={onExportYamlDownload ? handleExportYamlDownload : undefined}
      canvasId={canvasId}
    />
  );
}

type AbsoluteNodeRect = { x: number; y: number; w: number; h: number };
type NodeLike = {
  id: string;
  position: { x: number; y: number };
  measured?: { width?: number; height?: number };
  width?: number;
  height?: number;
};
type InternalNodeFull = {
  internals: { positionAbsolute: { x: number; y: number } };
  measured?: { width?: number; height?: number };
};

function resolveNodeWidth(internal: InternalNodeFull | undefined, node: NodeLike): number {
  return internal?.measured?.width ?? node.measured?.width ?? node.width ?? 240;
}

function resolveNodeHeight(internal: InternalNodeFull | undefined, node: NodeLike): number {
  return internal?.measured?.height ?? node.measured?.height ?? node.height ?? 80;
}

function resolveAbsoluteNodeRect(
  node: NodeLike,
  getInternalNode: (nodeId: string) => InternalNodeFull | undefined,
): AbsoluteNodeRect {
  const internal = getInternalNode(node.id);
  return {
    x: internal?.internals.positionAbsolute.x ?? node.position.x,
    y: internal?.internals.positionAbsolute.y ?? node.position.y,
    w: resolveNodeWidth(internal, node),
    h: resolveNodeHeight(internal, node),
  };
}

function CanvasContent({
  state,
  onSave,
  onNodeEdit,
  onNodeDelete,
  onNodesDelete,
  onDuplicateNodes,
  onAutoLayoutNodes,
  onEdgeCreate,
  hideHeader,
  onRun,
  onDuplicate,
  onConfigure,
  onDeactivate,
  onTogglePause,
  onToggleView,
  onToggleCollapse,
  onAnnotationUpdate,
  onAnnotationBlur,
  onGroupUpdate,
  onGroupNodes,
  onUngroupNodes,
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
  versionLabel,
  onCreateVersion,
  onPublishVersion,
  onDiscardVersion,
  createVersionDisabled,
  createVersionDisabledTooltip,
  publishVersionDisabled,
  publishVersionDisabledTooltip,
  discardVersionDisabled,
  discardVersionDisabledTooltip,
  headerMode,
  saveState,
  lastSavedAt,
  saveErrorMessage,
  onEnterEditMode,
  enterEditModeDisabled,
  enterEditModeDisabledTooltip,
  onExitEditMode,
  exitEditModeDisabled,
  exitEditModeDisabledTooltip,
  unpublishedDraftChangeCount,
  isVersionControlOpen,
  onOpenVersionControl,
  versionControlButtonTooltip,
  versionControlNotificationCount = 0,
  showBottomStatusControls = true,
  isAutoLayoutOnUpdateEnabled,
  onToggleAutoLayoutOnUpdate,
  autoLayoutOnUpdateDisabled,
  autoLayoutOnUpdateDisabledTooltip,
  readOnly,
  logEntries = [],
  focusRequest,
  initialFocusNodeId,
  runsEvents,
  runsTotalCount,
  runsHasNextPage,
  runsIsFetchingNextPage,
  onRunsLoadMore,
  runsNodes,
  runsComponentIconMap,
  runsNodeQueueItemsMap,
  onRunNodeSelect,
  onRunExecutionSelect,
  onAcknowledgeErrors,
  title,
  missingIntegrations,
  onConnectIntegration,
  canCreateIntegrations,
}: {
  state: CanvasPageState;
  onSave?: (nodes: CanvasNode[]) => void;
  onNodeEdit: (nodeId: string) => void;
  onNodeDelete?: (nodeId: string) => void;
  onNodesDelete?: (nodeIds: string[]) => void;
  onDuplicateNodes?: (nodeIds: string[]) => void;
  onAutoLayoutNodes?: (nodeIds: string[]) => void;
  onEdgeCreate?: (sourceId: string, targetId: string, sourceHandle?: string | null) => void;
  hideHeader?: boolean;
  onRun?: (nodeId: string) => void;
  onDuplicate?: (nodeId: string) => void;
  onConfigure?: (nodeId: string) => void;
  onDeactivate?: (nodeId: string) => void;
  onTogglePause?: (nodeId: string) => void;
  onToggleView?: (nodeId: string) => void;
  onToggleCollapse?: () => void;
  onDelete?: (nodeId: string) => void;
  onAnnotationUpdate?: (
    nodeId: string,
    updates: { text?: string; color?: string; width?: number; height?: number; x?: number; y?: number },
  ) => void;
  onAnnotationBlur?: () => void;
  onGroupUpdate?: (nodeId: string, updates: { label?: string; description?: string; color?: string }) => void;
  onGroupNodes?: (
    bounds: { x: number; y: number; width: number; height: number },
    nodePositions: Array<{ id: string; x: number; y: number }>,
  ) => void;
  onUngroupNodes?: (groupNodeId: string) => void;
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
  setCurrentTab?: (tab: "latest" | "settings" | "docs") => void;
  onUndo?: () => void;
  canUndo?: boolean;
  organizationId?: string;
  unsavedMessage?: string;
  saveIsPrimary?: boolean;
  saveButtonHidden?: boolean;
  saveDisabled?: boolean;
  saveDisabledTooltip?: string;
  versionLabel?: string;
  onCreateVersion?: () => void;
  onPublishVersion?: () => void;
  onDiscardVersion?: () => void;
  createVersionDisabled?: boolean;
  createVersionDisabledTooltip?: string;
  publishVersionDisabled?: boolean;
  publishVersionDisabledTooltip?: string;
  discardVersionDisabled?: boolean;
  discardVersionDisabledTooltip?: string;
  headerMode?: "default" | "version-live" | "version-edit" | "versioning-disabled";
  saveState?: "saved" | "saving" | "unsaved" | "error";
  lastSavedAt?: Date | string | null;
  saveErrorMessage?: string | null;
  onEnterEditMode?: () => void;
  enterEditModeDisabled?: boolean;
  enterEditModeDisabledTooltip?: string;
  onExitEditMode?: () => void;
  exitEditModeDisabled?: boolean;
  exitEditModeDisabledTooltip?: string;
  unpublishedDraftChangeCount?: number;
  isVersionControlOpen?: boolean;
  onOpenVersionControl?: () => void;
  versionControlButtonTooltip?: string;
  versionControlNotificationCount?: number;
  showBottomStatusControls?: boolean;
  isAutoLayoutOnUpdateEnabled?: boolean;
  onToggleAutoLayoutOnUpdate?: () => void;
  autoLayoutOnUpdateDisabled?: boolean;
  autoLayoutOnUpdateDisabledTooltip?: string;
  readOnly?: boolean;
  logEntries?: LogEntry[];
  focusRequest?: FocusRequest | null;
  onExecutionChainHandled?: () => void;
  initialFocusNodeId?: string | null;
  runsEvents?: CanvasesCanvasEventWithExecutions[];
  runsTotalCount?: number;
  runsHasNextPage?: boolean;
  runsIsFetchingNextPage?: boolean;
  onRunsLoadMore?: () => void;
  runsNodes?: ComponentsNode[];
  runsComponentIconMap?: Record<string, string>;
  runsNodeQueueItemsMap?: Record<string, CanvasesCanvasNodeQueueItem[]>;
  onRunNodeSelect?: (nodeId: string) => void;
  onRunExecutionSelect?: (options: {
    nodeId: string;
    eventId: string;
    executionId: string;
    triggerEvent?: SidebarEvent;
  }) => void;
  onAcknowledgeErrors?: (executionIds: string[]) => void;
  title?: string;
  missingIntegrations?: MissingIntegration[];
  onConnectIntegration?: (integrationName: string) => void;
  canCreateIntegrations?: boolean;
}) {
  const { fitView, screenToFlowPosition, getViewport, getInternalNode } = useReactFlow();
  const { zoom } = useViewport();
  const isReadOnly = readOnly ?? false;

  // Determine selection key code to support both Control (Windows/Linux) and Meta (Mac)
  // Similar to existing keyboard shortcuts that check (e.ctrlKey || e.metaKey)
  const selectionKey = useMemo(() => {
    const isMac = navigator.platform.toLowerCase().includes("mac");
    return isMac ? "Meta" : "Control";
  }, []);

  const computeSelectionBounds = useCallback(
    (nodes: CanvasNode[]) => {
      let minX = Infinity,
        minY = Infinity,
        maxX = -Infinity,
        maxY = -Infinity;
      const nodePositions = nodes.map((n) => {
        const rect = resolveAbsoluteNodeRect(n, getInternalNode);
        if (rect.x < minX) minX = rect.x;
        if (rect.y < minY) minY = rect.y;
        if (rect.x + rect.w > maxX) maxX = rect.x + rect.w;
        if (rect.y + rect.h > maxY) maxY = rect.y + rect.h;
        return { id: n.id, x: rect.x, y: rect.y };
      });
      return {
        bounds: { x: minX, y: minY, width: maxX - minX, height: maxY - minY },
        nodePositions,
      };
    },
    [getInternalNode],
  );

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
  const lastReportedZoomRef = useRef<number | null>(null);
  const reportZoom = useCallback(
    (zoom: number) => {
      if (!onZoomChange) {
        return;
      }

      if (lastReportedZoomRef.current === zoom) {
        return;
      }

      lastReportedZoomRef.current = zoom;
      onZoomChange(zoom);
    },
    [onZoomChange],
  );

  // Track if we've initialized to prevent flicker
  const [isInitialized, setIsInitialized] = useState(hasFitToViewRef.current);
  const [isLogSidebarOpen, setIsLogSidebarOpen] = useState(() => {
    const saved = localStorage.getItem(CONSOLE_OPEN_STORAGE_KEY);
    return saved !== null ? saved === "true" : false;
  });
  const [consoleTab, setConsoleTab] = useState<ConsoleTab>("runs");
  const [logSearch, setLogSearch] = useState("");
  const [logSidebarHeight, setLogSidebarHeight] = useState(() => {
    const saved = localStorage.getItem(CONSOLE_HEIGHT_STORAGE_KEY);
    return saved ? parseInt(saved, 10) : 320;
  });
  const [isSnapToGridEnabled, setIsSnapToGridEnabled] = useState(true);
  const { isMinimapVisible, setIsMinimapVisible } = useMinimapVisibility(false);

  useEffect(() => {
    if (showBottomStatusControls) {
      localStorage.setItem(CONSOLE_OPEN_STORAGE_KEY, String(isLogSidebarOpen));
    }
  }, [isLogSidebarOpen, showBottomStatusControls]);

  useEffect(() => {
    localStorage.setItem(CONSOLE_HEIGHT_STORAGE_KEY, String(logSidebarHeight));
  }, [logSidebarHeight]);

  const runsCountInfo = useMemo(() => {
    const events = runsEvents || [];
    let running = 0;
    for (const event of events) {
      const execs = event.executions || [];
      if (execs.some((e) => e.state === "STATE_STARTED" || e.state === "STATE_PENDING")) {
        running++;
      }
    }
    return { total: runsTotalCount || events.length, running };
  }, [runsEvents, runsTotalCount]);

  const unacknowledgedErrorCount = useMemo(() => countUnacknowledgedErrors(runsEvents || []), [runsEvents]);

  useEffect(() => {
    if (!showBottomStatusControls) {
      setIsLogSidebarOpen(false);
    }
  }, [showBottomStatusControls]);

  const [multiSelectedNodes, setMultiSelectedNodes] = useState<ReactFlowNode[]>([]);
  const [isSelecting, setIsSelecting] = useState(false);
  const previouslySelectedRef = useRef<Set<string>>(new Set());

  const stopCanvasPointerEvent = useCallback((event: SyntheticEvent) => {
    event.preventDefault();
    event.stopPropagation();
  }, []);

  useOnSelectionChange({
    onChange: useCallback(({ nodes }: { nodes: ReactFlowNode[] }) => {
      setMultiSelectedNodes(nodes.length >= 2 ? nodes : []);
    }, []),
  });

  const multiSelectedNodeIds = useMemo(() => new Set(multiSelectedNodes.map((n) => n.id)), [multiSelectedNodes]);

  const selectionToolbarFlowPos = useMemo(() => {
    if (multiSelectedNodeIds.size < 2) return null;

    let minY = Infinity;
    let maxX = -Infinity;

    for (const node of state.nodes) {
      if (!multiSelectedNodeIds.has(node.id)) continue;
      const rect = resolveAbsoluteNodeRect(node, getInternalNode);
      if (rect.y < minY) minY = rect.y;
      if (rect.x + rect.w > maxX) maxX = rect.x + rect.w;
    }

    return { x: maxX, y: minY };
  }, [multiSelectedNodeIds, state.nodes, getInternalNode]);

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
    (nodeId: string, e?: React.MouseEvent) => {
      const isMultiSelectClick = e && (e.ctrlKey || e.metaKey);
      if (isMultiSelectClick) return;

      const clickedNode = stateRef.current.nodes?.find((n) => n.id === nodeId);
      const isPendingConnection = clickedNode?.data?.isPendingConnection;
      const isAnnotationNode = clickedNode?.data?.type === "annotation";
      const isGroupNode = clickedNode?.data?.type === "group";

      const workflowNode = workflowNodes?.find((n) => n.id === nodeId);
      const isPlaceholder = workflowNode?.name === "New Component" && !workflowNode.component?.name;

      const isTemplateNode = clickedNode?.data?.isTemplate && !clickedNode?.data?.isPendingConnection;

      const currentTemplateNode = templateNodeId ? stateRef.current.nodes?.find((n) => n.id === templateNodeId) : null;
      const isCurrentTemplateConfigured =
        currentTemplateNode?.data?.isTemplate && !currentTemplateNode?.data?.isPendingConnection;

      if (
        isCurrentTemplateConfigured &&
        nodeId !== templateNodeId &&
        !isPendingConnection &&
        !isTemplateNode &&
        !isPlaceholder
      ) {
        return;
      }

      if (isAnnotationNode || isGroupNode) {
        return;
      }

      if (isPendingConnection && onPendingConnectionNodeClick) {
        onPendingConnectionNodeClick(nodeId);
      } else if (isPlaceholder && onPendingConnectionNodeClick) {
        onPendingConnectionNodeClick(nodeId);
      } else {
        if (isTemplateNode && onTemplateNodeClick) {
          onTemplateNodeClick(nodeId);
        } else {
          stateRef.current.componentSidebar.open(nodeId);

          const nodeData = clickedNode?.data as {
            component?: { error?: string };
            composite?: { error?: string };
            trigger?: { error?: string };
          } | null;
          const hasConfigurationWarning = Boolean(
            nodeData?.component?.error || nodeData?.composite?.error || nodeData?.trigger?.error,
          );

          if (setCurrentTab) {
            setCurrentTab(hasConfigurationWarning ? "settings" : "latest");
          }

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
  const onGroupUpdateRef = useRef(onGroupUpdate);
  onGroupUpdateRef.current = onGroupUpdate;
  const onUngroupNodesRef = useRef(onUngroupNodes);
  onUngroupNodesRef.current = onUngroupNodes;

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
      viewportRef.current = newViewport;
      reportZoom(newViewport.zoom);
    },
    [reportZoom, viewportRef],
  );

  const handleToggleCollapse = useCallback(() => {
    state.toggleCollapse();
    onToggleCollapse?.();
  }, [state.toggleCollapse, onToggleCollapse]);

  const handleToggleAutoLayoutOnUpdate = useCallback(() => {
    if (isReadOnly || !onToggleAutoLayoutOnUpdate || autoLayoutOnUpdateDisabled) {
      return;
    }
    onToggleAutoLayoutOnUpdate();
  }, [isReadOnly, onToggleAutoLayoutOnUpdate, autoLayoutOnUpdateDisabled]);

  const isAutoLayoutToggleDisabled = isReadOnly || !onToggleAutoLayoutOnUpdate || autoLayoutOnUpdateDisabled;
  const autoLayoutTooltipMessage =
    autoLayoutOnUpdateDisabledTooltip ||
    (isAutoLayoutOnUpdateEnabled
      ? "Auto-layout on add is enabled. New nodes reflow their connected graph."
      : "Auto-layout on add is disabled. Click to enable connected-graph layout for newly added nodes.");

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

    previouslySelectedRef.current = new Set();

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
          reportZoom(initialViewport.zoom);
        } else {
          const defaultViewport = viewportRef.current ?? { x: 0, y: 0, zoom: DEFAULT_CANVAS_ZOOM };
          viewportRef.current = defaultViewport;
          reactFlowInstance.setViewport(defaultViewport);
          reportZoom(defaultViewport.zoom);
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
    [fitView, getViewport, reportZoom, hasFitToViewRef, viewportRef, initialFocusNodeId],
  );

  const showHeader = !isReadOnly;

  const hasMultiSelection = multiSelectedNodes.length >= 2;

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
    onGroupUpdate: onGroupUpdateRef,
    onUngroupNodes: onUngroupNodesRef,
    aiState: state.ai,
    runDisabled,
    runDisabledTooltip,
    showHeader,
    hasMultiSelection,
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
    onGroupUpdate: onGroupUpdateRef,
    onUngroupNodes: onUngroupNodesRef,
    aiState: state.ai,
    runDisabled,
    runDisabledTooltip,
    showHeader,
    hasMultiSelection,
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
  }, [state.nodes, hoveredEdge, connectingFrom, state.edges, highlightedNodeIds, hasMultiSelection]);

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
      const prev = previouslySelectedRef.current;
      const nodes = stateRef.current.nodes ?? [];

      if (prev.size > 0) {
        changes = changes.map((c) => {
          if (c.type === "select" && !c.selected && prev.has(c.id)) {
            return { ...c, selected: true };
          }
          return c;
        });
      }

      if (!isReadOnly) {
        state.onNodesChange(clampGroupChildNodePositionChanges(changes, nodes));
        resizeGroupsAfterChildChanges(changes, nodes, state.setNodes);
        return;
      }

      const filteredChanges = changes.filter((change) => change.type === "select" || change.type === "dimensions");
      if (filteredChanges.length > 0) {
        state.onNodesChange(filteredChanges);
      }

      resizeGroupsAfterChildChanges(changes, stateRef.current.nodes ?? [], state.setNodes);
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
    if (!query) return logEntries;

    const matchesSearch = (value?: string) => (value || "").toLowerCase().includes(query);
    return logEntries.filter(
      (entry) => matchesSearch(entry.searchText) || matchesSearch(typeof entry.title === "string" ? entry.title : ""),
    );
  }, [logEntries, logSearch]);

  const handleLogButtonClick = useCallback((tab: ConsoleTab) => {
    setConsoleTab(tab);
    setIsLogSidebarOpen(true);
  }, []);

  const showVersionControlTrigger = showBottomStatusControls && !!onOpenVersionControl && !isVersionControlOpen;

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
          versionLabel={versionLabel}
          onCreateVersion={onCreateVersion}
          onPublishVersion={onPublishVersion}
          onDiscardVersion={onDiscardVersion}
          createVersionDisabled={createVersionDisabled}
          createVersionDisabledTooltip={createVersionDisabledTooltip}
          publishVersionDisabled={publishVersionDisabled}
          publishVersionDisabledTooltip={publishVersionDisabledTooltip}
          discardVersionDisabled={discardVersionDisabled}
          discardVersionDisabledTooltip={discardVersionDisabledTooltip}
          mode={headerMode}
          saveState={saveState}
          lastSavedAt={lastSavedAt}
          saveErrorMessage={saveErrorMessage}
          onEnterEditMode={onEnterEditMode}
          enterEditModeDisabled={enterEditModeDisabled}
          enterEditModeDisabledTooltip={enterEditModeDisabledTooltip}
          onExitEditMode={onExitEditMode}
          exitEditModeDisabled={exitEditModeDisabled}
          exitEditModeDisabledTooltip={exitEditModeDisabledTooltip}
          unpublishedDraftChangeCount={unpublishedDraftChangeCount}
        />
      )}

      <div className={hideHeader ? "h-full" : "pt-12 h-full"}>
        <div className="h-full w-full">
          <ReactFlow
            nodes={nodesWithCallbacks}
            edges={styledEdges}
            nodeTypes={nodeTypes}
            edgeTypes={edgeTypes}
            minZoom={MIN_CANVAS_ZOOM}
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
            onSelectionStart={() => {
              setIsSelecting(true);
              const selected = (stateRef.current.nodes || []).filter((n) => n.selected).map((n) => n.id);
              previouslySelectedRef.current = new Set(selected);
            }}
            onSelectionEnd={() => {
              setIsSelecting(false);
              previouslySelectedRef.current = new Set();
            }}
            onEdgeMouseEnter={handleEdgeMouseEnter}
            onEdgeMouseLeave={handleEdgeMouseLeave}
            defaultViewport={viewport}
            fitView={false}
            style={{ opacity: isInitialized ? 1 : 0 }}
            className="h-full w-full"
          >
            <Background gap={8} size={2} bgColor="#F1F5F9" color="#d9d9d9ff" />
            <CanvasMiniMap nodes={state.nodes} edges={state.edges} isVisible={isMinimapVisible} />
            <Panel
              position="bottom-left"
              className="!bg-transparent !outline-none !shadow-none p-0 flex flex-col items-start gap-4"
            >
              {missingIntegrations && missingIntegrations.length > 0 && onConnectIntegration && (
                <div style={isLogSidebarOpen ? { marginBottom: logSidebarHeight } : undefined}>
                  <IntegrationStatusIndicator
                    missingIntegrations={missingIntegrations}
                    onConnect={onConnectIntegration}
                    readOnly={isReadOnly}
                    canCreateIntegrations={canCreateIntegrations}
                  />
                </div>
              )}
              <div className="flex items-center gap-3">
                {showVersionControlTrigger ? (
                  <div className="bg-white text-gray-800 outline-1 outline-slate-950/15 flex items-center rounded-md p-0.5 h-8">
                    <Tooltip>
                      <TooltipTrigger asChild>
                        <span className="relative inline-flex">
                          <Button
                            variant="ghost"
                            size="sm"
                            className="h-8 items-center text-xs font-medium gap-1.5"
                            onClick={onOpenVersionControl}
                            aria-label="Open version control"
                          >
                            <GitBranch className="h-3 w-3" />
                          </Button>
                          {versionControlNotificationCount > 0 ? (
                            <span className="absolute left-6 -top-2 inline-flex min-w-[1.125rem] items-center justify-center rounded-full bg-orange-600 px-1 text-[10px] font-semibold leading-4 text-white">
                              {versionControlNotificationCount > 99 ? "99+" : versionControlNotificationCount}
                            </span>
                          ) : null}
                        </span>
                      </TooltipTrigger>
                      <TooltipContent>{versionControlButtonTooltip || "Open version control"}</TooltipContent>
                    </Tooltip>
                  </div>
                ) : null}
                <ZoomSlider
                  orientation="horizontal"
                  className="!static !m-0"
                  screenshotName={title}
                  isSnapToGridEnabled={isSnapToGridEnabled}
                  onSnapToGridToggle={() => setIsSnapToGridEnabled((prev) => !prev)}
                  leadingContent={
                    <Tooltip>
                      <TooltipTrigger asChild>
                        <Button
                          variant={isMinimapVisible ? "secondary" : "ghost"}
                          size="sm"
                          className={`h-8 w-8 px-0 ${
                            isMinimapVisible
                              ? "bg-emerald-50 text-emerald-700 hover:bg-emerald-100"
                              : "text-slate-600 hover:text-slate-900"
                          }`}
                          onClick={() => setIsMinimapVisible((prev: boolean) => !prev)}
                          aria-pressed={isMinimapVisible}
                        >
                          <MapIcon className="h-3 w-3" />
                        </Button>
                      </TooltipTrigger>
                      <TooltipContent>{isMinimapVisible ? "Hide minimap" : "Show minimap"}</TooltipContent>
                    </Tooltip>
                  }
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
                          variant={isAutoLayoutOnUpdateEnabled ? "secondary" : "ghost"}
                          size="sm"
                          className={`h-8 w-8 px-0 ${
                            isAutoLayoutOnUpdateEnabled
                              ? "bg-emerald-50 text-emerald-700 hover:bg-emerald-100"
                              : "text-slate-600 hover:text-slate-900"
                          }`}
                          onClick={handleToggleAutoLayoutOnUpdate}
                          disabled={isAutoLayoutToggleDisabled}
                          aria-pressed={isAutoLayoutOnUpdateEnabled}
                        >
                          <Workflow className="h-3 w-3" />
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
                {showBottomStatusControls && !isLogSidebarOpen ? (
                  <div className="bg-white text-gray-800 outline-1 outline-slate-950/15 flex items-center gap-1 rounded-md p-0.5 h-8">
                    <Tooltip>
                      <TooltipTrigger asChild>
                        <Button
                          variant="ghost"
                          size="sm"
                          className={cn(
                            "h-8 items-center text-xs font-medium",
                            runsCountInfo.running > 0 && "text-blue-600",
                          )}
                          onClick={() => handleLogButtonClick("runs")}
                        >
                          {runsCountInfo.running > 0 ? (
                            <Loader2 className="h-3 w-3 animate-spin" />
                          ) : (
                            <Play className="h-3 w-3" />
                          )}
                          <span
                            className={cn(
                              "tabular-nums",
                              runsCountInfo.running > 0 ? "text-blue-600" : "text-gray-800",
                            )}
                          >
                            {runsCountInfo.running > 0 ? runsCountInfo.running : runsCountInfo.total}
                          </span>
                        </Button>
                      </TooltipTrigger>
                      <TooltipContent>
                        {runsCountInfo.running > 0 ? `${runsCountInfo.running} running` : `${runsCountInfo.total} runs`}
                      </TooltipContent>
                    </Tooltip>
                    <Tooltip>
                      <TooltipTrigger asChild>
                        <Button
                          variant="ghost"
                          size="sm"
                          className={cn(
                            "h-8 items-center text-xs font-medium",
                            unacknowledgedErrorCount > 0 && "text-red-500",
                          )}
                          onClick={() => handleLogButtonClick("errors")}
                        >
                          <CircleX
                            className={unacknowledgedErrorCount > 0 ? "h-3 w-3 text-red-500" : "h-3 w-3 text-gray-800"}
                          />
                          <span
                            className={
                              unacknowledgedErrorCount > 0 ? "tabular-nums text-red-500" : "tabular-nums text-gray-800"
                            }
                          >
                            {unacknowledgedErrorCount}
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
                          onClick={() => handleLogButtonClick("warnings")}
                        >
                          <TriangleAlert
                            className={logCounts.warning > 0 ? "h-3 w-3 text-orange-500" : "h-3 w-3 text-gray-800"}
                          />
                          <span
                            className={
                              logCounts.warning > 0 ? "tabular-nums text-orange-500" : "tabular-nums text-gray-800"
                            }
                          >
                            {logCounts.warning}
                          </span>
                        </Button>
                      </TooltipTrigger>
                      <TooltipContent>Warnings</TooltipContent>
                    </Tooltip>
                  </div>
                ) : null}
              </div>
            </Panel>
            {selectionToolbarFlowPos &&
              !isSelecting &&
              !isReadOnly &&
              (onNodesDelete || onNodeDelete || onAutoLayoutNodes || onDuplicateNodes || onGroupNodes) && (
                <ViewportPortal>
                  <div
                    style={{
                      position: "absolute",
                      left: selectionToolbarFlowPos.x,
                      top: selectionToolbarFlowPos.y,
                      transform: "translate(-100%, -100%) translateY(-24px)",
                    }}
                  >
                    <div
                      className="nodrag nopan flex items-center gap-2"
                      onPointerDown={stopCanvasPointerEvent}
                      onMouseDown={stopCanvasPointerEvent}
                      style={{
                        transform: `scale(${1 / zoom})`,
                        transformOrigin: "bottom right",
                        pointerEvents: "all",
                      }}
                    >
                      {onGroupNodes &&
                        multiSelectedNodes.filter((n) => n.data?.type !== "group" && !n.parentId).length >= 2 && (
                          <button
                            type="button"
                            data-testid="multi-select-group"
                            onPointerDown={stopCanvasPointerEvent}
                            onMouseDown={stopCanvasPointerEvent}
                            onClick={(event) => {
                              event.preventDefault();
                              event.stopPropagation();
                              const groupable = multiSelectedNodes.filter(
                                (n) => n.data?.type !== "group" && !n.parentId,
                              );
                              const { bounds, nodePositions } = computeSelectionBounds(groupable);
                              onGroupNodes(bounds, nodePositions);
                              setMultiSelectedNodes([]);
                            }}
                            className="flex items-center justify-center p-1 text-gray-500 transition hover:text-gray-800"
                          >
                            <Group className="h-4 w-4" />
                          </button>
                        )}
                      {onAutoLayoutNodes && (
                        <button
                          type="button"
                          data-testid="multi-select-auto-layout"
                          onPointerDown={stopCanvasPointerEvent}
                          onMouseDown={stopCanvasPointerEvent}
                          onClick={(event) => {
                            event.preventDefault();
                            event.stopPropagation();
                            onAutoLayoutNodes(multiSelectedNodes.map((n) => n.id));
                          }}
                          className="flex items-center justify-center p-1 text-gray-500 transition hover:text-gray-800"
                        >
                          <LayoutGrid className="h-4 w-4" />
                        </button>
                      )}
                      {onDuplicateNodes && (
                        <button
                          type="button"
                          data-testid="multi-select-duplicate"
                          onPointerDown={stopCanvasPointerEvent}
                          onMouseDown={stopCanvasPointerEvent}
                          onClick={(event) => {
                            event.preventDefault();
                            event.stopPropagation();
                            onDuplicateNodes(multiSelectedNodes.map((n) => n.id));
                          }}
                          className="flex items-center justify-center p-1 text-gray-500 transition hover:text-gray-800"
                        >
                          <Copy className="h-4 w-4" />
                        </button>
                      )}
                      {(onNodesDelete || onNodeDelete) && (
                        <button
                          type="button"
                          data-testid="multi-select-delete"
                          onPointerDown={stopCanvasPointerEvent}
                          onMouseDown={stopCanvasPointerEvent}
                          onClick={(event) => {
                            event.preventDefault();
                            event.stopPropagation();
                            const nodeIds = multiSelectedNodes.map((n) => n.id);
                            if (onNodesDelete) {
                              onNodesDelete(nodeIds);
                            } else {
                              for (const id of nodeIds) {
                                onNodeDelete?.(id);
                              }
                            }
                            stateRef.current.setNodes((nodes) => nodes.map((node) => ({ ...node, selected: false })));
                            setMultiSelectedNodes([]);
                          }}
                          className="flex items-center justify-center p-1 text-gray-500 transition hover:text-gray-800"
                        >
                          <Trash2 className="h-4 w-4" />
                        </button>
                      )}
                    </div>
                  </div>
                </ViewportPortal>
              )}
          </ReactFlow>
        </div>
      </div>
      {showBottomStatusControls ? (
        <CanvasLogSidebar
          isOpen={isLogSidebarOpen}
          onClose={() => setIsLogSidebarOpen(false)}
          height={logSidebarHeight}
          onHeightChange={setLogSidebarHeight}
          searchValue={logSearch}
          onSearchChange={setLogSearch}
          entries={filteredLogEntries}
          counts={logCounts}
          activeTab={consoleTab}
          onTabChange={setConsoleTab}
          runsEvents={runsEvents}
          runsTotalCount={runsTotalCount}
          runsHasNextPage={runsHasNextPage}
          runsIsFetchingNextPage={runsIsFetchingNextPage}
          onRunsLoadMore={onRunsLoadMore}
          runsNodes={runsNodes}
          runsComponentIconMap={runsComponentIconMap}
          runsNodeQueueItemsMap={runsNodeQueueItemsMap}
          onRunNodeSelect={onRunNodeSelect}
          onRunExecutionSelect={onRunExecutionSelect}
          onAcknowledgeErrors={onAcknowledgeErrors}
        />
      ) : null}
    </div>
  );
}

export type { BuildingBlock } from "../BuildingBlocksSidebar";
export type { MissingIntegration } from "../IntegrationStatusIndicator";
export { CanvasPage };
