import { describe, expect, it } from "bun:test";

import type { CommandSection } from "@/ui/CanvasPage/RunnerLiveLogDialog/types";

import { notesFromLiveLogSections } from "../streamNotesFromLiveLog";
import { activityFromAgentStep, activityFromTranscript } from "./activityFromAgentStep";
import { agentStepsFromNotes, type AgentStep } from "./automationsViewModel";

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
    const activity = activityFromAgentStep(step({ output: "git clone https://example.com/repo.git", duration: "2s" }));

    expect(activity?.items).toEqual([
      expect.objectContaining({
        type: "tool",
        kind: "bash",
        input: "git clone https://example.com/repo.git",
        output: "",
        durationMs: 2_000,
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
      expect.objectContaining({ type: "tool", input: "git clone https://example.com/repo.git", output: "" }),
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

  it("keeps a multi-line bash script apart from its stdout", () => {
    const script = ["set -euo pipefail", "", 'git clone --depth 1 --branch "${BASE:-main}" "${REPO_URL}" repo'].join(
      "\n",
    );
    const stdout = ["Cloning into 'repo'...", "remote: Enumerating objects: 9384, done."].join("\n");
    const section: CommandSection = {
      index: 1,
      text: "Clone Repo",
      kind: "bash",
      preview: script,
      lines: stdout.split("\n"),
      events: [],
      status: "failed",
      duration_ms: 20,
      started_at: 1,
      collapsed: true,
    };
    const step = agentStepsFromNotes(notesFromLiveLogSections("agent", [section]))[0];
    const command = activityFromAgentStep(step!)?.items[0];

    expect(step?.output).toBe(`${script}\n\n${stdout}`);
    expect(command).toEqual(
      expect.objectContaining({
        type: "tool",
        kind: "bash",
        input: script,
        output: stdout,
        status: "failed",
      }),
    );
  });

  it("keeps an unnamed bash preview as the script when it matches the title", () => {
    const script = ["git clone --depth 1 https://example.com/repo.git", "cd repo"].join("\n");
    const stdout = ["Cloning into 'repo'...", "done."].join("\n");
    const section: CommandSection = {
      index: 1,
      text: script,
      kind: "bash",
      preview: script,
      lines: stdout.split("\n"),
      events: [],
      status: "passed",
      duration_ms: 20,
      started_at: 1,
      collapsed: true,
    };
    const step = agentStepsFromNotes(notesFromLiveLogSections("agent", [section]))[0];
    const command = activityFromAgentStep(step!)?.items[0];

    expect(command).toEqual(
      expect.objectContaining({
        type: "tool",
        kind: "bash",
        input: script,
        output: stdout,
      }),
    );
  });
});
