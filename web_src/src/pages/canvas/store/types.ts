import { CanvasData } from "../types";
import { SuperplaneCanvas, SuperplaneStage, SuperplaneEventSource, SuperplaneStageEvent } from "@/api-client/types.gen";
import { ReadyState } from "react-use-websocket";
import { AllNodeType, EdgeType } from "../types/flow";
import { OnEdgesChange, OnNodesChange, Connection } from "@xyflow/react";

// Define the store state type
export interface CanvasState {
  canvas: SuperplaneCanvas;
  stages: StageWithEventQueue[];
  event_sources: SuperplaneEventSource[];
  nodePositions: Record<string, { x: number, y: number }>;
  selectedStage: StageWithEventQueue | null;
  webSocketConnectionStatus: ReadyState;
  
  // Actions
  initialize: (data: CanvasData) => void;
  addStage: (stage: SuperplaneStage) => void;
  updateStage: (stage: SuperplaneStage) => void;
  addEventSource: (eventSource: SuperplaneEventSource) => void;
  updateEventSource: (eventSource: SuperplaneEventSource) => void;
  updateCanvas: (canvas: SuperplaneCanvas) => void;
  updateNodePosition: (nodeId: string, position: { x: number, y: number }) => void;
  approveStageEvent: (stageEventId: string, stageId: string) => void;
  selectStage: (stageId: string) => void;
  cleanSelectedStage: () => void;
  updateWebSocketConnectionStatus: (status: ReadyState) => void;

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
  syncToReactFlow: () => void;
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

export type StageWithEventQueue = SuperplaneStage & {queue: Array<SuperplaneStageEvent>}