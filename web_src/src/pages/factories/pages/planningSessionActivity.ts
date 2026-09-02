import { CREATE_WITH_AGENT_COPY } from "./createWithAgentCopy";
import type { CreateWithAgentMessage, CreateWithAgentView } from "./createWithAgentTypes";
import type { SplitRunPhase, SplitRunStreamLine } from "./work-order-split-run/splitRunMocks";

export const PLANNING_SESSION_PHASE_ID = "planning";
export const PLANNING_SESSION_AGENT_LINE_ID = "agent";

const PLANNING_AGENT_NAME = "Agent";

export function planningSessionPhase(
  view: Pick<CreateWithAgentView, "canvasId" | "executionId" | "messages" | "machineStatus">,
): SplitRunPhase {
  return {
    id: PLANNING_SESSION_PHASE_ID,
    name: CREATE_WITH_AGENT_COPY.menu,
    status: view.machineStatus === "waiting" ? "waiting" : "running",
    duration: "",
    componentName: PLANNING_AGENT_NAME,
    artifacts: [],
    stream: [planningAgentStreamLine(view), ...planningSessionTalkLines(view.messages)],
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
    // Stay running while the machine is on. A waiting status tears down the
    // live log stream and the full log flickers back as collapsed tool calls.
    status: "running",
  };
}

export function planningSessionTalkLines(messages: CreateWithAgentMessage[]): SplitRunStreamLine[] {
  return messages.flatMap((message) => {
    if (message.kind !== "text") {
      return [];
    }
    const text = message.text.trim();
    if (!text) {
      return [];
    }
    return [
      {
        id: message.id,
        nodeId: PLANNING_SESSION_AGENT_LINE_ID,
        at: "",
        note: true,
        componentName: text,
        componentType: message.role === "user" ? "prompt" : "note",
        status: "passed" as const,
        detail: message.role === "user" ? text : undefined,
      },
    ];
  });
}
