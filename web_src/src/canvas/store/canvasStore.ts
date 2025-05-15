import { create } from 'zustand';
import { CanvasInitialData, Stage, EventSource } from "../types";
import { CanvasState } from './types';
import { setupEventHandlers } from './handlers/setup';

// Create the store
export const useCanvasStore = create<CanvasState>((set, get) => ({
  // Initial state
  canvas: {},
  stages: [],
  event_sources: [],
  nodePositions: {},
  handleEvent: undefined,
  removeHandleEvent: undefined,
  pushEvent: undefined,
  
  // Actions (equivalent to the reducer actions in the context implementation)
  initialize: (data: CanvasInitialData) => {
    console.log("Initializing Canvas with data:", data);
    set({
      canvas: data.canvas || {},
      stages: data.stages || [],
      event_sources: data.event_sources || [],
      nodePositions: {},
      handleEvent: data.handleEvent,
      removeHandleEvent: data.removeHandleEvent,
      pushEvent: data.pushEvent,
    });
    console.log("Canvas initialized with stages:", data.stages?.length || 0);
  },
  
  addStage: (stage: Stage) => {
    console.log("Adding stage:", stage);
    set((state) => ({
      stages: [...state.stages, stage]
    }));
  },
  
  updateStage: (stage: Stage) => {
    console.log("Updating stage:", stage);
    set((state) => ({
      stages: state.stages.map(s => s.id === stage.id ? stage : s)
    }));
  },
  
  addEventSource: (eventSource: EventSource) => {
    console.log("Adding event source:", eventSource);
    set((state) => ({
      event_sources: [...state.event_sources, eventSource]
    }));
  },
  
  updateEventSource: (eventSource: EventSource) => {
    console.log("Updating event source:", eventSource);
    set((state) => ({
      event_sources: state.event_sources.map(es => 
        es.id === eventSource.id ? eventSource : es
      )
    }));
  },
  
  updateCanvas: (newCanvas: Record<string, any>) => {
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
    
    const { pushEvent } = get();
    if (pushEvent) {
      console.log("send trough websocket stage approval for stage: ", stageId, "event: ", stageEventId);
    } else {
      console.error("pushEvent function is not available");
    }
  },
  
  // Setup LiveView event handlers and return a cleanup function
  setupLiveViewHandlers: (initialData: CanvasInitialData) => {
    return setupEventHandlers(initialData);
  }
}));
