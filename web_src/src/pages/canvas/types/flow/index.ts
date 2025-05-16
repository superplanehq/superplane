import { Node, Edge } from "@xyflow/react";

export type AllNodeType = EventSourceNodeType | StageNodeType;
export type EdgeType = Edge;

// Event source node
type LastEvent = { type: string; release: string; timestamp: string };

export type EventSourceNodeData = {
  id: string;
  repoName: string;
  repoUrl: string;
  lastEvent: LastEvent;
}

export type EventSourceNodeType = Node<EventSourceNodeData, 'event_source'>;

// Stage node 
export type StageData = {
  label: string;
  labels: string[];
  status?: string;
  timestamp?: string;
  icon?: string;
  queues: Event[];
  connections: Connection[];
  conditions: Condition[];
  run_template: RunTemplate;
  approve_stage_event: (event: Event) => void;
}

export type Event = {
  id: string;
  state: string;
  source_id: string;
  created_at: string;
  source_type: string;
  state_reason: string;
  approvals: any[];
}

export enum QueueState {
  PENDING = 'STATE_PENDING',
  WAITING = 'STATE_WAITING',
  PROCESSED = 'STATE_PROCESSED',
}

export type RunTemplate = {
  type: RunTemplateType;
  semaphore: SemaphoreRunTemplate;
}

export enum RunTemplateType {
  SEMAPHORE = 'TYPE_SEMAPHORE',
}

export type SemaphoreRunTemplate = {
  project_id: string;
  branch: string;
  pipeline_file: string;
  task_id: string;
  parameters: Array<Record<string, string>>;
}

export type Connection = {
  name: string;
  type: string;
  filters: string[];
  filter_operator: ConnectionFilterOperator;
}

export enum ConnectionFilterOperator {
  AND = 'FILTER_OPERATOR_AND',
  OR = 'FILTER_OPERATOR_OR'
}

export type Condition = {
  type: ConditionType;
  approval: Approval;
  time_window: TimeWindow;
}

export enum ConditionType {
  APPROVAL = 'CONDITION_TYPE_APPROVAL',
  TIME_WINDOW = 'CONDITION_TYPE_TIME_WINDOW'
}

export type Approval = {
  count: number;
}

export type TimeWindow = {
  start: string;
  end: string;
  timezone: string;
  week_days: string[];
}


export type StageNodeType = Node<StageData, 'stage'>;

export type HandleType = 'source' | 'target';

export type HandleProps = {
  type: HandleType;
  conditions?: Condition[];
  connections?: Connection[];
}