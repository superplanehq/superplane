import { describe, expect, it } from "bun:test";

import { DRAFT_WORK_ORDER } from "../../../__fixtures__/factoryPageResponses";
import {
  LINE_BOARD_DONE_RECEIPTS_ORDER,
  LINE_BOARD_VERIFY_ENUM_ORDER,
} from "../../../__fixtures__/lineMetricsFactoriesFixture";
import { SPLIT_RUN_RUNNING, splitRunFixtureForWorkOrder, type SplitRunPhase } from "../splitRunMocks";
import {
  allStages,
  isConsoleTaskStage,
  outcomeSummary,
  settleStoppedSteps,
  stagesByConsoleColumn,
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

describe("automations view model", () => {
  it("keeps task stages separate from the pull request runs", () => {
    const fixture = splitRunFixtureForWorkOrder(LINE_BOARD_DONE_RECEIPTS_ORDER);
    const groups = stagesFromFixture(fixture);

    expect(groups.taskStages.length).toBeGreaterThan(0);
    expect(groups.taskStages.some((stage) => stage.name === "Implement")).toBe(true);
    expect(allStages(groups)).toHaveLength(
      groups.taskStages.length + groups.pullRequestGroups.flatMap((group) => group.stages).length,
    );
  });

  it("summarizes the run status, models, and spend from the fixture", () => {
    const outcome = outcomeSummary(SPLIT_RUN_RUNNING);

    expect(outcome.statusLabel).toBe("Running");
    expect(outcome.spend).toBe("$0.73");
    expect(outcome.tokens).toBe("2.7k tokens");
    expect(outcome.models).toEqual(["claude-sonnet-4-6"]);
    expect(outcome.headline).toBe("Implement is running");
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

function columnStageIds(fixture: ReturnType<typeof splitRunFixtureForWorkOrder>) {
  const columns = stagesByConsoleColumn(stagesFromFixture(fixture));
  return {
    backlog: columns.backlog.map((stage) => stage.id),
    implement: columns.implement.map((stage) => stage.id),
    verify: columns.verify.map((stage) => stage.id),
    done: columns.done.map((stage) => stage.id),
  };
}

describe("console column placement", () => {
  it("puts a Verify line step in Verify, not off the timeline", () => {
    const ids = columnStageIds(splitRunFixtureForWorkOrder(LINE_BOARD_VERIFY_ENUM_ORDER));

    expect(ids.verify).toContain("verify-1");
    expect(ids.implement).toContain("implement-0");
  });

  it("puts a Done line step in Done", () => {
    const ids = columnStageIds(splitRunFixtureForWorkOrder(LINE_BOARD_DONE_RECEIPTS_ORDER));

    expect(ids.done).toContain("done-2");
    expect(ids.verify).toContain("verify-1");
  });

  it("keeps a custom-named line step on the timeline", () => {
    const qa: SplitRunPhase = {
      id: "qa-custom",
      name: "QA",
      status: "passed",
      duration: "1m",
      componentName: "Quality Gate",
      appId: "app-qa",
      artifacts: [],
      stream: [],
      canvasSteps: [],
    };
    const fixture = {
      ...SPLIT_RUN_RUNNING,
      phases: [...SPLIT_RUN_RUNNING.phases, qa],
    };
    const ids = columnStageIds(fixture);

    expect([...ids.backlog, ...ids.implement, ...ids.verify, ...ids.done]).toContain("qa-custom");
  });
});
