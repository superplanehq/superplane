import "reactflow/dist/style.css";

import ReactFlow, {
  Background,
  BackgroundVariant,
  MarkerType,
  type Edge,
  type Node,
} from "reactflow";

import { DefaultBlock, InputBlock, OutputBlock } from "./Block";
import { useCanvasState } from "./useCanvasState";

namespace CanvasPage {
  export interface Props {
    nodes?: Node[];
    edges?: Edge[];
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
};

function CanvasPage(props: CanvasPage.Props) {
  const { nodes, edges, onNodesChange, onEdgesChange } = useCanvasState(props);

  return (
    <div className="h-[100vh] w-full overflow-hidden">
      <ReactFlow
        nodes={nodes}
        edges={edges?.map((e) => ({ ...e, ...EDGE_STYLE }))}
        nodeTypes={{
          default: DefaultBlock,
          input: InputBlock,
          output: OutputBlock,
        }}
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
