import {
  Background,
  ReactFlow,
  ReactFlowProvider,
  useReactFlow,
  type Edge as ReactFlowEdge,
  type Node as ReactFlowNode,
} from "@xyflow/react";

import { useCallback, useMemo, useRef } from "react";

import { AiSidebar } from "../ai";
import { ComponentSidebar } from "../componentSidebar";
import { ViewToggle } from "../ViewToggle";
import { Block, BlockData } from "./Block";
import "./canvas-reset.css";
import { Header, type BreadcrumbItem } from "./Header";
import { Simulation } from "./storybooks/useSimulation";
import { CanvasPageState, useCanvasState } from "./useCanvasState";
import type { ChildEventsInfo } from "../childEvents";
import type { MetadataItem } from "../metadataList";

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
}

export interface CanvasNode extends ReactFlowNode {
  __simulation?: Simulation;
}

export interface CanvasEdge extends ReactFlowEdge {}

export interface CanvasPageProps {
  nodes: CanvasNode[];
  edges: CanvasEdge[];

  startCollapsed?: boolean;
  title?: string;
  breadcrumbs?: BreadcrumbItem[];

  onNodeExpand?: (nodeId: string, nodeData: unknown) => void;
  getSidebarData?: (nodeId: string) => SidebarData | null;
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
        <CanvasContent state={state} />
      </ReactFlowProvider>

      <AiSidebar />
      <Sidebar state={state} getSidebarData={props.getSidebarData} />
    </div>
  );
}

function Sidebar({
  state,
  getSidebarData
}: {
  state: CanvasPageState;
  getSidebarData?: (nodeId: string) => SidebarData | null;
}) {
  const sidebarData = useMemo(() => {
    if (!state.componentSidebar.selectedNodeId || !getSidebarData) {
      return null;
    }
    return getSidebarData(state.componentSidebar.selectedNodeId);
  }, [state.componentSidebar.selectedNodeId, getSidebarData]);

  if (!sidebarData) {
    return null;
  }

  return (
    <ComponentSidebar
      isOpen={state.componentSidebar.isOpen}
      onClose={state.componentSidebar.close}
      latestEvents={sidebarData.latestEvents}
      nextInQueueEvents={sidebarData.nextInQueueEvents}
      metadata={sidebarData.metadata}
      title={sidebarData.title}
      iconSrc={sidebarData.iconSrc}
      iconSlug={sidebarData.iconSlug}
      iconColor={sidebarData.iconColor}
      iconBackground={sidebarData.iconBackground}
      moreInQueueCount={sidebarData.moreInQueueCount}
    />
  );
}

function CanvasContent({ state }: { state: CanvasPageState }) {
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
      default: (nodeProps: { data: unknown; id: string }) => (
        <Block
          data={nodeProps.data as BlockData}
          onExpand={handleNodeExpand}
          nodeId={nodeProps.id}
          onClick={() => handleNodeClick(nodeProps.id)}
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
