import { describe, expect, it } from "bun:test";

import type { WorkOrderCheckPresentation } from "../../../lib/workOrderChecks";
import { SPLIT_RUN_RUNNING } from "../splitRunMocks";
import { stagesFromFixture, type AutomationStage } from "./automationsViewModel";
import { consolePanes, defaultConsolePaneId } from "./consolePanes";

const RISK_CHECK: WorkOrderCheckPresentation = {
  id: "check-risk",
  name: "Risk score",
  score: 3,
  maxScore: 5,
  level: "caution",
  summary: "Moderate risk.",
};

function runningStage(id: string): AutomationStage {
  const stage = stagesFromFixture(SPLIT_RUN_RUNNING).taskStages.find((candidate) => candidate.id === id);
  if (!stage) {
    throw new Error(`Stage ${id} is not in the running fixture`);
  }
  return stage;
}

describe("consolePanes", () => {
  it("gives the Backlog creation card one Description pane", () => {
    const panes = consolePanes(runningStage("backlog"));

    expect(panes.map((pane) => pane.kind)).toEqual(["description"]);
    expect(defaultConsolePaneId(panes)).toBe("description");
  });

  it("starts a running canvas card on the agent log", () => {
    const implement = runningStage("implement");
    const panes = consolePanes(implement);

    expect(panes[0]?.kind).toBe("log");
    expect(panes.some((pane) => pane.kind === "artifact")).toBe(true);
    expect(defaultConsolePaneId(panes)).toBe("log");
  });

  it("orders description, log, artifacts, scores, then the pull request", () => {
    const implement = runningStage("implement");
    const stage: AutomationStage = {
      ...implement,
      description: "Retry the refund writer.",
      checks: [RISK_CHECK],
      outputs: {
        ...implement.outputs,
        pullRequests: [{ id: "pr-12", number: "12" }],
      },
    };
    const panes = consolePanes(stage);

    expect(panes.map((pane) => pane.kind)).toEqual(["description", "log", "artifact", "check", "pullRequest"]);
    expect(panes.find((pane) => pane.kind === "check")?.label).toBe("Risk score");
    expect(defaultConsolePaneId(panes)).toBe("log");
  });

  it("keeps a description.md artifact on cards other than Backlog", () => {
    const implement = runningStage("implement");
    const stage: AutomationStage = {
      ...implement,
      outputs: {
        pullRequests: [],
        artifacts: [{ id: "art-description", type: "TYPE_MARKDOWN", data: { title: "description.md", body: "x" } }],
      },
    };

    expect(consolePanes(stage).map((pane) => pane.label)).toEqual(["Agent log", "description.md"]);
  });
});
