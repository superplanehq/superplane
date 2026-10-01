import { describe, expect, it } from "bun:test";

import type { WorkOrderCheckPresentation } from "../../../lib/workOrderChecks";
import { SPLIT_RUN_RUNNING } from "../splitRunMocks";
import { stagesFromFixture, type AutomationStage } from "./automationsViewModel";
import {
  artifactOpenHref,
  artifactsPageCount,
  consoleArtifactKind,
  consoleCardArtifacts,
  consolePages,
  isExpandableArtifact,
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
  it("omits the task document from the Backlog card and keeps other files", () => {
    const backlog = runningStage("backlog");
    const note: StageArtifact = {
      id: "art-note",
      type: "TYPE_FILE",
      data: { filename: "trace.log", url: "https://example.com/trace.log" },
    };
    const withNote: AutomationStage = {
      ...backlog,
      outputs: { ...backlog.outputs, artifacts: [...backlog.outputs.artifacts, note] },
    };
    const implement = runningStage("implement");
    const implementWithDocument: AutomationStage = {
      ...implement,
      outputs: {
        ...implement.outputs,
        artifacts: [...implement.outputs.artifacts, ...backlog.outputs.artifacts],
      },
    };

    const detailsOnly: AutomationStage = {
      ...backlog,
      outputs: {
        ...backlog.outputs,
        artifacts: [{ id: "art-details", type: "TYPE_MARKDOWN", data: { title: "details.md", body: "x" } }],
      },
    };

    expect(consoleCardArtifacts(backlog)).toEqual([]);
    expect(artifactsPageCount(backlog)).toBe(0);
    expect(artifactsPageCount(detailsOnly)).toBe(0);
    expect(consolePages(backlog)).toEqual([]);
    expect(consoleCardArtifacts(withNote)).toEqual([note]);
    expect(artifactsPageCount(withNote)).toBe(1);
    expect(consolePages(withNote)).toEqual(["artifacts"]);
    expect(artifactsPageCount(implementWithDocument)).toBe(
      implement.outputs.artifacts.length + backlog.outputs.artifacts.length,
    );
  });

  it("starts a running canvas card on Agent runs", () => {
    const pages = consolePages(runningStage("implement"));

    expect(pages[0]).toBe("agent");
    expect(pages).toContain("artifacts");
  });

  it("keeps Agent runs when any grouped run has a log", () => {
    const implement = runningStage("implement");
    const older: AutomationStage = { ...implement, id: "implement-old", agentSteps: [] };

    expect(consolePages(implement, undefined, [implement, older])).toContain("agent");
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

  it("treats description.md and details.md as the task text only on the creation card", () => {
    const description: StageArtifact = {
      id: "art-description",
      type: "TYPE_MARKDOWN",
      data: { title: "description.md", body: "x" },
    };
    const details: StageArtifact = {
      id: "art-details",
      type: "TYPE_MARKDOWN",
      data: { title: "details.md", body: "x" },
    };

    expect(isTaskDocument(runningStage("backlog"), description)).toBe(true);
    expect(isTaskDocument(runningStage("backlog"), details)).toBe(true);
    expect(isTaskDocument(runningStage("implement"), description)).toBe(false);
  });

  it("tells documents, media, and other artifacts apart", () => {
    expect(consoleArtifactKind({ type: "TYPE_MARKDOWN", data: { body: "x" } })).toBe("markdown");
    expect(consoleArtifactKind({ type: "TYPE_FILE", data: { contentType: "image/png" } })).toBe("image");
    expect(consoleArtifactKind({ type: "TYPE_FILE", data: { contentType: "video/mp4" } })).toBe("video");
    expect(consoleArtifactKind({ type: "TYPE_BRANCH", data: { name: "feature/x" } })).toBe("branch");
    expect(consoleArtifactKind({ type: "TYPE_LINK", data: { url: "https://example.com" } })).toBe("link");
    expect(consoleArtifactKind({ type: "TYPE_FILE", data: { filename: "notes.pdf" } })).toBe("file");
  });

  it("expands documents and media, and opens links and other files", () => {
    expect(isExpandableArtifact("markdown")).toBe(true);
    expect(isExpandableArtifact("image")).toBe(true);
    expect(isExpandableArtifact("video")).toBe(true);
    expect(isExpandableArtifact("link")).toBe(false);
    expect(isExpandableArtifact("branch")).toBe(false);
    expect(isExpandableArtifact("file")).toBe(false);
    expect(artifactOpenHref({ type: "TYPE_LINK", data: { url: "https://example.com" } })).toBe("https://example.com");
    expect(artifactOpenHref({ type: "TYPE_BRANCH", data: { name: "feature/x", repository: "acme/app" } })).toBe(
      "https://github.com/acme/app/tree/feature/x",
    );
  });
});
