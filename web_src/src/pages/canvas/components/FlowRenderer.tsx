import React, { useEffect, useRef } from "react";
import { ReactFlow, Background } from "@xyflow/react";
import '@xyflow/react/dist/style.css';

import StageNode from './nodes/stage';
import GithubIntegration from './nodes/event_source';
import { FlowDevTools } from './devtools';
import { useFlowStore } from "../store/flowStore";
import { useFlowHandlers } from "../hooks/useFlowHandlers";
import { useFlowData } from "../hooks/useFlowData";
import { useAutoLayout } from "../hooks/useAutoLayout";
import { useFlowTransformation } from "../hooks/useFlowTransformation";
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
    onNodesChange,
    onEdgesChange,
    onConnect
  } = useFlowStore();

  const { layoutedNodes, flowEdges } = useFlowData();
  const { applyElkAutoLayout } = useAutoLayout();
  const { updateNodesAndEdges } = useFlowTransformation();
  const { onNodeDragStop, onInit } = useFlowHandlers();
  
  const [hasInitialLayout, setHasInitialLayout] = React.useState(false);
  const prevDataRef = useRef<{
    nodeCount: number;
    edgeCount: number;
    nodeIds: string;
    edgeIds: string;
  }>({
    nodeCount: 0,
    edgeCount: 0,
    nodeIds: '',
    edgeIds: ''
  });

  const currentNodeIds = layoutedNodes.map(n => n.id).sort().join('|');
  const currentEdgeIds = flowEdges.map(e => e.id).sort().join('|');
  
  useEffect(() => {
    const hasDataChanged = 
      prevDataRef.current.nodeCount !== layoutedNodes.length ||
      prevDataRef.current.edgeCount !== flowEdges.length ||
      prevDataRef.current.nodeIds !== currentNodeIds ||
      prevDataRef.current.edgeIds !== currentEdgeIds;

    if (hasDataChanged && (layoutedNodes.length > 0 || flowEdges.length > 0)) {
      if (!hasInitialLayout) {
        updateNodesAndEdges(layoutedNodes, flowEdges);
        applyElkAutoLayout(layoutedNodes, flowEdges);
        setHasInitialLayout(true);
      } else {
        updateNodesAndEdges(layoutedNodes, flowEdges);
      }
      
      // update the ref to prevent infinite loops
      prevDataRef.current = {
        nodeCount: layoutedNodes.length,
        edgeCount: flowEdges.length,
        nodeIds: currentNodeIds,
        edgeIds: currentEdgeIds
      };
    }
  }, [
    layoutedNodes.length, 
    flowEdges.length, 
    currentNodeIds, 
    currentEdgeIds,
    hasInitialLayout,
    applyElkAutoLayout, 
    updateNodesAndEdges,
    layoutedNodes, 
    flowEdges
  ]);

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