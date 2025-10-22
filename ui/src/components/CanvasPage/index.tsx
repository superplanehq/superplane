import {
  ReactFlow,
  Background,
  BackgroundVariant,
  MarkerType,
  type Edge,
  type Node,
} from "@xyflow/react";

import { Block } from "./Block";
import { useCanvasState } from "./useCanvasState";

namespace CanvasPage {
  export interface Props {
    nodes?: Node[];
    edges?: Edge[];
    nodeTypes?: Record<string, React.ComponentType<any>>;
  }
}

const EDGE_STYLE = {
  type: "smoothstep" as const,
  style: { stroke: "#9AA5B1", strokeWidth: 1 },
  markerEnd: {
    type: MarkerType.Arrow,
    width: 0,
    height: 0,
    color: "#9AA5B1",
  },
} as const;

function CanvasPage(props: CanvasPage.Props) {
  const { nodes, edges, onNodesChange, onEdgesChange } = useCanvasState(props);

  return (
    <div className="h-[100vh] w-full overflow-hidden">
      <ReactFlow
        nodes={nodes}
        edges={edges?.map((e) => ({ ...e, ...EDGE_STYLE }))}
        nodeTypes={props.nodeTypes ?? { default: Block }}
        fitView={true}
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
