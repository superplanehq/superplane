export type WorkOrderState = "draft" | "ready" | "running" | "successful" | "unsuccessful";

export interface PullRequest {
  repository: string;
  number: number;
  url: string;
}

export interface SoftwareFactory {
  id: string;
  name: string;
  description?: string;
}

export type WorkOrderAutomationState = "planned" | "running" | "done" | "failed";

export interface WorkOrderAutomation {
  id: string;
  name: string;
  state: WorkOrderAutomationState;
}

export interface WorkOrder {
  id: string;
  title: string;
  description: string;
  state: WorkOrderState;
  createdByUserId: string;
  createdByName: string;
  createdAt: string;
  updatedAt: string;
  automations: WorkOrderAutomation[];
  primaryPullRequest?: PullRequest;
}

export interface Automation {
  id: string;
  name: string;
  description: string;
  state: "idle" | "running" | "paused";
  runningCount: number;
  queuedCount: number;
  lastRunAt?: string;
}

export type WorkOrderEventKind = "created" | "approved" | "started" | "pull-request" | "outcome";

export interface WorkOrderEvent {
  id: string;
  kind: WorkOrderEventKind;
  summary: string;
  actor: string;
  occurredAt: string;
  detail?: string;
  pullRequest?: PullRequest;
  outcome?: Extract<WorkOrderState, "successful" | "unsuccessful">;
}

export interface NewWorkOrderInput {
  title: string;
  description: string;
  automationIds: string[];
}

export interface NewFactoryInput {
  name: string;
  description: string;
}
