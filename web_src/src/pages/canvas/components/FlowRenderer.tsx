import React, { useEffect, useCallback, useRef, useMemo } from "react";
import { ReactFlow, Controls, Background, Node, ReactFlowInstance, Edge, OnInit, ControlButton } from "@xyflow/react";
import { useCanvasStore } from "../store/canvasStore";
import { useFlowStore } from "../store/flowStore";
import '@xyflow/react/dist/style.css';
import Elk, { ElkNode, ElkExtendedEdge } from "elkjs";

import StageNode from './nodes/stage';
import GithubIntegration from './nodes/event_source';
import { FlowDevTools } from './devtools';
import { AllNodeType, EdgeType } from "../types/flow";
import { SuperplaneStageEvent } from "@/api-client/types.gen";

export const nodeTypes = {
  deploymentCard: StageNode,
  githubIntegration: GithubIntegration,
}

// ELK.js instance with layout configuration
const elk = new Elk({
  defaultLayoutOptions: {
    "elk.algorithm": "layered",
    "elk.direction": "RIGHT",
    "elk.spacing.nodeNode": "80",
    "elk.layered.spacing.nodeNodeBetweenLayers": "100", 
    "elk.layered.spacing": "80",
    "elk.layered.mergeEdges": "true",
    "elk.spacing": "80",
    "elk.spacing.individual": "80",
    "elk.edgeRouting": "SPLINES",
  },
});

// Default node dimensions
const DEFAULT_WIDTH = 300;
const DEFAULT_HEIGHT = 200;

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

  const reactFlowInstanceRef = useRef<ReactFlowInstance<AllNodeType, EdgeType> | null>(null);
  
  // Memoized computation of nodes and edges with auto-layout
  const { layoutedNodes, flowEdges } = useMemo(() => {
    // Convert data model to React Flow nodes
    const rawNodes = [
      ...event_sources.map((es, idx) => ({
        id: es.id,
        type: 'githubIntegration',
        data: {
          id: es.name,
          repoName: "repo/name",
          repoUrl: "repo/url",
          eventType: 'push',
          release: 'v1.0.0',
          timestamp: '2023-01-01T00:00:00'
        },
        position: nodePositions[es.id!] || { x: 0, y: idx * 320 },
        draggable: true
      })),
      ...stages.map((st, idx) => ({
        id: st.id,
        type: 'deploymentCard',
        data: {
          label: st.name,
          labels: [],
          status: "",
          icon: "storage",
          queues: st.queue || [],
          connections: st.connections || [],
          conditions: st.conditions || [],
          runTemplate: st.runTemplate,
          approveStageEvent: (event: SuperplaneStageEvent) => {
            approveStageEvent(event.id!, st.id!);
          }
        },
        position: nodePositions[st.id!] || { x: 600 * ((st.connections?.length || 1)), y: (idx -1) * 400 },
        draggable: true
      })) 
    ] as AllNodeType[];
    
    // Convert data model to React Flow edges
    const rawEdges = stages.flatMap((st) =>
      (st.connections || []).map((conn) => {
        const isEvent = event_sources.some((es) => es.name === conn.name);
        const sourceObj =
          event_sources.find((es) => es.name === conn.name) ||
          stages.find((s) => s.name === conn.name);
        const sourceId = sourceObj?.id ?? conn.name!;
        return { 
          id: `e-${conn.name}-${st.id}`, 
          source: sourceId, 
          target: st.id!, 
          type: "smoothstep", 
          animated: true, 
          style: isEvent ? { stroke: '#FF0000', strokeWidth: 2 } : undefined 
        };
      })
    );

    // Apply auto-layout only if nodes don't have stored positions
    const needsAutoLayout = rawNodes.some(node => !nodePositions[node.id]);
    
    if (needsAutoLayout && rawNodes.length > 0) {
      // For now, we'll use a simple grid layout since ELK is async
      // You could also use a different synchronous layout algorithm
      const layoutedNodes = rawNodes.map((node, index) => {
        if (nodePositions[node.id]) {
          return node; // Keep existing position
        }
        
        // Simple grid layout
        const cols = Math.ceil(Math.sqrt(rawNodes.length));
        const row = Math.floor(index / cols);
        const col = index % cols;
        
        return {
          ...node,
          position: {
            x: col * (DEFAULT_WIDTH + 100),
            y: row * (DEFAULT_HEIGHT + 100)
          }
        };
      });
      
      return { layoutedNodes, flowEdges: rawEdges };
    }
    
    return { layoutedNodes: rawNodes, flowEdges: rawEdges };
  }, [event_sources, stages, nodePositions, approveStageEvent]);

  // Async auto-layout function using ELK (for manual trigger)
  const applyElkAutoLayout = useCallback(async (layoutedNodes: AllNodeType[], flowEdges: Edge[]) => {
    if (layoutedNodes.length === 0) return;

    const elkNodes: ElkNode[] = layoutedNodes.map((node) => ({
      id: node.id,
      width: DEFAULT_WIDTH,
      height: DEFAULT_HEIGHT,
    }));

    const elkEdges: ElkExtendedEdge[] = flowEdges.map((edge) => ({
      id: edge.id,
      sources: [edge.source],
      targets: [edge.target],
    }));

    try {
      const layoutedGraph = await elk.layout({
        id: "root",
        children: elkNodes,
        edges: elkEdges,
      });

      const newNodes = layoutedNodes.map((node) => {
        const elkNode = layoutedGraph.children?.find((n) => n.id === node.id);
        
        if (elkNode?.x !== undefined && elkNode?.y !== undefined) {
          const newPosition = {
            x: elkNode.x - (elkNode.width || DEFAULT_WIDTH) / 2 + Math.random() / 1000,
            y: elkNode.y - (elkNode.height || DEFAULT_HEIGHT) / 2,
          };
          
          return {
            ...node,
            position: newPosition,
          };
        }
        
        return node;
      });

      setNodes(newNodes);
      
      // Update node positions in canvas store
      newNodes.forEach((node) => {
        updateNodePosition(node.id, node.position);
      });
    } catch (error) {
      console.error('ELK auto-layout failed:', error);
    }
  }, [setNodes, updateNodePosition]);
  
  useEffect(() => {
    setEdges(flowEdges);
    applyElkAutoLayout(layoutedNodes, flowEdges);
  }, [applyElkAutoLayout]);

  // Handler for when node dragging stops - propagate position to canvasStore
  const onNodeDragStop = useCallback(
    (_: React.MouseEvent, node: Node) => {
      updateNodePosition(node.id, node.position);
    },
    [updateNodePosition]
  );

  // Handle ReactFlow instance initialization
  const onInit: OnInit<AllNodeType, EdgeType> = useCallback((instance) => {
    reactFlowInstanceRef.current = instance;
    instance.fitView();
  }, []);

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
          <Controls>
            <ControlButton
              onClick={() => applyElkAutoLayout(layoutedNodes, flowEdges)}
              title="ELK Auto Layout"
              
            >
              <span className="material-icons" style={{fontSize:20}}>account_tree</span>

            </ControlButton>
          </Controls>
          <Background />
          <FlowDevTools />
        </ReactFlow>
    </div>
  );
};