import { describe, expect, it } from "vitest";

import {
  PLANNING_SESSION_AGENT_LINE_ID,
  PLANNING_SESSION_PHASE_ID,
  planningSessionPhase,
} from "./planningSessionActivity";

describe("planningSessionPhase", () => {
  it("builds a running Automations phase before the runner has an execution", () => {
    const phase = planningSessionPhase({
      canvasId: "",
      executionId: "",
      messages: [],
      machineStatus: "starting",
    });

    expect(phase.id).toBe(PLANNING_SESSION_PHASE_ID);
    expect(phase.name).toBe("Planning session");
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
    const phase = planningSessionPhase({
      canvasId: "canvas-1",
      executionId: "execution-1",
      messages: [],
      machineStatus: "running",
    });

    expect(phase.appId).toBe("canvas-1");
    expect(phase.stream[0]?.executionId).toBe("execution-1");
    expect(phase.stream[0]?.status).toBe("running");
  });

  it("marks the agent line passed when the first analysis result already exists", () => {
    const phase = planningSessionPhase({
      canvasId: "canvas-1",
      executionId: "execution-1",
      messages: [],
      machineStatus: "passed",
    });

    expect(phase.status).toBe("passed");
    expect(phase.stream[0]?.status).toBe("passed");
  });

  it("marks the agent line failed when the machine stopped", () => {
    const phase = planningSessionPhase({
      canvasId: "canvas-1",
      executionId: "execution-1",
      messages: [],
      machineStatus: "failed",
    });

    expect(phase.status).toBe("failed");
    expect(phase.stream[0]?.status).toBe("failed");
  });

  it("keeps the agent line running while SuperPlane waits so the live log does not reset", () => {
    const phase = planningSessionPhase({
      canvasId: "canvas-1",
      executionId: "execution-1",
      messages: [],
      machineStatus: "waiting",
    });

    expect(phase.status).toBe("waiting");
    expect(phase.stream[0]?.status).toBe("running");
    expect(phase.stream[0]?.executionId).toBe("execution-1");
  });

  it("keeps stored messages out of the transient run stream", () => {
    const phase = planningSessionPhase({
      canvasId: "canvas-1",
      executionId: "execution-1",
      machineStatus: "waiting",
      messages: [
        { id: "agent-1", kind: "text", role: "agent", text: "Ready when you are." },
        { id: "user-1", kind: "text", role: "user", text: "Add a Size field" },
      ],
    });

    expect(phase.stream).toHaveLength(1);
    expect(phase.stream[0]?.id).toBe(PLANNING_SESSION_AGENT_LINE_ID);
  });
});
