import { create } from 'zustand';
import { CanvasData } from "../types";
import { CanvasState, ConnectionGroupWithEvents, EventSourceWithEvents, StageWithEventQueue } from './types';
import { SuperplaneCanvas, SuperplaneExecutionFilter, SuperplaneExecutionState, SuperplaneStageEventState, SuperplaneStageEventStateReason } from "@/api-client/types.gen";
import { superplaneApproveStageEvent, superplaneListStageEvents, superplaneListEvents, superplaneCancelStageEvent } from '@/api-client';
import { withOrganizationHeader } from '@/utils/withOrganizationHeader';
import { ReadyState } from 'react-use-websocket';
import { Connection, Viewport, applyNodeChanges, applyEdgeChanges } from '@xyflow/react';
import { AllNodeType, EdgeType } from '../types/flow';
import { autoLayoutNodes, transformConnectionGroupsToNodes, transformEventSourcesToNodes, transformStagesToNodes, transformToEdges } from '../utils/flowTransformers';

const SYNC_EVENTS_LIMIT = 5;
const SYNC_STAGE_EVENTS_LIMIT = 20;

type SyncStageEventRequest = {
  states: SuperplaneStageEventState[]
  stateReasons?: SuperplaneStageEventStateReason[]
  executionState?: SuperplaneExecutionState[]
  executionFilter?: SuperplaneExecutionFilter
  limit: number
}

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
  selectedEventSourceId: null,
  focusedNodeId: null,
  editingStageId: null,
  editingEventSourceId: null,
  editingConnectionGroupId: null,
  webSocketConnectionStatus: ReadyState.UNINSTANTIATED,

  // reactflow state
  nodes: [],
  edges: [],
  handleDragging: undefined,
  lockedNodes: true,


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

  addStage: (stage: StageWithEventQueue, draft = false, autoLayout = false) => {
    set((state) => ({
      stages: [...state.stages, {
        ...stage,
        queue: [],
        events: [],
        isDraft: draft
      }]
    }));
    get().syncToReactFlow({ autoLayout });
  },

  removeStage: (stageId: string) => {
    set((state) => ({
      stages: state.stages.filter(s => s.metadata?.id !== stageId),
      edges: state.edges.filter(e => e.source !== stageId && e.target !== stageId),
      // Also clear selection and editing state if this stage was selected/being edited
      selectedStageId: state.selectedStageId === stageId ? null : state.selectedStageId,
      editingStageId: state.editingStageId === stageId ? null : state.editingStageId
    }));
    get().syncToReactFlow();
  },

  updateStage: (stage: StageWithEventQueue) => {
    set((state) => ({
      stages: state.stages.map((s) => s.metadata!.id === stage.metadata!.id ? {
        ...stage, queue: s.queue, events: s.events
      } : s)
    }));
    get().syncToReactFlow();
  },

  addConnectionGroup: (connectionGroup: ConnectionGroupWithEvents) => {
    set((state) => ({
      connectionGroups: [...state.connectionGroups, { ...connectionGroup, events: [] }],
    }));
    get().syncToReactFlow();
  },

  removeConnectionGroup: (connectionGroupId: string) => {
    set((state) => ({
      connectionGroups: state.connectionGroups.filter(cg => cg.metadata?.id !== connectionGroupId),
      edges: state.edges.filter(e => e.source !== connectionGroupId && e.target !== connectionGroupId)
    }));
    get().syncToReactFlow();
  },

  updateConnectionGroup: (connectionGroup: ConnectionGroupWithEvents) => {
    set((state) => ({
      connectionGroups: state.connectionGroups.map(cg => cg.metadata?.id === connectionGroup.metadata?.id ? connectionGroup : cg)
    }));
    get().syncToReactFlow();
  },

  addEventSource: (eventSource: EventSourceWithEvents, autoLayout = false) => {
    set((state) => ({
      eventSources: [...state.eventSources, { ...eventSource, events: [] }]
    }));
    get().syncToReactFlow({ autoLayout });
  },

  removeEventSource: (eventSourceId: string) => {
    set((state) => ({
      eventSources: state.eventSources.filter(es => es.metadata?.id !== eventSourceId),
      edges: state.edges.filter(e => e.source !== eventSourceId && e.target !== eventSourceId),
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
    superplaneApproveStageEvent(withOrganizationHeader({
      path: {
        canvasIdOrName: get().canvas.metadata!.id!,
        stageIdOrName: stageId,
        eventId: stageEventId
      },
      body: {}
    }));
  },

  cancelStageEvent: (stageEventId: string, stageId: string) => {
    superplaneCancelStageEvent(withOrganizationHeader({
      path: {
        canvasIdOrName: get().canvas.metadata!.id!,
        stageIdOrName: stageId,
        eventId: stageEventId
      },
      body: {}
    }));
  },

  selectStageId: (stageId: string) => {
    set({ selectedStageId: stageId, selectedEventSourceId: null });
  },

  cleanSelectedStageId: () => {
    set({ selectedStageId: null });
  },
  selectEventSourceId: (eventSourceId: string) => {
    set({ selectedEventSourceId: eventSourceId, selectedStageId: null });
  },
  cleanSelectedEventSourceId: () => {
    set({ selectedEventSourceId: null });
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

    const recentRunsRequest: SyncStageEventRequest = {
      states: ['STATE_PROCESSED', 'STATE_WAITING', "STATE_PENDING"],
      stateReasons: ['STATE_REASON_EXECUTION', "STATE_REASON_UNKNOWN"],
      executionState: ["STATE_CANCELLED", "STATE_FINISHED", "STATE_FINISHED"],
      executionFilter: "EXECUTION_FILTER_WITH_EXECUTION",
      limit: SYNC_EVENTS_LIMIT
    }

   const queueEventsRequest: SyncStageEventRequest = {
      states: ['STATE_PROCESSED', 'STATE_WAITING', "STATE_PENDING"],
      stateReasons: ['STATE_REASON_APPROVAL', 'STATE_REASON_TIME_WINDOW', "STATE_REASON_CANCELLED", "STATE_REASON_UNKNOWN"],
      executionFilter: "EXECUTION_FILTER_WITHOUT_EXECUTION",
      limit: SYNC_STAGE_EVENTS_LIMIT
    }

    const requestingStates = [
      recentRunsRequest,
      queueEventsRequest
    ]

    const responsePromises = requestingStates.map(request => {
      return superplaneListStageEvents(withOrganizationHeader({
        path: {
          canvasIdOrName: canvasId,
          stageIdOrName: stageId,
        },
        query: {
          limit: request.limit,
          states: request.states,
          stateReasons: request.stateReasons
        } 
      }));
    })

    const responses = await Promise.all(responsePromises)
    const queueEvents = responses.flatMap(res => res.data.events || [])

    set((state) => ({
      stages: state.stages.map((s) => s.metadata!.id === stageId ? {
        ...updatingStage,
        queue: queueEvents
      } : s)
    }));
  },

  syncEventSourceEvents: async (canvasId: string, eventSourceId: string) => {
    const { eventSources } = get();
    const updatingEventSource = eventSources.find(es => es.metadata!.id === eventSourceId);

    if (!updatingEventSource) {
      return;
    }

    const eventsResponse = await superplaneListEvents(withOrganizationHeader({
      path: { canvasIdOrName: canvasId },
      query: { 
        sourceType: 'EVENT_SOURCE_TYPE_EVENT_SOURCE' as const,
        sourceId: eventSourceId,
        limit: SYNC_EVENTS_LIMIT
      }
    }));

    set((state) => ({
      eventSources: state.eventSources.map((es) => es.metadata!.id === eventSourceId ? {
        ...updatingEventSource,
        events: eventsResponse.data?.events || []
      } : es)
    }));
  },

  syncStagePlainEvents: async (canvasId: string, stageId: string) => {
    const { stages } = get();
    const updatingStage = stages.find(stage => stage.metadata!.id === stageId);

    if (!updatingStage) {
      return;
    }

    const eventsResponse = await superplaneListEvents(withOrganizationHeader({
      path: { canvasIdOrName: canvasId },
      query: { 
        sourceType: 'EVENT_SOURCE_TYPE_STAGE' as const,
        sourceId: stageId,
        limit: SYNC_EVENTS_LIMIT
      }
    }));

    set((state) => ({
      stages: state.stages.map((s) => s.metadata!.id === stageId ? {
        ...updatingStage,
        events: eventsResponse.data?.events || []
      } : s)
    }));
  },

  syncConnectionGroupPlainEvents: async (canvasId: string, connectionGroupId: string) => {
    const { connectionGroups } = get();
    const updatingConnectionGroup = connectionGroups.find(cg => cg.metadata!.id === connectionGroupId);

    if (!updatingConnectionGroup) {
      return;
    }

    const eventsResponse = await superplaneListEvents(withOrganizationHeader({
      path: { canvasIdOrName: canvasId },
      query: { 
        sourceType: 'EVENT_SOURCE_TYPE_CONNECTION_GROUP' as const,
        sourceId: connectionGroupId,
        limit: SYNC_EVENTS_LIMIT
      }
    }));

    set((state) => ({
      connectionGroups: state.connectionGroups.map((cg) => cg.metadata!.id === connectionGroupId ? {
        ...updatingConnectionGroup,
        events: eventsResponse.data?.events || []
      } : cg)
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

  // Flow instance reference
  fitViewNodeRef: null as ((nodeId: string) => void) | null,
  
  setFitViewNodeRef: (fitViewNodeFn: (nodeId: string) => void) => {
    set({ fitViewNodeRef: fitViewNodeFn });
  },

  fitViewNode: (nodeId: string) => {
    const { fitViewNodeRef } = get();
    if (fitViewNodeRef) {
      fitViewNodeRef(nodeId);
    }
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
  },

  setLockedNodes: (locked: boolean) => {
    set({ lockedNodes: locked });
  }
}));
