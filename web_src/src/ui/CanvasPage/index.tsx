import {
  Background,
  ReactFlow,
  ReactFlowProvider,
  useReactFlow,
  type Edge as ReactFlowEdge,
  type Node as ReactFlowNode,
} from "@xyflow/react";

import { useCallback, useMemo } from "react";

import { AiSidebar } from "../ai";
import { ViewToggle } from "../ViewToggle";
import { Block, BlockData } from "./Block";
import { Header, type BreadcrumbItem } from "./Header";
import { Simulation } from "./storybooks/useSimulation";
import { useCanvasState } from "./useCanvasState";
import "./canvas-reset.css";

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
}

const EDGE_STYLE = {
  type: "default",
  style: { stroke: "#C9D5E1", strokeWidth: 3 },
} as const;

function CanvasContent(props: CanvasPageProps) {
  const {
    nodes,
    edges,
    onNodesChange,
    onEdgesChange,
    isCollapsed,
    toggleCollapse,
    toggleNodeCollapse,
  } = useCanvasState(props);

  const { fitView } = useReactFlow();
  const { onNodeExpand, title, breadcrumbs: propsBreadcrumbs } = props;

  const defaultBreadcrumbs: BreadcrumbItem[] = [
    { label: "Workflows" },
    { label: title || "Untitled Workflow" },
  ];

  const breadcrumbs = propsBreadcrumbs || defaultBreadcrumbs;

  const handleNodeExpand = useCallback(
    (nodeId: string) => {
      const node = nodes?.find((n) => n.id === nodeId);
      if (node && onNodeExpand) {
        onNodeExpand(nodeId, node.data);
        fitView();
      }
    },
    [nodes, onNodeExpand, fitView]
  );

  const nodeTypes = useMemo(
    () => ({
      default: (nodeProps: { data: unknown; id: string }) => (
        <Block
          data={nodeProps.data as BlockData}
          onExpand={handleNodeExpand}
          nodeId={nodeProps.id}
        />
      ),
      // eslint-disable-next-line react-hooks/exhaustive-deps
    }),
    []
  );

  const edgeTypes = useMemo(() => ({}), []);
  const styledEdges = useMemo(
    () => edges?.map((e) => ({ ...e, ...EDGE_STYLE })),
    [edges]
  );

  return (
    <>
      {/* Header */}
      <Header breadcrumbs={breadcrumbs} />

      {/* Toggle button */}
      <div className="absolute top-14 left-1/2 transform -translate-x-1/2 z-10">
        <ViewToggle isCollapsed={isCollapsed} onToggle={toggleCollapse} />
      </div>

      <div className="pt-12 h-full">
        <div className="h-full w-full">
          <ReactFlow
            nodes={nodes}
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
            onNodesChange={onNodesChange}
            onEdgesChange={onEdgesChange}
            onNodeDoubleClick={(_, node) => toggleNodeCollapse(node.id)}
          >
            <Background bgColor="#F1F5F9" color="#F1F5F9" />
          </ReactFlow>
        </div>
      </div>
    </>
  );
}

function CanvasPage(props: CanvasPageProps) {
  return (
    <div className="h-[100vh] w-[100vw] overflow-hidden sp-canvas relative">
      <ReactFlowProvider>
        <CanvasContent {...props} />
      </ReactFlowProvider>

      <AiSidebar />
    </div>
  );
}

export { CanvasPage };
