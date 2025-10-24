import {
  Background,
  EdgeMarker,
  ReactFlow,
  type Edge as ReactFlowEdge,
  type Node as ReactFlowNode,
} from "@xyflow/react";

import { Block } from "./Block";
import { useCanvasState } from "./useCanvasState";
import { ViewToggle } from "../ViewToggle";
import { Header, type BreadcrumbItem } from "./Header";

namespace CanvasPage {
  export type Node = ReactFlowNode;
  export type Edge = ReactFlowEdge;

  export interface Props {
    nodes?: Node[];
    edges?: Edge[];
    startCollapsed?: boolean;
    title?: string;
    breadcrumbs?: BreadcrumbItem[];
    onNodeExpand?: (nodeId: string, nodeData: any) => void;
  }
}

const EDGE_STYLE = {
  type: "bezier" as const,
  style: { stroke: "#C9D5E1", strokeWidth: 3 },
  markerEnd: {
    width: 20,
    height: 20,
    color: "#6B7280",
  } as EdgeMarker,
} as const;

function CanvasPage(props: CanvasPage.Props) {
  const { nodes, edges, onNodesChange, onEdgesChange, isCollapsed, toggleCollapse, toggleNodeCollapse } = useCanvasState(props);

  const defaultBreadcrumbs: BreadcrumbItem[] = [
    { label: "Workflows" },
    { label: props.title || "Untitled Workflow" }
  ];

  const breadcrumbs = props.breadcrumbs || defaultBreadcrumbs;

  const handleNodeExpand = (nodeId: string) => {
    const node = nodes?.find(n => n.id === nodeId);
    if (node && props.onNodeExpand) {
      props.onNodeExpand(nodeId, node.data);
    }
  };

  return (
    <div className="h-[100vh] w-[100vw] overflow-hidden sp-canvas relative">
      {/* Header */}
      <Header breadcrumbs={breadcrumbs} />

      {/* Toggle button */}
      <div className="absolute top-14 left-1/2 transform -translate-x-1/2 z-10">
        <ViewToggle isCollapsed={isCollapsed} onToggle={toggleCollapse} />
      </div>

      <div className="pt-12 h-full">
        <ReactFlow
          nodes={nodes}
          edges={edges?.map((e) => ({ ...e, ...EDGE_STYLE }))}
          nodeTypes={{
            default: (nodeProps) => (
              <Block
                data={nodeProps.data}
                onExpand={handleNodeExpand}
                nodeId={nodeProps.id}
              />
            )
          }}
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
  );
}

export { CanvasPage };
