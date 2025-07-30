import { Node, Edge } from "@xyflow/react";
import { 
  SuperplaneStageEvent,
  SuperplaneStageEventState,
  SuperplaneExecutorSpec,
  SuperplaneExecutorSpecType,
  SuperplaneConnection,
  SuperplaneFilterOperator,
  SuperplaneCondition,
  SuperplaneConditionType,
  SuperplaneInputDefinition,
  SuperplaneOutputDefinition,
  SpecGroupBy
} from "@/api-client/types.gen";

export type AllNodeType = EventSourceNodeType | StageNodeType | ConnectionGroupNodeType;
export type EdgeType = Edge;

// Connection group node
export type ConnectionGroupNodeData = {
  id: string;
  name: string;
  connections: SuperplaneConnection[];
  groupBy: SpecGroupBy;
}

export type ConnectionGroupNodeType = Node<ConnectionGroupNodeData, 'connection_group'>;

// Event source node
export type EventSourceNodeData = {
  id: string;
  name: string;
  events: SuperplaneStageEvent[];
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
  isDraft?: boolean;
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
  SuperplaneFilterOperator,
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
