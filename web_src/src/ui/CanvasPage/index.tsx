/* eslint-disable @typescript-eslint/no-explicit-any */
import {
  Background,
  ReactFlow,
  ReactFlowProvider,
  useReactFlow,
  type Edge as ReactFlowEdge,
  type Node as ReactFlowNode,
} from "@xyflow/react";

import { Loader2 } from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";

import { ConfigurationField } from "@/api-client";
import { AiSidebar } from "../ai";
import { BuildingBlock, BuildingBlockCategory, BuildingBlocksSidebar } from "../BuildingBlocksSidebar";
import { ComponentSidebar } from "../componentSidebar";
import { TabData } from "../componentSidebar/SidebarEventItem/SidebarEventItem";
import { EmitEventModal } from "../EmitEventModal";
import type { MetadataItem } from "../metadataList";
import { ViewToggle } from "../ViewToggle";
import { Block, BlockData } from "./Block";
import "./canvas-reset.css";
import { CustomEdge } from "./CustomEdge";
import { Header, type BreadcrumbItem } from "./Header";
import { NodeConfigurationModal } from "./NodeConfigurationModal";
import { Simulation } from "./storybooks/useSimulation";
import { CanvasPageState, useCanvasState } from "./useCanvasState";

export interface SidebarEvent {
  id: string;
  title: string;
  subtitle?: string;
  state: "processed" | "discarded" | "waiting" | "running";
  isOpen: boolean;
  receivedAt?: Date;
  values?: Record<string, string>;
}

export interface SidebarData {
  latestEvents: SidebarEvent[];
  nextInQueueEvents: SidebarEvent[];
  metadata: MetadataItem[];
  title: string;
  iconSrc?: string;
  iconSlug?: string;
  iconColor?: string;
  iconBackground?: string;
  moreInQueueCount: number;
  hideQueueEvents?: boolean;
  isLoading?: boolean;
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
}

export interface NewNodeData {
  buildingBlock: BuildingBlock;
  nodeName: string;
  displayLabel?: string;
  configuration: Record<string, any>;
  position?: { x: number; y: number };
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
  onNodeConfigurationSave?: (nodeId: string, configuration: Record<string, any>, nodeName: string) => void;
  onSave?: (nodes: CanvasNode[]) => void;
  onEdgeCreate?: (sourceId: string, targetId: string, sourceHandle?: string | null) => void;
  onNodeDelete?: (nodeId: string) => void;
  onEdgeDelete?: (edgeIds: string[]) => void;
  onNodePositionChange?: (nodeId: string, position: { x: number; y: number }) => void;
  onCancelQueueItem?: (nodeId: string, queueItemId: string) => void;
  onDirty?: () => void;

  onRun?: (nodeId: string, channel: string, data: any) => void | Promise<void>;
  onDuplicate?: (nodeId: string) => void;
  onDocs?: (nodeId: string) => void;
  onEdit?: (nodeId: string) => void;
  onConfigure?: (nodeId: string) => void;
  onDeactivate?: (nodeId: string) => void;
  onToggleView?: (nodeId: string) => void;
  onToggleCollapse?: () => void;

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
}

export const CANVAS_SIDEBAR_STORAGE_KEY = "canvasSidebarOpen";

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
  const [editingNodeData, setEditingNodeData] = useState<NodeEditData | null>(null);
  const [newNodeData, setNewNodeData] = useState<NewNodeData | null>(null);

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
      // Try the modal-based edit first (for node configuration)
      if (props.getNodeEditData) {
        const editData = props.getNodeEditData(nodeId);
        if (editData) {
          setEditingNodeData(editData);
          return;
        }
      }

      // Fall back to the simple onEdit callback
      if (props.onEdit) {
        props.onEdit(nodeId);
      }
    },
    [props],
  );

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

  const handleBuildingBlockDrop = useCallback((block: BuildingBlock, position?: { x: number; y: number }) => {
    setNewNodeData({
      buildingBlock: block,
      nodeName: block.name || "",
      displayLabel: block.label || block.name || "",
      configuration: {},
      position,
    });
  }, []);

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
    (configuration: Record<string, any>, nodeName: string) => {
      if (editingNodeData && props.onNodeConfigurationSave) {
        props.onNodeConfigurationSave(editingNodeData.nodeId, configuration, nodeName);
      }
      setEditingNodeData(null);
    },
    [editingNodeData, props],
  );

  const handleSaveNewNode = useCallback(
    (configuration: Record<string, any>, nodeName: string) => {
      if (newNodeData && props.onNodeAdd) {
        props.onNodeAdd({
          buildingBlock: newNodeData.buildingBlock,
          nodeName,
          configuration,
          position: newNodeData.position,
        });
      }
      setNewNodeData(null);
    },
    [newNodeData, props],
  );

  const handleToggleView = useCallback(
    (nodeId: string) => {
      state.toggleNodeCollapse(nodeId);
      props.onToggleView?.(nodeId);
    },
    [state.toggleNodeCollapse, props.onToggleView],
  );

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
              onZoomChange={setCanvasZoom}
              hasFitToViewRef={hasFitToViewRef}
              viewportRefProp={props.viewportRef}
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
            onCancelQueueItem={
              props.onCancelQueueItem && state.componentSidebar.selectedNodeId
                ? (id) => props.onCancelQueueItem!(state.componentSidebar.selectedNodeId!, id)
                : undefined
            }
            onRun={handleNodeRun}
            onDuplicate={props.onDuplicate}
            onDocs={props.onDocs}
            onConfigure={props.onConfigure}
            onDeactivate={props.onDeactivate}
            onDelete={handleNodeDelete}
            runDisabled={props.runDisabled}
            runDisabledTooltip={props.runDisabledTooltip}
          />
        </div>
      </div>

      {/* Edit existing node modal */}
      {editingNodeData && (
        <NodeConfigurationModal
          mode="edit"
          isOpen={true}
          onClose={() => setEditingNodeData(null)}
          nodeName={editingNodeData.nodeName}
          nodeLabel={editingNodeData.displayLabel}
          configuration={editingNodeData.configuration}
          configurationFields={editingNodeData.configurationFields}
          onSave={handleSaveConfiguration}
          domainId={props.organizationId}
          domainType="DOMAIN_TYPE_ORGANIZATION"
        />
      )}

      {/* Add new node modal */}
      {newNodeData && (
        <NodeConfigurationModal
          mode="create"
          isOpen={true}
          onClose={() => setNewNodeData(null)}
          nodeName={newNodeData.nodeName}
          nodeLabel={newNodeData.displayLabel}
          configuration={newNodeData.configuration}
          configurationFields={newNodeData.buildingBlock.configuration || []}
          onSave={handleSaveNewNode}
          domainId={props.organizationId}
          domainType="DOMAIN_TYPE_ORGANIZATION"
        />
      )}

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

function Sidebar({
  state,
  getSidebarData,
  loadSidebarData,
  getTabData,
  onCancelQueueItem,
  onRun,
  onDuplicate,
  onDocs,
  onConfigure,
  onDeactivate,
  onToggleView,
  onDelete,
  runDisabled,
  runDisabledTooltip,
}: {
  state: CanvasPageState;
  getSidebarData?: (nodeId: string) => SidebarData | null;
  loadSidebarData?: (nodeId: string) => void;
  getTabData?: (nodeId: string, event: SidebarEvent) => TabData | undefined;
  onCancelQueueItem?: (id: string) => void;
  onRun?: (nodeId: string) => void;
  onDuplicate?: (nodeId: string) => void;
  onDocs?: (nodeId: string) => void;
  onConfigure?: (nodeId: string) => void;
  onDeactivate?: (nodeId: string) => void;
  onToggleView?: (nodeId: string) => void;
  onDelete?: (nodeId: string) => void;
  runDisabled?: boolean;
  runDisabledTooltip?: string;
}) {
  const sidebarData = useMemo(() => {
    if (!state.componentSidebar.selectedNodeId || !getSidebarData) {
      return null;
    }
    return getSidebarData(state.componentSidebar.selectedNodeId);
  }, [state.componentSidebar.selectedNodeId, getSidebarData]);

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
  if (sidebarData.isLoading) {
    return (
      <div
        className="border-l-1 border-gray-200 border-border absolute right-0 top-0 h-full z-20 overflow-y-auto overflow-x-hidden bg-white shadow-2xl"
        style={{ width: "420px" }}
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
      isOpen={state.componentSidebar.isOpen}
      onClose={state.componentSidebar.close}
      latestEvents={latestEvents}
      nextInQueueEvents={nextInQueueEvents}
      metadata={sidebarData.metadata}
      title={sidebarData.title}
      iconSrc={sidebarData.iconSrc}
      iconSlug={sidebarData.iconSlug}
      iconColor={sidebarData.iconColor}
      iconBackground={sidebarData.iconBackground}
      moreInQueueCount={sidebarData.moreInQueueCount}
      hideQueueEvents={sidebarData.hideQueueEvents}
      getTabData={
        getTabData && state.componentSidebar.selectedNodeId
          ? (event) => getTabData(state.componentSidebar.selectedNodeId!, event)
          : undefined
      }
      onCancelQueueItem={onCancelQueueItem}
      onEventClick={(event) => {
        setLatestEvents((prev) => {
          return prev.map((e) => {
            if (e.id === event.id) {
              return { ...e, isOpen: !e.isOpen };
            }
            return e;
          });
        });
        setNextInQueueEvents((prev) => {
          return prev.map((e) => {
            if (e.title === event.title) {
              return { ...e, isOpen: !e.isOpen };
            }
            return e;
          });
        });
      }}
      onRun={onRun ? () => onRun(state.componentSidebar.selectedNodeId!) : undefined}
      runDisabled={runDisabled}
      runDisabledTooltip={runDisabledTooltip}
      onDuplicate={onDuplicate ? () => onDuplicate(state.componentSidebar.selectedNodeId!) : undefined}
      onDocs={onDocs ? () => onDocs(state.componentSidebar.selectedNodeId!) : undefined}
      onConfigure={onConfigure ? () => onConfigure(state.componentSidebar.selectedNodeId!) : undefined}
      onDeactivate={onDeactivate ? () => onDeactivate(state.componentSidebar.selectedNodeId!) : undefined}
      onToggleView={onToggleView ? () => onToggleView(state.componentSidebar.selectedNodeId!) : undefined}
      onDelete={onDelete ? () => onDelete(state.componentSidebar.selectedNodeId!) : undefined}
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
  onZoomChange,
  hasFitToViewRef,
  viewportRefProp,
  runDisabled,
  runDisabledTooltip,
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
  onZoomChange?: (zoom: number) => void;
  hasFitToViewRef: React.MutableRefObject<boolean>;
  viewportRefProp?: React.MutableRefObject<{ x: number; y: number; zoom: number } | undefined>;
  runDisabled?: boolean;
  runDisabledTooltip?: string;
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

  const handleNodeClick = useCallback((nodeId: string) => {
    stateRef.current.componentSidebar.open(nodeId);
  }, []);

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
    stateRef.current.componentSidebar.close();
  }, []);

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
  const nodesWithCallbacks = useMemo(() => {
    return state.nodes.map((node) => ({
      ...node,
      data: {
        ...node.data,
        _callbacksRef: callbacksRef,
      },
    }));
  }, [state.nodes]);

  const edgeTypes = useMemo(
    () => ({
      custom: CustomEdge,
    }),
    [],
  );
  const styledEdges = useMemo(() => state.edges?.map((e) => ({ ...e, ...EDGE_STYLE })), [state.edges]);

  return (
    <>
      {/* Header */}
      {!hideHeader && <Header breadcrumbs={state.breadcrumbs} onSave={onSave ? handleSave : undefined} />}

      {/* Toggle button */}
      <div className={`absolute ${hideHeader ? "top-2" : "top-14"} left-1/2 transform -translate-x-1/2 z-10`}>
        <ViewToggle isCollapsed={state.isCollapsed} onToggle={handleToggleCollapse} />
      </div>

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
            onDragOver={handleDragOver}
            onDrop={handleDrop}
            onMove={handleMove}
            onInit={handleInit}
            onPaneClick={handlePaneClick}
            defaultViewport={viewport}
            fitView={false}
            style={{ opacity: isInitialized ? 1 : 0 }}
          >
            <Background bgColor="#F1F5F9" color="#F1F5F9" />
          </ReactFlow>
        </div>
      </div>
    </>
  );
}

export type { BuildingBlock } from "../BuildingBlocksSidebar";
export { CanvasPage };
