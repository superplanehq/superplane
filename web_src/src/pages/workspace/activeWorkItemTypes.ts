export type WorkRunState = "awaiting-approval" | "running" | "paused" | "stopped";
export type TimelineEventKind = "request" | "agent" | "user" | "system" | "approval" | "progress";
export type PlanStepState = "complete" | "changed" | "queued";

export interface WorkTimelineEvent {
  id: string;
  time: string;
  actor: string;
  title: string;
  description: string;
  kind: TimelineEventKind;
  details?: string[];
}

export interface WorkPlanStep {
  id: string;
  title: string;
  description: string;
  state: PlanStepState;
}

export interface WorkPlan {
  version: number;
  summary: string;
  changesFromPrevious: string[];
  steps: WorkPlanStep[];
}

export interface WorkAgent {
  name: string;
  role: string;
  status: string;
  active: boolean;
}

export interface ActiveWorkItemData {
  id: string;
  title: string;
  goal: string;
  projectName: string;
  repository: string;
  branch: string;
  baseBranch: string;
  elapsed: string;
  state: WorkRunState;
  filesChanged: number;
  checksPassed: number;
  checksTotal: number;
  pullRequest?: string;
  timeline: WorkTimelineEvent[];
  plan: WorkPlan;
  agents: WorkAgent[];
}
