export type CreateWithAgentMachineStatus = "starting" | "running" | "waiting" | "failed";

export type CreateWithAgentCreatedOrder = {
  id: string;
  key: string;
  title: string;
  description: string;
};

export type CreateWithAgentDraft = {
  title: string;
  description: string;
};

export type CreateWithAgentRightPane =
  | { kind: "empty" }
  | { kind: "draft"; draft: CreateWithAgentDraft }
  | { kind: "preview"; order: CreateWithAgentCreatedOrder };

export type CreateWithAgentMessage = {
  id: string;
  kind: "text";
  role: "user" | "agent";
  text: string;
  origin?: "survey";
  /**
   * Epoch ms this message was created, when known. Lets the transcript
   * merge order this message against agent notes by true chronology
   * instead of guessing from wait-slot position.
   */
  createdAtMs?: number;
};

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
};
