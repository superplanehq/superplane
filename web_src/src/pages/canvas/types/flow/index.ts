import { Node, Edge } from "@xyflow/react";
import {
  SuperplaneStageEvent,
  SuperplaneEvent,
  SuperplaneStageEventState,
  SuperplaneConnection,
  SuperplaneFilterOperator,
  SuperplaneCondition,
  SuperplaneConditionType,
  SuperplaneInputDefinition,
  SuperplaneOutputDefinition,
  SuperplaneExecutor,
  SuperplaneValueDefinition,
  SuperplaneInputMapping,
  SpecGroupBy,
  IntegrationsIntegrationRef,
  IntegrationsResourceRef,
  EventSourceEventType,
  SuperplaneEventSourceSchedule
} from "@/api-client/types.gen";

export type AllNodeType = EventSourceNodeType | StageNodeType | ConnectionGroupNodeType;
export type EdgeType = Edge;

// Connection group node
export type ConnectionGroupNodeData = {
  id: string;
  name: string;
  description?: string;
  connections: SuperplaneConnection[];
  groupBy: SpecGroupBy;
}

export type ConnectionGroupNodeType = Node<ConnectionGroupNodeData, 'connection_group'>;

// Event source node
export type EventSourceNodeData = {
  id: string;
  name: string;
  description?: string;
  events: SuperplaneEvent[];
  eventFilters?: EventSourceEventType[];
  integration: IntegrationsIntegrationRef | null;
  resource: IntegrationsResourceRef | null;
  eventSourceType?: string;
  schedule?: SuperplaneEventSourceSchedule | null;
}

export type EventSourceNodeType = Node<EventSourceNodeData, 'event_source'>;

// Stage node 
export type StageData = {
  name: string;
  description?: string;
  labels: string[];
  status?: string;
  timestamp?: string;
  icon?: string;
  queues: SuperplaneStageEvent[];
  connections: SuperplaneConnection[];
  conditions: SuperplaneCondition[];
  inputs: SuperplaneInputDefinition[];
  outputs: SuperplaneOutputDefinition[];
  secrets: SuperplaneValueDefinition[];
  inputMappings?: SuperplaneInputMapping[];
  executor?: SuperplaneExecutor;
  dryRun?: boolean;
  spec: object;
  approveStageEvent: (event: SuperplaneStageEvent) => void;
  isDraft?: boolean;
}

export type StageNodeType = Node<StageData, 'stage'>;

export type HandleType = 'source' | 'target';

export type HandleProps = {
  type: HandleType;
  conditions?: SuperplaneCondition[];
  connections?: SuperplaneConnection[];
  internalPadding?: boolean;
}

export {
  SuperplaneStageEventState as QueueState,
  SuperplaneFilterOperator,
  SuperplaneConditionType as ConditionType
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
