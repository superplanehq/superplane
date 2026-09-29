import { describe, expect, it } from "bun:test";

import { SPLIT_RUN_RUNNING } from "../splitRunMocks";
import { stagesFromFixture } from "./automationsViewModel";
import { outputCountLabel, runMetaLine, stepOutputSummary } from "./consoleCardText";

function runningStage(id: string) {
  const stage = stagesFromFixture(SPLIT_RUN_RUNNING).taskStages.find((candidate) => candidate.id === id);
  if (!stage) {
    throw new Error(`Stage ${id} is not in the running fixture`);
  }
  return stage;
}

describe("stepOutputSummary", () => {
  it("counts agent runs, artifacts, and checks", () => {
    const implement = runningStage("implement");

    expect(stepOutputSummary(implement, 2)).toEqual({
      runCount: 2,
      artifactCount: implement.outputs.artifacts.length,
      checkCount: implement.checks.length,
    });
  });

  it("labels a count with the singular or plural noun", () => {
    expect(outputCountLabel(1, "agent run", "agent runs")).toBe("1 agent run");
    expect(outputCountLabel(4, "agent run", "agent runs")).toBe("4 agent runs");
  });
});

describe("runMetaLine", () => {
  it("shows the duration and leaves spend to the summary panel", () => {
    const stage = { ...runningStage("implement"), status: "passed" as const, duration: "40m 18s", cost: "$15.97" };

    expect(runMetaLine(stage)).toBe("40m 18s");
  });

  it("marks a running duration as still counting", () => {
    const stage = { ...runningStage("implement"), status: "running" as const, duration: "10m 56s" };

    expect(runMetaLine(stage)).toBe("10m 56s so far");
  });
});
