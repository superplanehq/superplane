import { Node, Edge } from "@xyflow/react";
import { 
  SuperplaneStageEvent,
  SuperplaneStageEventState,
  SuperplaneRunTemplate,
  SuperplaneRunTemplateType,
  SuperplaneConnection,
  ConnectionFilterOperator,
  SuperplaneCondition,
  SuperplaneConditionType
} from "@/api-client/types.gen";

export type AllNodeType = EventSourceNodeType | StageNodeType;
export type EdgeType = Edge;

// Event source node
export type EventSourceNodeData = {
  id: string;
  repoName: string;
  repoUrl: string;
  eventType: string;
  release: string;
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
  runTemplate: SuperplaneRunTemplate;
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
  SuperplaneRunTemplateType as RunTemplateType
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