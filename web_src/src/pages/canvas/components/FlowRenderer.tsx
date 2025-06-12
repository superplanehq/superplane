import React, { useEffect, useMemo, useRef } from "react";
import { ReactFlow, Background } from "@xyflow/react";
import '@xyflow/react/dist/style.css';

import StageNode from './nodes/stage';
import GithubIntegration from './nodes/event_source';
import { FlowDevTools } from './devtools';
import { useCanvasStore } from "../store/canvasStore";
import { useFlowHandlers } from "../hooks/useFlowHandlers";
import { useAutoLayout } from "../hooks/useAutoLayout";
import { useFlowTransformation } from "../hooks/useFlowTransformation";
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
  const { updateNodesAndEdges } = useFlowTransformation();
  const { onNodeDragStop, onInit } = useFlowHandlers();
  
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

  const currentNodeIds = useMemo(() => nodes.map(n => n.id).sort().join('|'), [nodes]);
  const currentEdgeIds = useMemo(() => edges.map(e => e.id).sort().join('|'), [edges]);
  
  useEffect(() => {
    const hasDataChanged = 
      prevDataRef.current.nodeCount !== nodes.length ||
      prevDataRef.current.edgeCount !== edges.length ||
      prevDataRef.current.nodeIds !== currentNodeIds ||
      prevDataRef.current.edgeIds !== currentEdgeIds;

    if (hasDataChanged && (nodes.length > 0 || edges.length > 0)) {
      updateNodesAndEdges(nodes);

      prevDataRef.current = {
        nodeCount: nodes.length,
        edgeCount: edges.length,
        nodeIds: currentNodeIds,
        edgeIds: currentEdgeIds
      };
    }
  }, [
    nodes.length, 
    edges.length, 
    currentNodeIds, 
    currentEdgeIds,
    updateNodesAndEdges,
    nodes, 
    edges
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