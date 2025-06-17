import React from "react";
import { ReactFlow, Background } from "@xyflow/react";
import '@xyflow/react/dist/style.css';

import StageNode from './nodes/stage';
import GithubIntegration from './nodes/event_source';
import { FlowDevTools } from './devtools';
import { useCanvasStore } from "../store/canvasStore";
import { useFlowHandlers } from "../hooks/useFlowHandlers";
import { useAutoLayout } from "../hooks/useAutoLayout";
import { FlowControls } from "./FlowControls";
import { ConnectionStatus } from "./ConnectionStatus";

export const nodeTypes = {
  deploymentCard: StageNode,
  githubIntegration: GithubIntegration,
};

export const FlowRenderer: React.FC = () => {
  const nodes = useCanvasStore((state) => state.nodes);
  const edges = useCanvasStore((state) => state.edges);
  const onNodesChange = useCanvasStore((state) => state.onNodesChange);
  const onEdgesChange = useCanvasStore((state) => state.onEdgesChange);
  const onConnect = useCanvasStore((state) => state.onConnect);

  const { applyElkAutoLayout } = useAutoLayout();
  const { onNodeDragStop, onInit } = useFlowHandlers();
 
  return (
    <div style={{ width: "100vw", height: "100vh", minWidth: 0, minHeight: 0 }}>
      <ReactFlow
        nodes={nodes}
        edges={edges}
        nodeTypes={nodeTypes}
        onNodesChange={onNodesChange}
        onEdgesChange={onEdgesChange}
        onConnect={onConnect}
        onNodeDragStop={onNodeDragStop}
        onInit={onInit}
        fitView
        minZoom={0.4}
        maxZoom={1.5}
        colorMode="light"
      >
        <FlowControls
          onAutoLayout={applyElkAutoLayout}
          nodes={nodes}
          edges={edges}
        />
        <Background />
        <FlowDevTools />
        <ConnectionStatus />
      </ReactFlow>
    </div>
  );
};