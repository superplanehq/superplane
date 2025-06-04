import React, { useEffect } from "react";
import { ReactFlow, Background } from "@xyflow/react";
import '@xyflow/react/dist/style.css';

import StageNode from './nodes/stage';
import GithubIntegration from './nodes/event_source';
import { FlowDevTools } from './devtools';
import { useFlowStore } from "../store/flowStore";
import { useFlowHandlers } from "../hooks/useFlowHandlers";
import { useFlowData } from "../hooks/useFlowData";
import { useAutoLayout } from "../hooks/useAutoLayout";
import { FlowControls } from "./FlowControls";
import { ConnectionStatus } from "./ConnectionStatus";

export const nodeTypes = {
  deploymentCard: StageNode,
  githubIntegration: GithubIntegration,
};

export const FlowRenderer: React.FC = () => {
  const { 
    nodes, 
    edges, 
    setEdges,
    onNodesChange,
    onEdgesChange,
    onConnect
  } = useFlowStore();

  const { layoutedNodes, flowEdges } = useFlowData();
  const { applyElkAutoLayout } = useAutoLayout();
  const { onNodeDragStop, onInit } = useFlowHandlers();
  const [firstAutoLayout, setFirstAutoLayout] = React.useState(true);
  
  useEffect(() => {
    if (firstAutoLayout) {
      setFirstAutoLayout(false);
      setEdges(flowEdges);
      applyElkAutoLayout(layoutedNodes, flowEdges);
    }
  }, [applyElkAutoLayout, setEdges, layoutedNodes, flowEdges, firstAutoLayout]);

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
          nodes={layoutedNodes}
          edges={flowEdges}
        />
        <Background />
        <FlowDevTools />
        <ConnectionStatus />
      </ReactFlow>
    </div>
  );
};