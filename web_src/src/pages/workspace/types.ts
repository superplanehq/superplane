export type FactoryStageState = "complete" | "active" | "queued";
export type WorkItemStatus = "running" | "waiting" | "attention";

export interface WorkspaceProject {
  name: string;
  factoryName: string;
  repository: string;
  defaultBranch: string;
}

export interface FactoryMetric {
  label: string;
  value: string;
  detail: string;
  tone: "positive" | "neutral";
}

export interface FactoryStage {
  id: string;
  label: string;
  detail: string;
  state: FactoryStageState;
}

export interface FactoryWorkItem {
  id: string;
  title: string;
  stage: string;
  branch: string;
  agentCount: number;
  elapsed: string;
  status: WorkItemStatus;
}

export interface ThroughputPoint {
  label: string;
  value: number;
}

export interface DeliveryEvent {
  id: string;
  title: string;
  reference: string;
  timestamp: string;
}

export interface WorkspacePageData {
  project: WorkspaceProject;
  metrics: FactoryMetric[];
  stages: FactoryStage[];
  workItems: FactoryWorkItem[];
  throughput: ThroughputPoint[];
  recentDeliveries: DeliveryEvent[];
}

export interface CreateWorkRequest {
  title: string;
  goal: string;
}
