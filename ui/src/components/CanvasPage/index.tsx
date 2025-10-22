import "reactflow/dist/style.css";

import ReactFlow, {
  Background,
  BackgroundVariant,
  type Edge,
  type Node,
} from "reactflow";
import { useCanvasState } from "./useCanvasState";

namespace CanvasPage {
  export interface Props {
    nodes?: Node[];
    edges?: Edge[];
  }
}

function CanvasPage(props: CanvasPage.Props) {
  const { nodes, edges, onNodesChange, onEdgesChange } = useCanvasState(props);

  return (
    <div className="h-[100vh] w-full overflow-hidden">
      <ReactFlow
        nodes={nodes}
        edges={edges}
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
