import { CREATE_WITH_AGENT_COPY } from "./createWithAgentCopy";
import type { CreateWithAgentView } from "./createWithAgentTypes";

export const CREATE_WITH_AGENT_DEMO_REPOSITORY = "acme/payments";

export function emptyCreateWithAgentView(repository = CREATE_WITH_AGENT_DEMO_REPOSITORY): CreateWithAgentView {
  return {
    repository,
    machineStatus: "starting",
    canvasId: "",
    canvasRunId: "",
    executionId: "",
    messages: [],
    composer: "",
    created: [],
    right: { kind: "empty" },
    endConfirmOpen: false,
    selectableModelKey: "hosted::anthropic::claude-sonnet-4-6",
    refining: false,
  };
}

export function runningCreateWithAgentView(overrides: Partial<CreateWithAgentView> = {}): CreateWithAgentView {
  return {
    ...emptyCreateWithAgentView(),
    machineStatus: "running",
    canvasId: "canvas-demo",
    canvasRunId: "run-demo",
    executionId: "execution-demo",
    messages: [{ id: "greet", kind: "text", role: "agent", text: CREATE_WITH_AGENT_COPY.greeting }],
    ...overrides,
  };
}
