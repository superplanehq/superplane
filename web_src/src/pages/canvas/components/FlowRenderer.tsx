import React, { useMemo } from "react";
import { ReactFlow, Background, BackgroundVariant } from "@xyflow/react";
import '@xyflow/react/dist/style.css';

import StageNode from './nodes/stage';
import GithubIntegration from './nodes/event_source';
import ConnectionGroupNode from './nodes/connection_group';
import { FlowDevTools } from './devtools';
import { useCanvasStore } from "../store/canvasStore";
import { useFlowHandlers } from "../hooks/useFlowHandlers";
import { useAutoLayout } from "../hooks/useAutoLayout";
import { FlowControls } from "./FlowControls";
import { ConnectionStatus } from "./ConnectionStatus";

export const nodeTypes = {
  connectionGroup: ConnectionGroupNode,
  deploymentCard: StageNode,
  githubIntegration: GithubIntegration,
};

export const FlowRenderer: React.FC = () => {
  const nodes = useCanvasStore((state) => state.nodes);
  const edges = useCanvasStore((state) => state.edges);
  const stages = useCanvasStore((state) => state.stages);
  const onNodesChange = useCanvasStore((state) => state.onNodesChange);
  const onEdgesChange = useCanvasStore((state) => state.onEdgesChange);
  const onConnect = useCanvasStore((state) => state.onConnect);
  const setFocusedNodeId = useCanvasStore((state) => state.setFocusedNodeId);
  const fitViewNode = useCanvasStore((state) => state.fitViewNode);
  const setFitViewNodeRef = useCanvasStore((state) => state.setFitViewNodeRef);
  const lockedNodes = useCanvasStore((state) => state.lockedNodes);
  const setLockedNodes = useCanvasStore((state) => state.setLockedNodes);

  const { applyElkAutoLayout } = useAutoLayout();
  const { onNodeDragStop, onInit, fitViewToNode } = useFlowHandlers();

  React.useEffect(() => {
    setFitViewNodeRef(fitViewToNode);
  }, [fitViewToNode, setFitViewNodeRef]);

  const animatedEdges = useMemo(() => {
    const runningEdges = new Set<string>();

    stages.forEach(stage => {
      const allExecutions = stage.queue?.flatMap(event => ({ ...event.execution, sourceId: event.sourceId }))
        .filter(execution => execution)
        .sort((a, b) => new Date(b?.createdAt || '').getTime() - new Date(a?.createdAt || '').getTime()) || [];

      const executionsRunning = allExecutions.filter(execution => execution?.state === 'STATE_STARTED');
      const sourceIdStageIdPairs = executionsRunning.map(execution => `${execution.sourceId}-${stage.metadata?.id}`);
      const isRunning = sourceIdStageIdPairs.length > 0;

      if (isRunning) {
        sourceIdStageIdPairs.forEach(pair => runningEdges.add(pair));
      }
    });

    return edges.map(edge => ({
      ...edge,
      animated: runningEdges.has(`${edge.source}-${edge.target}`)
    }));
  }, [edges, stages]);

  return (
    <div style={{ width: "100vw", height: "100%", minWidth: 0, minHeight: 0 }}>
      <ReactFlow
        nodes={nodes}
        edges={animatedEdges}
        nodeTypes={nodeTypes}
        onNodesChange={onNodesChange}
        onEdgesChange={onEdgesChange}
        onConnect={onConnect}
        onNodeDragStop={onNodeDragStop}
        onNodeClick={(_, node) => {
          setFocusedNodeId(node.id);
          fitViewNode(node.id);
        }}
        onNodeDrag={(_, node) => {
          setFocusedNodeId(node.id)
        }}
        onInit={onInit}
        nodesDraggable={!lockedNodes}
        fitView
        minZoom={0.4}
        maxZoom={1.5}
        colorMode={"system"}
      >
        <FlowControls
          onAutoLayout={applyElkAutoLayout}
          nodes={nodes}
          edges={animatedEdges}
          onLockToggle={setLockedNodes}
          isLocked={lockedNodes}
        />
        <Background
          variant={BackgroundVariant.Dots}
          gap={24}
          size={1}
        />
        <FlowDevTools />
        <ConnectionStatus />
      </ReactFlow>
    </div>
  );
};