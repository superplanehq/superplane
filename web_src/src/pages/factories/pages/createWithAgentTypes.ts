export type CreateWithAgentMachineStatus = "starting" | "running" | "waiting" | "passed" | "failed";

export type CreateWithAgentCreatedOrder = {
  id: string;
  key: string;
  title: string;
  description: string;
  /** Factory-scoped sequence number; builds the task permalink. */
  number?: number;
};

export type CreateWithAgentDraft = {
  title: string;
  description: string;
};

export type CreateWithAgentRightPane =
  | { kind: "empty" }
  | { kind: "draft"; draft: CreateWithAgentDraft }
  | { kind: "preview"; order: CreateWithAgentCreatedOrder };

export type CreateWithAgentTextMessage = {
  id: string;
  kind: "text";
  role: "user" | "agent";
  text: string;
  origin?: "survey";
  userId?: string;
  activityId?: string;
  /**
   * Epoch ms this message was created, when known. Lets the transcript
   * merge order this message against agent notes by true chronology
   * instead of guessing from wait-slot position.
   */
  createdAtMs?: number;
};

export type CreateWithAgentPlanMessage = {
  id: string;
  kind: "plan";
  role: "plan";
  score: number;
  createdAtMs?: number;
};

/**
 * A task the agent split off this draft. The transcript shows it as a card
 * at the moment it was created, so the user can open it right away.
 */
export type CreateWithAgentTaskMessage = {
  id: string;
  kind: "task";
  role: "task";
  workOrderId: string;
  key: string;
  title: string;
  number?: number;
  activityId?: string;
  createdAtMs?: number;
};

export type CreateWithAgentMessage =
  | CreateWithAgentTextMessage
  | CreateWithAgentPlanMessage
  | CreateWithAgentTaskMessage;

export type CreateWithAgentSurveyQuestion = {
  prompt: string;
  options: string[];
};

export type CreateWithAgentSurvey = {
  id?: string;
  questions: CreateWithAgentSurveyQuestion[];
};

export type CreateWithAgentView = {
  repository: string;
  machineStatus: CreateWithAgentMachineStatus;
  canvasId: string;
  canvasRunId: string;
  executionId: string;
  messages: CreateWithAgentMessage[];
  survey?: CreateWithAgentSurvey;
  composer: string;
  created: CreateWithAgentCreatedOrder[];
  right: CreateWithAgentRightPane;
  endConfirmOpen: boolean;
  selectableModelKey: string;
  refining: boolean;
  activities?: AgentActivity[];
};
import type { AgentActivity } from "./work-order-split-run/agentActivity";
