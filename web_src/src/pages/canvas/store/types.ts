import { CanvasData } from "../types";
import { SuperplaneCanvas, SuperplaneStage, SuperplaneStageEvent } from "@/api-client/types.gen";
import { ReadyState } from "react-use-websocket";

// Define the store state type
export interface CanvasState {
  canvas: SuperplaneCanvas;
  stages: StageWithEventQueue[];
  event_sources: EventSourceWithEvents[];
  nodePositions: Record<string, { x: number, y: number }>;
  selectedStage: StageWithEventQueue | null;
  webSocketConnectionStatus: ReadyState;
  
  // Actions
  initialize: (data: CanvasData) => void;
  addStage: (stage: SuperplaneStage) => void;
  updateStage: (stage: SuperplaneStage) => void;
  addEventSource: (eventSource: EventSourceWithEvents) => void;
  updateEventSource: (eventSource: EventSourceWithEvents) => void;
  updateCanvas: (canvas: SuperplaneCanvas) => void;
  updateNodePosition: (nodeId: string, position: { x: number, y: number }) => void;
  approveStageEvent: (stageEventId: string, stageId: string) => void;
  selectStage: (stageId: string) => void;
  cleanSelectedStage: () => void;
  updateWebSocketConnectionStatus: (status: ReadyState) => void;
  
  // State and action for event handlers setup
  eventHandlersSetup: boolean;
  markEventHandlersAsSetup: () => void;
}

export type StageWithEventQueue = SuperplaneStage & {queue: Array<SuperplaneStageEvent>}
export type EventSourceWithEvents = SuperplaneStage & {events: Array<SuperplaneStageEvent>}