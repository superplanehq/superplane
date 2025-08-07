import { create } from 'zustand';
import { CanvasData } from "../types";
import { CanvasState, EventSourceWithEvents } from './types';
import { SuperplaneCanvas, SuperplaneConnectionGroup, SuperplaneStage } from "@/api-client/types.gen";
import { superplaneApproveStageEvent, superplaneListStageEvents } from '@/api-client';
import { ReadyState } from 'react-use-websocket';
import { Connection, Viewport, applyNodeChanges, applyEdgeChanges } from '@xyflow/react';
import { AllNodeType, EdgeType } from '../types/flow';
import { autoLayoutNodes, transformConnectionGroupsToNodes, transformEventSourcesToNodes, transformStagesToNodes, transformToEdges } from '../utils/flowTransformers';

// Create the store
export const useCanvasStore = create<CanvasState>((set, get) => ({
  // Initial state
  canvas: {},
  canvasId: '',
  stages: [],
  eventSources: [],
  connectionGroups: [],
  nodePositions: {},
  eventSourceKeys: {},
  selectedStageId: null,
  focusedNodeId: null,
  editingStageId: null,
  editingEventSourceId: null,
  editingConnectionGroupId: null,
  webSocketConnectionStatus: ReadyState.UNINSTANTIATED,

  // reactflow state
  nodes: [],
  edges: [],
  handleDragging: undefined,


  // Actions (equivalent to the reducer actions in the context implementation)
  initialize: (data: CanvasData) => {
    set({
      canvas: data.canvas || {},
      canvasId: data.canvas?.metadata?.id || '',
      stages: data.stages || [],
      eventSources: data.eventSources || [],
      connectionGroups: data.connectionGroups || [],
      nodePositions: {},
      eventSourceKeys: {},
    });
    get().syncToReactFlow({ autoLayout: true });
  },

  addStage: (stage: SuperplaneStage, draft = false, autoLayout = true) => {
    set((state) => ({
      stages: [...state.stages, {
        ...stage,
        queue: [],
        isDraft: draft
      }]
    }));
    get().syncToReactFlow({ autoLayout });
  },

  removeStage: (stageId: string) => {
    set((state) => ({
      stages: state.stages.filter(s => s.metadata?.id !== stageId),
      // Also clear selection and editing state if this stage was selected/being edited
      selectedStageId: state.selectedStageId === stageId ? null : state.selectedStageId,
      editingStageId: state.editingStageId === stageId ? null : state.editingStageId
    }));
    get().syncToReactFlow();
  },

  updateStage: (stage: SuperplaneStage) => {
    set((state) => ({
      stages: state.stages.map((s) => s.metadata!.id === stage.metadata!.id ? {
        ...stage, queue: s.queue
      } : s)
    }));
    get().syncToReactFlow();
  },

  addConnectionGroup: (connectionGroup: SuperplaneConnectionGroup) => {
    set((state) => ({
      connectionGroups: [...state.connectionGroups, connectionGroup]
    }));
    get().syncToReactFlow();
  },

  removeConnectionGroup: (connectionGroupId: string) => {
    set((state) => ({
      connectionGroups: state.connectionGroups.filter(cg => cg.metadata?.id !== connectionGroupId)
    }));
    get().syncToReactFlow();
  },

  updateConnectionGroup: (connectionGroup: SuperplaneConnectionGroup) => {
    set((state) => ({
      connectionGroups: state.connectionGroups.map(cg => cg.metadata?.id === connectionGroup.metadata?.id ? connectionGroup : cg)
    }));
    get().syncToReactFlow();
  },

  addEventSource: (eventSource: EventSourceWithEvents, autoLayout = false) => {
    set((state) => ({
      eventSources: [...state.eventSources, eventSource]
    }));
    get().syncToReactFlow({ autoLayout });
  },

  removeEventSource: (eventSourceId: string) => {
    set((state) => ({
      eventSources: state.eventSources.filter(es => es.metadata?.id !== eventSourceId),
      // Also clear editing state if this event source was being edited
      editingEventSourceId: state.editingEventSourceId === eventSourceId ? null : state.editingEventSourceId
    }));
    get().syncToReactFlow();
  },

  updateEventSource: (eventSource: EventSourceWithEvents) => {
    set((state) => ({
      eventSources: state.eventSources.map(es =>
        es.metadata!.id === eventSource.metadata!.id ? eventSource : es
      )
    }));
    get().syncToReactFlow();
  },

  updateCanvas: (newCanvas: Partial<SuperplaneCanvas>) => {
    set((state) => ({
      canvas: { ...state.canvas, ...newCanvas }
    }));
  },

  updateNodePosition: (nodeId: string, position: { x: number, y: number }) => {
    set((state) => ({
      nodePositions: {
        ...state.nodePositions,
        [nodeId]: position
      }
    }));
  },

  approveStageEvent: (stageEventId: string, stageId: string) => {

    // use post request to approve stage event
    // defined in @/api-client/api
    superplaneApproveStageEvent({
      path: {
        canvasIdOrName: get().canvas.metadata!.id!,
        stageIdOrName: stageId,
        eventId: stageEventId
      },
      body: {}
    });
  },

  selectStageId: (stageId: string) => {
    set({ selectedStageId: stageId });
  },

  cleanSelectedStageId: () => {
    set({ selectedStageId: null });
  },

  setFocusedNodeId: (stageId: string | null) => {
    set({ focusedNodeId: stageId });
  },

  cleanFocusedNodeId: () => {
    set({ focusedNodeId: null });
  },

  setEditingStage: (stageId: string | null) => {
    set({ editingStageId: stageId });
  },

  setEditingEventSource: (eventSourceId: string | null) => {
    set({ editingEventSourceId: eventSourceId });
  },

  setEditingConnectionGroup: (connectionGroupId: string | null) => {
    set({ editingConnectionGroupId: connectionGroupId });
  },

  updateWebSocketConnectionStatus: (status) => {
    set({ webSocketConnectionStatus: status });
  },

  syncStageEvents: async (canvasId: string, stageId: string) => {
    const { stages } = get();
    const updatingStage = stages.find(stage => stage.metadata!.id === stageId);

    if (!updatingStage) {
      return;
    }

    const stageEventsResponse = await superplaneListStageEvents({
      path: {
        canvasIdOrName: canvasId,
        stageIdOrName: stageId
      }
    });
    set((state) => ({
      stages: state.stages.map((s) => s.metadata!.id === stageId ? {
        ...updatingStage,
        queue: stageEventsResponse.data?.events || []
      } : s)
    }));
  },

  syncToReactFlow: async (options?: { autoLayout?: boolean }) => {
    const { stages, connectionGroups, eventSources, nodePositions, approveStageEvent } = get();

    // Use the transformer functions from flowTransformers.ts
    const stageNodes = transformStagesToNodes(stages, nodePositions, approveStageEvent);
    const connectionGroupNodes = transformConnectionGroupsToNodes(connectionGroups, nodePositions);
    const eventSourceNodes = transformEventSourcesToNodes(eventSources, nodePositions);

    // Get edges based on connections
    const edges = transformToEdges(stages, connectionGroups, eventSources);
    const unlayoutedNodes = [...stageNodes, ...connectionGroupNodes, ...eventSourceNodes];
    const nodes = options?.autoLayout ?
      await autoLayoutNodes(unlayoutedNodes, edges) :
      unlayoutedNodes;
    const newNodePositions = nodes.reduce((acc, node) => {
      acc[node.id] = node.position;
      return acc;
    }, {} as Record<string, { x: number; y: number }>);

    set({
      nodes,
      edges,
      nodePositions: newNodePositions
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
  fitViewNode: () => {
    // Will be replaced when setReactFlowInstance is called
  },

  updateEventSourceKey: (eventSourceId: string, key: string) => {
    set((state) => ({
      eventSourceKeys: {
        ...state.eventSourceKeys,
        [eventSourceId]: key
      }
    }));
  },

  resetEventSourceKey: (eventSourceId: string) => {
    set((state) => {
      const updatedEventSourceKeys = { ...state.eventSourceKeys };
      delete updatedEventSourceKeys[eventSourceId];
      return { eventSourceKeys: updatedEventSourceKeys };
    });
  }
}));
