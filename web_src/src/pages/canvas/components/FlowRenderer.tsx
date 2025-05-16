import React, { useEffect, useCallback } from "react";
import { ReactFlow, Controls, Background, Node } from "@xyflow/react";
import { useCanvasStore } from "../store/canvasStore";
import { useFlowStore } from "../store/flowStore";
import '@xyflow/react/dist/style.css';

import StageNode from './nodes/stage';
import GithubIntegration from './nodes/event_source';
import { FlowDevTools } from './devtools';
import { AllNodeType, Event } from "../types/flow";

export const nodeTypes = {
  deploymentCard: StageNode,
  githubIntegration: GithubIntegration,
}

/**
 * Renders the canvas data as React Flow nodes and edges.
 */
export const FlowRenderer: React.FC = () => {
  // Get data from canvasStore (our data model)
  const { stages, event_sources, nodePositions, updateNodePosition, approveStageEvent } = useCanvasStore();
  
  // Get flow methods from flowStore (our UI flow state)
  const { 
    nodes, 
    edges, 
    setNodes, 
    setEdges,
    onNodesChange,
    onEdgesChange,
    onConnect
  } = useFlowStore();
  
  // Sync canvasStore data with flowStore nodes and edges
  useEffect(() => {
    // Convert data model to React Flow nodes and explicitly cast to AllNodeType
    // We're ensuring the node structure matches what's expected in the types
    const flowNodes = [
      ...event_sources.map((es, idx) => ({
        id: es.id,
        type: 'githubIntegration',
        data: {
          label: es.name,
          repoName: es.name,
          repoUrl: es.url,
          lastEvent: es.lastEvent || {
            type: 'push',
            release: 'v1.0.0',
            timestamp: '2023-01-01T00:00:00'
          }
        },
        position: nodePositions[es.id] || { x: 0, y: idx * 320 },
        draggable: true
      })),
      ...stages.map((st, idx) => ({
        id: st.id,
        type: 'deploymentCard',
        data: {
          label: st.name,
          labels: st.labels || [],
          status: st.status,
          timestamp: st.timestamp,
          icon: st.icon || "storage",
          queues: st.queues || [],
          connections: st.connections || [],
          conditions: st.conditions || [],
          run_template: st.run_template,
          approve_stage_event: (event: Event) => {
            console.log('Approve stage event', event);
            approveStageEvent(event.id, st.id);
          }
        },
        position: nodePositions[st.id] || { x: 600 * ((st.connections?.length || 1)), y: (idx -1) * 400 },
        draggable: true
      }))
    ] as AllNodeType[]; // Use type assertion to resolve the complex type issue
    
    // Convert data model to React Flow edges
    const flowEdges = stages.flatMap((st) =>
      (st.connections || []).map((conn) => {
        const isEvent = event_sources.some((es) => es.name === conn.name);
        const sourceObj =
          event_sources.find((es) => es.name === conn.name) ||
          stages.find((s) => s.name === conn.name);
        const sourceId = sourceObj?.id ?? conn.name;
        return { 
          id: `e-${conn.name}-${st.id}`, 
          source: sourceId, 
          target: st.id, 
          type: "smoothstep", 
          animated: true, 
          style: isEvent ? { stroke: '#FF0000', strokeWidth: 2 } : undefined 
        };
      })
    );
    
    // Update the flow store with new nodes and edges
    setNodes(flowNodes);
    setEdges(flowEdges);
    
  }, [event_sources, stages, nodePositions, setNodes, setEdges]);
  
  // Handler for when node dragging stops - propagate position to canvasStore
  const onNodeDragStop = useCallback(
    (_: React.MouseEvent, node: Node) => {
      updateNodePosition(node.id, node.position);
    },
    [updateNodePosition]
  );

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
          onInit={(instance) => instance.fitView()}
          fitView
          minZoom={0.4}
          maxZoom={1.5}
        >
          <Controls />
          <Background />
          <FlowDevTools />
        </ReactFlow>
    </div>
  );
};
