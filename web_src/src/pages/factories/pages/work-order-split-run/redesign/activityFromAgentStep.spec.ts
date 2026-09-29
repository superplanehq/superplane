import { describe, expect, it } from "bun:test";

import { activityFromAgentStep } from "./activityFromAgentStep";
import type { AgentStep } from "./automationsViewModel";

function step(overrides: Partial<AgentStep>): AgentStep {
  return {
    id: "step-1",
    title: "Clone Repo",
    type: "bash",
    status: "passed",
    summary: "",
    toolCount: 0,
    events: [],
    ...overrides,
  };
}

describe("activityFromAgentStep", () => {
  it("turns a bash step output into a command tool", () => {
    const activity = activityFromAgentStep(step({ output: "git clone https://example.com/repo.git" }));

    expect(activity?.items).toEqual([
      expect.objectContaining({
        type: "tool",
        kind: "bash",
        input: "git clone https://example.com/repo.git",
      }),
    ]);
  });

  it("keeps notes and skips a Thinking placeholder", () => {
    const activity = activityFromAgentStep(
      step({
        title: "Implementation",
        type: "prompt",
        events: [
          { kind: "note", id: "n1", text: "Thinking" },
          { kind: "note", id: "n2", text: "Let me read the factory handler." },
        ],
      }),
    );

    expect(activity?.items).toEqual([
      expect.objectContaining({
        type: "content",
        kind: "assistant",
        text: "Let me read the factory handler.",
        status: "passed",
      }),
    ]);
  });

  it("keeps notes settled while the step is still running", () => {
    const activity = activityFromAgentStep(
      step({
        title: "Implementation",
        type: "prompt",
        status: "running",
        events: [{ kind: "note", id: "n1", text: "Let me read the factory handler." }],
      }),
    );

    expect(activity?.status).toBe("running");
    expect(activity?.items).toEqual([
      expect.objectContaining({ type: "content", status: "passed", text: "Let me read the factory handler." }),
    ]);
  });
});
