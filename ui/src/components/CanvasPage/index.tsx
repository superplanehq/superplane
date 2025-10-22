import "reactflow/dist/style.css";

import ReactFlow, {
  Background,
  BackgroundVariant,
  type Edge,
  type Node,
} from "reactflow";

namespace CanvasPage {
  export interface Props {
    nodes?: Node[];
    edges?: Edge[];
  }
}

function CanvasPage({ nodes, edges }: CanvasPage.Props) {
  return (
    <div className="h-[100vh] w-full overflow-hidden">
      <ReactFlow nodes={nodes} edges={edges} fitView={true}>
        <Background variant={BackgroundVariant.Dots} gap={24} size={1} />
      </ReactFlow>
    </div>
  );
}

export { CanvasPage };
