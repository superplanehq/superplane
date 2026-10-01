import { describe, expect, it } from "bun:test";

import { activityFromAgentStep, activityFromTranscript } from "./activityFromAgentStep";
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

  it("keeps the command detail when the step also has child events", () => {
    const activity = activityFromAgentStep(
      step({
        title: "Clone Repo",
        output: "git clone https://example.com/repo.git",
        events: [{ kind: "note", id: "n1", text: "Cloned the repository." }],
      }),
    );

    expect(activity?.items).toEqual([
      expect.objectContaining({ type: "tool", input: "git clone https://example.com/repo.git" }),
      expect.objectContaining({ type: "content", text: "Cloned the repository." }),
    ]);
  });

  it("joins earlier turns into the open transcript", () => {
    const first = activityFromAgentStep(
      step({
        id: "turn-1",
        events: [{ kind: "note", id: "n1", text: "First I inspect the host." }],
      }),
    );
    const second = activityFromAgentStep(
      step({
        id: "turn-2",
        status: "running",
        events: [{ kind: "note", id: "n2", text: "Then I install bun." }],
      }),
    );

    const activity = activityFromTranscript([first!, second!]);

    expect(activity?.id).toBe("live-transcript");
    expect(activity?.status).toBe("running");
    expect(activity?.items.map((item) => ("text" in item ? item.text : ""))).toEqual([
      "First I inspect the host.",
      "Then I install bun.",
    ]);
  });
});
