import {
  Background,
  ReactFlow,
  ReactFlowProvider,
  useReactFlow,
  type Edge as ReactFlowEdge,
  type Node as ReactFlowNode,
} from "@xyflow/react";

import { useCallback, useEffect, useMemo, useRef, useState } from "react";

import { ComponentsConfigurationField } from "@/api-client";
import { AiSidebar } from "../ai";
import { BuildingBlock, BuildingBlockCategory, BuildingBlocksSidebar } from "../BuildingBlocksSidebar";
import type { ChildEventsInfo } from "../childEvents";
import { ComponentSidebar } from "../componentSidebar";
import { EmitEventModal } from "../EmitEventModal";
import type { MetadataItem } from "../metadataList";
import { ViewToggle } from "../ViewToggle";
import { Block, BlockData } from "./Block";
import "./canvas-reset.css";
import { Header, type BreadcrumbItem } from "./Header";
import { NodeConfigurationModal } from "./NodeConfigurationModal";
import { Simulation } from "./storybooks/useSimulation";
import { CanvasPageState, useCanvasState } from "./useCanvasState";

export interface SidebarEvent {
  title: string;
  subtitle?: string;
  state: "processed" | "discarded" | "waiting";
  isOpen: boolean;
  receivedAt?: Date;
  values?: Record<string, string>;
  childEventsInfo?: ChildEventsInfo;
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
  configuration: Record<string, any>;
  configurationFields: ComponentsConfigurationField[];
}

export interface NewNodeData {
  buildingBlock: BuildingBlock;
  nodeName: string;
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

  onNodeExpand?: (nodeId: string, nodeData: unknown) => void;
  getSidebarData?: (nodeId: string) => SidebarData | null;
  getNodeEditData?: (nodeId: string) => NodeEditData | null;
  onNodeConfigurationSave?: (nodeId: string, configuration: Record<string, any>, nodeName: string) => void;
  onSave?: (nodes: CanvasNode[]) => void;
  onDelete?: () => void;
  onEdgeCreate?: (sourceId: string, targetId: string, sourceHandle?: string | null) => void;
  onNodeDelete?: (nodeId: string) => void;
  onEdgeDelete?: (edgeIds: string[]) => void;
  onNodePositionChange?: (nodeId: string, position: { x: number; y: number }) => void;

  onRun?: (nodeId: string, channel: string, data: any) => void | Promise<void>;
  onDuplicate?: (nodeId: string) => void;
  onDocs?: (nodeId: string) => void;
  onEdit?: (nodeId: string) => void;
  onDeactivate?: (nodeId: string) => void;
  onToggleView?: (nodeId: string) => void;

  ai?: AiProps;

  // Building blocks for adding new nodes
  buildingBlocks: BuildingBlockCategory[];
  onNodeAdd?: (newNodeData: NewNodeData) => void;

  // Refs to persist state across re-renders
  hasFitToViewRef?: React.MutableRefObject<boolean>;
  hasUserToggledSidebarRef?: React.MutableRefObject<boolean>;
  isSidebarOpenRef?: React.MutableRefObject<boolean | null>;
  viewportRef?: React.MutableRefObject<{ x: number; y: number; zoom: number } | undefined>;
}

const EDGE_STYLE = {
  type: "default",
  style: { stroke: "#C9D5E1", strokeWidth: 3 },
} as const;

function CanvasPage(props: CanvasPageProps) {
  const state = useCanvasState(props);
  const [editingNodeData, setEditingNodeData] = useState<NodeEditData | null>(null);
  const [newNodeData, setNewNodeData] = useState<NewNodeData | null>(null);

  // Use refs from props if provided, otherwise create local ones
  const hasFitToViewRef = props.hasFitToViewRef || useRef(false);
  const hasUserToggledSidebarRef = props.hasUserToggledSidebarRef || useRef(false);
  const isSidebarOpenRef = props.isSidebarOpenRef || useRef<boolean | null>(null);

  // Initialize sidebar state from ref if available, otherwise based on whether nodes exist
  const [isBuildingBlocksSidebarOpen, setIsBuildingBlocksSidebarOpen] = useState(() => {
    // If we have a persisted state in the ref, use it
    if (isSidebarOpenRef.current !== null) {
      return isSidebarOpenRef.current;
    }
    // Otherwise, open if no nodes exist
    return props.nodes.length === 0;
  });

  const [canvasZoom, setCanvasZoom] = useState(1);
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
    [state.nodes],
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
      configuration: {},
      position,
    });
  }, []);

  const handleSidebarToggle = useCallback(
    (open: boolean) => {
      hasUserToggledSidebarRef.current = true;
      isSidebarOpenRef.current = open;
      setIsBuildingBlocksSidebarOpen(open);
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

  return (
    <div className="h-[100vh] w-[100vw] overflow-hidden sp-canvas relative flex flex-col">
      {/* Header at the top spanning full width */}
      <div className="relative z-20">
        <CanvasContentHeader state={state} onSave={props.onSave} onDelete={props.onDelete} organizationId={props.organizationId} />
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
          <ReactFlowProvider key="canvas-flow-provider">
            <CanvasContent
              state={state}
              onSave={props.onSave}
              onNodeEdit={handleNodeEdit}
              onNodeDelete={handleNodeDelete}
              onEdgeCreate={props.onEdgeCreate}
              hideHeader={true}
              onToggleView={props.onToggleView}
              onRun={handleNodeRun}
              onDuplicate={props.onDuplicate}
              onDeactivate={props.onDeactivate}
              onBuildingBlockDrop={handleBuildingBlockDrop}
              onZoomChange={setCanvasZoom}
              hasFitToViewRef={hasFitToViewRef}
              viewportRefProp={props.viewportRef}
            />
          </ReactFlowProvider>

          <AiSidebar
            isOpen={state.ai.sidebarOpen}
            setIsOpen={state.ai.setSidebarOpen}
            showNotifications={state.ai.showNotifications}
            notificationMessage={state.ai.notificationMessage}
          />

          <Sidebar
            state={state}
            getSidebarData={props.getSidebarData}
            onRun={handleNodeRun}
            onDuplicate={props.onDuplicate}
            onDocs={props.onDocs}
            onDeactivate={props.onDeactivate}
            onDelete={handleNodeDelete}
          />
        </div>
      </div>

      {/* Edit existing node modal */}
      {editingNodeData && (
        <NodeConfigurationModal
          isOpen={true}
          onClose={() => setEditingNodeData(null)}
          nodeName={editingNodeData.nodeName}
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
          isOpen={true}
          onClose={() => setNewNodeData(null)}
          nodeName={newNodeData.nodeName}
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
  onRun,
  onDuplicate,
  onDocs,
  onDeactivate,
  onToggleView,
  onDelete,
}: {
  state: CanvasPageState;
  getSidebarData?: (nodeId: string) => SidebarData | null;
  onRun?: (nodeId: string) => void;
  onDuplicate?: (nodeId: string) => void;
  onDocs?: (nodeId: string) => void;
  onDeactivate?: (nodeId: string) => void;
  onToggleView?: (nodeId: string) => void;
  onDelete?: (nodeId: string) => void;
}) {
  const sidebarData = useMemo(() => {
    if (!state.componentSidebar.selectedNodeId || !getSidebarData) {
      return null;
    }
    return getSidebarData(state.componentSidebar.selectedNodeId);
  }, [state.componentSidebar.selectedNodeId, getSidebarData]);

  const [latestEvents, setLatestEvents] = useState<SidebarEvent[]>(sidebarData?.latestEvents || []);
  const [nextInQueueEvents, setNextInQueueEvents] = useState<SidebarEvent[]>(sidebarData?.nextInQueueEvents || []);

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
      onEventClick={(event) => {
        setLatestEvents((prev) => {
          return prev.map((e) => {
            if (e.title === event.title) {
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
      onDuplicate={onDuplicate ? () => onDuplicate(state.componentSidebar.selectedNodeId!) : undefined}
      onDocs={onDocs ? () => onDocs(state.componentSidebar.selectedNodeId!) : undefined}
      onDeactivate={onDeactivate ? () => onDeactivate(state.componentSidebar.selectedNodeId!) : undefined}
      onToggleView={onToggleView ? () => onToggleView(state.componentSidebar.selectedNodeId!) : undefined}
      onDelete={onDelete ? () => onDelete(state.componentSidebar.selectedNodeId!) : undefined}
    />
  );
}

function CanvasContentHeader({ state, onSave, onDelete, organizationId }: { state: CanvasPageState; onSave?: (nodes: CanvasNode[]) => void; onDelete?: () => void; organizationId?: string }) {
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

  return <Header breadcrumbs={state.breadcrumbs} onSave={onSave ? handleSave : undefined} onDelete={onDelete} onLogoClick={organizationId ? handleLogoClick : undefined} />;
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
  onDeactivate,
  onToggleView,
  onBuildingBlockDrop,
  onZoomChange,
  hasFitToViewRef,
  viewportRefProp,
}: {
  state: CanvasPageState;
  onSave?: (nodes: CanvasNode[]) => void;
  onNodeEdit: (nodeId: string) => void;
  onNodeDelete?: (nodeId: string) => void;
  onEdgeCreate?: (sourceId: string, targetId: string, sourceHandle?: string | null) => void;
  hideHeader?: boolean;
  onRun?: (nodeId: string) => void;
  onDuplicate?: (nodeId: string) => void;
  onDeactivate?: (nodeId: string) => void;
  onToggleView?: (nodeId: string) => void;
  onDelete?: (nodeId: string) => void;
  onBuildingBlockDrop?: (block: BuildingBlock, position?: { x: number; y: number }) => void;
  onZoomChange?: (zoom: number) => void;
  hasFitToViewRef: React.MutableRefObject<boolean>;
  viewportRefProp?: React.MutableRefObject<{ x: number; y: number; zoom: number } | undefined>;
}) {
  const { fitView, screenToFlowPosition, getViewport } = useReactFlow();

  // Use refs to avoid recreating callbacks when state changes
  const stateRef = useRef(state);
  stateRef.current = state;

  // Use viewport ref from props if provided, otherwise create local one
  const viewportRef = viewportRefProp || useRef<{ x: number; y: number; zoom: number } | undefined>(undefined);

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

  // Handle fit to view on ReactFlow initialization
  const handleInit = useCallback(
    (reactFlowInstance: any) => {
      if (!hasFitToViewRef.current) {
        // Fit to view but don't zoom in too much (max zoom of 1.0)
        fitView({ maxZoom: 1.0, padding: 0.5 });

        // Store the initial viewport after fit
        const initialViewport = getViewport();
        viewportRef.current = initialViewport;

        if (onZoomChange) {
          onZoomChange(initialViewport.zoom);
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

  const nodeTypes = useMemo(
    () => ({
      default: (nodeProps: { data: unknown; id: string; selected?: boolean }) => (
        <Block
          data={nodeProps.data as BlockData}
          onExpand={handleNodeExpand}
          nodeId={nodeProps.id}
          onClick={() => handleNodeClick(nodeProps.id)}
          onEdit={() => onNodeEdit(nodeProps.id)}
          onDelete={() => onNodeDelete?.(nodeProps.id)}
          selected={nodeProps.selected}
          onRun={onRunRef.current ? () => onRunRef.current?.(nodeProps.id) : undefined}
          onDuplicate={onDuplicate ? () => onDuplicate(nodeProps.id) : undefined}
          onDeactivate={onDeactivate ? () => onDeactivate(nodeProps.id) : undefined}
          onToggleView={onToggleView ? () => onToggleView(nodeProps.id) : undefined}
          ai={{
            show: state.ai.sidebarOpen,
            suggestion: state.ai.suggestions[nodeProps.id] || null,
            onApply: () => state.ai.onApply(nodeProps.id),
            onDismiss: () => state.ai.onDismiss(nodeProps.id),
          }}
        />
      ),
    }),
    [handleNodeExpand, handleNodeClick, onNodeEdit, state.ai, onDuplicate, onDeactivate, onToggleView, onNodeDelete],
  );

  const edgeTypes = useMemo(() => ({}), []);
  const styledEdges = useMemo(() => state.edges?.map((e) => ({ ...e, ...EDGE_STYLE })), [state.edges]);

  return (
    <>
      {/* Header */}
      {!hideHeader && <Header breadcrumbs={state.breadcrumbs} onSave={onSave ? handleSave : undefined} />}

      {/* Toggle button */}
      <div className={`absolute ${hideHeader ? "top-2" : "top-14"} left-1/2 transform -translate-x-1/2 z-10`}>
        <ViewToggle isCollapsed={state.isCollapsed} onToggle={state.toggleCollapse} />
      </div>

      <div className={hideHeader ? "h-full" : "pt-12 h-full"}>
        <div className="h-full w-full">
          <ReactFlow
            nodes={state.nodes}
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
            onNodeDoubleClick={(_, node) => state.toggleNodeCollapse(node.id)}
            onDragOver={handleDragOver}
            onDrop={handleDrop}
            onMove={handleMove}
            onInit={handleInit}
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
