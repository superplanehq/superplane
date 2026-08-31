import type { WorkOrderSurveyView } from "../lib/workOrderSurvey";

export type CreateWithAgentMachineStatus = "starting" | "running";

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
  | { kind: "list" }
  | { kind: "preview"; order: CreateWithAgentCreatedOrder };

export type CreateWithAgentTextMessage = {
  id: string;
  kind: "text";
  role: "user" | "agent";
  text: string;
};

export type CreateWithAgentSurveyMessage = {
  id: string;
  kind: "survey";
  survey: WorkOrderSurveyView;
  answered?: boolean;
};

export type CreateWithAgentMessage = CreateWithAgentTextMessage | CreateWithAgentSurveyMessage;

export type CreateWithAgentView = {
  repository: string;
  machineStatus: CreateWithAgentMachineStatus;
  messages: CreateWithAgentMessage[];
  composer: string;
  created: CreateWithAgentCreatedOrder[];
  right: CreateWithAgentRightPane;
  endConfirmOpen: boolean;
};
