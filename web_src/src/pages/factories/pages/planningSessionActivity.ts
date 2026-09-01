import { CREATE_WITH_AGENT_COPY } from "./createWithAgentCopy";
import type { CreateWithAgentView } from "./createWithAgentTypes";
import type { SplitRunPhase, SplitRunStreamLine } from "./work-order-split-run/splitRunMocks";

export const PLANNING_SESSION_PHASE_ID = "planning";
export const PLANNING_SESSION_AGENT_LINE_ID = "agent";

const PLANNING_AGENT_NAME = "Agent";

export function planningSessionPhase(
  view: Pick<CreateWithAgentView, "canvasId" | "executionId">,
): SplitRunPhase {
  return {
    id: PLANNING_SESSION_PHASE_ID,
    name: CREATE_WITH_AGENT_COPY.menu,
    status: "running",
    duration: "",
    componentName: PLANNING_AGENT_NAME,
    artifacts: [],
    stream: [planningAgentStreamLine(view)],
    canvasSteps: [],
    appId: view.canvasId || undefined,
  };
}

function planningAgentStreamLine(view: Pick<CreateWithAgentView, "executionId">): SplitRunStreamLine {
  return {
    id: PLANNING_SESSION_AGENT_LINE_ID,
    nodeId: PLANNING_SESSION_AGENT_LINE_ID,
    at: "",
    componentName: PLANNING_AGENT_NAME,
    componentType: "Run Claude Code",
    component: "runnerClaudeCode",
    executionId: view.executionId || undefined,
    status: "running",
  };
}
