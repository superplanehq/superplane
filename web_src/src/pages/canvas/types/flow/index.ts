import { Node, Edge } from "@xyflow/react";
import { 
  SuperplaneStageEvent,
  SuperplaneStageEventState,
  SuperplaneExecutorSpec,
  SuperplaneExecutorSpecType,
  SuperplaneConnection,
  ConnectionFilterOperator,
  SuperplaneCondition,
  SuperplaneConditionType,
  SuperplaneInputDefinition,
  SuperplaneOutputDefinition
} from "@/api-client/types.gen";

export type AllNodeType = EventSourceNodeType | StageNodeType;
export type EdgeType = Edge;

// Event source node
export type EventSourceNodeData = {
  id: string;
  name: string;
  timestamp: string;
}

export type EventSourceNodeType = Node<EventSourceNodeData, 'event_source'>;

// Stage node 
export type StageData = {
  label: string;
  labels: string[];
  status?: string;
  timestamp?: string;
  icon?: string;
  queues: SuperplaneStageEvent[];
  connections: SuperplaneConnection[];
  conditions: SuperplaneCondition[];
  inputs: SuperplaneInputDefinition[];
  outputs: SuperplaneOutputDefinition[];
  executorSpec: SuperplaneExecutorSpec;
  approveStageEvent: (event: SuperplaneStageEvent) => void;
}

export type StageNodeType = Node<StageData, 'stage'>;

export type HandleType = 'source' | 'target';

export type HandleProps = {
  type: HandleType;
  conditions?: SuperplaneCondition[];
  connections?: SuperplaneConnection[];
}

export {
  SuperplaneStageEventState as QueueState,
  ConnectionFilterOperator,
  SuperplaneConditionType as ConditionType,
  SuperplaneExecutorSpecType as ExecutorSpecType
};

export interface FlowEdge extends Edge {
  id: string;
  source: string;
  target: string;
}

export interface LayoutedFlowData {
  layoutedNodes: AllNodeType[];
  flowEdges: FlowEdge[];
}
