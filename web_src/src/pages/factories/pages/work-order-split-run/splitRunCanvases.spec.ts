import { describe, expect, it } from "vitest";

import {
  canvasKeyForAutomation,
  canvasKeyForPhase,
  parseSplitRunCanvasKey,
  richStreamForCanvas,
  splitRunCanvasForPhase,
} from "./splitRunCanvases";
import { SPLIT_RUN_RUNNING } from "./splitRunMocks";

describe("parseSplitRunCanvasKey", () => {
  it("accepts known canvas keys", () => {
    expect(parseSplitRunCanvasKey("implementation")).toBe("implementation");
  });

  it("returns undefined for unknown keys", () => {
    expect(parseSplitRunCanvasKey("canvas")).toBeUndefined();
  });
});

describe("canvasKeyForAutomation", () => {
  it("maps planner, implementer, and closure apps", () => {
    expect(canvasKeyForAutomation({ id: "app-refund-planner" })).toBe("planning");
    expect(canvasKeyForAutomation({ id: "app-refund-implementer" })).toBe("implementation");
    expect(canvasKeyForAutomation({ id: "app-pr-closure", name: "PR Closure" })).toBe("closure");
  });

  it("returns undefined when the app does not name a canvas", () => {
    expect(canvasKeyForAutomation({ id: "app-custom" })).toBeUndefined();
  });
});

describe("splitRunCanvasForPhase", () => {
  it("opens the implementation canvas for the running implement step", () => {
    const implement = SPLIT_RUN_RUNNING.phases.find((phase) => phase.id === "implement");
    expect(implement).toBeDefined();
    expect(canvasKeyForPhase(implement!)).toBe("implementation");

    const canvas = splitRunCanvasForPhase(implement!);
    expect(canvas.title).toBe("Implement");
    expect(canvas.nodes.map((node) => node.name)).toContain("Create Branch");
    expect(canvas.nodes.map((node) => node.name)).toContain("From GH issue?");
    expect(canvas.statuses["create-branch"]).toBe("passed");
    expect(canvas.statuses["implementation-agent"]).toBe("did_not_run");
    expect(canvas.statuses["implementation-agent-no-issue"]).toBe("running");
  });

  it("opens the planning canvas for a completed plan step", () => {
    const plan = SPLIT_RUN_RUNNING.phases.find((phase) => phase.id === "plan");
    expect(canvasKeyForPhase(plan!)).toBe("planning");

    const canvas = splitRunCanvasForPhase(plan!);
    expect(canvas.title).toBe("Plan");
    expect(canvas.statuses["onrun-create-plan"]).toBe("triggered");
    expect(canvas.statuses["planner-agent"]).toBe("did_not_run");
    expect(canvas.statuses["planner-agent-no-issue"]).toBe("passed");
  });

  it("writes one log line per canvas node and extra Claude Code notes", () => {
    const canvas = splitRunCanvasForPhase({
      id: "done",
      name: "Done",
      status: "passed",
      duration: "1m 12s",
      componentName: "PR Closure",
      artifacts: [],
      stream: [],
      canvasSteps: [],
    });
    const stream = richStreamForCanvas(canvas);

    expect(canvas.nodes.length).toBeGreaterThanOrEqual(5);
    expect(stream.filter((line) => !line.note).length).toBe(canvas.nodes.length);
    expect(stream.filter((line) => line.nodeId === "find-work-order").map((line) => line.id)).toEqual([
      "find-work-order",
    ]);
    expect(stream.some((line) => line.artifact?.type === "TYPE_PR")).toBe(true);
    expect(
      stream.some(
        (line) =>
          line.artifact?.data && "name" in line.artifact.data && line.artifact.data.name === "merge-screenshot.png",
      ),
    ).toBe(true);
    expect(stream.some((line) => line.artifact?.type === "TYPE_MARKDOWN")).toBe(true);
  });

  it("adds a verbose transcript for Claude Code and a score for checks", () => {
    const canvas = splitRunCanvasForPhase({
      id: "verify",
      name: "Verify",
      status: "passed",
      duration: "2m",
      componentName: "Risk Assessment",
      artifacts: [],
      stream: [],
      canvasSteps: [],
    });
    const stream = richStreamForCanvas(canvas);
    const agentNotes = stream.filter((line) => line.nodeId === "assess-pr-risk" && line.note);
    expect(agentNotes.map((line) => line.componentName)).toEqual([
      "Reading the pull request diff.",
      "Scoring retry-policy risk.",
      "Writing the risk review.",
    ]);
    expect(stream.find((line) => line.id === "report-risk-check")?.componentName).toBe("Risk review  65/100");
  });
});
