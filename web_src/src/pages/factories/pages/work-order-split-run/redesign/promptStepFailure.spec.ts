import { describe, expect, it } from "bun:test";

import type { CommandSection } from "@/ui/CanvasPage/RunnerLiveLogDialog/types";

import { notesFromLiveLogSections } from "../streamNotesFromLiveLog";
import type { SplitRunPhase, SplitRunPhaseStatus, SplitRunStreamLine } from "../splitRunMocks";
import { agentStepsFromNotes, stageFromPhase } from "./automationsViewModel";

function promptSection(status: CommandSection["status"] = "failed"): CommandSection {
  return {
    index: 1,
    text: "Refine Task",
    kind: "prompt",
    preview: "You refine draft tasks.",
    lines: [],
    events: [{ kind: "note", text: "OpenCode started" }],
    status,
    duration_ms: 20,
    started_at: 1,
    collapsed: true,
  };
}

function bashSection(status: CommandSection["status"] = "failed"): CommandSection {
  return {
    index: 2,
    text: "Clone repository",
    kind: "bash",
    preview: "git clone https://example.com/repo.git",
    lines: [],
    events: [],
    status,
    duration_ms: 20,
    started_at: 2,
    collapsed: true,
  };
}

function canvasNode(id: string, componentName: string, status: SplitRunPhaseStatus): SplitRunStreamLine {
  return { id, nodeId: id, at: "12:00", componentName, status };
}

function failedPromptNote(): SplitRunStreamLine {
  return {
    id: "refine",
    nodeId: "agent",
    at: "12:01",
    componentName: "Refine Task",
    componentType: "prompt",
    status: "failed",
    note: true,
  };
}

function phase(stream: SplitRunStreamLine[]): SplitRunPhase {
  return {
    id: "implement",
    name: "Implement",
    status: "failed",
    duration: "1m",
    componentName: "Implement",
    artifacts: [],
    stream,
    canvasSteps: [],
  };
}

describe("prompt step failure", () => {
  it("keeps a failed prompt off the step when the run passed", () => {
    const notes = notesFromLiveLogSections("agent", [promptSection()], "passed");

    expect(notes[0]).toMatchObject({ status: "passed", promptStatus: "failed" });
    expect(notes.some((note) => note.componentType === "note")).toBe(true);
  });

  it("keeps Failed on the prompt step that stopped a failed run", () => {
    const notes = notesFromLiveLogSections("agent", [promptSection()], "failed");

    expect(notes[0]).toMatchObject({ status: "failed", promptStatus: "failed" });
  });

  it("does not mark an earlier failed prompt when a later bash step stopped the run", () => {
    const notes = notesFromLiveLogSections("agent", [promptSection(), bashSection()], "failed");

    expect(notes.find((note) => note.componentType === "prompt")?.status).toBe("passed");
    expect(notes.find((note) => note.componentType === "bash")?.status).toBe("failed");
  });

  it("keeps a failed bash step failed when the run passed", () => {
    const notes = notesFromLiveLogSections("agent", [bashSection()], "passed");

    expect(notes[0]).toMatchObject({ status: "failed", componentType: "bash" });
    expect(notes[0]?.promptStatus).toBeUndefined();
  });

  it("does not mark an earlier prompt Failed when a later node stopped the run", () => {
    const stage = stageFromPhase(
      phase([
        canvasNode("agent", "Run agent", "failed"),
        failedPromptNote(),
        canvasNode("checks", "Wait for checks", "failed"),
      ]),
    );

    expect(stage.agentSteps[0]).toMatchObject({ status: "passed", promptStatus: "failed" });
    expect(stage.steps.map((step) => [step.type, step.status])).toEqual([
      ["prompt", "passed"],
      ["node", "failed"],
    ]);
  });

  it("does not mark a prompt Failed when the live canvas has a later node", () => {
    const steps = agentStepsFromNotes([failedPromptNote()], "failed", [
      canvasNode("agent", "Run agent", "failed"),
      canvasNode("checks", "Wait for checks", "failed"),
    ]);

    expect(steps[0]).toMatchObject({ status: "passed", promptStatus: "failed" });
  });

  it("keeps Failed on the prompt when an earlier node did not stop the run", () => {
    const steps = agentStepsFromNotes(
      [canvasNode("trigger", "Webhook", "passed"), canvasNode("agent", "Run agent", "failed"), failedPromptNote()],
      "failed",
    );

    expect(steps[0]?.status).toBe("failed");
  });
});
