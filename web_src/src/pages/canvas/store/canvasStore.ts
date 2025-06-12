import { create } from 'zustand';
import { CanvasData } from "../types";
import { CanvasState, EventSourceWithEvents } from './types';
import { SuperplaneCanvas, SuperplaneStage } from "@/api-client/types.gen";
import { superplaneApproveStageEvent } from '@/api-client';
import { ReadyState } from 'react-use-websocket';
import { Connection, Viewport, applyNodeChanges, applyEdgeChanges } from '@xyflow/react';
import { AllNodeType, EdgeType } from '../types/flow';
import { autoLayoutNodes, transformEventSourcesToNodes, transformStagesToNodes, transformToEdges } from '../utils/flowTransformers';

function generateFakeUUID() {
  return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, function(c) {
    const r = Math.random() * 16 | 0;
    const v = c == 'x' ? r : (r & 0x3 | 0x8);
    return v.toString(16);
  });
}

// Create the store
export const useCanvasStore = create<CanvasState>((set, get) => ({
  // Initial state
  canvas: {},
  stages: [],
  event_sources: [],
  nodePositions: {},
  selectedStage: null,
  webSocketConnectionStatus: ReadyState.UNINSTANTIATED,

  // reactflow state
  nodes: [],
  edges: [],
  handleDragging: undefined,

  
  // Actions (equivalent to the reducer actions in the context implementation)
  initialize: (data: CanvasData) => {
    console.log("Initializing Canvas with data:", data);
    set({
      canvas: data.canvas || {},
      stages: data.stages || [],
      event_sources: data.event_sources || [],
      nodePositions: {},
    });
    get().syncToReactFlow({ autoLayout: true });
    console.log("Canvas initialized with stages:", data.stages?.length || 0);
  },
  
  addStage: (stage: SuperplaneStage) => {
    console.log("Adding stage:", stage);
    set((state) => ({
      stages: [...state.stages, {
        ...stage,
        queue: []
      }]
    }));
    get().syncToReactFlow();
  },
  
  updateStage: (stage: SuperplaneStage) => {
    console.log("Updating stage:", stage);
    set((state) => ({
      stages: state.stages.map((s) => s.metadata!.id === stage.metadata!.id ? {
        ...stage, queue: s.queue} : s)
    }));
    get().syncToReactFlow();
  },
  
  addEventSource: (eventSource: EventSourceWithEvents) => {
    set((state) => ({
      event_sources: [...state.event_sources, eventSource]
    }));
    get().syncToReactFlow();
  },
  
  updateEventSource: (eventSource: EventSourceWithEvents) => {
    console.log("Updating event source:", eventSource);
    set((state) => ({
      event_sources: state.event_sources.map(es => 
        es.metadata!.id === eventSource.metadata!.id ? eventSource : es
      )
    }));
    get().syncToReactFlow();
  },
  
  updateCanvas: (newCanvas: Partial<SuperplaneCanvas>) => {
    console.log("Updating canvas:", newCanvas);
    set((state) => ({
      canvas: { ...state.canvas, ...newCanvas }
    }));
  },
  
  updateNodePosition: (nodeId: string, position: { x: number, y: number }) => {
    // console.log("Updating node position:", nodeId, position);
    set((state) => ({
      nodePositions: {
        ...state.nodePositions,
        [nodeId]: position
      }
    }));
  },

  approveStageEvent: (stageEventId: string, stageId: string) => {
    console.log("[client action] Approving stage event:", stageEventId);
    
    // use post request to approve stage event
    // defined in @/api-client/api
    superplaneApproveStageEvent({
      path: {
        canvasIdOrName: get().canvas.metadata!.id!,
        stageIdOrName: stageId,
        eventId: stageEventId
      },
      body: {
        requesterId: generateFakeUUID(),
        // Both fields are optional, but the 'body' property itself is required
      }
    });
  },
  
  selectStage: (stageId: string) => {
    set((state) => ({ selectedStage: state.stages.find(stage => stage.metadata!.id === stageId) }));
  },

  cleanSelectedStage: () => {
    set({ selectedStage: null });
  },
  
  updateWebSocketConnectionStatus: (status) => {
    set({ webSocketConnectionStatus: status });
  },

  syncToReactFlow: async (options?: { autoLayout?: boolean }) => {
    const { stages, event_sources, nodePositions, approveStageEvent } = get();

    // Use the transformer functions from flowTransformers.ts
    const stageNodes = transformStagesToNodes(stages, nodePositions, approveStageEvent);
    const eventSourceNodes = transformEventSourcesToNodes(event_sources, nodePositions);
    
    // Get edges based on connections
    const edges = transformToEdges(stages, event_sources);
    const unlayoutedNodes = [...stageNodes, ...eventSourceNodes];
    const nodes = options?.autoLayout ?
      await autoLayoutNodes(unlayoutedNodes, edges) :
      unlayoutedNodes;
    
    set({
        nodes,
        edges
    });
},


  onNodesChange: (changes) => {
    set({
      nodes: applyNodeChanges(changes, get().nodes) as AllNodeType[],
    });
  },

  onEdgesChange: (changes) => {
    set({
      edges: applyEdgeChanges(changes, get().edges) as EdgeType[],
    });
  },

  setNodes: (update: AllNodeType[]) => {
    set({ nodes: update });
  },

  // Edge operations
  onConnect: (connection: Connection) => {
    // Create a new edge when a connection is made
    const newEdge: EdgeType = {
      id: `e-${connection.source}-${connection.target}-${Math.floor(Math.random() * 1000)}`,
      source: connection.source || '',
      target: connection.target || '',
      sourceHandle: connection.sourceHandle || undefined,
      targetHandle: connection.targetHandle || undefined,
      type: 'smoothstep',
      animated: true
    };
    
    set({
      edges: [...get().edges, newEdge],
    });
  },

  // Flow utilities
  cleanFlow: () => {
    set({ nodes: [], edges: [] });
  },

  unselectAll: () => {
    set({
      nodes: get().nodes.map(node => ({ ...node, selected: false })),
      edges: get().edges.map(edge => ({ ...edge, selected: false })),
    });
  },

  getFlow: () => {
    const defaultViewport: Viewport = { x: 0, y: 0, zoom: 1 };
    return { 
      nodes: get().nodes, 
      edges: get().edges, 
      viewport: defaultViewport // Note: you might want to store the actual viewport from React Flow
    };
  },

  getNodePosition: (nodeId: string) => {
    const node = get().nodes.find(n => n.id === nodeId);
    return node?.position || { x: 0, y: 0 };
  },

  // Handle dragging state for connections
  setHandleDragging: (data) => {
    set({ handleDragging: data });
  },

  // Initialization with default properties
  fitViewNode: (nodeId: string) => {
    // Will be replaced when setReactFlowInstance is called
    console.warn('fitViewNode called before ReactFlow instance was set', nodeId);
  },
}));
