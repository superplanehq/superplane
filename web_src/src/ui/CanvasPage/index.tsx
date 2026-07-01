import {
  Background,
  Panel,
  ReactFlow,
  ReactFlowProvider,
  ViewportPortal,
  useOnSelectionChange,
  useReactFlow,
  useViewport,
  type Connection,
  type EdgeChange,
  type NodeChange,
  type Edge as ReactFlowEdge,
  type Node as ReactFlowNode,
  type Viewport,
} from "@xyflow/react";

import { GlobalCommandPaletteCanvasNodeSearch } from "@/components/GlobalCommandPalette/canvasNodeSearch";
import { openGlobalCommandPalette } from "@/components/GlobalCommandPalette/controller";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogDescription, DialogTitle } from "@/components/ui/dialog";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { ZoomSlider } from "@/components/zoom-slider";
import { getDraftDiffEdgeStyle } from "@/lib/draftDiff";
import { cn } from "@/lib/utils";
import { CircleX, Copy, LayoutDashboard, LayoutGrid, Loader2, Search, Trash2, CircleAlert } from "lucide-react";
import {
  Component,
  memo,
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
  type ErrorInfo,
  type ReactNode,
  type SyntheticEvent,
} from "react";

import type {
  CanvasesCanvasRun,
  CanvasesCanvasNodeExecution,
  ActionsAction,
  ComponentsIntegrationRef,
  SuperplaneComponentsNode as ComponentsNode,
  ConfigurationField,
  OrganizationsIntegration,
  TriggersTrigger,
} from "@/api-client";
import { CanvasRunsSidebar } from "@/components/CanvasRunsSidebar";
import type { CanvasRunsSidebarState } from "@/components/CanvasRunsSidebar/useCanvasRunsSidebarState";
import { useCanvasRunsSidebarState } from "@/components/CanvasRunsSidebar/useCanvasRunsSidebarState";
import { CanvasVersionsSidebar } from "@/components/CanvasVersionsSidebar";
import type { CanvasVersionsSidebarState } from "@/components/CanvasVersionsSidebar/useCanvasVersionsSidebarState";
import { useCanvasVersionsSidebarState } from "@/components/CanvasVersionsSidebar/useCanvasVersionsSidebarState";
import { CanvasToolSidebar } from "@/components/CanvasToolSidebar";
import {
  useCanvasToolSidebarState,
  type CanvasToolSidebarState,
} from "@/components/CanvasToolSidebar/useCanvasToolSidebarState";
import { buildSidebarComponentDocsPayload } from "@/lib/componentDocsUrl";
import { parseDefaultValues } from "@/lib/components";
import { countUnacknowledgedErrors } from "@/pages/app/lib/canvas-runs";
import { findFreePositionInViewport } from "@/pages/app/lib/find-free-position-in-viewport";
import {
  allowsBuildingBlocksSidebar,
  blocksBuildingBlocksShortcut,
  isCanvasWorkflowTab,
  isPanelHeaderMode,
  normalizeCanvasHeaderMode,
} from "@/pages/app/viewState";
import { CANVAS_NODE_FALLBACK_MESSAGE } from "@/pages/app/mappers/safeMappers";
import { LIVE_CANVAS_FIT_VIEW_OPTIONS, RUN_CANVAS_FIT_VIEW_OPTIONS } from "@/ui/CanvasPage/canvasFitOptions";
import { Sentry } from "@/sentry";
import { useSidebarLayoutStore, useSidebarMount } from "@/stores/sidebarLayoutStore";
import { getActiveNoteId, restoreActiveNoteFocus } from "@/ui/annotationComponent/noteFocus";
import type { BuildingBlock, BuildingBlockCategory } from "../BuildingBlocksSidebar";
import { BuildingBlocksSidebar } from "../BuildingBlocksSidebar";
import { CanvasLogSidebar, type ConsoleTab, type LogEntry } from "../CanvasLogSidebar";
import type { EventState, EventStateMap } from "../componentBase";
import { ComponentSidebar } from "../componentSidebar";
import type { TabData } from "../componentSidebar/SidebarEventItem/SidebarEventItem";
import type { SidebarEvent } from "../componentSidebar/types";
import { IntegrationStatusIndicator, type MissingIntegration } from "../IntegrationStatusIndicator";
import { RunNodeDetailPane } from "../Runs/RunNodeDetailPane";
import { ResizableBottomPane } from "./ResizableBottomPane";
import { LiveBottomInspectorEmptyState } from "./LiveBottomInspectorEmptyState";
import { Block, type BlockData, type BlockProps, type CanvasBlockData } from "./Block";
import "./canvas-reset.css";
import { CustomEdge } from "./CustomEdge";
import { Header } from "./Header";
import { isComponentSidebarVisibleMode } from "./canvasTabHeaderMode";
import { isCanvasNodeHighlighted, shouldBlankCanvasNodeBody } from "./nodeDimming";
import { RightSideControls } from "./RightSideControls";
import { useBuildingBlocksShortcut } from "./useBuildingBlocksShortcut";
import type { CanvasPageState } from "./useCanvasState";
import { useCanvasState } from "./useCanvasState";
import type { TriggerActionModal } from "@/pages/app/mappers/types";

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

/* eslint-disable-next-line @typescript-eslint/no-empty-object-type --
   Having a specific type allows us to extend it with additional properties without breaking consumers.
 */
export interface CanvasNode extends ReactFlowNode {}

export interface CanvasEdge extends ReactFlowEdge {
  sourceHandle?: string | null;
  targetHandle?: string | null;
}

interface FocusRequest {
  nodeId: string;
  requestId: number;
  tab?: "latest" | "settings";
}

export interface NodeEditData {
  nodeId: string;
  nodeName: string;
  displayLabel?: string;
  configuration: Record<string, unknown>;
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
  configuration: Record<string, unknown>;
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
  /** Display name for the canvas header (center title). */
  title?: string;
  headerBanner?: React.ReactNode;
  organizationId?: string;
  canvasId?: string;
  onPublishVersion?: () => void;
  onDiscardVersion?: () => void;
  onShowDiff?: () => void;
  onShowConsoleDiff?: () => void;
  onShowNodeDiff?: (nodeId: string) => void;
  visualDiffEnabled?: boolean;
  draftVisualDiff?: {
    diffCounts: { added: number; updated: number; removed: number };
    diffToggles: {
      showDeletedNodes: boolean;
      toggleShowDeletedNodes: () => void;
      showEdgeDiff: boolean;
      toggleShowEdgeDiff: () => void;
    };
  };
  draftConsoleDiff?: {
    diffCounts: { added: number; updated: number; removed: number };
  };
  onToggleVisualDiff?: () => void;
  publishVersionDisabled?: boolean;
  publishVersionDisabledTooltip?: string;
  discardVersionDisabled?: boolean;
  discardVersionDisabledTooltip?: string;
  /** True when the active draft has uncommitted staged spec edits. Shows the Commit/Reset controls. */
  hasStagingChanges?: boolean;
  /** Commits staged canvas.yaml/console.yaml edits into the draft version row. */
  onCommitStaging?: () => void;
  commitStagingPending?: boolean;
  resetStagingPending?: boolean;
  /** Discards staged edits, reverting to the last committed draft. */
  onResetStaging?: () => void;
  headerMode?: "default" | "version-live" | "console" | "memory" | "files";
  /** Node settings sidebar: canvas uses debounced autosave without closing the panel after each save. */
  configurationSaveMode?: "manual" | "auto";
  onEnterEditMode?: () => void;
  enterEditModeDisabled?: boolean;
  enterEditModeDisabledTooltip?: string;
  onExitEditMode?: () => void;
  exitEditModeDisabled?: boolean;
  exitEditModeDisabledTooltip?: string;
  /** Switches back to the Canvas tab without changing edit mode. */
  onSelectCanvasView?: () => void;
  isRunInspectionMode?: boolean;
  onSelectConsole?: () => void;
  /** Switches the canvas surface to the Memory tab. Omitted on templates. */
  onSelectMemory?: () => void;
  /** Switches the canvas surface to the Files tab. Omitted on templates. */
  onSelectFiles?: () => void;
  /** Opens the console YAML modal when `headerMode` is `console`. */
  onConsoleOpenYaml?: () => void;
  /** DOM slot for Files mode actions owned by the files editor overlay. */
  filesHeaderActionsSlotId?: string;
  publishVersionLabel?: string;
  hasUnpublishedDraftChanges?: boolean;
  hasUnpublishedCanvasDraftChanges?: boolean;
  hasUnpublishedConsoleDraftChanges?: boolean;
  /** True when a non-spec repository file is staged; shows a dot on the Files tab. */
  hasFilesStagingChanges?: boolean;
  hasUncommittedCanvasDraftChanges?: boolean;
  hasUncommittedConsoleDraftChanges?: boolean;
  hasUncommittedFilesDraftChanges?: boolean;
  hasCommittedCanvasDraftChanges?: boolean;
  hasCommittedConsoleDraftChanges?: boolean;
  hasCommittedFilesDraftChanges?: boolean;
  editTabTone?: "uncommitted" | "ready" | "neutral";
  activeDraftBranchLabel?: string;
  activeDraftBranchShortSha?: string;
  isAutoLayoutOnUpdateEnabled?: boolean;
  onToggleAutoLayoutOnUpdate?: () => void;
  autoLayoutOnUpdateDisabled?: boolean;
  autoLayoutOnUpdateDisabledTooltip?: string;
  canvasStateMode?: "default" | "editing" | "previewing-previous-version";
  /** When true, enables inline rename and app settings in the project switcher. */
  showCanvasSettingsMenu?: boolean;
  showBottomStatusControls?: boolean;
  readOnly?: boolean;
  hideAddControls?: boolean;
  /** Hide the Agent / Versions left panel toggle (templates only). */
  hideCanvasToolSidebar?: boolean;
  canReadIntegrations?: boolean;
  canCreateIntegrations?: boolean;
  canUpdateIntegrations?: boolean;
  missingIntegrations?: MissingIntegration[];
  onConnectIntegration?: (integrationName: string) => void;

  getSidebarData?: (nodeId: string) => SidebarData | null;
  loadSidebarData?: (nodeId: string) => void;
  getTabData?: (nodeId: string, event: SidebarEvent) => TabData | undefined;
  getNodeEditData?: (nodeId: string) => NodeEditData | null;
  getAutocompleteExampleObj?: (nodeId: string) => Record<string, unknown> | null;
  onNodeConfigurationSave?: (
    nodeId: string,
    configuration: Record<string, unknown>,
    nodeName: string,
    integrationRef?: ComponentsIntegrationRef,
  ) => void | Promise<void>;
  onAnnotationUpdate?: (
    nodeId: string,
    updates: { text?: string; color?: string; width?: number; height?: number; x?: number; y?: number },
  ) => void;
  onAnnotationBlur?: () => void;
  getCustomField?: (nodeId: string, integration?: OrganizationsIntegration) => (() => React.ReactNode) | null;
  onNodeClick?: (nodeId: string) => void;
  integrations?: OrganizationsIntegration[];
  onEdgeCreate?: (sourceId: string, targetId: string, sourceHandle?: string | null) => void;
  onNodeDelete?: (nodeId: string) => void;
  onNodesDelete?: (nodeIds: string[]) => void;
  onDuplicateNodes?: (nodeIds: string[]) => void;
  onAutoLayoutNodes?: (nodeIds: string[]) => void;
  onEdgeDelete?: (edgeIds: string[]) => void;
  logRuns?: CanvasesCanvasRun[];
  runsNodes?: ComponentsNode[];
  runsComponentIconMap?: Record<string, string>;
  toolSidebarRunsContent?: React.ReactNode;
  toolSidebarVersionsContent?: React.ReactNode;
  onRunNodeSelect?: (nodeId: string) => void;
  onRunExecutionSelect?: (options: { runId: string; nodeId: string }) => void;
  onAcknowledgeErrors?: (executionIds: string[]) => void;
  onNodePositionChange?: (nodeId: string, position: { x: number; y: number }) => void;
  onNodesPositionChange?: (updates: Array<{ nodeId: string; position: { x: number; y: number } }>) => void;
  onCancelQueueItem?: (nodeId: string, queueItemId: string) => void;
  onCancelExecution?: (nodeId: string, executionId: string) => void;

  onDuplicate?: (nodeId: string) => void;
  onToggleView?: (nodeId: string, collapsed: boolean) => void;
  onReEmit?: (nodeId: string, eventOrExecutionId: string) => void;
  onRunItemOpen?: (nodeId: string | undefined, executionStatus: string, errorMessage?: string) => void;
  resolveRunIdForSidebarEvent?: (event: SidebarEvent) => string | null;
  fetchRunIdForSidebarEvent?: (event: SidebarEvent) => Promise<string | null>;
  onSelectRunFromSidebarEvent?: (runId: string, options?: { nodeId?: string }) => void;

  // Building blocks for adding new nodes
  buildingBlocks: BuildingBlockCategory[];
  /** When true, the canvas draft is active across Canvas, Console, and Memory tabs. */
  isEditing: boolean;
  /** True while an edit session is active (editing a draft or previewing a version from the versions sidebar). Drives the permanent versions sidebar and the Edit/Exit header affordance. */
  isEditSessionActive?: boolean;
  /** Active canvas version id (draft when editing); drives agent build mode. */
  activeCanvasVersionId: string;
  onNodeAdd?: (newNodeData: NewNodeData) => Promise<string>;
  onPlaceholderAdd?: (data: {
    position: { x: number; y: number };
    sourceNodeId?: string;
    sourceHandleId?: string | null;
  }) => Promise<string>;
  onPlaceholderConfigure?: (data: {
    placeholderId: string;
    buildingBlock: BuildingBlock;
    nodeName: string;
    configuration: Record<string, unknown>;
    integrationName?: string;
  }) => Promise<void>;

  // Refs to persist state across re-renders
  hasFitToViewRef?: React.MutableRefObject<boolean>;
  hasUserToggledSidebarRef?: React.MutableRefObject<boolean>;
  isSidebarOpenRef?: React.MutableRefObject<boolean | null>;
  viewportRef?: React.MutableRefObject<{ x: number; y: number; zoom: number } | undefined>;

  // Optional: control and observe component sidebar state
  onSidebarChange?: (isOpen: boolean, selectedNodeId: string | null) => void;
  onTriggerModalHostReady?: (openModal: (modal: TriggerActionModal) => void) => void;
  initialSidebar?: { isOpen?: boolean; nodeId?: string | null };
  initialFocusNodeId?: string | null;
  /** Bump this counter to fit all currently-rendered nodes into view (e.g., when run selection changes). */
  fitAllRequest?: number | null;
  /** When set with a bumped `fitAllRequest`, fit the viewport to this subset of nodes (e.g. run participants). */
  fitAllFocusNodeIds?: string[];
  /** In runs view, nodes not in this list are dimmed unless edge-hover highlighting is active. */
  runParticipantNodeIds?: string[];
  /** Shows a loading indicator over the canvas (not the sidebar) while run executions are being fetched. */
  runCanvasLoading?: boolean;
  /** Runs mode: selected run for the bottom node detail pane. */
  runNodeDetailRun?: CanvasesCanvasRun | null;
  runNodeDetailNodeId?: string | null;
  runNodeDetailCanvasId?: string;
  onRunNodeDetailClose?: () => void;
  onRunNodeDetailNavigate?: (nodeId: string) => void;
  runNodeDetailPaneHeight?: number;
  onRunNodeDetailPaneHeightChange?: (height: number) => void;

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

  // State registry function for determining execution states
  getExecutionState?: (
    nodeId: string,
    execution: CanvasesCanvasNodeExecution,
  ) => { map: EventStateMap; state: EventState };

  workflowNodes?: ComponentsNode[];
  components?: ActionsAction[];
  triggers?: TriggersTrigger[];
  logEntries?: LogEntry[];
  focusRequest?: FocusRequest | null;

  /** Returns to the current live canvas version from a published history preview. */
  onSeeCurrentVersion?: () => void;
}

export const CANVAS_SIDEBAR_STORAGE_KEY = "canvasSidebarOpen";
/**
 * @deprecated Width is now coordinated by the shared sidebar layout store.
 *  Kept exported for backward compatibility with existing test mocks.
 */
export const COMPONENT_SIDEBAR_WIDTH_STORAGE_KEY = "componentSidebarWidth";
export const CONSOLE_OPEN_STORAGE_KEY = "consoleOpen";
export const CONSOLE_HEIGHT_STORAGE_KEY = "consoleHeight";

function ComponentSidebarLoadingSkeleton({ layout = "sidebar" }: { layout?: "sidebar" | "bottom" }) {
  const sidebarWidth = useSidebarLayoutStore((state) => state.rightWidth);
  useSidebarMount("right", layout !== "bottom");

  if (layout === "bottom") {
    return (
      <div className="flex min-h-0 flex-1 flex-col items-center justify-center bg-white">
        <div className="flex flex-col items-center gap-3">
          <Loader2 className="h-8 w-8 animate-spin text-gray-500" />
          <p className="text-sm text-gray-500">Loading events...</p>
        </div>
      </div>
    );
  }

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

const EDGE_STYLE = {
  type: "custom",
  style: { stroke: "#C9D5E1", strokeWidth: 3 },
} as const;

const DEFAULT_CANVAS_ZOOM = 0.8;
const MIN_CANVAS_ZOOM = 0.1;
const SNAP_GRID_STEP_PX = 24;

type CanvasAnnotationUpdate = {
  text?: string;
  color?: string;
  width?: number;
  height?: number;
  x?: number;
  y?: number;
};

type CanvasModalRequest = TriggerActionModal;

type CanvasNodeRendererCallbacks = {
  handleNodeClick: (nodeId: string, event?: React.MouseEvent) => void;
  onAppendFromNode?: (nodeId: string, sourceHandleId?: string | null) => void | Promise<void>;
  onNodeDelete: React.MutableRefObject<CanvasPageProps["onNodeDelete"] | undefined>;
  onDuplicate: React.MutableRefObject<CanvasPageProps["onDuplicate"] | undefined>;
  onToggleView: React.MutableRefObject<((nodeId: string) => void) | undefined>;
  onShowNodeDiff: React.MutableRefObject<CanvasPageProps["onShowNodeDiff"] | undefined>;
  onAnnotationUpdate: React.MutableRefObject<CanvasPageProps["onAnnotationUpdate"] | undefined>;
  onAnnotationBlur: React.MutableRefObject<CanvasPageProps["onAnnotationBlur"] | undefined>;
  showHeader: boolean;
  hasMultiSelection: boolean;
  canvasMode: "live" | "edit";
};

type CanvasBlockNodeData = CanvasBlockData &
  Record<string, unknown> & {
    _callbacksRef?: React.MutableRefObject<CanvasNodeRendererCallbacks>;
    nodeName?: string;
  };

type CanvasConnectionState = {
  nodeId: string;
  handleId: string | null;
  handleType: "source" | "target" | null;
};

type EnrichedCanvasNodeCacheEntry = {
  sourceNode: ReactFlowNode;
  sourceData: ReactFlowNode["data"];
  node: ReactFlowNode;
  data: CanvasBlockNodeData;
  hoveredEdge: CanvasEdge | null;
  connectingFrom: CanvasConnectionState | null;
  edges: CanvasEdge[];
  isHighlighted: boolean;
  hasHighlightedNodes: boolean;
  runParticipantKey: string;
};

function canReuseEnrichedNodeData({
  cachedNode,
  node,
  hoveredEdge,
  connectingFrom,
  edges,
  isHighlighted,
  hasHighlightedNodes,
  runParticipantKey,
}: {
  cachedNode: EnrichedCanvasNodeCacheEntry | undefined;
  node: ReactFlowNode;
  hoveredEdge: CanvasEdge | null;
  connectingFrom: CanvasConnectionState | null;
  edges: CanvasEdge[];
  isHighlighted: boolean;
  hasHighlightedNodes: boolean;
  runParticipantKey: string;
}) {
  return (
    cachedNode &&
    cachedNode.sourceData === node.data &&
    cachedNode.hoveredEdge === hoveredEdge &&
    cachedNode.connectingFrom === connectingFrom &&
    cachedNode.edges === edges &&
    cachedNode.isHighlighted === isHighlighted &&
    cachedNode.hasHighlightedNodes === hasHighlightedNodes &&
    cachedNode.runParticipantKey === runParticipantKey
  );
}

type CollapsibleNodeData = {
  type?: unknown;
  component?: { collapsed?: boolean };
  trigger?: { collapsed?: boolean };
  composite?: { collapsed?: boolean };
};

function createNodeRenderFallbackData(data: BlockData): BlockData {
  return {
    ...data,
    label: typeof data.label === "string" && data.label.trim() ? data.label : "Component",
    outputChannels: Array.isArray(data.outputChannels) ? data.outputChannels : undefined,
    trigger: undefined,
    component: undefined,
    composite: undefined,
    annotation: undefined,
    renderFallback: {
      source: "node-render",
      message: CANVAS_NODE_FALLBACK_MESSAGE,
    },
  };
}

function isCanvasNodeCollapsed(node: ReactFlowNode | undefined): boolean {
  const data = node?.data as CollapsibleNodeData | undefined;
  if (!data) {
    return false;
  }

  const nodeType = data.type;
  if (nodeType !== "component" && nodeType !== "trigger" && nodeType !== "composite") {
    return false;
  }

  return Boolean(data[nodeType]?.collapsed);
}

function getNonEmptyString(value: unknown): string | undefined {
  return typeof value === "string" && value.trim() ? value : undefined;
}

const NODE_ERROR_LABEL_GETTERS: Record<BlockData["type"], (data: BlockData) => string | undefined> = {
  trigger: (data) => getNonEmptyString(data.trigger?.title),
  component: (data) => getNonEmptyString(data.component?.title),
  composite: (data) => getNonEmptyString(data.composite?.title),
  annotation: (data) => getNonEmptyString(data.annotation?.title),
};

function getNodeErrorDisplayName(data: BlockData): string {
  const labelGetter = NODE_ERROR_LABEL_GETTERS[data.type as BlockData["type"]];

  return labelGetter?.(data) || getNonEmptyString(data.label) || "unknown";
}

function areOutputChannelsEqual(previous: string[] | undefined, next: string[] | undefined) {
  if (previous === next) {
    return true;
  }

  if (!previous || !next || previous.length !== next.length) {
    return false;
  }

  return previous.every((channel, index) => channel === next[index]);
}

function didNodeErrorBoundaryDataChange(previous: BlockData, next: BlockData) {
  return (
    previous.type !== next.type ||
    previous.label !== next.label ||
    previous.trigger !== next.trigger ||
    previous.component !== next.component ||
    previous.composite !== next.composite ||
    previous.annotation !== next.annotation ||
    previous.renderFallback?.source !== next.renderFallback?.source ||
    previous.renderFallback?.message !== next.renderFallback?.message ||
    !areOutputChannelsEqual(previous.outputChannels, next.outputChannels)
  );
}

function getNodeAction<TArgs extends unknown[]>(
  actionRef: React.MutableRefObject<((...args: TArgs) => void) | undefined> | undefined,
  ...args: TArgs
) {
  return actionRef?.current ? () => actionRef.current?.(...args) : undefined;
}

function getVoidAction(actionRef: React.MutableRefObject<(() => void) | undefined> | undefined) {
  return actionRef?.current ? () => actionRef.current?.() : undefined;
}

function getAnnotationUpdateAction(callbacks?: CanvasNodeRendererCallbacks) {
  return callbacks?.onAnnotationUpdate.current
    ? (annotationNodeId: string, updates: CanvasAnnotationUpdate) =>
        callbacks.onAnnotationUpdate.current?.(annotationNodeId, updates)
    : undefined;
}

function buildInteractiveNodeBlockProps(
  callbacks: CanvasNodeRendererCallbacks | undefined,
  nodeId: string,
): Omit<BlockProps, "data" | "nodeId" | "selected"> {
  if (!callbacks) {
    return {};
  }

  return {
    showHeader: callbacks.showHeader && !callbacks.hasMultiSelection,
    canvasMode: callbacks.canvasMode,
    onAppendFromNode: callbacks.onAppendFromNode,
    onClick: (event) => callbacks.handleNodeClick(nodeId, event),
    onDelete: getNodeAction(callbacks.onNodeDelete, nodeId),
    onDuplicate: getNodeAction(callbacks.onDuplicate, nodeId),
    onToggleView: getNodeAction(callbacks.onToggleView, nodeId),
    onShowDiff: getNodeAction(callbacks.onShowNodeDiff, nodeId),
    onAnnotationUpdate: getAnnotationUpdateAction(callbacks),
    onAnnotationBlur: getVoidAction(callbacks.onAnnotationBlur),
  };
}

function buildDefaultNodeBlockProps(args: {
  nodeId: string;
  selected?: boolean;
  callbacks?: CanvasNodeRendererCallbacks;
}): Omit<BlockProps, "data"> {
  const { nodeId, selected, callbacks } = args;

  return {
    nodeId,
    selected,
    ...buildInteractiveNodeBlockProps(callbacks, nodeId),
  };
}

type CanvasNodeErrorBoundaryProps = {
  nodeId: string;
  nodeData: BlockData;
  fallback: ReactNode;
  children: ReactNode;
};

type CanvasNodeErrorBoundaryState = {
  hasError: boolean;
};

export class CanvasNodeErrorBoundary extends Component<CanvasNodeErrorBoundaryProps, CanvasNodeErrorBoundaryState> {
  state: CanvasNodeErrorBoundaryState = {
    hasError: false,
  };

  static getDerivedStateFromError(): CanvasNodeErrorBoundaryState {
    return { hasError: true };
  }

  componentDidCatch(error: Error, errorInfo: ErrorInfo) {
    const nodeType = this.props.nodeData.type || "unknown";
    const nodeLabel = getNodeErrorDisplayName(this.props.nodeData);

    console.error(`[CanvasPage] Node "${this.props.nodeId}" failed to render:`, error);

    Sentry.withScope((scope) => {
      scope.setTag("canvas.node_id", this.props.nodeId);
      scope.setTag("canvas.node_type", nodeType);
      scope.setExtra("nodeLabel", nodeLabel);
      scope.setExtra("componentStack", errorInfo.componentStack);
      Sentry.captureException(error);
    });
  }

  componentDidUpdate(prevProps: CanvasNodeErrorBoundaryProps) {
    if (
      this.state.hasError &&
      (prevProps.nodeId !== this.props.nodeId ||
        didNodeErrorBoundaryDataChange(prevProps.nodeData, this.props.nodeData))
    ) {
      this.setState({ hasError: false });
    }
  }

  render() {
    if (this.state.hasError) {
      return <div data-testid="canvas-node-fallback">{this.props.fallback}</div>;
    }

    return this.props.children;
  }
}

/*
 * nodeTypes must be defined outside of the component to prevent
 * react-flow from remounting the node types on every render.
 */
type DefaultNodeRendererProps = {
  data: CanvasBlockNodeData;
  id: string;
  selected?: boolean;
};

function areDefaultNodeRendererPropsEqual(
  previousProps: DefaultNodeRendererProps,
  nextProps: DefaultNodeRendererProps,
): boolean {
  return (
    previousProps.id === nextProps.id &&
    previousProps.selected === nextProps.selected &&
    previousProps.data === nextProps.data
  );
}

const DefaultNodeRenderer = memo(function DefaultNodeRenderer(nodeProps: DefaultNodeRendererProps) {
  const { _callbacksRef, ...blockData } = nodeProps.data;
  const callbacks = _callbacksRef?.current;
  const blockProps = buildDefaultNodeBlockProps({
    nodeId: nodeProps.id,
    selected: nodeProps.selected,
    callbacks,
  });
  const fallback = <Block {...blockProps} data={createNodeRenderFallbackData(blockData)} />;

  return (
    <CanvasNodeErrorBoundary nodeId={nodeProps.id} nodeData={blockData} fallback={fallback}>
      <Block {...blockProps} data={blockData} />
    </CanvasNodeErrorBoundary>
  );
}, areDefaultNodeRendererPropsEqual);

const nodeTypes = {
  default: DefaultNodeRenderer,
};

function CanvasPage(props: CanvasPageProps) {
  const state = useCanvasState(props);
  const readOnly = props.readOnly ?? false;
  const workflowHeaderMode = normalizeCanvasHeaderMode(props.headerMode);
  const [currentTab, setCurrentTab] = useState<"latest" | "settings" | "docs">(() =>
    props.canvasStateMode === "editing" ? "settings" : "latest",
  );
  const [liveNodeDetailPaneHeight, setLiveNodeDetailPaneHeight] = useState(320);
  const [templateNodeId, setTemplateNodeId] = useState<string | null>(null);
  const canvasWrapperRef = useRef<HTMLDivElement | null>(null);
  const localHasFitToViewRef = useRef(false);
  const localHasUserToggledSidebarRef = useRef(false);
  const localIsSidebarOpenRef = useRef<boolean | null>(null);

  // Use refs from props if provided, otherwise create local ones
  const hasFitToViewRef = props.hasFitToViewRef ?? localHasFitToViewRef;
  const hasUserToggledSidebarRef = props.hasUserToggledSidebarRef ?? localHasUserToggledSidebarRef;
  const isSidebarOpenRef = props.isSidebarOpenRef ?? localIsSidebarOpenRef;

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

  const toolSidebarState = useCanvasToolSidebarState({
    isEditing: props.isEditing,
    hideCanvasToolSidebar: props.hideCanvasToolSidebar ?? false,
    readOnly,
    canvasId: props.canvasId,
    organizationId: props.organizationId,
  });
  const runsSidebarBaseState = useCanvasRunsSidebarState();
  const showRunsSidebar = isCanvasWorkflowTab(props.headerMode) && props.toolSidebarRunsContent != null;
  const runsSidebarState = {
    ...runsSidebarBaseState,
    showRunsSidebarToggle: showRunsSidebar,
  };
  const isRunsSidebarOpen = showRunsSidebar && runsSidebarBaseState.isRunsSidebarOpen;

  const versionsSidebarBaseState = useCanvasVersionsSidebarState();
  // Versions content is only produced during an edit session; within that session
  // the sidebar can be shown/hidden with the header toggle.
  const versionsContentAvailable = props.toolSidebarVersionsContent != null;
  const showVersionsSidebarToggle = versionsContentAvailable;
  const versionsSidebarState = {
    ...versionsSidebarBaseState,
    showVersionsSidebarToggle,
  };
  const isVersionsSidebarOpen = versionsContentAvailable && versionsSidebarBaseState.isVersionsSidebarOpen;

  // The collapse state is intentionally not persisted: the versions sidebar always
  // starts expanded whenever the user (re)enters an edit session.
  const isEditSessionActive = props.isEditSessionActive;
  const { openVersionsSidebar } = versionsSidebarBaseState;
  useEffect(() => {
    if (isEditSessionActive) {
      openVersionsSidebar();
    }
  }, [isEditSessionActive, openVersionsSidebar]);

  const initialCanvasZoom = props.nodes.length === 0 ? DEFAULT_CANVAS_ZOOM : 1;
  const [canvasZoom, setCanvasZoom] = useState(initialCanvasZoom);
  const [canvasModalRequest, setCanvasModalRequest] = useState<CanvasModalRequest | null>(null);
  const openCanvasModal = useCallback((modal: CanvasModalRequest) => {
    setCanvasModalRequest(modal);
  }, []);
  const closeCanvasModal = useCallback(() => {
    setCanvasModalRequest(null);
  }, []);
  useEffect(() => {
    props.onTriggerModalHostReady?.(openCanvasModal);
  }, [props, openCanvasModal]);
  useEffect(() => {
    if (!props.focusRequest?.tab) {
      return;
    }

    setCurrentTab(props.focusRequest.tab);
  }, [props.focusRequest?.requestId, props.focusRequest?.tab]);

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
      if (templateNodeId === nodeId) {
        setTemplateNodeId(null);
        setIsBuildingBlocksSidebarOpen(false);
        isSidebarOpenRef.current = false;
      }

      if (state.componentSidebar.selectedNodeId === nodeId) {
        state.componentSidebar.close();
      }

      props.onNodeDelete?.(nodeId);
    },
    [props, templateNodeId, state.componentSidebar, setTemplateNodeId, isSidebarOpenRef],
  );

  const handleNodesDelete = useCallback(
    (nodeIds: string[]) => {
      nodeIds.forEach((nodeId) => {
        if (templateNodeId === nodeId) {
          setTemplateNodeId(null);
          setIsBuildingBlocksSidebarOpen(false);
          isSidebarOpenRef.current = false;
        }

        if (state.componentSidebar.selectedNodeId === nodeId) {
          state.componentSidebar.close();
        }
      });

      props.onNodesDelete?.(nodeIds);
    },
    [props, templateNodeId, state.componentSidebar, setTemplateNodeId, isSidebarOpenRef],
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
      const isPlaceholder = workflowNode?.name === "New Component" && !workflowNode.component;

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

    const position = findFreePositionInViewport({
      viewport: props.viewportRef?.current ?? { x: 0, y: 0, zoom: DEFAULT_CANVAS_ZOOM },
      canvasRect: canvasWrapperRef.current?.getBoundingClientRect() ?? null,
      nodes: state.nodes || [],
      nodeSize: { width: 320, height: 160 },
      fallbackCanvasSize: { width: window.innerWidth, height: window.innerHeight },
    });

    const annotationBlock: BuildingBlock = {
      name: "annotation",
      label: "Annotation",
      type: "component",
    };

    await props.onNodeAdd({
      buildingBlock: annotationBlock,
      nodeName: "Note",
      configuration: {},
      position,
    });
  }, [props, state.nodes, readOnly]);

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
        await props.onNodeAdd({
          buildingBlock: block,
          nodeName: block.name || "",
          configuration: defaultConfiguration,
          position,
          integrationName: block.integrationName,
        });
      }
    },
    [props, readOnly],
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

  /**
   * Keyboard equivalent of dropping a block onto the canvas via drag-and-drop.
   * When a placeholder / pending-connection is active we route through the
   * existing click path so the block fills the placeholder instead of spawning
   * a free-floating node. Otherwise we place the new node at the viewport
   * center using the same algorithm as "Add Note".
   */
  const handleBuildingBlockSelect = useCallback(
    (block: BuildingBlock) => {
      if (readOnly) return;

      if (templateNodeId) {
        void handleBuildingBlockClick(block);
        return;
      }

      const position = findFreePositionInViewport({
        viewport: props.viewportRef?.current ?? { x: 0, y: 0, zoom: DEFAULT_CANVAS_ZOOM },
        canvasRect: canvasWrapperRef.current?.getBoundingClientRect() ?? null,
        nodes: state.nodes || [],
        nodeSize: { width: 420, height: 200 },
        fallbackCanvasSize: { width: window.innerWidth, height: window.innerHeight },
      });

      void handleBuildingBlockDrop(block, position);
    },
    [readOnly, templateNodeId, handleBuildingBlockClick, handleBuildingBlockDrop, props.viewportRef, state.nodes],
  );

  const handleBuildingBlocksShortcutOpen = useCallback(() => {
    if (readOnly) {
      return;
    }

    handleSidebarToggle(true);
    state.componentSidebar.close();
  }, [readOnly, state.componentSidebar, handleSidebarToggle]);

  useBuildingBlocksShortcut({
    disabled:
      readOnly ||
      Boolean(props.hideAddControls) ||
      !props.isEditing ||
      blocksBuildingBlocksShortcut(workflowHeaderMode) ||
      state.componentSidebar.isOpen,
    isSidebarOpen: isBuildingBlocksSidebarOpen,
    onOpen: handleBuildingBlocksShortcutOpen,
  });

  const handleSaveConfiguration = useCallback(
    (configuration: Record<string, unknown>, nodeName: string, integrationRef?: ComponentsIntegrationRef) => {
      if (!editingNodeData?.nodeId || !props.onNodeConfigurationSave) {
        return;
      }
      const result = props.onNodeConfigurationSave(editingNodeData.nodeId, configuration, nodeName, integrationRef);
      if (props.configurationSaveMode !== "auto") {
        state.componentSidebar.close();
      }
      return result;
    },
    [editingNodeData?.nodeId, props, state.componentSidebar],
  );

  const canvasNodesForToggle = state.nodes;
  const toggleNodeCollapse = state.toggleNodeCollapse;
  const onToggleView = props.onToggleView;
  const handleToggleView = useCallback(
    (nodeId: string) => {
      const node = canvasNodesForToggle.find((candidate) => candidate.id === nodeId);
      const collapsed = !isCanvasNodeCollapsed(node);
      toggleNodeCollapse(nodeId);
      onToggleView?.(nodeId, collapsed);
    },
    [canvasNodesForToggle, toggleNodeCollapse, onToggleView],
  );

  const onCancelQueueItemHandler = props.onCancelQueueItem;
  const onCancelExecutionHandler = props.onCancelExecution;

  const handleCancelQueueItem = useCallback(
    (queueId: string) => {
      const selectedNodeId = state.componentSidebar.selectedNodeId;
      if (selectedNodeId && onCancelQueueItemHandler) {
        onCancelQueueItemHandler(selectedNodeId, queueId);
      }
    },
    [onCancelQueueItemHandler, state.componentSidebar.selectedNodeId],
  );

  const handleCancelExecution = useCallback(
    (executionId: string) => {
      const selectedNodeId = state.componentSidebar.selectedNodeId;
      if (selectedNodeId && onCancelExecutionHandler) {
        onCancelExecutionHandler(selectedNodeId, executionId);
      }
    },
    [onCancelExecutionHandler, state.componentSidebar.selectedNodeId],
  );

  const handleSidebarClose = useCallback(() => {
    const selectedNodeId = state.componentSidebar.selectedNodeId;
    // Check if the currently open node is a pending connection
    const currentNode = state.nodes.find((n) => n.id === selectedNodeId);
    const isPendingConnection = currentNode?.data?.isPendingConnection;

    state.componentSidebar.close();
    // Reset to latest tab when sidebar closes
    setCurrentTab(props.canvasStateMode === "editing" ? "settings" : "latest");

    // Only remove the node if it's a pending connection node (not yet configured)
    if (isPendingConnection && selectedNodeId) {
      const nodeIdToRemove = selectedNodeId;
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
  }, [props.canvasStateMode, state, templateNodeId]);

  const previousHeaderModeForSidebarRef = useRef<CanvasPageProps["headerMode"]>(props.headerMode);

  useEffect(() => {
    const previousMode = previousHeaderModeForSidebarRef.current;
    const currentMode = props.headerMode;
    previousHeaderModeForSidebarRef.current = currentMode;

    if (isComponentSidebarVisibleMode(previousMode) && !isComponentSidebarVisibleMode(currentMode)) {
      if (state.componentSidebar.isOpen) {
        handleSidebarClose();
      }
    }
  }, [props.headerMode, state.componentSidebar.isOpen, handleSidebarClose]);

  useEffect(() => {
    if (props.isRunInspectionMode && props.isEditing && state.componentSidebar.isOpen) {
      handleSidebarClose();
    }
  }, [props.isEditing, props.isRunInspectionMode, state.componentSidebar.isOpen, handleSidebarClose]);

  const canvasStateMode = props.canvasStateMode || "default";
  const showPreviewFloatingBar = canvasStateMode === "previewing-previous-version" && !!props.onSeeCurrentVersion;

  const liveBottomInspectorOpen = !props.isRunInspectionMode && !props.isEditing && state.componentSidebar.isOpen;

  const runNodeDetailPaneOpen =
    props.isRunInspectionMode &&
    !!props.runNodeDetailRun &&
    !!props.runNodeDetailNodeId &&
    !!props.runNodeDetailCanvasId;

  const bottomDetailPaneOpen = runNodeDetailPaneOpen || liveBottomInspectorOpen;

  const renderInspectorSidebar = useCallback(
    (layout: "sidebar" | "bottom") => (
      <Sidebar
        layout={layout}
        state={state}
        getSidebarData={props.getSidebarData}
        loadSidebarData={props.loadSidebarData}
        getTabData={props.getTabData}
        getAutocompleteExampleObj={props.getAutocompleteExampleObj}
        onCancelQueueItem={handleCancelQueueItem}
        onCancelExecution={handleCancelExecution}
        getAllHistoryEvents={props.getAllHistoryEvents}
        onLoadMoreHistory={props.onLoadMoreHistory}
        getHasMoreHistory={props.getHasMoreHistory}
        getLoadingMoreHistory={props.getLoadingMoreHistory}
        onLoadMoreQueue={props.onLoadMoreQueue}
        getAllQueueEvents={props.getAllQueueEvents}
        getHasMoreQueue={props.getHasMoreQueue}
        getLoadingMoreQueue={props.getLoadingMoreQueue}
        onReEmit={props.onReEmit}
        onRunItemOpen={props.onRunItemOpen}
        resolveRunId={props.resolveRunIdForSidebarEvent}
        fetchRunId={props.fetchRunIdForSidebarEvent}
        onSelectRun={props.onSelectRunFromSidebarEvent}
        getExecutionState={props.getExecutionState}
        onSidebarClose={handleSidebarClose}
        editingNodeData={editingNodeData}
        onSaveConfiguration={handleSaveConfiguration}
        configurationSaveMode={props.configurationSaveMode}
        currentTab={currentTab}
        onTabChange={setCurrentTab}
        canvasMode={props.isEditing ? "edit" : "live"}
        organizationId={props.organizationId}
        getCustomField={props.getCustomField}
        integrations={props.integrations}
        workflowNodes={props.workflowNodes}
        components={props.components}
        triggers={props.triggers}
        readOnly={readOnly}
        canReadIntegrations={props.canReadIntegrations}
        canCreateIntegrations={props.canCreateIntegrations}
        canUpdateIntegrations={props.canUpdateIntegrations}
      />
    ),
    [
      currentTab,
      editingNodeData,
      handleCancelExecution,
      handleCancelQueueItem,
      handleSaveConfiguration,
      handleSidebarClose,
      props.canCreateIntegrations,
      props.canReadIntegrations,
      props.canUpdateIntegrations,
      props.configurationSaveMode,
      props.fetchRunIdForSidebarEvent,
      props.getAllHistoryEvents,
      props.getAllQueueEvents,
      props.getAutocompleteExampleObj,
      props.getCustomField,
      props.getExecutionState,
      props.getHasMoreHistory,
      props.getHasMoreQueue,
      props.getLoadingMoreHistory,
      props.getLoadingMoreQueue,
      props.getSidebarData,
      props.getTabData,
      props.integrations,
      props.components,
      props.triggers,
      props.isEditing,
      props.loadSidebarData,
      props.onLoadMoreHistory,
      props.onLoadMoreQueue,
      props.onReEmit,
      props.onRunItemOpen,
      props.onSelectRunFromSidebarEvent,
      props.organizationId,
      props.resolveRunIdForSidebarEvent,
      props.workflowNodes,
      readOnly,
      state,
    ],
  );

  return (
    <div
      ref={canvasWrapperRef}
      className={cn(
        "h-full w-full overflow-hidden sp-canvas relative flex flex-col",
        (props.headerMode === "version-live" ||
          props.headerMode === "console" ||
          props.headerMode === "memory" ||
          props.headerMode === "files") &&
          "sp-canvas-live",
        props.isRunInspectionMode && "sp-canvas-live",
        props.isEditing && "sp-canvas-editing",
      )}
    >
      {/* Header at the top spanning full width */}
      <div className="relative z-40">
        <CanvasContentHeader
          canvasName={props.title ?? ""}
          organizationId={props.organizationId}
          onPublishVersion={props.onPublishVersion}
          onDiscardVersion={props.onDiscardVersion}
          onShowDiff={props.onShowDiff}
          onShowConsoleDiff={props.onShowConsoleDiff}
          visualDiffEnabled={props.visualDiffEnabled}
          draftVisualDiff={props.draftVisualDiff}
          draftConsoleDiff={props.draftConsoleDiff}
          onToggleVisualDiff={props.onToggleVisualDiff}
          publishVersionDisabled={props.publishVersionDisabled}
          publishVersionDisabledTooltip={props.publishVersionDisabledTooltip}
          discardVersionDisabled={props.discardVersionDisabled}
          discardVersionDisabledTooltip={props.discardVersionDisabledTooltip}
          hasStagingChanges={props.hasStagingChanges}
          onCommitStaging={props.onCommitStaging}
          commitStagingPending={props.commitStagingPending}
          resetStagingPending={props.resetStagingPending}
          onResetStaging={props.onResetStaging}
          headerMode={props.headerMode}
          isEditing={props.isEditing}
          isEditSessionActive={props.isEditSessionActive}
          onSelectCanvasView={props.onSelectCanvasView}
          onEnterEditMode={props.onEnterEditMode}
          enterEditModeDisabled={props.enterEditModeDisabled}
          enterEditModeDisabledTooltip={props.enterEditModeDisabledTooltip}
          onExitEditMode={props.onExitEditMode}
          exitEditModeDisabled={props.exitEditModeDisabled}
          exitEditModeDisabledTooltip={props.exitEditModeDisabledTooltip}
          onSelectConsole={props.onSelectConsole}
          onSelectMemory={props.onSelectMemory}
          onSelectFiles={props.onSelectFiles}
          filesHeaderActionsSlotId={props.filesHeaderActionsSlotId}
          publishVersionLabel={props.publishVersionLabel}
          hasUnpublishedDraftChanges={props.hasUnpublishedDraftChanges}
          hasUnpublishedCanvasDraftChanges={props.hasUnpublishedCanvasDraftChanges}
          hasUnpublishedConsoleDraftChanges={props.hasUnpublishedConsoleDraftChanges}
          hasFilesStagingChanges={props.hasFilesStagingChanges}
          hasUncommittedCanvasDraftChanges={props.hasUncommittedCanvasDraftChanges}
          hasUncommittedConsoleDraftChanges={props.hasUncommittedConsoleDraftChanges}
          hasUncommittedFilesDraftChanges={props.hasUncommittedFilesDraftChanges}
          hasCommittedCanvasDraftChanges={props.hasCommittedCanvasDraftChanges}
          hasCommittedConsoleDraftChanges={props.hasCommittedConsoleDraftChanges}
          hasCommittedFilesDraftChanges={props.hasCommittedFilesDraftChanges}
          editTabTone={props.editTabTone}
          activeDraftBranchLabel={props.activeDraftBranchLabel}
          activeDraftBranchShortSha={props.activeDraftBranchShortSha}
          showCanvasSettingsMenu={props.showCanvasSettingsMenu}
          toolSidebarState={toolSidebarState}
          runsSidebarState={runsSidebarState}
          versionsSidebarState={versionsSidebarState}
        />
        {props.headerBanner ? <div className="border-b border-black/20">{props.headerBanner}</div> : null}
      </div>

      {/* Main content area with sidebar and canvas */}
      <div className="relative flex min-h-0 flex-1 overflow-hidden">
        <CanvasToolSidebar toolSidebarState={toolSidebarState} />

        <CanvasRunsSidebar isOpen={isRunsSidebarOpen}>{props.toolSidebarRunsContent ?? null}</CanvasRunsSidebar>

        <CanvasVersionsSidebar isOpen={isVersionsSidebarOpen}>
          {props.toolSidebarVersionsContent ?? null}
        </CanvasVersionsSidebar>

        {isPanelHeaderMode(workflowHeaderMode) ? null : props.isEditing ? (
          props.headerMode === "console" ? null : (
            <RightSideControls
              mode="edit"
              canvasEditControls
              onSidebarOpen={handleBuildingBlocksShortcutOpen}
              onAddNote={handleAddNote}
            />
          )
        ) : (
          <RightSideControls
            mode={readOnly ? "live" : "edit"}
            onSidebarOpen={handleBuildingBlocksShortcutOpen}
            onAddNote={handleAddNote}
          />
        )}
        {props.hideAddControls || !isBuildingBlocksSidebarOpen ? null : (
          <BuildingBlocksSidebar
            isOpen={isBuildingBlocksSidebarOpen && !!props.isEditing && allowsBuildingBlocksSidebar(workflowHeaderMode)}
            onToggle={handleSidebarToggle}
            blocks={props.buildingBlocks || []}
            integrations={props.integrations}
            canvasZoom={canvasZoom}
            disabled={readOnly}
            disabledMessage="You don't have permission to edit this canvas."
            onBlockClick={handleBuildingBlockSelect}
            onEnterSubmit={handleBuildingBlockSelect}
          />
        )}

        <div className="flex min-h-0 flex-1 flex-col overflow-hidden">
          <div className="relative min-h-0 flex-1">
            {props.runCanvasLoading && props.isRunInspectionMode ? (
              <div className="absolute inset-0 z-30 pointer-events-none flex items-center justify-center">
                <div className="rounded-lg bg-white/80 p-3 shadow-sm backdrop-blur-sm">
                  <Loader2 className="h-5 w-5 animate-spin text-slate-500" />
                </div>
              </div>
            ) : null}
            {showPreviewFloatingBar ? (
              <div className="pointer-events-none absolute inset-x-0 top-0 z-[19] flex justify-center pt-3">
                <div className="pointer-events-auto flex max-w-[min(100vw-2rem,42rem)] items-center gap-2 rounded-full bg-gray-500 pl-3 pr-1.5 py-1.5">
                  <span className="flex min-w-0 max-w-full shrink-0 truncate items-center gap-1 text-[13px] font-medium text-white">
                    Previewing previous version
                  </span>
                  <Button
                    type="button"
                    variant="outline"
                    size="xs"
                    className="shrink-0 border-0 shadow-none"
                    onClick={() => {
                      props.onSeeCurrentVersion?.();
                    }}
                  >
                    See Current Version
                  </Button>
                </div>
              </div>
            ) : null}
            {props.headerMode === "files" ? (
              <div className="absolute inset-0 bg-slate-50" data-testid="canvas-files-backdrop" aria-hidden />
            ) : (
              <ReactFlowProvider key="canvas-flow-provider" data-testid="canvas-drop-area">
                <CanvasContent
                  state={state}
                  onNodeDelete={handleNodeDelete}
                  onNodesDelete={handleNodesDelete}
                  onDuplicateNodes={props.onDuplicateNodes}
                  onAutoLayoutNodes={props.onAutoLayoutNodes}
                  onEdgeCreate={props.onEdgeCreate}
                  onToggleView={handleToggleView}
                  onShowNodeDiff={props.onShowNodeDiff}
                  onDuplicate={props.onDuplicate}
                  onAnnotationUpdate={props.onAnnotationUpdate}
                  onAnnotationBlur={props.onAnnotationBlur}
                  onBuildingBlockDrop={handleBuildingBlockDrop}
                  onBuildingBlocksSidebarToggle={handleSidebarToggle}
                  onConnectionDropInEmptySpace={handleConnectionDropInEmptySpace}
                  onPendingConnectionNodeClick={handlePendingConnectionNodeClick}
                  onNodeClick={props.onNodeClick}
                  onZoomChange={setCanvasZoom}
                  hasFitToViewRef={hasFitToViewRef}
                  viewportRefProp={props.viewportRef}
                  workflowNodes={props.workflowNodes}
                  setCurrentTab={setCurrentTab}
                  showBottomStatusControls={props.showBottomStatusControls}
                  isRunInspectionMode={props.isRunInspectionMode}
                  isEditing={props.isEditing}
                  isAutoLayoutOnUpdateEnabled={props.isAutoLayoutOnUpdateEnabled}
                  onToggleAutoLayoutOnUpdate={props.onToggleAutoLayoutOnUpdate}
                  autoLayoutOnUpdateDisabled={props.autoLayoutOnUpdateDisabled}
                  autoLayoutOnUpdateDisabledTooltip={props.autoLayoutOnUpdateDisabledTooltip}
                  readOnly={props.readOnly}
                  logEntries={props.logEntries}
                  focusRequest={props.focusRequest}
                  initialFocusNodeId={props.initialFocusNodeId}
                  fitAllRequest={props.fitAllRequest}
                  fitAllFocusNodeIds={props.fitAllFocusNodeIds}
                  runParticipantNodeIds={props.runParticipantNodeIds}
                  runSelectedNodeId={props.isRunInspectionMode ? props.runNodeDetailNodeId : null}
                  runNodeDetailPaneOpen={bottomDetailPaneOpen}
                  logRuns={props.logRuns}
                  runsNodes={props.runsNodes}
                  runsComponentIconMap={props.runsComponentIconMap}
                  onRunNodeSelect={props.onRunNodeSelect}
                  onRunExecutionSelect={props.onRunExecutionSelect}
                  onAcknowledgeErrors={props.onAcknowledgeErrors}
                  missingIntegrations={props.missingIntegrations}
                  onConnectIntegration={props.onConnectIntegration}
                  canCreateIntegrations={props.canCreateIntegrations}
                />
              </ReactFlowProvider>
            )}
            {isComponentSidebarVisibleMode(props.headerMode) && !props.isRunInspectionMode && props.isEditing
              ? renderInspectorSidebar("sidebar")
              : null}
          </div>
          {runNodeDetailPaneOpen ? (
            <RunNodeDetailPane
              canvasId={props.runNodeDetailCanvasId!}
              run={props.runNodeDetailRun!}
              nodeId={props.runNodeDetailNodeId!}
              workflowNodes={props.workflowNodes}
              componentIconMap={props.runsComponentIconMap}
              onClose={() => props.onRunNodeDetailClose?.()}
              onNavigateNode={props.onRunNodeDetailNavigate}
              height={props.runNodeDetailPaneHeight}
              onHeightChange={props.onRunNodeDetailPaneHeightChange}
            />
          ) : null}
          {liveBottomInspectorOpen ? (
            <ResizableBottomPane
              height={liveNodeDetailPaneHeight}
              onHeightChange={setLiveNodeDetailPaneHeight}
              testId="live-node-detail-pane"
              resizeHandleTestId="live-node-detail-pane-resize-handle"
            >
              {state.componentSidebar.selectedNodeId ? (
                renderInspectorSidebar("bottom")
              ) : (
                <LiveBottomInspectorEmptyState onClose={handleSidebarClose} />
              )}
            </ResizableBottomPane>
          ) : null}
        </div>
      </div>

      {/* Edit existing node modal - now handled by settings sidebar */}

      <Dialog open={!!canvasModalRequest} onOpenChange={(isOpen) => !isOpen && closeCanvasModal()}>
        <DialogContent className="max-w-3xl max-h-[80vh]">
          {canvasModalRequest?.title ? <DialogTitle>{canvasModalRequest.title}</DialogTitle> : null}
          {canvasModalRequest?.description ? (
            <DialogDescription>{canvasModalRequest.description}</DialogDescription>
          ) : null}
          {canvasModalRequest ? canvasModalRequest.content({ close: closeCanvasModal }) : null}
        </DialogContent>
      </Dialog>
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
  onCancelExecution,
  onReEmit,
  onRunItemOpen,
  resolveRunId,
  fetchRunId,
  onSelectRun,
  getAllHistoryEvents,
  onLoadMoreHistory,
  getHasMoreHistory,
  getLoadingMoreHistory,
  onLoadMoreQueue,
  getAllQueueEvents,
  getHasMoreQueue,
  getLoadingMoreQueue,
  getExecutionState,
  onSidebarClose,
  editingNodeData,
  onSaveConfiguration,
  configurationSaveMode = "manual",
  currentTab,
  onTabChange,
  canvasMode,
  organizationId,
  getCustomField,
  integrations,
  workflowNodes,
  components,
  triggers,
  readOnly,
  canReadIntegrations,
  canCreateIntegrations,
  canUpdateIntegrations,
  layout = "sidebar",
}: {
  layout?: "sidebar" | "bottom";
  state: CanvasPageState;
  getSidebarData?: (nodeId: string) => SidebarData | null;
  loadSidebarData?: (nodeId: string) => void;
  getTabData?: (nodeId: string, event: SidebarEvent) => TabData | undefined;
  getAutocompleteExampleObj?: (nodeId: string) => Record<string, unknown> | null;
  onCancelQueueItem?: (id: string) => void;
  onCancelExecution?: (executionId: string) => void;
  onReEmit?: (nodeId: string, eventOrExecutionId: string) => void;
  onRunItemOpen?: (nodeId: string | undefined, executionStatus: string, errorMessage?: string) => void;
  resolveRunId?: (event: SidebarEvent) => string | null;
  fetchRunId?: (event: SidebarEvent) => Promise<string | null>;
  onSelectRun?: (runId: string, options?: { nodeId?: string }) => void;
  getAllHistoryEvents?: (nodeId: string) => SidebarEvent[];
  onLoadMoreHistory?: (nodeId: string) => void;
  getHasMoreHistory?: (nodeId: string) => boolean;
  getLoadingMoreHistory?: (nodeId: string) => boolean;
  onLoadMoreQueue?: (nodeId: string) => void;
  getAllQueueEvents?: (nodeId: string) => SidebarEvent[];
  getHasMoreQueue?: (nodeId: string) => boolean;
  getLoadingMoreQueue?: (nodeId: string) => boolean;
  getExecutionState?: (
    nodeId: string,
    execution: CanvasesCanvasNodeExecution,
  ) => { map: EventStateMap; state: EventState };
  onSidebarClose?: () => void;
  editingNodeData?: NodeEditData | null;
  onSaveConfiguration?: (
    configuration: Record<string, unknown>,
    nodeName: string,
    integrationRef?: ComponentsIntegrationRef,
  ) => void | Promise<void>;
  configurationSaveMode?: "manual" | "auto";
  currentTab?: "latest" | "settings" | "docs";
  onTabChange?: (tab: "latest" | "settings" | "docs") => void;
  canvasMode: "live" | "edit";
  organizationId?: string;
  getCustomField?: (nodeId: string, integration?: OrganizationsIntegration) => (() => React.ReactNode) | null;
  integrations?: OrganizationsIntegration[];
  workflowNodes?: ComponentsNode[];
  components?: ActionsAction[];
  triggers?: TriggersTrigger[];
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
    return selectedNode?.type === "TYPE_WIDGET" && selectedNode?.component === "annotation";
  }, [state.componentSidebar.selectedNodeId, workflowNodes]);

  const [latestEvents, setLatestEvents] = useState<SidebarEvent[]>(sidebarData?.latestEvents || []);
  const [nextInQueueEvents, setNextInQueueEvents] = useState<SidebarEvent[]>(sidebarData?.nextInQueueEvents || []);
  const shouldShowRunsSidebar = canvasMode === "live" && !isAnnotationNode;

  // Trigger data loading when sidebar opens for a node
  useEffect(() => {
    if (shouldShowRunsSidebar && state.componentSidebar.selectedNodeId && loadSidebarData) {
      loadSidebarData(state.componentSidebar.selectedNodeId);
    }
  }, [state.componentSidebar.selectedNodeId, loadSidebarData, shouldShowRunsSidebar]);

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
  }, [editingNodeData, components, triggers]);

  if (!sidebarData) {
    return null;
  }

  // Show loading state when data is being fetched (skip for annotation nodes)
  if (sidebarData.isLoading && currentTab === "latest" && shouldShowRunsSidebar) {
    return <ComponentSidebarLoadingSkeleton layout={layout} />;
  }

  return (
    <ComponentSidebar
      key={state.componentSidebar.selectedNodeId}
      layout={layout}
      isOpen={state.componentSidebar.isOpen}
      canvasMode={canvasMode}
      onClose={onSidebarClose || state.componentSidebar.close}
      latestEvents={latestEvents}
      nextInQueueEvents={nextInQueueEvents}
      nodeId={state.componentSidebar.selectedNodeId || undefined}
      iconSrc={sidebarData.iconSrc}
      iconSlug={isAnnotationNode ? "sticky-note" : sidebarData.iconSlug}
      totalInQueueCount={sidebarData.totalInQueueCount}
      totalInHistoryCount={sidebarData.totalInHistoryCount}
      hideQueueEvents={sidebarData.hideQueueEvents}
      getTabData={
        getTabData && state.componentSidebar.selectedNodeId ? (event) => getTabData(event.nodeId!, event) : undefined
      }
      onCancelQueueItem={onCancelQueueItem}
      onCancelExecution={onCancelExecution}
      getAllHistoryEvents={() => getAllHistoryEvents?.(state.componentSidebar.selectedNodeId!) || []}
      onLoadMoreHistory={() => onLoadMoreHistory?.(state.componentSidebar.selectedNodeId!)}
      getHasMoreHistory={() => getHasMoreHistory?.(state.componentSidebar.selectedNodeId!) || false}
      getLoadingMoreHistory={() => getLoadingMoreHistory?.(state.componentSidebar.selectedNodeId!) || false}
      onLoadMoreQueue={() => onLoadMoreQueue?.(state.componentSidebar.selectedNodeId!)}
      getAllQueueEvents={() => getAllQueueEvents?.(state.componentSidebar.selectedNodeId!) || []}
      getHasMoreQueue={() => getHasMoreQueue?.(state.componentSidebar.selectedNodeId!) || false}
      getLoadingMoreQueue={() => getLoadingMoreQueue?.(state.componentSidebar.selectedNodeId!) || false}
      onReEmit={onReEmit}
      onEventClick={(event) => {
        if (event.kind === "trigger" || event.kind === "execution") {
          onRunItemOpen?.(
            state.componentSidebar.selectedNodeId ?? undefined,
            event.state ?? "unknown",
            event.originalExecution?.resultMessage,
          );
        }
      }}
      getExecutionState={getExecutionState}
      showSettingsTab={true}
      nodeConfigMode="edit"
      nodeName={editingNodeData?.nodeName || ""}
      nodeLabel={editingNodeData?.displayLabel}
      blockName={editingNodeData?.blockName}
      nodeConfiguration={editingNodeData?.configuration || {}}
      nodeConfigurationFields={editingNodeData?.configurationFields ?? []}
      onNodeConfigSave={onSaveConfiguration}
      onNodeConfigCancel={undefined}
      configurationSaveMode={configurationSaveMode}
      domainId={organizationId}
      domainType="DOMAIN_TYPE_ORGANIZATION"
      customField={
        getCustomField && state.componentSidebar.selectedNodeId
          ? getCustomField(
              state.componentSidebar.selectedNodeId,
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
      hideRunsTab={isAnnotationNode}
      hideDocsTab={isAnnotationNode}
      hideNodeId={isAnnotationNode}
      readOnly={readOnly}
      resolveRunId={resolveRunId}
      fetchRunId={fetchRunId}
      onSelectRun={onSelectRun}
    />
  );
}

function CanvasContentHeader({
  canvasName,
  organizationId,
  onPublishVersion,
  onDiscardVersion,
  onShowDiff,
  onShowConsoleDiff,
  visualDiffEnabled,
  draftVisualDiff,
  draftConsoleDiff,
  onToggleVisualDiff,
  publishVersionDisabled,
  publishVersionDisabledTooltip,
  discardVersionDisabled,
  discardVersionDisabledTooltip,
  hasStagingChanges,
  onCommitStaging,
  commitStagingPending,
  resetStagingPending,
  onResetStaging,
  headerMode,
  isEditing,
  isEditSessionActive,
  onSelectCanvasView,
  onEnterEditMode,
  enterEditModeDisabled,
  enterEditModeDisabledTooltip,
  onExitEditMode,
  exitEditModeDisabled,
  exitEditModeDisabledTooltip,
  onSelectConsole,
  onSelectMemory,
  onSelectFiles,
  filesHeaderActionsSlotId,
  publishVersionLabel,
  hasUnpublishedDraftChanges,
  hasUnpublishedCanvasDraftChanges,
  hasUnpublishedConsoleDraftChanges,
  hasFilesStagingChanges,
  hasUncommittedCanvasDraftChanges,
  hasUncommittedConsoleDraftChanges,
  hasUncommittedFilesDraftChanges,
  hasCommittedCanvasDraftChanges,
  hasCommittedConsoleDraftChanges,
  hasCommittedFilesDraftChanges,
  editTabTone,
  activeDraftBranchLabel,
  activeDraftBranchShortSha,
  showCanvasSettingsMenu,
  toolSidebarState,
  runsSidebarState,
  versionsSidebarState,
}: {
  canvasName: string;
  organizationId?: string;
  onPublishVersion?: () => void;
  onDiscardVersion?: () => void;
  onShowDiff?: () => void;
  onShowConsoleDiff?: () => void;
  visualDiffEnabled?: boolean;
  draftVisualDiff?: {
    diffCounts: { added: number; updated: number; removed: number };
    diffToggles: {
      showDeletedNodes: boolean;
      toggleShowDeletedNodes: () => void;
      showEdgeDiff: boolean;
      toggleShowEdgeDiff: () => void;
    };
  };
  draftConsoleDiff?: {
    diffCounts: { added: number; updated: number; removed: number };
  };
  onToggleVisualDiff?: () => void;
  publishVersionDisabled?: boolean;
  publishVersionDisabledTooltip?: string;
  discardVersionDisabled?: boolean;
  discardVersionDisabledTooltip?: string;
  hasStagingChanges?: boolean;
  onCommitStaging?: () => void;
  commitStagingPending?: boolean;
  resetStagingPending?: boolean;
  onResetStaging?: () => void;
  headerMode?: CanvasPageProps["headerMode"];
  isEditing?: boolean;
  isEditSessionActive?: boolean;
  onSelectCanvasView?: () => void;
  onEnterEditMode?: () => void;
  enterEditModeDisabled?: boolean;
  enterEditModeDisabledTooltip?: string;
  onExitEditMode?: () => void;
  exitEditModeDisabled?: boolean;
  exitEditModeDisabledTooltip?: string;
  onSelectConsole?: () => void;
  onSelectMemory?: () => void;
  onSelectFiles?: () => void;
  filesHeaderActionsSlotId?: string;
  publishVersionLabel?: string;
  hasUnpublishedDraftChanges?: boolean;
  hasUnpublishedCanvasDraftChanges?: boolean;
  hasUnpublishedConsoleDraftChanges?: boolean;
  hasFilesStagingChanges?: boolean;
  hasUncommittedCanvasDraftChanges?: boolean;
  hasUncommittedConsoleDraftChanges?: boolean;
  hasUncommittedFilesDraftChanges?: boolean;
  hasCommittedCanvasDraftChanges?: boolean;
  hasCommittedConsoleDraftChanges?: boolean;
  hasCommittedFilesDraftChanges?: boolean;
  editTabTone?: "uncommitted" | "ready" | "neutral";
  activeDraftBranchLabel?: string;
  activeDraftBranchShortSha?: string;
  showCanvasSettingsMenu?: boolean;
  toolSidebarState: CanvasToolSidebarState;
  runsSidebarState: CanvasRunsSidebarState;
  versionsSidebarState: CanvasVersionsSidebarState;
}) {
  return (
    <Header
      canvasName={canvasName}
      organizationId={organizationId}
      onPublishVersion={onPublishVersion}
      onDiscardVersion={onDiscardVersion}
      onShowDiff={onShowDiff}
      onShowConsoleDiff={onShowConsoleDiff}
      visualDiffEnabled={visualDiffEnabled}
      onToggleVisualDiff={onToggleVisualDiff}
      draftVisualDiff={draftVisualDiff}
      draftConsoleDiff={draftConsoleDiff}
      publishVersionDisabled={publishVersionDisabled}
      publishVersionDisabledTooltip={publishVersionDisabledTooltip}
      discardVersionDisabled={discardVersionDisabled}
      discardVersionDisabledTooltip={discardVersionDisabledTooltip}
      hasStagingChanges={hasStagingChanges}
      onCommitStaging={onCommitStaging}
      commitStagingPending={commitStagingPending}
      resetStagingPending={resetStagingPending}
      onResetStaging={onResetStaging}
      mode={headerMode}
      isEditing={isEditing}
      isEditSessionActive={isEditSessionActive}
      onSelectCanvasView={onSelectCanvasView}
      onEnterEditMode={onEnterEditMode}
      enterEditModeDisabled={enterEditModeDisabled}
      enterEditModeDisabledTooltip={enterEditModeDisabledTooltip}
      onExitEditMode={onExitEditMode}
      exitEditModeDisabled={exitEditModeDisabled}
      exitEditModeDisabledTooltip={exitEditModeDisabledTooltip}
      onSelectConsole={onSelectConsole}
      onSelectMemory={onSelectMemory}
      onSelectFiles={onSelectFiles}
      filesHeaderActionsSlotId={filesHeaderActionsSlotId}
      publishVersionLabel={publishVersionLabel}
      hasUnpublishedDraftChanges={hasUnpublishedDraftChanges}
      hasUnpublishedCanvasDraftChanges={hasUnpublishedCanvasDraftChanges}
      hasUnpublishedConsoleDraftChanges={hasUnpublishedConsoleDraftChanges}
      hasFilesStagingChanges={hasFilesStagingChanges}
      hasUncommittedCanvasDraftChanges={hasUncommittedCanvasDraftChanges}
      hasUncommittedConsoleDraftChanges={hasUncommittedConsoleDraftChanges}
      hasUncommittedFilesDraftChanges={hasUncommittedFilesDraftChanges}
      hasCommittedCanvasDraftChanges={hasCommittedCanvasDraftChanges}
      hasCommittedConsoleDraftChanges={hasCommittedConsoleDraftChanges}
      hasCommittedFilesDraftChanges={hasCommittedFilesDraftChanges}
      editTabTone={editTabTone}
      activeDraftBranchLabel={activeDraftBranchLabel}
      activeDraftBranchShortSha={activeDraftBranchShortSha}
      showCanvasSettingsMenu={showCanvasSettingsMenu}
      toolSidebarState={toolSidebarState}
      runsSidebarState={runsSidebarState}
      versionsSidebarState={versionsSidebarState}
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

type ComponentSidebarTab = "latest" | "settings" | "docs";

type NodeConfigurationWarningData = {
  component?: { error?: string };
  composite?: { error?: string };
  trigger?: { error?: string };
} | null;

function shouldOpenSidebarSettingsTab(nodeData: NodeConfigurationWarningData, isEditMode: boolean): boolean {
  return Boolean(nodeData?.component?.error || nodeData?.composite?.error || nodeData?.trigger?.error) || isEditMode;
}

function applySidebarTabOnNodeOpen(
  setCurrentTab: ((tab: ComponentSidebarTab) => void) | undefined,
  wasSidebarOpen: boolean,
  shouldOpenSettings: boolean,
): void {
  if (!setCurrentTab) {
    return;
  }
  if (!wasSidebarOpen) {
    setCurrentTab(shouldOpenSettings ? "settings" : "latest");
    return;
  }
  if (shouldOpenSettings) {
    setCurrentTab("settings");
  }
}

function CanvasContent({
  state,
  onNodeDelete,
  onNodesDelete,
  onDuplicateNodes,
  onAutoLayoutNodes,
  onEdgeCreate,
  onDuplicate,
  onToggleView,
  onShowNodeDiff,
  onAnnotationUpdate,
  onAnnotationBlur,
  onBuildingBlockDrop,
  onBuildingBlocksSidebarToggle,
  onConnectionDropInEmptySpace,
  onZoomChange,
  hasFitToViewRef,
  viewportRefProp,
  onPendingConnectionNodeClick,
  onNodeClick,
  workflowNodes,
  setCurrentTab,
  showBottomStatusControls = true,
  isRunInspectionMode = false,
  isEditing = false,
  isAutoLayoutOnUpdateEnabled,
  onToggleAutoLayoutOnUpdate,
  autoLayoutOnUpdateDisabled,
  autoLayoutOnUpdateDisabledTooltip,
  readOnly,
  logEntries = [],
  focusRequest,
  initialFocusNodeId,
  fitAllRequest,
  fitAllFocusNodeIds,
  runParticipantNodeIds,
  runSelectedNodeId,
  runNodeDetailPaneOpen,
  logRuns,
  runsNodes,
  runsComponentIconMap,
  onRunNodeSelect,
  onRunExecutionSelect,
  onAcknowledgeErrors,
  missingIntegrations,
  onConnectIntegration,
  canCreateIntegrations,
}: {
  state: CanvasPageState;
  onNodeDelete?: (nodeId: string) => void;
  onNodesDelete?: (nodeIds: string[]) => void;
  onDuplicateNodes?: (nodeIds: string[]) => void;
  onAutoLayoutNodes?: (nodeIds: string[]) => void;
  onEdgeCreate?: (sourceId: string, targetId: string, sourceHandle?: string | null) => void;
  onDuplicate?: (nodeId: string) => void;
  onToggleView?: (nodeId: string) => void;
  onShowNodeDiff?: (nodeId: string) => void;
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
  onPendingConnectionNodeClick?: (nodeId: string) => void;
  onNodeClick?: (nodeId: string) => void;
  workflowNodes?: ComponentsNode[];
  setCurrentTab?: (tab: "latest" | "settings" | "docs") => void;
  showBottomStatusControls?: boolean;
  isRunInspectionMode?: boolean;
  isEditing?: boolean;
  isAutoLayoutOnUpdateEnabled?: boolean;
  onToggleAutoLayoutOnUpdate?: () => void;
  autoLayoutOnUpdateDisabled?: boolean;
  autoLayoutOnUpdateDisabledTooltip?: string;
  readOnly?: boolean;
  logEntries?: LogEntry[];
  focusRequest?: FocusRequest | null;
  initialFocusNodeId?: string | null;
  fitAllRequest?: number | null;
  fitAllFocusNodeIds?: string[];
  runParticipantNodeIds?: string[];
  runSelectedNodeId?: string | null;
  runNodeDetailPaneOpen?: boolean;
  logRuns?: CanvasesCanvasRun[];
  runsNodes?: ComponentsNode[];
  runsComponentIconMap?: Record<string, string>;
  onRunNodeSelect?: (nodeId: string) => void;
  onRunExecutionSelect?: (options: { runId: string; nodeId: string }) => void;
  onAcknowledgeErrors?: (executionIds: string[]) => void;
  missingIntegrations?: MissingIntegration[];
  onConnectIntegration?: (integrationName: string) => void;
  canCreateIntegrations?: boolean;
}) {
  const { fitView, screenToFlowPosition, getViewport, getInternalNode, getNodes, setViewport } = useReactFlow();
  const { zoom } = useViewport();
  const isReadOnly = readOnly ?? false;

  // Determine selection key code to support both Control (Windows/Linux) and Meta (Mac)
  // Similar to existing keyboard shortcuts that check (e.ctrlKey || e.metaKey)
  const selectionKey = useMemo(() => {
    const isMac = navigator.platform.toLowerCase().includes("mac");
    return isMac ? "Meta" : "Control";
  }, []);

  // Use refs to avoid recreating callbacks when state changes
  const stateRef = useRef(state);
  stateRef.current = state;
  const localViewportRef = useRef<{ x: number; y: number; zoom: number } | undefined>(undefined);

  // Use viewport ref from props if provided, otherwise create local one
  const viewportRef = viewportRefProp ?? localViewportRef;

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
  const lastFitAllRequestRef = useRef<{ nonce: number; runMode: boolean } | null>(null);
  const [isLogSidebarOpen, setIsLogSidebarOpen] = useState(() => {
    const saved = localStorage.getItem(CONSOLE_OPEN_STORAGE_KEY);
    return saved !== null ? saved === "true" : false;
  });
  const [consoleTab, setConsoleTab] = useState<ConsoleTab>("errors");
  const [logSearch, setLogSearch] = useState("");
  const [logSidebarHeight, setLogSidebarHeight] = useState(() => {
    const saved = localStorage.getItem(CONSOLE_HEIGHT_STORAGE_KEY);
    return saved ? parseInt(saved, 10) : 320;
  });
  const [isSnapToGridEnabled, setIsSnapToGridEnabled] = useState(true);
  const isEditMode = isEditing;
  const runSelectableSet = useMemo(() => {
    if (!isRunInspectionMode || !runParticipantNodeIds || runParticipantNodeIds.length === 0) {
      return null;
    }
    return new Set(runParticipantNodeIds);
  }, [isRunInspectionMode, runParticipantNodeIds]);

  useEffect(() => {
    if (isEditMode) {
      return;
    }

    setHoveredEdgeId(null);
  }, [isEditMode]);

  useEffect(() => {
    if (showBottomStatusControls) {
      localStorage.setItem(CONSOLE_OPEN_STORAGE_KEY, String(isLogSidebarOpen));
    }
  }, [isLogSidebarOpen, showBottomStatusControls]);

  useEffect(() => {
    localStorage.setItem(CONSOLE_HEIGHT_STORAGE_KEY, String(logSidebarHeight));
  }, [logSidebarHeight]);

  const unacknowledgedErrorCount = useMemo(() => countUnacknowledgedErrors(logRuns || []), [logRuns]);

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

  const handleNodeClick = useCallback(
    (nodeId: string, e?: React.MouseEvent) => {
      const isMultiSelectClick = e && (e.ctrlKey || e.metaKey);
      if (isMultiSelectClick) return;

      const clickedNode = stateRef.current.nodes?.find((n) => n.id === nodeId);
      const isPendingConnection = clickedNode?.data?.isPendingConnection;
      const isAnnotationNode = clickedNode?.data?.type === "annotation";

      const workflowNode = workflowNodes?.find((n) => n.id === nodeId);
      const isPlaceholder = workflowNode?.name === "New Component" && !workflowNode.component;

      if (isAnnotationNode) {
        return;
      }

      if (runSelectableSet && onNodeClick && !runSelectableSet.has(nodeId)) {
        return;
      }

      if (isPendingConnection && onPendingConnectionNodeClick) {
        onPendingConnectionNodeClick(nodeId);
      } else if (isPlaceholder && onPendingConnectionNodeClick) {
        onPendingConnectionNodeClick(nodeId);
      } else if (onNodeClick) {
        onNodeClick(nodeId);
      } else {
        const wasSidebarOpen = stateRef.current.componentSidebar.isOpen;
        stateRef.current.componentSidebar.open(nodeId);

        const nodeData = clickedNode?.data as NodeConfigurationWarningData;
        applySidebarTabOnNodeOpen(setCurrentTab, wasSidebarOpen, shouldOpenSidebarSettingsTab(nodeData, isEditMode));
        onBuildingBlocksSidebarToggle?.(false);
      }

      stateRef.current.setNodes((nodes) =>
        nodes.map((node) => ({
          ...node,
          selected: node.id === nodeId,
        })),
      );
    },
    [
      workflowNodes,
      onBuildingBlocksSidebarToggle,
      onPendingConnectionNodeClick,
      onNodeClick,
      setCurrentTab,
      isEditMode,
      runSelectableSet,
    ],
  );

  const onNodeDeleteRef = useRef(onNodeDelete);
  onNodeDeleteRef.current = onNodeDelete;

  const onDuplicateRef = useRef(onDuplicate);
  onDuplicateRef.current = onDuplicate;

  const onToggleViewRef = useRef(onToggleView);
  onToggleViewRef.current = onToggleView;
  const onShowNodeDiffRef = useRef(onShowNodeDiff);
  onShowNodeDiffRef.current = onShowNodeDiff;

  const onAnnotationUpdateRef = useRef(onAnnotationUpdate);
  onAnnotationUpdateRef.current = onAnnotationUpdate;
  const onAnnotationBlurRef = useRef(onAnnotationBlur);
  onAnnotationBlurRef.current = onAnnotationBlur;

  const handleConnect = useCallback(
    (connection: Connection) => {
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
    (_event: unknown, newViewport: Viewport) => {
      viewportRef.current = newViewport;
      reportZoom(newViewport.zoom);
    },
    [reportZoom, viewportRef],
  );

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
  const suppressNextPaneClickRef = useRef(false);
  const suppressNextPaneClickTimeoutRef = useRef<number | null>(null);
  const runCanvasNodeIdsKey = useMemo(() => state.nodes.map((node) => node.id).join("|"), [state.nodes]);

  useEffect(() => {
    if (!focusRequest) {
      return;
    }

    const targetNode =
      getNodes().find((node) => node.id === focusRequest.nodeId) ??
      stateRef.current.nodes?.find((node) => node.id === focusRequest.nodeId);
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
  }, [focusRequest, fitView, getNodes, runCanvasNodeIdsKey]);

  useEffect(() => {
    if (!isRunInspectionMode) {
      return;
    }

    stateRef.current.setNodes((nodes) => {
      if (!runSelectedNodeId) {
        if (nodes.every((node) => !node.selected)) {
          return nodes;
        }
        return nodes.map((node) => ({ ...node, selected: false }));
      }

      if (!nodes.some((node) => node.id === runSelectedNodeId)) {
        return nodes;
      }

      const alreadyCorrect = nodes.every((node) => node.selected === (node.id === runSelectedNodeId));
      if (alreadyCorrect) {
        return nodes;
      }

      return nodes.map((node) => ({
        ...node,
        selected: node.id === runSelectedNodeId,
      }));
    });
  }, [isRunInspectionMode, runSelectedNodeId, runCanvasNodeIdsKey]);

  // Listen for agent sidebar node chip clicks to zoom to nodes
  useEffect(() => {
    const handler = (e: Event) => {
      const nodeId = (e as CustomEvent).detail?.nodeId;
      if (!nodeId) return;
      const targetNode = stateRef.current.nodes?.find((n) => n.id === nodeId);
      if (!targetNode) return;
      stateRef.current.setNodes((nodes) => nodes.map((n) => ({ ...n, selected: n.id === nodeId })));
      fitView({ nodes: [targetNode], duration: 500, maxZoom: 1.2 });
    };
    window.addEventListener("agent:focus-node", handler);
    return () => window.removeEventListener("agent:focus-node", handler);
  }, [fitView]);

  useEffect(() => {
    return () => {
      if (suppressNextPaneClickTimeoutRef.current !== null && typeof window !== "undefined") {
        window.clearTimeout(suppressNextPaneClickTimeoutRef.current);
      }
    };
  }, []);

  const suppressNextPaneClick = useCallback(() => {
    suppressNextPaneClickRef.current = true;

    if (typeof window === "undefined") {
      return;
    }

    if (suppressNextPaneClickTimeoutRef.current !== null) {
      window.clearTimeout(suppressNextPaneClickTimeoutRef.current);
    }

    suppressNextPaneClickTimeoutRef.current = window.setTimeout(() => {
      suppressNextPaneClickRef.current = false;
      suppressNextPaneClickTimeoutRef.current = null;
    }, 250);
  }, []);

  const handlePaneClick = useCallback(() => {
    if (suppressNextPaneClickRef.current) {
      suppressNextPaneClickRef.current = false;
      return;
    }

    previouslySelectedRef.current = new Set();

    if (isRunInspectionMode && runSelectedNodeId) {
      return;
    }

    const isLiveBottomInspectorOpen = !isRunInspectionMode && !isEditMode && stateRef.current.componentSidebar.isOpen;

    if (isLiveBottomInspectorOpen) {
      stateRef.current.setNodes((nodes) =>
        nodes.map((node) => ({
          ...node,
          selected: false,
        })),
      );
      stateRef.current.componentSidebar.clearSelection();
      return;
    }

    if (!isEditMode && stateRef.current.componentSidebar.isOpen) {
      return;
    }

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
  }, [isEditMode, isRunInspectionMode, onBuildingBlocksSidebarToggle, runSelectedNodeId]);

  // Handle fit to view on ReactFlow initialization
  const handleInit = useCallback(
    (reactFlowInstance: { setViewport: (viewport: Viewport) => void }) => {
      if (!hasFitToViewRef.current) {
        const hasNodes = (stateRef.current.nodes?.length ?? 0) > 0;

        const focusNodeId = initialFocusNodeId;
        const focusNode = focusNodeId ? stateRef.current.nodes?.find((node) => node.id === focusNodeId) : null;

        if (focusNode) {
          fitView({ nodes: [focusNode], duration: 500, maxZoom: 1.2 });
        } else if (hasNodes) {
          fitView({ ...LIVE_CANVAS_FIT_VIEW_OPTIONS, duration: 500 });
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

  // Fit all currently-rendered nodes into view whenever the parent bumps `fitAllRequest`.
  // Wait a microtask so ReactFlow has measured the just-swapped node set (e.g. switching
  // between runs whose participating nodes have different coordinates) before fitting.
  useEffect(() => {
    if (fitAllRequest == null) {
      lastFitAllRequestRef.current = null;
      return;
    }
    const last = lastFitAllRequestRef.current;
    if (last?.nonce === fitAllRequest && last.runMode === isRunInspectionMode) return;
    if (!hasFitToViewRef.current) return;
    lastFitAllRequestRef.current = { nonce: fitAllRequest, runMode: isRunInspectionMode };
    const id = window.setTimeout(() => {
      const focusIds = fitAllFocusNodeIds?.length ? new Set(fitAllFocusNodeIds) : null;
      const renderedNodes = getNodes();
      const nodeSubset =
        focusIds && focusIds.size > 0 ? renderedNodes.filter((n) => n.id && focusIds.has(n.id)) : undefined;
      const fitOptions = isRunInspectionMode ? RUN_CANVAS_FIT_VIEW_OPTIONS : LIVE_CANVAS_FIT_VIEW_OPTIONS;
      fitView({
        ...(nodeSubset && nodeSubset.length > 0 ? { nodes: nodeSubset } : {}),
        ...fitOptions,
        duration: 500,
      });
    }, 0);
    return () => window.clearTimeout(id);
  }, [fitAllRequest, fitAllFocusNodeIds, fitView, getNodes, hasFitToViewRef, isRunInspectionMode]);

  const showHeader = !isReadOnly;

  const hasMultiSelection = multiSelectedNodes.length >= 2;

  const handleAppendFromNode = useCallback(
    (sourceNodeId: string, sourceHandleId?: string | null) => {
      if (isReadOnly || !onConnectionDropInEmptySpace) {
        return;
      }

      const sourceNode = stateRef.current.nodes?.find((node) => node.id === sourceNodeId);
      if (!sourceNode) {
        return;
      }

      const sourceWidth = sourceNode.width ?? 240;
      const appendGapX = 300;
      const appendAlignmentY = 30;
      const placeholderPosition = {
        x: sourceNode.position.x + sourceWidth + appendGapX,
        y: sourceNode.position.y + appendAlignmentY,
      };

      const currentViewport = getViewport();
      const canvasWidth =
        typeof document === "undefined" ? 0 : (document.querySelector(".react-flow")?.clientWidth ?? window.innerWidth);
      const rightSidebarSafeArea = 560;
      const placeholderEstimatedWidth = 420;
      const viewportBuffer = 48;
      const placeholderRightScreenX =
        (placeholderPosition.x + placeholderEstimatedWidth) * currentViewport.zoom + currentViewport.x;
      const maxVisibleScreenX = canvasWidth - rightSidebarSafeArea - viewportBuffer;

      if (canvasWidth > 0 && placeholderRightScreenX > maxVisibleScreenX) {
        const overflow = placeholderRightScreenX - maxVisibleScreenX;
        const nextViewport = { ...currentViewport, x: currentViewport.x - overflow };
        setViewport(nextViewport, { duration: 180 });
        viewportRef.current = nextViewport;
      }

      onConnectionDropInEmptySpace(placeholderPosition, {
        nodeId: sourceNodeId,
        handleId: sourceHandleId ?? "default",
      });
    },
    [getViewport, isReadOnly, onConnectionDropInEmptySpace, setViewport, viewportRef],
  );

  // Store callback handlers in a ref so they can be accessed without being in node data
  const callbacksRef = useRef({
    handleNodeClick,
    onAppendFromNode: handleAppendFromNode,
    onNodeDelete: onNodeDeleteRef,
    onDuplicate: onDuplicateRef,
    onToggleView: onToggleViewRef,
    onShowNodeDiff: onShowNodeDiffRef,
    onAnnotationUpdate: onAnnotationUpdateRef,
    onAnnotationBlur: onAnnotationBlurRef,
    showHeader,
    hasMultiSelection,
    canvasMode: isEditMode ? ("edit" as const) : ("live" as const),
  });
  callbacksRef.current = {
    handleNodeClick,
    onAppendFromNode: handleAppendFromNode,
    onNodeDelete: onNodeDeleteRef,
    onDuplicate: onDuplicateRef,
    onToggleView: onToggleViewRef,
    onShowNodeDiff: onShowNodeDiffRef,
    onAnnotationUpdate: onAnnotationUpdateRef,
    onAnnotationBlur: onAnnotationBlurRef,
    showHeader,
    hasMultiSelection,
    canvasMode: isEditMode ? "edit" : "live",
  };

  // Just pass the state nodes directly - callbacks will be added in nodeTypes
  const [hoveredEdgeId, setHoveredEdgeId] = useState<string | null>(null);
  const [connectingFrom, setConnectingFrom] = useState<CanvasConnectionState | null>(null);

  // Track connection completion for empty space drop detection
  const connectionCompletedRef = useRef(false);
  const connectingFromRef = useRef<CanvasConnectionState | null>(null);
  const blockConnectingFrom = useMemo(
    () =>
      connectingFrom
        ? {
            nodeId: connectingFrom.nodeId,
            handleId: connectingFrom.handleId,
            handleType: connectingFrom.handleType ?? undefined,
          }
        : undefined,
    [connectingFrom],
  );

  const handleEdgeMouseEnter = useCallback((_event: React.MouseEvent, edge: CanvasEdge) => {
    setHoveredEdgeId(edge.id);
  }, []);

  const handleEdgeMouseLeave = useCallback(() => {
    setHoveredEdgeId(null);
  }, []);

  const handleConnectStart = useCallback(
    (
      _event: unknown,
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
            suppressNextPaneClick();
            onConnectionDropInEmptySpace(canvasPosition, currentConnectingFrom);
          }
        }
      }

      setConnectingFrom(null);
      connectingFromRef.current = null;
      connectionCompletedRef.current = false;
    },
    [screenToFlowPosition, onConnectionDropInEmptySpace, suppressNextPaneClick, isReadOnly],
  );

  // Find the hovered edge to get its source and target
  const hoveredEdge = useMemo(() => {
    if (!hoveredEdgeId) return null;
    return (state.edges?.find((e) => e.id === hoveredEdgeId) as CanvasEdge | undefined) ?? null;
  }, [hoveredEdgeId, state.edges]);

  const enrichedNodeCacheRef = useRef<Map<string, EnrichedCanvasNodeCacheEntry>>(new Map());
  const nodesWithCallbacks = useMemo(() => {
    const runParticipantKey =
      runParticipantNodeIds !== undefined && runParticipantNodeIds.length > 0
        ? [...runParticipantNodeIds].sort().join("|")
        : "";
    const runParticipantSet =
      runParticipantNodeIds !== undefined && runParticipantNodeIds.length > 0 ? new Set(runParticipantNodeIds) : null;
    const edgeHoverActive = false;
    const runDimActive = runParticipantSet !== null;
    const hasHighlightedNodes = edgeHoverActive || runDimActive;
    const visibleNodeIds = new Set<string>();
    const enrichedNodes = state.nodes.map((node) => {
      visibleNodeIds.add(node.id);

      const isHighlighted = isCanvasNodeHighlighted({
        nodeId: node.id,
        edgeHoverActive,
        highlightedNodeIds: new Set<string>(),
        runDimActive,
        runParticipantSet,
      });
      const shouldBlankBody = shouldBlankCanvasNodeBody({
        nodeId: node.id,
        edgeHoverActive,
        runDimActive,
        runParticipantSet,
      });
      const cachedNode = enrichedNodeCacheRef.current.get(node.id);
      const canReuseData = canReuseEnrichedNodeData({
        cachedNode,
        node,
        hoveredEdge,
        connectingFrom,
        edges: state.edges,
        isHighlighted,
        hasHighlightedNodes,
        runParticipantKey,
      });

      if (canReuseData && cachedNode && cachedNode.sourceNode === node) {
        return cachedNode.node;
      }

      const sourceData = node.data as CanvasBlockNodeData;
      const data =
        canReuseData && cachedNode
          ? cachedNode.data
          : {
              ...sourceData,
              _callbacksRef: callbacksRef,
              _hoveredEdge: hoveredEdge ?? undefined,
              _connectingFrom: blockConnectingFrom,
              _allEdges: state.edges,
              _isHighlighted: isHighlighted,
              _hasHighlightedNodes: hasHighlightedNodes,
              _dimBodyBelowHeader: shouldBlankBody,
            };
      const enrichedNode: ReactFlowNode = {
        ...node,
        selectable: runSelectableSet ? runSelectableSet.has(node.id) : (node.selectable ?? true),
        data: data as ReactFlowNode["data"],
      };

      enrichedNodeCacheRef.current.set(node.id, {
        sourceNode: node,
        sourceData: node.data,
        node: enrichedNode,
        data,
        hoveredEdge,
        connectingFrom,
        edges: state.edges,
        isHighlighted,
        hasHighlightedNodes,
        runParticipantKey,
      });

      return enrichedNode;
    });

    for (const nodeId of enrichedNodeCacheRef.current.keys()) {
      if (!visibleNodeIds.has(nodeId)) {
        enrichedNodeCacheRef.current.delete(nodeId);
      }
    }

    return enrichedNodes;
  }, [
    state.nodes,
    hoveredEdge,
    connectingFrom,
    state.edges,
    blockConnectingFrom,
    runParticipantNodeIds,
    runSelectableSet,
  ]);

  const edgeTypes = useMemo(
    () => ({
      custom: CustomEdge,
    }),
    [],
  );
  const onEdgesChangeRef = useRef(state.onEdgesChange);
  onEdgesChangeRef.current = state.onEdgesChange;
  const stableEdgeDelete = useCallback(
    (edgeId: string) => onEdgesChangeRef.current([{ id: edgeId, type: "remove" }]),
    [],
  );
  const styledEdges = useMemo(() => {
    return state.edges?.map((e) => {
      const diffStatus = (e.data as Record<string, unknown> | undefined)?._draftDiffStatus;
      const diffStyle = getDraftDiffEdgeStyle(diffStatus) ?? {};

      return {
        ...e,
        ...EDGE_STYLE,
        style: { ...EDGE_STYLE.style, ...diffStyle },
        data: {
          ...e.data,
          isHovered: e.id === hoveredEdgeId,
          canDelete: isEditMode && !isReadOnly && diffStatus !== "removed",
          onDelete: isEditMode && !isReadOnly && diffStatus !== "removed" ? stableEdgeDelete : undefined,
        },
        zIndex: e.id === hoveredEdgeId ? 1000 : 0,
      };
    });
  }, [state.edges, hoveredEdgeId, stableEdgeDelete, isEditMode, isReadOnly]);

  const isConnectionEditingEnabled = isEditMode && !isReadOnly && !!onEdgeCreate;
  const { onNodesChange, onEdgesChange } = state;

  const handleNodesChange = useCallback(
    (incomingChanges: NodeChange[]) => {
      let changes = incomingChanges;
      if (runSelectableSet) {
        changes = changes.filter((c) => !(c.type === "select" && c.selected === true && !runSelectableSet.has(c.id)));
      }

      const prev = previouslySelectedRef.current;

      if (prev.size > 0) {
        changes = changes.map((c) => {
          if (c.type === "select" && !c.selected && prev.has(c.id)) {
            return { ...c, selected: true };
          }
          return c;
        });
      }

      if (!isReadOnly) {
        onNodesChange(changes);
        return;
      }

      const filteredChanges = changes.filter((change) => change.type === "select" || change.type === "dimensions");
      if (filteredChanges.length > 0) {
        onNodesChange(filteredChanges);
      }
    },
    [isReadOnly, onNodesChange, runSelectableSet],
  );

  const handleEdgesChange = useCallback(
    (changes: EdgeChange[]) => {
      if (!isReadOnly) {
        onEdgesChange(changes);
        return;
      }

      const filteredChanges = changes.filter((change) => change.type === "select");
      if (filteredChanges.length > 0) {
        onEdgesChange(filteredChanges);
      }
    },
    [isReadOnly, onEdgesChange],
  );

  const logCounts = useMemo(() => {
    return logEntries.reduce(
      (acc, entry) => {
        acc.total += 1;
        if (entry.type === "error") acc.error += 1;
        if (entry.type === "warning") acc.warning += 1;
        if (entry.type === "success") acc.success += 1;
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
  const handleSnapToGridToggle = useCallback(() => setIsSnapToGridEnabled((prev) => !prev), []);
  const handleNodeSearch = useCallback((searchString: string) => {
    const query = searchString.toLowerCase();
    return stateRef.current.nodes.filter((node) => {
      const nodeData = node.data as unknown as CanvasBlockNodeData | undefined;
      const label = (nodeData?.label || "").toLowerCase();
      const nodeName = (nodeData?.nodeName || "").toLowerCase();
      const id = (node.id || "").toLowerCase();
      return label.includes(query) || nodeName.includes(query) || id.includes(query);
    });
  }, []);
  const handleNodeSearchSelect = useCallback((node: ReactFlowNode) => {
    const nodeData = node.data as unknown as CanvasBlockNodeData | undefined;
    const isAnnotationNode = nodeData?.type === "annotation";
    if (isAnnotationNode) {
      return;
    }
    stateRef.current.componentSidebar.open(node.id);
  }, []);
  const handleOpenCommandPalette = useCallback(() => {
    openGlobalCommandPalette();
  }, []);
  const autoLayoutToggleControl = useMemo(() => {
    if (!isEditMode) {
      return null;
    }

    return (
      <Tooltip>
        <TooltipTrigger asChild>
          <span className="inline-flex">
            <Button
              variant="ghost"
              size="sm"
              className="h-7 w-7 px-0 text-slate-600 hover:text-slate-900"
              onClick={handleToggleAutoLayoutOnUpdate}
              disabled={isAutoLayoutToggleDisabled}
              aria-pressed={isAutoLayoutOnUpdateEnabled}
            >
              {isAutoLayoutOnUpdateEnabled ? (
                <LayoutGrid className="h-3 w-3" />
              ) : (
                <LayoutDashboard className="h-3 w-3" />
              )}
            </Button>
          </span>
        </TooltipTrigger>
        <TooltipContent>{autoLayoutTooltipMessage}</TooltipContent>
      </Tooltip>
    );
  }, [
    autoLayoutTooltipMessage,
    handleToggleAutoLayoutOnUpdate,
    isAutoLayoutOnUpdateEnabled,
    isAutoLayoutToggleDisabled,
    isEditMode,
  ]);
  const commandPaletteSearchControl = useMemo(
    () => (
      <Tooltip>
        <TooltipTrigger asChild>
          <Button
            variant="ghost"
            size="icon-sm"
            className="h-7 w-7"
            onClick={handleOpenCommandPalette}
            aria-label="Search commands and components"
          >
            <Search className="h-3 w-3" />
          </Button>
        </TooltipTrigger>
        <TooltipContent>Search Components</TooltipContent>
      </Tooltip>
    ),
    [handleOpenCommandPalette],
  );
  const zoomSliderContent = useMemo(
    () => (
      <>
        {autoLayoutToggleControl}
        {commandPaletteSearchControl}
      </>
    ),
    [autoLayoutToggleControl, commandPaletteSearchControl],
  );
  const reactFlowStyle = useMemo(() => ({ opacity: isInitialized ? 1 : 0 }), [isInitialized]);
  const handleSelectionStart = useCallback(() => {
    setIsSelecting(true);
    const selected = (stateRef.current.nodes || []).filter((n) => n.selected).map((n) => n.id);
    previouslySelectedRef.current = new Set(selected);
  }, []);
  const handleSelectionEnd = useCallback(() => {
    setIsSelecting(false);
    previouslySelectedRef.current = new Set();
  }, []);

  return (
    <div className="h-full w-full relative">
      <div className="h-full">
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
            snapGrid={[SNAP_GRID_STEP_PX, SNAP_GRID_STEP_PX]}
            panOnScrollSpeed={0.8}
            nodesDraggable={!isReadOnly}
            nodesConnectable={isConnectionEditingEnabled}
            elementsSelectable={true}
            onlyRenderVisibleElements={true}
            onNodesChange={handleNodesChange}
            onEdgesChange={handleEdgesChange}
            onConnect={isConnectionEditingEnabled ? handleConnect : undefined}
            onConnectStart={isConnectionEditingEnabled ? handleConnectStart : undefined}
            onConnectEnd={isConnectionEditingEnabled ? handleConnectEnd : undefined}
            onDragOver={isReadOnly ? undefined : handleDragOver}
            onDrop={isReadOnly ? undefined : handleDrop}
            onMove={handleMove}
            onInit={handleInit}
            deleteKeyCode={null}
            onPaneClick={handlePaneClick}
            onSelectionStart={handleSelectionStart}
            onSelectionEnd={handleSelectionEnd}
            onEdgeMouseEnter={isEditMode ? handleEdgeMouseEnter : undefined}
            onEdgeMouseLeave={isEditMode ? handleEdgeMouseLeave : undefined}
            defaultViewport={viewport}
            fitView={false}
            style={reactFlowStyle}
            className="h-full w-full"
          >
            <Background gap={8} size={2} bgColor="#F1F5F9" color="#cbd5e1" />
            <GlobalCommandPaletteCanvasNodeSearch onSearch={handleNodeSearch} onSelectNode={handleNodeSearchSelect} />
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
              <div className="flex h-7 items-center gap-3">
                <ZoomSlider
                  orientation="horizontal"
                  className={cn(
                    "!static !m-0",
                    runNodeDetailPaneOpen && "opacity-50 transition-opacity hover:opacity-100",
                  )}
                  isSnapToGridEnabled={isEditMode ? isSnapToGridEnabled : undefined}
                  onSnapToGridToggle={isEditMode ? handleSnapToGridToggle : undefined}
                >
                  {zoomSliderContent}
                </ZoomSlider>
                {showBottomStatusControls && !isLogSidebarOpen ? (
                  <div className="bg-white text-gray-800 outline-1 outline-slate-950/15 flex h-7 items-center gap-1 rounded-md p-0.5">
                    <Tooltip>
                      <TooltipTrigger asChild>
                        <Button
                          variant="ghost"
                          size="sm"
                          className={cn(
                            "h-7 items-center text-xs font-medium",
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
                          className="h-7 items-center text-xs font-medium"
                          onClick={() => handleLogButtonClick("warnings")}
                        >
                          <CircleAlert
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
              (onNodesDelete || onNodeDelete || onAutoLayoutNodes || onDuplicateNodes) && (
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
                      {onAutoLayoutNodes && (
                        <Tooltip>
                          <TooltipTrigger asChild>
                            <button
                              type="button"
                              data-testid="multi-select-auto-layout"
                              aria-label="Tidy"
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
                          </TooltipTrigger>
                          <TooltipContent>Tidy</TooltipContent>
                        </Tooltip>
                      )}
                      {onDuplicateNodes && (
                        <Tooltip>
                          <TooltipTrigger asChild>
                            <button
                              type="button"
                              data-testid="multi-select-duplicate"
                              aria-label="Copy"
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
                          </TooltipTrigger>
                          <TooltipContent>Copy</TooltipContent>
                        </Tooltip>
                      )}
                      {(onNodesDelete || onNodeDelete) && (
                        <Tooltip>
                          <TooltipTrigger asChild>
                            <button
                              type="button"
                              data-testid="multi-select-delete"
                              aria-label="Delete Selected"
                              onPointerDown={stopCanvasPointerEvent}
                              onMouseDown={stopCanvasPointerEvent}
                              onClick={(event) => {
                                event.preventDefault();
                                event.stopPropagation();
                                if (
                                  !window.confirm(
                                    "Are you sure you want to delete the selected nodes? This action cannot be undone.",
                                  )
                                ) {
                                  return;
                                }
                                const nodeIds = multiSelectedNodes.map((n) => n.id);
                                if (onNodesDelete) {
                                  onNodesDelete(nodeIds);
                                } else {
                                  for (const id of nodeIds) {
                                    onNodeDelete?.(id);
                                  }
                                }
                                stateRef.current.setNodes((nodes) =>
                                  nodes.map((node) => ({ ...node, selected: false })),
                                );
                                setMultiSelectedNodes([]);
                              }}
                              className="flex items-center justify-center p-1 text-gray-500 transition hover:text-gray-800"
                            >
                              <Trash2 className="h-4 w-4" />
                            </button>
                          </TooltipTrigger>
                          <TooltipContent>Delete Selected</TooltipContent>
                        </Tooltip>
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
          logRuns={logRuns}
          runsNodes={runsNodes}
          runsComponentIconMap={runsComponentIconMap}
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
