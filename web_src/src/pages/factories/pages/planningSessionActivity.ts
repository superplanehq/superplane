import type { CreateWithAgentView } from "./createWithAgentTypes";
import type { SplitRunPhase, SplitRunStreamLine } from "./work-order-split-run/splitRunMocks";

export const PLANNING_SESSION_PHASE_ID = "planning";
export const PLANNING_SESSION_AGENT_LINE_ID = "agent";

const PLANNING_AGENT_NAME = "Agent";

export function planningSessionPhase(
  view: Pick<CreateWithAgentView, "canvasId" | "executionId" | "messages" | "machineStatus">,
): SplitRunPhase {
  return {
    id: PLANNING_SESSION_PHASE_ID,
    name: "Planning session",
    status: planningSessionPhaseStatus(view.machineStatus),
    duration: "",
    componentName: PLANNING_AGENT_NAME,
    artifacts: [],
    stream: [planningAgentStreamLine(view)],
    canvasSteps: [],
    appId: view.canvasId || undefined,
  };
}

function planningSessionPhaseStatus(machineStatus: CreateWithAgentView["machineStatus"]): SplitRunPhase["status"] {
  if (machineStatus === "failed") {
    return "failed";
  }
  if (machineStatus === "passed") {
    return "passed";
  }
  if (machineStatus === "waiting") {
    return "waiting";
  }
  return "running";
}

function planningAgentLineStatus(machineStatus: CreateWithAgentView["machineStatus"]): SplitRunStreamLine["status"] {
  if (machineStatus === "failed") {
    return "failed";
  }
  if (machineStatus === "passed") {
    return "passed";
  }
  return "running";
}

function planningAgentStreamLine(view: Pick<CreateWithAgentView, "executionId" | "machineStatus">): SplitRunStreamLine {
  return {
    id: PLANNING_SESSION_AGENT_LINE_ID,
    nodeId: PLANNING_SESSION_AGENT_LINE_ID,
    at: "",
    componentName: PLANNING_AGENT_NAME,
    componentType: "Run Claude Code",
    component: "runnerClaudeCode",
    executionId: view.executionId || undefined,
    // Stay running while the machine is on. A waiting status tears down the
    // live log stream and the full log flickers back as collapsed tool calls.
    status: planningAgentLineStatus(view.machineStatus),
  };
}
