import { create } from 'zustand';
import { CanvasData } from "../types";
import { CanvasState } from './types';
import { SuperplaneCanvas, SuperplaneEventSource, SuperplaneStage } from "@/api-client/types.gen";
import { superplaneApproveStageEvent } from '@/api-client';
import { ReadyState } from 'react-use-websocket';

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
  
  // Actions (equivalent to the reducer actions in the context implementation)
  initialize: (data: CanvasData) => {
    console.log("Initializing Canvas with data:", data);
    set({
      canvas: data.canvas || {},
      stages: data.stages || [],
      event_sources: data.event_sources || [],
      nodePositions: {},
    });
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
  },
  
  updateStage: (stage: SuperplaneStage) => {
    console.log("Updating stage:", stage);
    set((state) => ({
      stages: state.stages.map((s) => s.metadata!.id === stage.metadata!.id ? {
        ...stage, queue: s.queue} : s)
    }));
  },
  
  addEventSource: (eventSource: SuperplaneEventSource) => {
    console.log("Adding event source:", eventSource);
    set((state) => ({
      event_sources: [...state.event_sources, eventSource]
    }));
  },
  
  updateEventSource: (eventSource: SuperplaneEventSource) => {
    console.log("Updating event source:", eventSource);
    set((state) => ({
      event_sources: state.event_sources.map(es => 
        es.metadata!.id === eventSource.metadata!.id ? eventSource : es
      )
    }));
  },
  
  updateCanvas: (newCanvas: Partial<SuperplaneCanvas>) => {
    console.log("Updating canvas:", newCanvas);
    set((state) => ({
      canvas: { ...state.canvas, ...newCanvas }
    }));
  },
  
  updateNodePosition: (nodeId: string, position: { x: number, y: number }) => {
    console.log("Updating node position:", nodeId, position);
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
  
  // This is a flag that indicates whether event handlers have been set up
  // The actual setup will be done in a React component using the useSetupEventHandlers hook
  eventHandlersSetup: false,
  
  // Mark event handlers as set up
  markEventHandlersAsSetup: () => {
    set({ eventHandlersSetup: true });
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
}));
