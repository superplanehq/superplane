import { CanvasData } from "../types";
import { SuperplaneCanvas, SuperplaneConnectionGroup, SuperplaneEventSource, SuperplaneExecution, SuperplaneStage, SuperplaneStageEvent } from "@/api-client/types.gen";
import { ReadyState } from "react-use-websocket";
import { AllNodeType, EdgeType } from "../types/flow";
import { OnEdgesChange, OnNodesChange, Connection } from "@xyflow/react";

// Define the store state type
export interface CanvasState {
  canvas: SuperplaneCanvas;
  stages: StageWithEventQueue[];
  eventSources: EventSourceWithEvents[];
  connectionGroups: SuperplaneConnectionGroup[];
  nodePositions: Record<string, { x: number, y: number }>;
  selectedStageId: string | null;
  editingStageId: string | null;
  webSocketConnectionStatus: ReadyState;
  
  // Actions
  initialize: (data: CanvasData) => void;
  addStage: (stage: SuperplaneStage, draft?: boolean) => void;
  addConnectionGroup: (connectionGroup: SuperplaneConnectionGroup) => void;
  updateStage: (stage: SuperplaneStage) => void;
  addEventSource: (eventSource: EventSourceWithEvents) => void;
  updateEventSource: (eventSource: EventSourceWithEvents) => void;
  updateCanvas: (canvas: SuperplaneCanvas) => void;
  updateNodePosition: (nodeId: string, position: { x: number, y: number }) => void;
  approveStageEvent: (stageEventId: string, stageId: string) => void;
  selectStageId: (stageId: string) => void;
  cleanSelectedStageId: () => void;
  setEditingStage: (stageId: string | null) => void;
  updateWebSocketConnectionStatus: (status: ReadyState) => void;
  syncStageEvents: (canvasId: string, stageId: string) => Promise<void>;

  // flow fields
  nodes: AllNodeType[];
  edges: EdgeType[];
  handleDragging:
  | {
      source: string | undefined;
      sourceHandle: string | undefined;
      target: string | undefined;
      targetHandle: string | undefined;
      type: string;
      color: string;
    }
  | undefined;
  // flow actions
  syncToReactFlow: (options?: { autoLayout?: boolean }) => void;
  fitViewNode: (nodeId: string) => void;
  onNodesChange: OnNodesChange<AllNodeType>;
  onEdgesChange: OnEdgesChange<EdgeType>;
  onConnect: (connection: Connection) => void;
  setNodes: (nodes: AllNodeType[]) => void;
  setHandleDragging: (
    data:
      | {
          source: string | undefined;
          sourceHandle: string | undefined;
          target: string | undefined;
          targetHandle: string | undefined;
          type: string;
          color: string;
        }
      | undefined,
  ) => void;
}

export type StageWithEventQueue = SuperplaneStage & {queue: Array<SuperplaneStageEvent>; isDraft?: boolean}
export type EventSourceWithEvents = SuperplaneEventSource & {events: Array<SuperplaneStageEvent>}
export type ExecutionWithEvent = SuperplaneExecution & {event: SuperplaneStageEvent}