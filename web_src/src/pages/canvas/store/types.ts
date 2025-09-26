import { CanvasData } from "../types";
import { SuperplaneCanvas, SuperplaneConnectionGroup, SuperplaneEventSource, SuperplaneStage, SuperplaneStageEvent, SuperplaneEvent, SuperplaneExecution, EventSourceEventType } from "@/api-client/types.gen";
import { ReadyState } from "react-use-websocket";
import { AllNodeType, EdgeType } from "../types/flow";
import { OnEdgesChange, OnNodesChange, Connection } from "@xyflow/react";
import { EventSourceConfig } from "../components/ComponentSidebar";

// Define the store state type
export interface CanvasState {
  canvas: SuperplaneCanvas;
  canvasId: string;
  stages: Stage[];
  eventSources: EventSourceWithEvents[];
  connectionGroups: ConnectionGroupWithEvents[];
  nodePositions: Record<string, { x: number, y: number }>;
  selectedStageId: string | null;
  selectedEventSourceId: string | null;
  selectedConnectionGroupId: string | null;
  focusedNodeId: string | null;
  editingStageId: string | null;
  editingEventSourceId: string | null;
  editingConnectionGroupId: string | null;
  webSocketConnectionStatus: ReadyState;
  eventSourceKeys: Record<string, string>;
  
  // Actions
  initialize: (data: CanvasData) => void;
  addStage: (stage: Stage, draft?: boolean, autoLayout?: boolean) => void;
  removeStage: (stageId: string) => void;
  addConnectionGroup: (connectionGroup: ConnectionGroupWithEvents, autoLayout?: boolean) => void;
  removeConnectionGroup: (connectionGroupId: string) => void;
  updateConnectionGroup: (connectionGroup: ConnectionGroupWithEvents) => void;
  updateStage: (stage: Stage) => void;
  addEventSource: (eventSource: EventSourceWithEvents, autoLayout?: boolean) => void;
  removeEventSource: (eventSourceId: string) => void;
  updateEventSource: (eventSource: EventSourceWithEvents) => void;
  updateCanvas: (canvas: SuperplaneCanvas) => void;
  updateNodePosition: (nodeId: string, position: { x: number, y: number }) => void;
  approveStageEvent: (stageEventId: string, stageId: string) => Promise<any>;
  discardStageEvent: (stageEventId: string, stageId: string) => Promise<void>;
  cancelStageExecution: (executionId: string, stageId: string) => Promise<void>;
  selectStageId: (stageId: string) => void;
  cleanSelectedStageId: () => void;
  selectEventSourceId: (eventSourceId: string) => void;
  cleanSelectedEventSourceId: () => void;
  selectConnectionGroupId: (connectionGroupId: string) => void;
  cleanSelectedConnectionGroupId: () => void;
  setFocusedNodeId: (stageId: string | null) => void;
  cleanFocusedNodeId: () => void;
  setEditingStage: (stageId: string | null) => void;
  setEditingEventSource: (eventSourceId: string | null) => void;
  setEditingConnectionGroup: (connectionGroupId: string | null) => void;
  updateWebSocketConnectionStatus: (status: ReadyState) => void;
  syncStageEvents: (canvasId: string, stageId: string) => Promise<void>;
  syncStageExecutions: (canvasId: string, stageId: string) => Promise<void>;
  syncEventSourceEvents: (canvasId: string, eventSourceId: string) => Promise<void>;
  syncStagePlainEvents: (canvasId: string, stageId: string) => Promise<void>;
  syncConnectionGroupPlainEvents: (canvasId: string, connectionGroupId: string) => Promise<void>;

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
  lockedNodes: boolean;
  // flow actions
  syncToReactFlow: (options?: { autoLayout?: boolean }) => void;
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

  updateEventSourceKey: (eventSourceId: string, key: string) => void;
  resetEventSourceKey: (eventSourceId: string) => void;
  setLockedNodes: (locked: boolean) => void;
  updateConnectionSourceNames: (oldName: string, newName: string) => void;
}

export type Stage = SuperplaneStage & {queue: Array<SuperplaneStageEvent>; executions: Array<SuperplaneExecution>; isDraft?: boolean}
export type EventSourceWithEvents = SuperplaneEventSource & {events: Array<SuperplaneEvent>; eventSourceConfig?: EventSourceConfig; eventFilters?: Array<EventSourceEventType>; isDuplicate?: boolean}
export type ConnectionGroupWithEvents = SuperplaneConnectionGroup & {events: Array<SuperplaneEvent>}
