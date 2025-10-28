import {
  Background,
  ReactFlow,
  ReactFlowProvider,
  useReactFlow,
  type Edge as ReactFlowEdge,
  type Node as ReactFlowNode,
} from "@xyflow/react";

import { useCallback, useEffect, useMemo, useRef, useState } from "react";

import { AiSidebar } from "../ai";
import type { ChildEventsInfo } from "../childEvents";
import { ComponentSidebar } from "../componentSidebar";
import type { MetadataItem } from "../metadataList";
import { ViewToggle } from "../ViewToggle";
import { Block, BlockData } from "./Block";
import "./canvas-reset.css";
import { Header, type BreadcrumbItem } from "./Header";
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

export interface CanvasEdge extends ReactFlowEdge { }

export interface CanvasPageProps {
  nodes: CanvasNode[];
  edges: CanvasEdge[];

  startCollapsed?: boolean;
  title?: string;
  breadcrumbs?: BreadcrumbItem[];

  onNodeExpand?: (nodeId: string, nodeData: unknown) => void;
  getSidebarData?: (nodeId: string) => SidebarData | null;

  // Sidebar action handlers
  onRun?: (nodeId: string) => void;
  onDuplicate?: (nodeId: string) => void;
  onDocs?: (nodeId: string) => void;
  onDeactivate?: (nodeId: string) => void;
  onToggleView?: (nodeId: string) => void;
  onDelete?: (nodeId: string) => void;

  aiSidebar?: {
    showNotifications: boolean;
    notificationMessage?: string;
  };
}

const EDGE_STYLE = {
  type: "default",
  style: { stroke: "#C9D5E1", strokeWidth: 3 },
} as const;

function CanvasPage(props: CanvasPageProps) {
  const state = useCanvasState(props);

  return (
    <div className="h-[100vh] w-[100vw] overflow-hidden sp-canvas relative">
      <ReactFlowProvider>
        <CanvasContent state={state} props={props} />
      </ReactFlowProvider>

      <AiSidebar
        showNotifications={state.aiSidebar.showNotifications}
        notificationMessage={state.aiSidebar.notificationMessage}
      />

      <Sidebar
        state={state}
        getSidebarData={props.getSidebarData}
        onRun={props.onRun}
        onDuplicate={props.onDuplicate}
        onDocs={props.onDocs}
        onDeactivate={props.onDeactivate}
        onDelete={props.onDelete}
      />
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

  const [latestEvents, setLatestEvents] = useState<SidebarEvent[]>(
    sidebarData?.latestEvents || []
  );
  const [nextInQueueEvents, setNextInQueueEvents] = useState<SidebarEvent[]>(
    sidebarData?.nextInQueueEvents || []
  );

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

function CanvasContent({ state, props }: { state: CanvasPageState; props: CanvasPageProps }) {
  const { fitView } = useReactFlow();

  // Use refs to avoid recreating callbacks when state changes
  const stateRef = useRef(state);
  stateRef.current = state;

  const handleNodeExpand = useCallback(
    (nodeId: string) => {
      const node = stateRef.current.nodes?.find((n) => n.id === nodeId);
      if (node && stateRef.current.onNodeExpand) {
        stateRef.current.onNodeExpand(nodeId, node.data);
        fitView();
      }
    },
    [fitView]
  );

  const handleNodeClick = useCallback((nodeId: string) => {
    stateRef.current.componentSidebar.open(nodeId);
  }, []);

  const nodeTypes = useMemo(
    () => ({
      default: (nodeProps: {
        data: unknown;
        id: string;
        selected?: boolean;
      }) => (
        <Block
          data={nodeProps.data as BlockData}
          onExpand={handleNodeExpand}
          nodeId={nodeProps.id}
          onClick={() => handleNodeClick(nodeProps.id)}
          selected={nodeProps.selected}
          onRun={props.onRun ? () => props.onRun!(nodeProps.id) : undefined}
          onDuplicate={props.onDuplicate ? () => props.onDuplicate!(nodeProps.id) : undefined}
          onDeactivate={props.onDeactivate ? () => props.onDeactivate!(nodeProps.id) : undefined}
          onToggleView={props.onToggleView ? () => props.onToggleView!(nodeProps.id) : undefined}
          onDelete={props.onDelete ? () => props.onDelete!(nodeProps.id) : undefined}
        />
      ),
    }),
    [handleNodeExpand, handleNodeClick]
  );

  const edgeTypes = useMemo(() => ({}), []);
  const styledEdges = useMemo(
    () => state.edges?.map((e) => ({ ...e, ...EDGE_STYLE })),
    [state.edges]
  );

  return (
    <>
      {/* Header */}
      <Header breadcrumbs={state.breadcrumbs} />

      {/* Toggle button */}
      <div className="absolute top-14 left-1/2 transform -translate-x-1/2 z-10">
        <ViewToggle
          isCollapsed={state.isCollapsed}
          onToggle={state.toggleCollapse}
        />
      </div>

      <div className="pt-12 h-full">
        <div className="h-full w-full">
          <ReactFlow
            nodes={state.nodes}
            edges={styledEdges}
            nodeTypes={nodeTypes}
            edgeTypes={edgeTypes}
            fitView={true}
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
            nodesConnectable={false}
            elementsSelectable={true}
            onNodesChange={state.onNodesChange}
            onEdgesChange={state.onEdgesChange}
            onNodeDoubleClick={(_, node) => state.toggleNodeCollapse(node.id)}
          >
            <Background bgColor="#F1F5F9" color="#F1F5F9" />
          </ReactFlow>
        </div>
      </div>
    </>
  );
}

export { CanvasPage };
