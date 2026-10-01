import { describe, expect, it } from "bun:test";

import { formatWorkOrderDateTime } from "../../../lib/workOrderDateTime";
import { SPLIT_RUN_RUNNING } from "../splitRunMocks";
import { stagesFromFixture } from "./automationsViewModel";
import {
  outputCountLabel,
  runFooterLine,
  runFooterSpendLabel,
  runMetaLine,
  stepOutputSummary,
  tickingRunMetaLine,
} from "./consoleCardText";

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

describe("tickingRunMetaLine", () => {
  const sampledAt = 1_000;

  it("adds one second to a running duration and keeps the words", () => {
    const stage = { ...runningStage("implement"), status: "running" as const, duration: "5m 55s" };

    expect(tickingRunMetaLine(stage, sampledAt, sampledAt)).toBe("5m 55s so far");
    expect(tickingRunMetaLine(stage, sampledAt, sampledAt + 1_000)).toBe("5m 56s so far");
  });

  it("does not add seconds or the so far suffix when the stage is not running", () => {
    const stage = { ...runningStage("implement"), status: "passed" as const, duration: "5m 55s" };

    expect(tickingRunMetaLine(stage, sampledAt, sampledAt + 1_000)).toBe("5m 55s");
  });

  it("does not tick a label that is not a duration", () => {
    const running = { ...runningStage("implement"), status: "running" as const, duration: "Running" };
    const dash = { ...runningStage("implement"), status: "running" as const, duration: "—" };
    const empty = { ...runningStage("implement"), status: "running" as const, duration: "" };

    expect(tickingRunMetaLine(running, sampledAt, sampledAt + 1_000)).toBe("Running so far");
    expect(tickingRunMetaLine(dash, sampledAt, sampledAt + 1_000)).toBe("— so far");
    expect(tickingRunMetaLine(empty, sampledAt, sampledAt + 1_000)).toBe("");
  });

  it("strips a trailing so far before it adds seconds", () => {
    const stage = { ...runningStage("implement"), status: "running" as const, duration: "5m 55s so far" };

    expect(tickingRunMetaLine(stage, sampledAt, sampledAt + 1_000)).toBe("5m 56s so far");
  });
});

describe("runFooterLine", () => {
  it("shows the start date and time without a Started prefix", () => {
    const startedAt = "2026-09-30T00:59:00.000Z";
    const stage = {
      ...runningStage("implement"),
      startedAt,
      model: "claude-sonnet-4-6",
    };

    expect(runFooterLine(stage)).toBe(formatWorkOrderDateTime(new Date(startedAt)));
    expect(runFooterLine(stage)).not.toContain("Started");
    expect(runFooterLine(stage)).not.toContain("claude-sonnet-4-6");
  });
});

describe("runFooterSpendLabel", () => {
  it("joins cost and tokens", () => {
    expect(runFooterSpendLabel("$0.45", "2.1k")).toBe("$0.45 · 2.1k");
    expect(runFooterSpendLabel("$0.45")).toBe("$0.45");
    expect(runFooterSpendLabel(undefined, "2.1k")).toBe("2.1k");
  });
});
