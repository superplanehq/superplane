import { describe, expect, it } from "bun:test";

import { CLOSED_WORK_ORDER, DRAFT_WORK_ORDER, OPEN_WORK_ORDER } from "../../../__fixtures__/factoryPageResponses";
import {
  LINE_BOARD_DONE_RECEIPTS_ORDER,
  LINE_BOARD_VERIFY_ENUM_ORDER,
} from "../../../__fixtures__/lineMetricsFactoriesFixture";
import {
  SPLIT_RUN_RUNNING,
  splitRunFixtureForWorkOrder,
  type SplitRunPhase,
  type SplitRunStreamLine,
} from "../splitRunMocks";
import {
  allStages,
  agentStepsFromNotes,
  automationsFromStages,
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

function note(overrides: Partial<SplitRunStreamLine> & Pick<SplitRunStreamLine, "id">): SplitRunStreamLine {
  return {
    at: "12:00",
    componentName: "Step",
    status: "passed",
    note: true,
    ...overrides,
  };
}

describe("agentStepsFromNotes", () => {
  it("does not put a tool-count summary on a prompt step", () => {
    const steps = agentStepsFromNotes([
      {
        id: "think",
        at: "12:00",
        componentName: "Thinking",
        status: "running",
        note: true,
      },
      {
        id: "read-1",
        at: "12:01",
        componentName: "src/colors.ts",
        status: "passed",
        note: true,
        noteParentId: "think",
        componentType: "read",
      },
      {
        id: "bash-1",
        at: "12:02",
        componentName: "ls",
        status: "passed",
        note: true,
        noteParentId: "think",
        componentType: "bash",
      },
    ]);

    expect(steps).toHaveLength(1);
    expect(steps[0]?.title).toBe("Thinking");
    expect(steps[0]?.summary).toBe("");
    expect(steps[0]?.toolCount).toBe(2);
  });

  it("does not mark a prompt step Failed when the run passed", () => {
    const steps = agentStepsFromNotes(
      [
        note({ id: "refine", componentType: "prompt", componentName: "Refine Task", status: "failed" }),
        note({
          id: "refine-note",
          componentType: "note",
          componentName: "OpenCode started",
          status: "passed",
          noteParentId: "refine",
        }),
      ],
      "passed",
    );

    expect(steps[0]?.status).toBe("passed");
    expect(steps[0]?.promptStatus).toBe("failed");
  });

  it("keeps Failed on the prompt step that stopped the run", () => {
    const steps = agentStepsFromNotes(
      [
        note({ id: "first", componentType: "prompt", componentName: "Refine Task", status: "failed" }),
        note({ id: "stop", componentType: "prompt", componentName: "Implement", status: "failed" }),
      ],
      "failed",
    );

    expect(steps.map((step) => step.status)).toEqual(["passed", "failed"]);
    expect(steps[0]?.promptStatus).toBe("failed");
  });

  it("keeps a failed bash step failed", () => {
    const steps = agentStepsFromNotes(
      [note({ id: "clone", componentType: "bash", componentName: "Clone repository", status: "failed" })],
      "passed",
    );

    expect(steps[0]?.status).toBe("failed");
    expect(steps[0]?.type).toBe("bash");
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

  it("puts the close decision in Done when no closer automation ran", () => {
    const ids = columnStageIds(splitRunFixtureForWorkOrder(CLOSED_WORK_ORDER, { demoArtifacts: false }));

    expect(ids.done).toEqual(["done-closure"]);
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

describe("console automation grouping", () => {
  it("keeps comment replies as runs of one Address PR feedback card", () => {
    const fixture = splitRunFixtureForWorkOrder(OPEN_WORK_ORDER, {
      demoArtifacts: false,
      prFeedbackRuns: [
        {
          canvasId: "canvas-comment-1",
          handlerName: "Address PR feedback",
          title: "Read the requested changes",
          pullRequestNumber: "12",
          run: {
            id: "run-comment-1",
            canvasId: "canvas-comment-1",
            state: "STATE_FINISHED",
            result: "RESULT_PASSED",
            createdAt: "2026-08-26T11:00:00Z",
          },
        },
        {
          canvasId: "canvas-comment-2",
          handlerName: "Address PR feedback",
          title: "Address new review comment",
          pullRequestNumber: "12",
          run: {
            id: "run-comment-2",
            canvasId: "canvas-comment-2",
            state: "STATE_FINISHED",
            result: "RESULT_PASSED",
            createdAt: "2026-08-26T12:00:00Z",
          },
        },
      ],
    });
    const automations = automationsFromStages(stagesByConsoleColumn(stagesFromFixture(fixture)).verify);
    const address = automations.filter((automation) => automation.name === "Address PR feedback");

    expect(address).toHaveLength(1);
    expect(address[0]?.runs).toHaveLength(2);
    expect(address[0]?.runs.map((run) => run.id)).toEqual(["pr-feedback-run-comment-1", "pr-feedback-run-comment-2"]);
    expect(address[0]?.runs.map((run) => run.name)).toEqual([
      "Read the requested changes",
      "Address new review comment",
    ]);
  });

  it("keeps the live run on the grouped card when a newer run already finished", () => {
    const fixture = splitRunFixtureForWorkOrder(OPEN_WORK_ORDER, {
      demoArtifacts: false,
      prFeedbackRuns: [
        {
          canvasId: "canvas-comment-live",
          handlerName: "Address PR feedback",
          title: "Read the requested changes",
          pullRequestNumber: "12",
          run: {
            id: "run-comment-live",
            canvasId: "canvas-comment-live",
            state: "STATE_STARTED",
            createdAt: "2026-08-26T11:00:00Z",
          },
        },
        {
          canvasId: "canvas-comment-done",
          handlerName: "Address PR feedback",
          title: "Address new review comment",
          pullRequestNumber: "12",
          run: {
            id: "run-comment-done",
            canvasId: "canvas-comment-done",
            state: "STATE_FINISHED",
            result: "RESULT_PASSED",
            createdAt: "2026-08-26T12:00:00Z",
          },
        },
      ],
    });
    const automations = automationsFromStages(stagesByConsoleColumn(stagesFromFixture(fixture)).verify);
    const address = automations.find((automation) => automation.name === "Address PR feedback");

    expect(address?.latest.status).toBe("running");
    expect(address?.latest.id).toBe("pr-feedback-run-comment-live");
  });

  it("keeps a column app run off the Address PR feedback card", () => {
    const fixture = splitRunFixtureForWorkOrder(OPEN_WORK_ORDER, {
      demoArtifacts: false,
      columnApps: [{ id: "app-storybook", name: "Deploys Storybook", columnKey: "verify" }],
      prFeedbackRuns: [
        {
          canvasId: "app-storybook",
          title: "Storybook deployment ready",
          pullRequestNumber: "12",
          run: {
            id: "run-storybook",
            canvasId: "app-storybook",
            state: "STATE_FINISHED",
            result: "RESULT_PASSED",
            createdAt: "2026-08-26T11:30:00Z",
          },
        },
        {
          canvasId: "canvas-comment",
          handlerName: "Address PR feedback",
          title: "Address new review comment",
          pullRequestNumber: "12",
          run: {
            id: "run-comment",
            canvasId: "canvas-comment",
            state: "STATE_FINISHED",
            result: "RESULT_PASSED",
            createdAt: "2026-08-26T12:00:00Z",
          },
        },
      ],
    });
    const names = automationsFromStages(stagesByConsoleColumn(stagesFromFixture(fixture)).verify).map(
      (automation) => automation.name,
    );

    expect(names).toContain("Deploys Storybook");
    expect(names).toContain("Address PR feedback");
    expect(names.filter((name) => name === "Address PR feedback")).toHaveLength(1);
  });

  it("lists a column app check as a card in that column", () => {
    const fixture = splitRunFixtureForWorkOrder(OPEN_WORK_ORDER, {
      demoArtifacts: false,
      checks: [
        {
          id: "check-risk",
          key: "risk-review",
          name: "Risk score",
          score: 2,
          maxScore: 5,
          level: "LEVEL_POSITIVE",
          automation: { appId: "app-risk", appName: "Risk score" },
          runId: "run-risk",
          updatedAt: "2026-08-26T11:10:00Z",
        },
      ],
      columnApps: [
        { id: "app-risk", name: "Risk score", columnKey: "verify" },
        { id: "app-env", name: "Create env", columnKey: "done" },
      ],
    });
    const columns = stagesByConsoleColumn(stagesFromFixture(fixture));

    expect(automationsFromStages(columns.verify).map((automation) => automation.name)).toContain("Risk score");
    expect(automationsFromStages(columns.done).map((automation) => automation.name)).not.toContain("Create env");
  });

  it("keeps every check from one column app run on that card", () => {
    const fixture = splitRunFixtureForWorkOrder(OPEN_WORK_ORDER, {
      demoArtifacts: false,
      checks: [
        {
          id: "check-risk",
          key: "risk-review",
          name: "Risk score",
          score: 2,
          maxScore: 5,
          level: "LEVEL_POSITIVE",
          automation: { appId: "app-risk", appName: "Risk score" },
          runId: "run-risk",
          updatedAt: "2026-08-26T11:10:00Z",
        },
        {
          id: "check-diff",
          key: "risk-diff",
          name: "Diff size",
          score: 1,
          maxScore: 5,
          level: "LEVEL_POSITIVE",
          automation: { appId: "app-risk", appName: "Risk score" },
          runId: "run-risk",
          updatedAt: "2026-08-26T11:11:00Z",
        },
      ],
      columnApps: [{ id: "app-risk", name: "Risk score", columnKey: "verify" }],
    });
    const risk = automationsFromStages(stagesByConsoleColumn(stagesFromFixture(fixture)).verify).find(
      (automation) => automation.name === "Risk score",
    );

    expect(risk?.runs).toHaveLength(1);
    expect(risk?.latest.checks.map((check) => check.name)).toEqual(["Blast radius", "Diff size"]);
  });

  it("puts factory PR Closure in Done, not Verify", () => {
    const ids = columnStageIds(
      splitRunFixtureForWorkOrder(OPEN_WORK_ORDER, {
        demoArtifacts: false,
        prFeedbackRuns: [
          {
            canvasId: "app-pr-closure",
            handlerName: "PR Closure",
            title: "Pull request merged",
            pullRequestNumber: "12",
            run: {
              id: "run-closure",
              canvasId: "app-pr-closure",
              state: "STATE_FINISHED",
              result: "RESULT_PASSED",
              createdAt: "2026-08-26T13:00:00Z",
            },
          },
        ],
      }),
    );

    expect(ids.done).toContain("pr-feedback-run-closure");
    expect(ids.verify).not.toContain("pr-feedback-run-closure");
  });

  it("puts a Done-column app run in Done", () => {
    const fixture = splitRunFixtureForWorkOrder(OPEN_WORK_ORDER, {
      demoArtifacts: false,
      checks: [
        {
          id: "check-env",
          key: "env-ready",
          name: "Environment",
          score: 1,
          maxScore: 1,
          format: "FORMAT_BOOLEAN",
          level: "LEVEL_POSITIVE",
          automation: { appId: "app-env", appName: "Create env" },
          runId: "run-env",
          updatedAt: "2026-08-26T11:10:00Z",
        },
      ],
      columnApps: [{ id: "app-env", name: "Create env", columnKey: "done" }],
    });

    expect(
      automationsFromStages(stagesByConsoleColumn(stagesFromFixture(fixture)).done).map(
        (automation) => automation.name,
      ),
    ).toContain("Create env");
  });
});
