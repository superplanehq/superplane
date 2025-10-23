import {
  Background,
  BackgroundVariant,
  MarkerType,
  ReactFlow,
  type Edge as ReactFlowEdge,
  type Node as ReactFlowNode,
} from "@xyflow/react";

import { Block } from "./Block";
import { useCanvasState } from "./useCanvasState";

namespace CanvasPage {
  export type Node = ReactFlowNode;
  export type Edge = ReactFlowEdge;

  export interface Props {
    nodes?: Node[];
    edges?: Edge[];
  }
}

const EDGE_STYLE = {
  type: "bezier" as const,
  style: { stroke: "#C9D5E1", strokeWidth: 2 },
  markerEnd: {
    width: 20,
    height: 20,
    color: "#6B7280",
  },
} as const;

function CanvasPage(props: CanvasPage.Props) {
  const { nodes, edges, onNodesChange, onEdgesChange } = useCanvasState(props);

  return (
    <div className="h-[100vh] w-[100vw] overflow-hidden sp-canvas">
      <ReactFlow
        nodes={nodes}
        edges={edges?.map((e) => ({ ...e, ...EDGE_STYLE }))}
        nodeTypes={{ default: Block }}
        fitView={true}
        minZoom={0.4}
        maxZoom={1.5}
        zoomOnScroll={true}
        zoomOnPinch={true}
        zoomOnDoubleClick={true}
        panOnScroll={true}
        panOnDrag={true}
        selectionOnDrag={false}
        panOnScrollSpeed={0.8}
        nodesDraggable={true}
        nodesConnectable={false}
        elementsSelectable={true}
        onNodesChange={onNodesChange}
        onEdgesChange={onEdgesChange}
      >
        <Background variant={BackgroundVariant.Dots} gap={24} size={1} />
      </ReactFlow>
    </div>
  );
}

export { CanvasPage };
