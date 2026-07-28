export type FactoryStageState = "complete" | "active" | "queued";
export type WorkItemStatus = "attention" | "running" | "done" | "failed";
export type FactoryTab = "overview" | "work-orders" | "automations" | "velocity";
export type AutomationStatus = "active" | "paused" | "draft";
export type VelocityCohortId = "team" | "human" | "factory";

export interface WorkspaceProject {
  name: string;
  description?: string;
  status: "operational" | "degraded" | "offline";
  repositories: FactoryRepository[];
}

export interface FactoryRepository {
  id: string;
  name: string;
  defaultBranch: string;
}

export interface FactoryMetric {
  id: "throughput" | "success" | "active" | "cost";
  label: string;
  value: string;
  detail: string;
  tone: "positive" | "neutral";
}

export interface FactoryAutomation {
  id: string;
  canvasId: string;
  name: string;
  description: string;
  trigger: string;
  status: AutomationStatus;
  activeWorkItems: number;
  successRate: string;
  lastRun: string;
}

export interface VelocityCohortValue {
  value: string;
  detail: string;
  trend?: string;
  unavailable?: boolean;
}

export interface VelocityMetric {
  id: "throughput" | "cycle-time" | "success-rate" | "execution-cost";
  label: string;
  description: string;
  values: Record<VelocityCohortId, VelocityCohortValue>;
}

export interface VelocityCohort {
  id: VelocityCohortId;
  label: string;
  description: string;
}

export interface VelocityTrendPoint {
  label: string;
  team: number;
  human: number;
  factory: number;
}

export interface RepositoryVelocity {
  period: string;
  defaultRepositoryId: string;
  cohorts: VelocityCohort[];
  metrics: VelocityMetric[];
  trend: VelocityTrendPoint[];
  costBreakdown: {
    tokens: string;
    compute: string;
    total: string;
    note: string;
  };
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
  detail: string;
  updatedAt: string;
}

export interface ThroughputPoint {
  label: string;
  value: number;
}

export interface WorkspacePageData {
  project: WorkspaceProject;
  repositoryVelocity: RepositoryVelocity;
  metrics: FactoryMetric[];
  automations: FactoryAutomation[];
  stages: FactoryStage[];
  workItems: FactoryWorkItem[];
  throughput: ThroughputPoint[];
}

export interface CreateWorkRequest {
  title: string;
  goal: string;
}
