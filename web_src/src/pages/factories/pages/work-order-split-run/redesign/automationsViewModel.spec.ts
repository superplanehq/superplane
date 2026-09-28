import { describe, expect, it } from "bun:test";

import { DRAFT_WORK_ORDER } from "../../../__fixtures__/factoryPageResponses";
import { SPLIT_RUN_SUPER503 } from "../splitRunSuper503Fixture";
import { SUPER503_APPS } from "../splitRunSuper503Shared";
import { splitRunFixtureForWorkOrder } from "../splitRunMocks";
import {
  allStages,
  isConsoleTaskStage,
  outcomeSummary,
  settleStoppedSteps,
  stagesFromFixture,
  type AgentStep,
} from "./automationsViewModel";

const BACKLOG_COLUMN = ["Backlog", "Analysis"];

function backlogColumnNames(order: Parameters<typeof splitRunFixtureForWorkOrder>[0]) {
  const stages = stagesFromFixture(splitRunFixtureForWorkOrder(order, { demoArtifacts: false })).taskStages;
  return stages
    .filter((stage) => BACKLOG_COLUMN.includes(stage.name) && isConsoleTaskStage(stage))
    .map((stage) => stage.componentName);
}

describe("automations view model for SUPER-503", () => {
  it("keeps task stages separate from the pull request runs", () => {
    const groups = stagesFromFixture(SPLIT_RUN_SUPER503);

    expect(groups.taskStages.map((stage) => stage.name)).toEqual(["Backlog", "Analysis", "Implement"]);
    expect(groups.taskStages.map((stage) => stage.appId)).toEqual([
      undefined,
      SUPER503_APPS.backlog,
      SUPER503_APPS.implement,
    ]);
    expect(groups.pullRequestGroups).toHaveLength(1);
    expect(groups.pullRequestGroups[0].stages).toHaveLength(18);
    expect(groups.pullRequestGroups[0].stages.at(-1)?.componentName).toBe("PR Closure");
    expect(allStages(groups)).toHaveLength(21);
  });

  it("summarizes the merged pull request, the confidence check, and the spend", () => {
    const outcome = outcomeSummary(SPLIT_RUN_SUPER503);
    const implement = stagesFromFixture(SPLIT_RUN_SUPER503).taskStages.find((stage) => stage.id === "implement");

    expect(outcome.statusLabel).toBe("Completed");
    expect(outcome.pullRequests.map((pullRequest) => pullRequest.number)).toEqual(["7771"]);
    expect(outcome.checksPassed).toBe(1);
    expect(outcome.checksTotal).toBe(1);
    expect(outcome.models).toEqual(["grok-4.6"]);
    expect(implement?.agentSteps.map((step) => step.title)).toContain("Implementation");
    expect(implement?.rawLog).toContain("$ Implementation");
  });

  it("lists canvas nodes as steps and splices agent steps in place of the agent node", () => {
    const implement = stagesFromFixture(SPLIT_RUN_SUPER503).taskStages.find((stage) => stage.id === "implement");
    const titles = implement?.steps.map((step) => step.title) ?? [];

    expect(titles[0]).toBe("Start Implementation");
    expect(titles).toContain("Implementation");
    expect(titles).not.toContain("Implementation Agent");
    expect(titles.at(-1)).toBe("Comment Visual Evidence");
    expect(titles.indexOf("Implementation")).toBeLessThan(titles.indexOf("Add Branch Artifact"));
  });

  it("turns plumbing-only runs into node steps", () => {
    const checks = stagesFromFixture(SPLIT_RUN_SUPER503).pullRequestGroups[0].stages.find(
      (stage) => stage.componentName === "Fix pull request checks",
    );

    expect(checks?.agentSteps).toHaveLength(0);
    expect(checks?.steps.map((step) => step.title)).toEqual([
      "On Pull Request",
      "Find Pull Request",
      "Add PR Activity",
      "Wait For Checks",
      "Mark Checks Passed",
    ]);
    expect(checks?.steps.every((step) => step.type === "node")).toBe(true);
    expect(checks?.steps[0].iconSlug).toBe("github");
  });
});

describe("settleStoppedSteps", () => {
  const running = {
    id: "step",
    title: "Implementation",
    type: "prompt",
    status: "running",
    summary: "",
    toolCount: 0,
    events: [],
  } satisfies AgentStep;

  it("clears a running step when the stage is canceled", () => {
    expect(settleStoppedSteps([running], "cancelled").map((step) => step.status)).toEqual(["cancelled"]);
  });

  it("leaves a running step alone while the stage is still running", () => {
    expect(settleStoppedSteps([running], "running")).toEqual([running]);
  });
});

describe("Backlog column stages", () => {
  it("shows how the task was added before any later Backlog run", () => {
    expect(backlogColumnNames(DRAFT_WORK_ORDER)).toEqual(["Created manually"]);
    expect(
      backlogColumnNames({
        ...DRAFT_WORK_ORDER,
        origin: { url: "https://github.com/acme/payments/issues/12", label: "acme/payments#12" },
      }),
    ).toEqual(["Imported from GitHub"]);
    expect(
      backlogColumnNames({
        ...DRAFT_WORK_ORDER,
        createdBy: { automation: { appId: "app-github-issues", appName: "GitHub issues" } },
        origin: { url: "https://github.com/acme/payments/issues/12", label: "acme/payments#12" },
      }),
    ).toEqual(["GitHub issues"]);
    expect(
      backlogColumnNames({
        ...DRAFT_WORK_ORDER,
        createdBy: { automation: { appId: "app-sentry", appName: "Sentry exceptions" } },
      }),
    ).toEqual(["Sentry exceptions"]);
  });

  it("lists task creation before the Backlog analyzer", () => {
    const fixture = splitRunFixtureForWorkOrder(
      {
        ...DRAFT_WORK_ORDER,
        createdBy: { automation: { appId: "app-github-issues", appName: "GitHub issues" } },
      },
      {
        demoArtifacts: false,
        analysisRuns: [
          {
            canvasId: "canvas-backlog",
            workOrderId: DRAFT_WORK_ORDER.id ?? "",
            run: {
              id: "run-analysis",
              canvasId: "canvas-backlog",
              state: "STATE_FINISHED",
              result: "RESULT_PASSED",
              createdAt: "2026-08-28T12:00:00Z",
            },
          },
        ],
      },
    );
    const names = stagesFromFixture(fixture)
      .taskStages.filter((stage) => BACKLOG_COLUMN.includes(stage.name) && isConsoleTaskStage(stage))
      .map((stage) => stage.componentName);

    expect(names).toEqual(["GitHub issues", "Backlog"]);
  });
});
