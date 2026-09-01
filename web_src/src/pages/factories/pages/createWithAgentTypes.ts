export type CreateWithAgentMachineStatus = "starting" | "running" | "waiting";

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
};

export type CreateWithAgentView = {
  repository: string;
  machineStatus: CreateWithAgentMachineStatus;
  canvasId: string;
  executionId: string;
  messages: CreateWithAgentMessage[];
  composer: string;
  created: CreateWithAgentCreatedOrder[];
  right: CreateWithAgentRightPane;
  endConfirmOpen: boolean;
};
