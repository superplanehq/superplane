import { describe, expect, it } from "bun:test";

import { formatWorkOrderDateTime } from "../../../lib/workOrderDateTime";
import { SPLIT_RUN_RUNNING } from "../splitRunMocks";
import { stagesFromFixture, type AutomationStage } from "./automationsViewModel";
import {
  outputCountLabel,
  runFooterLine,
  runFooterSpendLabel,
  runMetaLine,
  stepOutputSummary,
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

  it("does not count the task document as a Backlog artifact", () => {
    const backlog = runningStage("backlog");
    const withExtra: AutomationStage = {
      ...backlog,
      outputs: {
        ...backlog.outputs,
        artifacts: [
          ...backlog.outputs.artifacts,
          { id: "art-log", type: "TYPE_FILE", data: { filename: "trace.log" } },
        ],
      },
    };

    expect(stepOutputSummary(backlog).artifactCount).toBe(0);
    expect(stepOutputSummary(withExtra).artifactCount).toBe(1);
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

  it("shows only the duration while a run is live", () => {
    const stage = { ...runningStage("implement"), status: "running" as const, duration: "10m 56s" };

    expect(runMetaLine(stage)).toBe("10m 56s");
  });

  it("sums every run on the card, not only the latest", () => {
    const latest = { ...runningStage("implement"), status: "passed" as const, duration: "9m 22s" };
    const runs = [
      { ...latest, id: "r1", duration: "22m 38s" },
      { ...latest, id: "r2", duration: "16m 35s" },
      { ...latest, id: "r3", duration: "14m 32s" },
      latest,
    ];

    expect(runMetaLine(latest, runs)).toBe("63m 7s");
  });

  it("keeps less than a second when every run is under one second", () => {
    const latest = { ...runningStage("implement"), status: "passed" as const, duration: "<1s" };
    const earlier = { ...latest, id: "r1", duration: "<1s" };

    expect(runMetaLine(latest, [earlier, latest])).toBe("<1s");
  });

  it("keeps less than a second for a legacy running label", () => {
    const stage = { ...runningStage("implement"), status: "running" as const, duration: "<1s so far" };

    expect(runMetaLine(stage)).toBe("<1s");
  });

  it("sums a live run with earlier runs", () => {
    const latest = { ...runningStage("implement"), status: "running" as const, duration: "2m" };
    const earlier = { ...latest, id: "r1", status: "passed" as const, duration: "10m" };

    expect(runMetaLine(latest, [earlier, latest])).toBe("12m");
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
