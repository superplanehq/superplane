import { describe, expect, it } from "vitest";

import { CREATE_WITH_AGENT_COPY } from "./createWithAgentCopy";
import { PLANNING_SESSION_AGENT_LINE_ID, PLANNING_SESSION_PHASE_ID, planningSessionPhase } from "./planningSessionActivity";

describe("planningSessionPhase", () => {
  it("builds a running Automations phase before the runner has an execution", () => {
    const phase = planningSessionPhase({ canvasId: "", executionId: "" });

    expect(phase.id).toBe(PLANNING_SESSION_PHASE_ID);
    expect(phase.name).toBe(CREATE_WITH_AGENT_COPY.menu);
    expect(phase.status).toBe("running");
    expect(phase.appId).toBeUndefined();
    expect(phase.stream).toEqual([
      {
        id: PLANNING_SESSION_AGENT_LINE_ID,
        nodeId: PLANNING_SESSION_AGENT_LINE_ID,
        at: "",
        componentName: "Agent",
        componentType: "Run Claude Code",
        component: "runnerClaudeCode",
        executionId: undefined,
        status: "running",
      },
    ]);
  });

  it("attaches the runner execution so live notes fill the stream", () => {
    const phase = planningSessionPhase({ canvasId: "canvas-1", executionId: "execution-1" });

    expect(phase.appId).toBe("canvas-1");
    expect(phase.stream[0]?.executionId).toBe("execution-1");
  });
});
