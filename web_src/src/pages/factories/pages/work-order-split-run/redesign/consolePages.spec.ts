import { describe, expect, it } from "bun:test";

import type { WorkOrderCheckPresentation } from "../../../lib/workOrderChecks";
import { SPLIT_RUN_RUNNING } from "../splitRunMocks";
import { stagesFromFixture, type AutomationStage } from "./automationsViewModel";
import {
  artifactsPageCount,
  consoleArtifactKind,
  consolePages,
  isTaskDocument,
  type StageArtifact,
} from "./consolePages";

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

describe("consolePages", () => {
  it("gives the Backlog creation card an Artifacts page and no Agent log", () => {
    expect(consolePages(runningStage("backlog"))).toEqual(["artifacts"]);
  });

  it("starts a running canvas card on the agent log", () => {
    const pages = consolePages(runningStage("implement"));

    expect(pages[0]).toBe("agent");
    expect(pages).toContain("artifacts");
  });

  it("keeps the agent page on a finished canvas run without stored steps", () => {
    const implement = runningStage("implement");
    const stage: AutomationStage = { ...implement, status: "passed", agentSteps: [] };

    expect(consolePages(stage)).toContain("agent");
  });

  it("orders the fixed pages agent, artifacts, checks", () => {
    const implement = runningStage("implement");
    const stage: AutomationStage = {
      ...implement,
      checks: [RISK_CHECK],
      outputs: {
        ...implement.outputs,
        pullRequests: [{ id: "pr-12", number: "12" }],
      },
    };

    expect(consolePages(stage)).toEqual(["agent", "artifacts", "checks"]);
  });

  it("counts pull requests on the Artifacts page", () => {
    const implement = runningStage("implement");
    const stage: AutomationStage = {
      ...implement,
      outputs: {
        ...implement.outputs,
        pullRequests: [{ id: "pr-12", number: "12" }],
      },
    };

    expect(artifactsPageCount(stage)).toBe(implement.outputs.artifacts.length + 1);
  });

  it("treats description.md as the task text only on the creation card", () => {
    const artifact: StageArtifact = {
      id: "art-description",
      type: "TYPE_MARKDOWN",
      data: { title: "description.md", body: "x" },
    };

    expect(isTaskDocument(runningStage("backlog"), artifact)).toBe(true);
    expect(isTaskDocument(runningStage("implement"), artifact)).toBe(false);
  });

  it("tells documents, media, and other artifacts apart", () => {
    expect(consoleArtifactKind({ type: "TYPE_MARKDOWN", data: { body: "x" } })).toBe("markdown");
    expect(consoleArtifactKind({ type: "TYPE_FILE", data: { contentType: "image/png" } })).toBe("image");
    expect(consoleArtifactKind({ type: "TYPE_FILE", data: { contentType: "video/mp4" } })).toBe("video");
    expect(consoleArtifactKind({ type: "TYPE_BRANCH", data: { name: "feature/x" } })).toBe("other");
  });
});
