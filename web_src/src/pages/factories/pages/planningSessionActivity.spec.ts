import { describe, expect, it } from "vitest";

import { CREATE_WITH_AGENT_COPY } from "./createWithAgentCopy";
import {
  PLANNING_SESSION_AGENT_LINE_ID,
  PLANNING_SESSION_PHASE_ID,
  planningSessionPhase,
  planningSessionTalkLines,
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

  it("nests user and agent text under the agent so the live log can mix them in order", () => {
    const phase = planningSessionPhase({
      canvasId: "canvas-1",
      executionId: "execution-1",
      machineStatus: "waiting",
      messages: [
        { id: "agent-1", kind: "text", role: "agent", text: "Ready when you are." },
        { id: "user-1", kind: "text", role: "user", text: "Add a Size field" },
      ],
    });

    expect(phase.stream[0]?.id).toBe(PLANNING_SESSION_AGENT_LINE_ID);
    expect(phase.stream.slice(1)).toEqual(
      planningSessionTalkLines([
        { id: "agent-1", kind: "text", role: "agent", text: "Ready when you are." },
        { id: "user-1", kind: "text", role: "user", text: "Add a Size field" },
      ]),
    );
    expect(phase.stream[1]).toMatchObject({
      id: "agent-1",
      nodeId: PLANNING_SESSION_AGENT_LINE_ID,
      note: true,
      componentType: "note",
      componentName: "Ready when you are.",
    });
    expect(phase.stream[2]).toMatchObject({
      id: "user-1",
      nodeId: PLANNING_SESSION_AGENT_LINE_ID,
      note: true,
      componentType: "prompt",
      componentName: "Add a Size field",
    });
  });
});
