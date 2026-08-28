import type { FactoriesWorkOrder, FactoriesWorkOrderExecution, FactoriesWorkOrderLineDispatch } from "@/api-client";
import { describe, expect, it } from "vitest";

import {
  APPROVAL_WORK_ORDER,
  DRAFT_WORK_ORDER,
  INGEST_DRAFT_WORK_ORDER,
  LINE_RUN_IMPLEMENT_NOTIFY_ID,
  OPEN_WORK_ORDER,
  RUNNING_WORK_ORDER,
  SENTRY_DRAFT_WORK_ORDER,
  SLACK_DRAFT_WORK_ORDER,
} from "../../__fixtures__/factoryPageResponses";
import {
  BOARD_DONE_CANCELED_ORDER,
  BOARD_DONE_REJECTED_ORDER,
  BOARD_IMPLEMENT_FAILED_ORDER,
  BOARD_IMPLEMENT_NOTIFY_ORDER,
} from "../../__fixtures__/lineMetricsBoardOrders";
import {
  LINE_BOARD_DONE_RECEIPTS_ORDER,
  LINE_BOARD_VERIFY_ENUM_ORDER,
  LINE_BOARD_VERIFY_PR_REVIEW_ORDER,
} from "../../__fixtures__/lineMetricsFactoriesFixture";
import {
  OPEN_WORK_ORDER_CHECKS,
  RUNNING_WORK_ORDER_CHECKS,
  VERIFY_STEP_CHECKS,
} from "../../__fixtures__/workOrderCheckFixtures";
import { REVIEW_CANDIDATE_WORK_ORDERS } from "../onboarding/first-run/reviewCandidates";
import { splitRunFixtureForWorkOrder, splitRunStatusLabel } from "./splitRunMocks";

function dispatch(state: FactoriesWorkOrderLineDispatch["state"], stepExecutions: FactoriesWorkOrderExecution[]) {
  return {
    id: "dispatch-1",
    line: { id: "line-1", name: "plan-and-implement" },
    state,
    stepExecutions,
  };
}

function order(overrides: FactoriesWorkOrder): FactoriesWorkOrder {
  return { id: "wo-1", ...overrides };
}

function artifactNames(artifacts: Array<{ data?: Record<string, unknown> }> | undefined): string[] {
  return (artifacts ?? []).map((artifact) => {
    const data = artifact.data ?? {};
    if (typeof data.name === "string") {
      return data.name;
    }
    if (typeof data.number === "number") {
      return `#${data.number}`;
    }
    return typeof data.title === "string" ? data.title : "";
  });
}

describe("splitRunFixtureForWorkOrder", () => {
  it("uses the designed running fixture when the order is missing", () => {
    const fixture = splitRunFixtureForWorkOrder();
    expect(fixture.title).toBe("Add refund reconciliation test");
    expect(fixture.currentPhaseId).toBe("implement");
    expect(fixture.phases.map((phase) => [phase.name, phase.status])).toEqual([
      ["Backlog", "passed"],
      ["Create plan", "passed"],
      ["Implement", "running"],
    ]);
  });

  it("maps the running reconciliation card from the work-order execution", () => {
    const fixture = splitRunFixtureForWorkOrder(RUNNING_WORK_ORDER);
    expect(fixture.title).toBe("Add refund reconciliation test");
    expect(fixture.costUsd).toBe("$0.73");
    expect(fixture.tokensLabel).toBe("2.7k tokens");
    expect(fixture.lineStatus).toBe("running");
    expect(fixture.currentPhaseId).toMatch(/^implement-/);
    const implement = fixture.phases.find((phase) => phase.name === "Implement");
    expect(implement?.status).toBe("running");
    expect(implement?.componentName).toBe("Implementation");
    expect(implement?.appId).toBe("app-refund-implementer");
    expect(implement?.runId).toBe(RUNNING_WORK_ORDER.lineDispatches?.[0]?.stepExecutions?.[0]?.run?.id);
    expect(implement?.stream.map((line) => line.componentName)).toEqual(["Implementation"]);
    expect(fixture.phases.map((phase) => [phase.id, phase.name, phase.status])).toEqual([
      ["ingest", "Ingest", "passed"],
      ["analyze", "Analyze", "passed"],
      ["plan", "Create plan", "passed"],
      ["score", "Score", "passed"],
      ["implement-0", "Implement", "running"],
    ]);
    expect(fixture.phases.find((phase) => phase.id === "ingest")?.artifacts).toEqual(
      expect.arrayContaining([
        expect.objectContaining({
          type: "TYPE_MARKDOWN",
          data: expect.objectContaining({ name: "details.md", body: RUNNING_WORK_ORDER.description }),
        }),
        expect.objectContaining({
          type: "TYPE_LINK",
          data: expect.objectContaining({ title: "RF-103" }),
        }),
      ]),
    );
    expect(fixture.waitingNotes).toEqual([]);
    expect(fixture.footerTone).toBe("running");
    expect(fixture.footer.note?.headline).toBe("Implement is running");
    expect(fixture.footer.run).toEqual({
      appId: "app-refund-implementer",
      runId: RUNNING_WORK_ORDER.lineDispatches?.[0]?.stepExecutions?.[0]?.run?.id,
    });
    expect(fixture.footer.actions.map((action) => action.label)).toEqual([]);
    expect(fixture.checks).toMatchObject([{ id: "wo-running-refunds-confidence", name: "Confidence score", score: 4 }]);
    expect(artifactNames(fixture.phases.find((phase) => phase.id === "plan")?.artifacts)).toEqual(["plan.md"]);
    expect(fixture.phases.find((phase) => phase.id === "score")?.checks?.[0]).toMatchObject({
      name: "Confidence score",
      score: 4,
    });
    expect(artifactNames(implement?.artifacts)).toEqual(["feature/rf-103", "#503"]);
  });

  it("keeps a single Backlog ingest row on an ingest draft", () => {
    const fixture = splitRunFixtureForWorkOrder(INGEST_DRAFT_WORK_ORDER);
    expect(fixture.phases.map((phase) => phase.id)).toEqual(["backlog"]);
  });

  it("puts complete ingest analysis on a scored review draft", () => {
    const fixture = splitRunFixtureForWorkOrder(REVIEW_CANDIDATE_WORK_ORDERS[0]);
    expect(fixture.phases.map((phase) => phase.id)).toEqual(["ingest", "analyze", "plan", "score"]);
    expect(fixture.footerTone).toBe("draft");
    expect(fixture.phases.find((phase) => phase.id === "score")?.checks?.[0]).toMatchObject({
      name: "Confidence score",
      score: 5,
      summary: "This issue is a good fit for an agent on this factory line.",
    });
    expect(fixture.checks[0]).toMatchObject({
      name: "Confidence score",
      summary: "This issue is a good fit for an agent on this factory line.",
    });
    expect(fixture.checks[0]?.analysis).toContain("The automation read this GitHub issue.");
    expect(fixture.checks[0]?.analysis).toContain("how suitable the work is for an agent");
    expect(artifactNames(fixture.phases.find((phase) => phase.id === "plan")?.artifacts)).toEqual(["plan.md"]);
  });

  it("narrates source, plan, and confidence reasons on a scored draft footer", () => {
    const fixture = splitRunFixtureForWorkOrder(REVIEW_CANDIDATE_WORK_ORDERS[0]);
    const note = fixture.footer.note;

    expect(note?.headline).toBe("Review the plan, then start");
    expect(note?.text).toContain("[PAY-842](https://github.com/acme/payments-service/issues/842)");
    expect(note?.text).toContain("**plan.md**");
    expect(note?.text).toContain("Confidence 5/5 (High):");
    expect(note?.text).toContain("- The GitHub issue names retryable status codes and a hard attempt limit.");
    expect(fixture.footer.actions.map((action) => action.label)).toEqual(["Start"]);
  });

  it("pins a pull request review on a waiting implement card", () => {
    const fixture = splitRunFixtureForWorkOrder(
      order({
        title: "Ship idempotent refund retries",
        state: "STATE_OPEN",
        statusNotes: OPEN_WORK_ORDER.statusNotes,
        lineDispatches: [
          dispatch("STATE_FINISHED", [
            { id: "e-impl", step: "Implement", stepIndex: 0, state: "STATE_FINISHED", result: "RESULT_PASSED" },
          ]),
        ],
      }),
    );
    expect(fixture.footerTone).toBe("waiting");
    expect(fixture.waitingNotes.map((note) => note.headline)).toEqual(["Listening for user review"]);
    expect(fixture.waitingNotes[0]?.cta?.label).toBe("Review PR #6812");
    expect(fixture.footer.note?.headline).toBe("Listening for user review");
    expect(fixture.footer.attentionCard).toBe(true);
    expect(fixture.footer.actions.map((action) => action.label)).toEqual([]);
    expect(fixture.checks).toEqual([]);
  });

  it("keeps a waiting state bar when a waiting order has no notes", () => {
    const fixture = splitRunFixtureForWorkOrder(
      order({
        title: "Ship idempotent refund retries",
        state: "STATE_OPEN",
        assignees: [{ id: "user-1", name: "Ada Lovelace" }],
        lineDispatches: [
          dispatch("STATE_FINISHED", [
            { id: "e-impl", step: "Implement", stepIndex: 0, state: "STATE_FINISHED", result: "RESULT_PASSED" },
          ]),
        ],
      }),
    );
    expect(fixture.footerTone).toBe("waiting");
    expect(fixture.waitingNotes).toEqual([]);
    expect(fixture.footer.note?.headline).toBe("A person must act");
    expect(fixture.footer.attentionCard).toBeUndefined();
    expect(fixture.footer.actions.map((action) => action.label)).toEqual([]);
  });

  it("does not treat a missing execution step index as the first step", () => {
    const fixture = splitRunFixtureForWorkOrder(
      order({
        state: "STATE_OPEN",
        title: "Later step",
        lineDispatches: [
          dispatch("STATE_ACTIVE", [
            { id: "e-plan", step: "Planning", stepIndex: 0, state: "STATE_FINISHED", result: "RESULT_PASSED" },
            { id: "e-impl", step: "Implement", state: "STATE_FINISHED", result: "RESULT_FAILED" },
            { id: "e-verify", step: "Verify", stepIndex: 2, state: "STATE_STARTED", result: "RESULT_UNKNOWN" },
          ]),
        ],
      }),
    );

    const implement = fixture.phases.find((phase) => phase.name === "Implement");
    expect(implement?.status).toBe("failed");
    expect(implement?.stepIndex).toBeUndefined();
    expect(fixture.currentStepIndex).toBe(2);
  });

  it("keeps a waiting state bar after a finished unnamed step while the order waits", () => {
    const fixture = splitRunFixtureForWorkOrder(
      order({
        title: "dasdas",
        state: "STATE_OPEN",
        assignees: [{ id: "user-1", name: "test test" }],
        lineDispatches: [
          dispatch("STATE_FINISHED", [
            { id: "e-1", step: "dasdasdas", stepIndex: 0, state: "STATE_FINISHED", result: "RESULT_PASSED" },
          ]),
        ],
      }),
    );
    expect(fixture.footerTone).toBe("waiting");
    expect(fixture.waitingNotes).toEqual([]);
    expect(fixture.footer.sentence).toBe("This work order needs attention.");
  });

  it("puts risk score and code quality on the verify step", () => {
    const fixture = splitRunFixtureForWorkOrder(
      order({
        title: "Verify job",
        state: "STATE_OPEN",
        lineDispatches: [
          dispatch("STATE_ACTIVE", [
            { id: "e-impl", step: "Implement", stepIndex: 0, state: "STATE_FINISHED", result: "RESULT_PASSED" },
            { id: "e-verify", step: "Verify", stepIndex: 1, state: "STATE_STARTED", result: "RESULT_UNKNOWN" },
          ]),
        ],
      }),
      { checks: OPEN_WORK_ORDER_CHECKS },
    );
    const verify = fixture.phases.find((phase) => phase.id === "verify-1");
    expect(verify?.checks?.map((check) => check.name)).toEqual(["Risk score", "Code quality"]);
    expect(fixture.phases.find((phase) => phase.id === "implement-0")?.checks).toBeUndefined();
    expect(fixture.checks.map((check) => check.name)).toEqual([
      "Risk score",
      "Code quality",
      "Test coverage",
      "Confidence score",
      "CI",
    ]);
  });

  it("shows no verify checks when the API supplies none", () => {
    const verify = splitRunFixtureForWorkOrder(
      order({
        title: "Verify job",
        state: "STATE_OPEN",
        lineDispatches: [
          dispatch("STATE_ACTIVE", [
            { id: "e-verify", step: "Verify", stepIndex: 2, state: "STATE_STARTED", result: "RESULT_UNKNOWN" },
          ]),
        ],
      }),
      { checks: [] },
    );
    expect(verify.phases.find((phase) => phase.id === "verify-2")?.checks).toEqual([]);
    expect(verify.checks).toEqual([]);

    const done = splitRunFixtureForWorkOrder(
      order({
        title: "Done job",
        state: "STATE_CLOSED",
        result: "RESULT_COMPLETED",
        lineDispatches: [
          dispatch("STATE_FINISHED", [
            { id: "e-done", step: "Done", stepIndex: 3, state: "STATE_FINISHED", result: "RESULT_PASSED" },
          ]),
        ],
      }),
    );
    expect(done.checks).toEqual([]);
    expect(done.waitingNotes).toEqual([]);
    expect(done.footerTone).toBe("done");
    expect(done.footer.sentence).toBe("Work order completed successfully.");
    expect(done.footer.actions.map((action) => action.label)).toEqual(["Reopen"]);
  });

  it("keeps a completed order on the done footer when a leftover step failed", () => {
    const fixture = splitRunFixtureForWorkOrder(
      order({
        title: "Done job",
        state: "STATE_CLOSED",
        result: "RESULT_COMPLETED",
        lineDispatches: [
          dispatch("STATE_FINISHED", [
            { id: "e-done", step: "Implement", stepIndex: 0, state: "STATE_FINISHED", result: "RESULT_FAILED" },
          ]),
        ],
      }),
    );

    expect(fixture.footerTone).toBe("done");
    expect(fixture.footer.actions.map((action) => action.label)).toEqual(["Reopen"]);
    expect(fixture.footer.status).toBe("completed");
  });

  it("uses the newest dispatch on the viewed line", () => {
    const fixture = splitRunFixtureForWorkOrder(
      order({
        title: "Two-line order",
        state: "STATE_OPEN",
        lineDispatches: [
          {
            id: "d-other",
            createdAt: "2026-08-21T12:00:00.000Z",
            line: { id: "line-other", name: "other-line" },
            state: "STATE_FINISHED",
            stepExecutions: [
              { id: "e-other", step: "Implement", stepIndex: 1, state: "STATE_FINISHED", result: "RESULT_PASSED" },
            ],
          },
          {
            id: "d-viewed",
            createdAt: "2026-08-21T11:00:00.000Z",
            line: { id: "line-1", name: "plan-and-implement" },
            state: "STATE_ACTIVE",
            stepExecutions: [
              { id: "e-impl", step: "Implement", stepIndex: 0, state: "STATE_STARTED", result: "RESULT_UNKNOWN" },
            ],
          },
        ],
      }),
      { lineId: "line-1" },
    );
    expect(fixture.lineName).toBe("plan-and-implement");
    expect(fixture.currentPhaseId).toMatch(/^implement-/);
    expect(fixture.phases.some((phase) => phase.name === "Implement")).toBe(true);
  });

  it("keeps earlier passed steps when the latest dispatch only reran a later step", () => {
    const fixture = splitRunFixtureForWorkOrder(
      order({
        title: "Improve AGENTS.md",
        state: "STATE_OPEN",
        lineDispatches: [
          {
            id: "d-full",
            createdAt: "2026-08-25T20:00:00.000Z",
            line: { id: "line-1", name: "Software delivery" },
            state: "STATE_FINISHED",
            steps: [{ name: "Planning" }, { name: "Implementation", stepIndex: 1 }, { name: "", stepIndex: 2 }],
            stepExecutions: [
              { id: "e-plan", step: "Planning", state: "STATE_FINISHED", result: "RESULT_PASSED" },
              {
                id: "e-impl-old",
                step: "Implementation",
                stepIndex: 1,
                state: "STATE_FINISHED",
                result: "RESULT_FAILED",
              },
            ],
          },
          {
            id: "d-rerun",
            createdAt: "2026-08-25T21:00:00.000Z",
            line: { id: "line-1", name: "Software delivery" },
            state: "STATE_FINISHED",
            steps: [{ name: "" }, { name: "Implementation", stepIndex: 1 }, { name: "", stepIndex: 2 }],
            stepExecutions: [
              {
                id: "e-impl-new",
                step: "Implementation",
                stepIndex: 1,
                state: "STATE_FINISHED",
                result: "RESULT_FAILED",
              },
            ],
          },
        ],
      }),
      { lineId: "line-1" },
    );

    expect(fixture.phases.map((phase) => phase.name)).toEqual(expect.arrayContaining(["Plan", "Implement"]));
  });

  it("prefers an older active dispatch over a newer finished rerun", () => {
    const fixture = splitRunFixtureForWorkOrder(
      order({
        title: "Improve AGENTS.md",
        state: "STATE_OPEN",
        lineDispatches: [
          {
            id: "d-full",
            createdAt: "2026-08-25T20:00:00.000Z",
            line: { id: "line-1", name: "Software delivery" },
            state: "STATE_ACTIVE",
            stepExecutions: [
              { id: "e-plan", step: "Planning", state: "STATE_FINISHED", result: "RESULT_PASSED" },
              {
                id: "e-impl-new",
                step: "Implementation",
                stepIndex: 1,
                state: "STATE_STARTED",
                result: "RESULT_UNKNOWN",
              },
            ],
          },
          {
            id: "d-rerun",
            createdAt: "2026-08-25T21:00:00.000Z",
            line: { id: "line-1", name: "Software delivery" },
            state: "STATE_FINISHED",
            stepExecutions: [
              {
                id: "e-impl-old",
                step: "Implementation",
                stepIndex: 1,
                state: "STATE_FINISHED",
                result: "RESULT_FAILED",
              },
            ],
          },
        ],
      }),
      { lineId: "line-1" },
    );

    expect(fixture.phases.map((phase) => phase.name)).toEqual(expect.arrayContaining(["Plan", "Implement"]));
    expect(fixture.phases.find((phase) => phase.name === "Implement")?.status).toBe("running");
  });

  it("gives a rerun of the same step its own phase and run", () => {
    const fixture = splitRunFixtureForWorkOrder(
      order({
        title: "Test work order 2",
        state: "STATE_OPEN",
        lineDispatches: [
          {
            id: "d-1",
            createdAt: "2026-08-26T05:58:02.000Z",
            line: { id: "line-1", name: "Software delivery" },
            state: "STATE_ACTIVE",
            stepExecutions: [
              {
                id: "e-plan",
                step: "Planning",
                state: "STATE_FINISHED",
                result: "RESULT_PASSED",
                run: { id: "run-plan" },
              },
              {
                id: "e-impl-old",
                step: "Implementation",
                stepIndex: 1,
                state: "STATE_FINISHED",
                result: "RESULT_FAILED",
                run: { id: "run-old" },
              },
              {
                id: "e-impl-new",
                step: "Implementation",
                stepIndex: 1,
                state: "STATE_STARTED",
                result: "RESULT_UNKNOWN",
                run: { id: "run-new" },
              },
            ],
          },
        ],
      }),
      { lineId: "line-1" },
    );

    const implementPhases = fixture.phases.filter((phase) => phase.name === "Implement");
    expect(implementPhases).toHaveLength(2);
    expect(implementPhases[0].id).not.toBe(implementPhases[1].id);
    expect(implementPhases[0].runId).toBe("run-old");
    expect(implementPhases[1].runId).toBe("run-new");
    expect(implementPhases[1].status).toBe("running");
    expect(fixture.currentPhaseId).toBe(implementPhases[1].id);
  });

  it("uses supplied checks instead of the fixture pills", () => {
    const fixture = splitRunFixtureForWorkOrder(
      order({
        title: "Verify job",
        state: "STATE_OPEN",
        lineDispatches: [
          dispatch("STATE_ACTIVE", [
            { id: "e-verify", step: "Verify", stepIndex: 2, state: "STATE_STARTED", result: "RESULT_UNKNOWN" },
          ]),
        ],
      }),
      { checks: [] },
    );
    expect(fixture.checks).toEqual([]);
  });

  it("opens a running implement on the current phase and hides later steps", () => {
    const fixture = splitRunFixtureForWorkOrder(
      order({
        title: "Implement job",
        state: "STATE_OPEN",
        lineDispatches: [
          dispatch("STATE_ACTIVE", [
            { id: "e-impl", step: "Implement", stepIndex: 0, state: "STATE_STARTED", result: "RESULT_UNKNOWN" },
          ]),
        ],
      }),
    );

    expect(fixture.title).toBe("Implement job");
    expect(fixture.lineStatus).toBe("running");
    expect(fixture.phases.map((phase) => [phase.name, phase.status])).toEqual([
      ["Backlog", "passed"],
      ["Implement", "running"],
    ]);
    expect(fixture.currentPhaseId).toBe("implement-0");
    expect(fixture.phases.at(-1)?.canvasSteps.at(-1)?.status).toBe("running");
  });

  it("marks a pending pull-request step as waiting", () => {
    const fixture = splitRunFixtureForWorkOrder(
      order({
        title: "Waiting job",
        state: "STATE_OPEN",
        lineDispatches: [
          dispatch("STATE_FINISHED", [
            { id: "e-impl", step: "Implement", stepIndex: 0, state: "STATE_FINISHED", result: "RESULT_PASSED" },
            { id: "e-pr", step: "Open pull request", stepIndex: 1, state: "STATE_PENDING", result: "RESULT_UNKNOWN" },
          ]),
        ],
      }),
    );

    expect(fixture.lineStatus).toBe("running");
    expect(fixture.phases.at(-1)?.status).toBe("pending");
    expect(splitRunStatusLabel(fixture.phases.at(-1)!.status)).toBe("Pending");
  });

  it("marks a failed implement step as failed", () => {
    const fixture = splitRunFixtureForWorkOrder(
      order({
        title: "Failed job",
        state: "STATE_OPEN",
        lineDispatches: [
          dispatch("STATE_FINISHED", [
            { id: "e-impl", step: "Implement", stepIndex: 0, state: "STATE_FINISHED", result: "RESULT_FAILED" },
          ]),
        ],
      }),
    );

    expect(fixture.lineStatus).toBe("waiting");
    expect(fixture.phases.at(-1)?.name).toBe("Implement");
    expect(fixture.phases.at(-1)?.status).toBe("failed");
    expect(fixture.phases.at(-1)?.canvasSteps.at(-1)?.status).toBe("failed");
    expect(fixture.footerTone).toBe("failed");
    expect(fixture.waitingNotes.map((note) => note.headline)).toEqual(["Implement did not pass"]);
    expect(fixture.waitingNotes[0]?.cta?.label).toBe("Review the run");
    expect(fixture.footer.attentionCard).toBe(true);
    expect(fixture.footer.actions.map((action) => action.label)).toEqual([]);
    expect(fixture.checks).toEqual([]);
  });

  it("logs a manual create when a person opens a draft", () => {
    const fixture = splitRunFixtureForWorkOrder(DRAFT_WORK_ORDER);
    const backlog = fixture.phases[0];

    expect(fixture.lineStatus).toBe("pending");
    expect(fixture.currentPhaseId).toBe("backlog");
    expect(fixture.footerTone).toBe("draft");
    expect(backlog).toMatchObject({
      id: "backlog",
      name: "Backlog",
      componentName: "Created manually",
      status: "passed",
      canvasKey: null,
    });
    expect(backlog?.stream.map((line) => line.componentName)).toEqual([
      "Leonardo DiCaprio created this work order manually.",
    ]);
    expect(backlog?.artifacts[0]?.data).toMatchObject({
      name: "description.md",
      body: DRAFT_WORK_ORDER.description,
    });
    expect(backlog?.stream[0]?.artifact?.data).toMatchObject({ name: "description.md" });
    expect(fixture.footer.note?.text).toContain("Leonardo DiCaprio created this work order manually.");
  });

  it("logs GitHub ingest as the backlog source", () => {
    const fixture = splitRunFixtureForWorkOrder(INGEST_DRAFT_WORK_ORDER);
    const backlog = fixture.phases[0];

    expect(backlog).toMatchObject({
      name: "Backlog",
      componentName: "Ingest",
      canvasKey: "intake",
      triggerName: "On Issue Label",
      appId: "app-refund-backlog",
    });
    expect(backlog?.artifacts[0]?.data).toMatchObject({
      name: "description.md",
      body: INGEST_DRAFT_WORK_ORDER.description,
    });
  });

  it("logs Sentry and Slack intake as other backlog sources", () => {
    const sentry = splitRunFixtureForWorkOrder(SENTRY_DRAFT_WORK_ORDER).phases[0];
    const slack = splitRunFixtureForWorkOrder(SLACK_DRAFT_WORK_ORDER).phases[0];

    expect(sentry).toMatchObject({
      name: "Backlog",
      componentName: "Sentry",
      canvasKey: "sentry",
      triggerName: "On Issue",
      appId: "app-refund-sentry",
    });
    expect(sentry?.artifacts[0]?.data).toMatchObject({ name: "description.md" });
    expect(slack).toMatchObject({
      name: "Backlog",
      componentName: "Slack",
      canvasKey: "slack",
      triggerName: "On Mention",
      appId: "app-refund-slack",
    });
    expect(slack?.artifacts[0]?.data).toMatchObject({ name: "description.md" });
  });

  it("still prompts a draft with no creator to start", () => {
    const fixture = splitRunFixtureForWorkOrder(
      order({
        title: OPEN_WORK_ORDER.title,
        description: OPEN_WORK_ORDER.description,
        state: "STATE_DRAFT",
      }),
    );
    const backlog = fixture.phases[0];

    expect(fixture.footerTone).toBe("draft");
    expect(backlog?.canvasKey).toBeNull();
    expect(backlog?.stream.map((line) => line.componentName)).toEqual(["Created this work order manually."]);
    expect(backlog?.artifacts[0]?.data).toMatchObject({
      name: "description.md",
      body: OPEN_WORK_ORDER.description,
    });
    expect(fixture.waitingNotes).toEqual([]);
    expect(fixture.footer.note?.headline).toBe("Start this work order");
    expect(fixture.footer.note?.text).toContain("A person created this work order manually.");
    expect(fixture.footer.actions.map((action) => action.label)).toEqual(["Start"]);
  });

  it("omits invented files and ledger pull requests for a live order", () => {
    const fixture = splitRunFixtureForWorkOrder(LINE_BOARD_DONE_RECEIPTS_ORDER, { demoArtifacts: false });
    const names = fixture.phases.flatMap((phase) => artifactNames(phase.artifacts));

    expect(names).not.toContain("merge-screenshot.png");
    expect(names).not.toContain("closure.md");
    expect(names).not.toContain("plan.md");
    expect(names).not.toContain("#510");
    expect(names.some((name) => name.startsWith("feature/"))).toBe(false);
    expect(names.filter((name) => name !== "description.md")).toEqual([]);
  });
});

describe("line board work-order examples", () => {
  it("keeps a plan, a branch, and a pull request on the running GitHub implement card", () => {
    const fixture = splitRunFixtureForWorkOrder(RUNNING_WORK_ORDER);
    expect(fixture.phases.map((phase) => phase.id)).toEqual(["ingest", "analyze", "plan", "score", "implement-0"]);
    expect(artifactNames(fixture.phases.find((phase) => phase.id === "plan")?.artifacts)).toEqual(["plan.md"]);
    expect(artifactNames(fixture.phases.find((phase) => phase.id === "implement-0")?.artifacts)).toEqual([
      "feature/rf-103",
      "#503",
    ]);
  });

  it("keeps ingest analysis, a branch, and a pull request on the approval implement card", () => {
    const fixture = splitRunFixtureForWorkOrder(APPROVAL_WORK_ORDER);
    expect(fixture.phases.map((phase) => phase.id)).toEqual(["ingest", "analyze", "plan", "score", "implement-0"]);
    expect(artifactNames(fixture.phases.find((phase) => phase.id === "plan")?.artifacts)).toEqual(["plan.md"]);
    expect(artifactNames(fixture.phases.find((phase) => phase.id === "implement-0")?.artifacts)).toEqual([
      "feature/rf-109",
      "#509",
    ]);
  });

  it("keeps ingest analysis, a branch, and a pull request on the failed implement card", () => {
    const fixture = splitRunFixtureForWorkOrder(BOARD_IMPLEMENT_FAILED_ORDER);
    expect(fixture.phases.map((phase) => phase.id)).toEqual(["ingest", "analyze", "plan", "score", "implement-0"]);
    expect(artifactNames(fixture.phases.find((phase) => phase.id === "plan")?.artifacts)).toEqual(["plan.md"]);
    expect(artifactNames(fixture.phases.find((phase) => phase.id === "implement-0")?.artifacts)).toEqual([
      "feature/rf-106",
      "#506",
    ]);
    expect(fixture.footerTone).toBe("failed");
    expect(fixture.footer.actions.map((action) => action.label)).toEqual(["Reopen"]);
  });

  it("keeps the branch and pull request on implement for the verify enum card", () => {
    const fixture = splitRunFixtureForWorkOrder(LINE_BOARD_VERIFY_ENUM_ORDER);
    expect(fixture.phases.map((phase) => phase.id)).toEqual([
      "ingest",
      "analyze",
      "plan",
      "score",
      "implement-0",
      "verify-1",
    ]);
    expect(artifactNames(fixture.phases.find((phase) => phase.id === "plan")?.artifacts)).toEqual(["plan.md"]);
    expect(artifactNames(fixture.phases.find((phase) => phase.id === "implement-0")?.artifacts)).toEqual([
      "feature/rf-102",
      "#502",
    ]);
    expect(artifactNames(fixture.phases.find((phase) => phase.id === "verify-1")?.artifacts)).toEqual([]);
    expect(fixture.phases.find((phase) => phase.id === "verify-1")?.checks?.map((check) => check.name)).toEqual([
      "Risk score",
      "Code quality",
    ]);
  });

  it("keeps the ingest confidence check when later steps report their own checks", () => {
    const verify = splitRunFixtureForWorkOrder(LINE_BOARD_VERIFY_ENUM_ORDER, { checks: VERIFY_STEP_CHECKS });
    const done = splitRunFixtureForWorkOrder(LINE_BOARD_DONE_RECEIPTS_ORDER, { checks: VERIFY_STEP_CHECKS });
    const failed = splitRunFixtureForWorkOrder(BOARD_IMPLEMENT_FAILED_ORDER, { checks: VERIFY_STEP_CHECKS });
    const running = splitRunFixtureForWorkOrder(RUNNING_WORK_ORDER, { checks: RUNNING_WORK_ORDER_CHECKS });

    expect(verify.checks.map((check) => check.name)).toEqual(["Confidence score", "Risk score", "Code quality"]);
    expect(done.checks.map((check) => check.name)).toEqual(["Confidence score", "Risk score", "Code quality"]);
    expect(failed.checks.map((check) => check.name)).toEqual(["Confidence score", "Risk score", "Code quality"]);
    expect(running.checks.map((check) => check.name)).toEqual(["Confidence score", "Risk score", "CI"]);
    expect(verify.checks[0]?.summary).toContain("fit for an agent");
    expect(running.checks.filter((check) => check.name === "Confidence score")).toHaveLength(1);
  });

  it("keeps PR #6812 on implement for the waiting verify card", () => {
    const fixture = splitRunFixtureForWorkOrder(LINE_BOARD_VERIFY_PR_REVIEW_ORDER);
    expect(fixture.phases.map((phase) => phase.id)).toEqual([
      "ingest",
      "analyze",
      "plan",
      "score",
      "implement-0",
      "verify-1",
    ]);
    expect(artifactNames(fixture.phases.find((phase) => phase.id === "plan")?.artifacts)).toEqual(["plan.md"]);
    expect(artifactNames(fixture.phases.find((phase) => phase.id === "implement-0")?.artifacts)).toEqual([
      "feature/rf-104",
      "#6812",
    ]);
    expect(artifactNames(fixture.phases.find((phase) => phase.id === "verify-1")?.artifacts)).toEqual([]);
    expect(fixture.phases.find((phase) => phase.id === "verify-1")?.checks?.map((check) => check.name)).toEqual([
      "Risk score",
      "Code quality",
    ]);
    expect(fixture.waitingNotes[0]?.cta?.label).toBe("Review PR #6812");
  });

  it("keeps ingest analysis and the merged receipts pull request on the done card", () => {
    const fixture = splitRunFixtureForWorkOrder(LINE_BOARD_DONE_RECEIPTS_ORDER);
    expect(fixture.phases.map((phase) => phase.id)).toEqual([
      "ingest",
      "analyze",
      "plan",
      "score",
      "implement-0",
      "verify-1",
      "done-2",
    ]);
    expect(artifactNames(fixture.phases.find((phase) => phase.id === "plan")?.artifacts)).toEqual(["plan.md"]);
    expect(artifactNames(fixture.phases.find((phase) => phase.id === "implement-0")?.artifacts)).toEqual([
      "feature/rf-88",
      "#510",
    ]);
    expect(artifactNames(fixture.phases.find((phase) => phase.id === "verify-1")?.artifacts)).toEqual([]);
    expect(fixture.phases.find((phase) => phase.id === "verify-1")?.checks?.map((check) => check.name)).toEqual([
      "Risk score",
      "Code quality",
    ]);
    expect(artifactNames(fixture.phases.find((phase) => phase.id === "done-2")?.artifacts)).toEqual(["#510"]);
  });

  it("keeps ingest analysis and a rejected pull request on the rejected done card", () => {
    const fixture = splitRunFixtureForWorkOrder(BOARD_DONE_REJECTED_ORDER);
    expect(fixture.phases.map((phase) => phase.id)).toEqual([
      "ingest",
      "analyze",
      "plan",
      "score",
      "implement-0",
      "verify-1",
      "done-2",
    ]);
    expect(artifactNames(fixture.phases.find((phase) => phase.id === "plan")?.artifacts)).toEqual(["plan.md"]);
    expect(artifactNames(fixture.phases.find((phase) => phase.id === "implement-0")?.artifacts)).toEqual([
      "feature/rf-112",
      "#512",
    ]);
    expect(artifactNames(fixture.phases.find((phase) => phase.id === "verify-1")?.artifacts)).toEqual([]);
    expect(artifactNames(fixture.phases.find((phase) => phase.id === "done-2")?.artifacts)).toEqual(["#512"]);
    expect(fixture.phases.find((phase) => phase.id === "done-2")?.artifacts[0]?.data).toMatchObject({
      state: "closed",
    });
  });

  it("keeps ingest analysis and a cancel note on the canceled done card", () => {
    const fixture = splitRunFixtureForWorkOrder(BOARD_DONE_CANCELED_ORDER);
    expect(fixture.phases.map((phase) => phase.id)).toEqual([
      "ingest",
      "analyze",
      "plan",
      "score",
      "implement-0",
      "verify-1",
      "done-2",
    ]);
    expect(artifactNames(fixture.phases.find((phase) => phase.id === "plan")?.artifacts)).toEqual(["plan.md"]);
    expect(artifactNames(fixture.phases.find((phase) => phase.id === "implement-0")?.artifacts)).toEqual([
      "feature/rf-113",
      "#513",
    ]);
    expect(artifactNames(fixture.phases.find((phase) => phase.id === "done-2")?.artifacts)).toEqual(["notes.md"]);
  });

  it("keeps a completed notify log on the extra implement card", () => {
    const fixture = splitRunFixtureForWorkOrder(BOARD_IMPLEMENT_NOTIFY_ORDER);
    expect(fixture.title).toBe("Notify on status change after a reopen");
    expect(fixture.lineStatus).toBe("passed");
    expect(fixture.currentPhaseId).toBe("pr-creation-2");
    expect(fixture.openPhaseId).toBe("pr-creation-2");
    expect(fixture.footerTone).toBe("done");
    expect(fixture.footer.run).toEqual({
      appId: "app-refund-implementer",
      runId: LINE_RUN_IMPLEMENT_NOTIFY_ID,
    });
    expect(fixture.phases.find((phase) => phase.id === "implementation-1")?.appId).toBe("app-refund-implementer");
    expect(fixture.phases.find((phase) => phase.id === "implementation-1")?.runId).toBe(LINE_RUN_IMPLEMENT_NOTIFY_ID);
    expect(fixture.footer.sentence).toBe("Work order completed successfully.");
    expect(fixture.footer.actions.map((action) => action.label)).toEqual(["Reopen"]);
    expect(
      fixture.phases.map((phase) => [phase.id, phase.name, phase.componentName, phase.status, phase.duration]),
    ).toEqual([
      ["backlog", "Backlog", "Created manually", "passed", "2s"],
      ["planning-0", "Plan", "Planning", "passed", "2m 59s"],
      ["implementation-1", "Implement", "Implementation", "passed", "23m 56s"],
      ["pr-creation-2", "PR Creation", "PR Creation", "passed", "1m 23s"],
      ["ci-loop-3", "Verify", "Risk Assessment", "passed", "10m 12s"],
      ["risk-assessment-4", "Verify", "Risk Assessment", "passed", "29s"],
      [
        "ui-preview-storybook-coverage-5",
        "UI Preview & Storybook Coverage",
        "UI Preview & Storybook Coverage",
        "passed",
        "1m 26s",
      ],
    ]);
    expect(artifactNames(fixture.phases.find((phase) => phase.id === "backlog")?.artifacts)).toEqual([
      "description.md",
    ]);
    expect(artifactNames(fixture.phases.find((phase) => phase.id === "planning-0")?.artifacts)).toEqual(["PLAN.md"]);
    expect(artifactNames(fixture.phases.find((phase) => phase.id === "implementation-1")?.artifacts)).toEqual([
      "fix/bug-not-getting-notified-for-status-change-when-re-1787246840-4193b6d9",
    ]);
    const prStream = fixture.phases.find((phase) => phase.id === "pr-creation-2")?.stream ?? [];
    expect(prStream.map((line) => [line.at, line.componentType, line.componentName, line.action])).toEqual([
      ["19:51:16", "On Run", "Create", "triggered"],
      ["19:51:16", "Filter", "PR does not exist?", "passed"],
      ["19:51:17", "Run Claude Code", "Generate PR title and description", "passed"],
      ["19:52:37", "github.createPullRequest", "Create Draft Pull Request", "passed"],
      ["19:52:38", "github.addIssueLabel", "Add Label to Pull Request", "passed"],
      ["19:52:39", "Add Work Order Artifact", "Attach PR to Work Order", "passed"],
      ["19:52:39", "setWorkOrderStatusNote", "Set PR closure note", "passed"],
    ]);
    expect(prStream.find((line) => line.componentName === "Attach PR to Work Order")?.artifact?.data).toMatchObject({
      number: 6837,
      state: "merged",
      url: "https://github.com/superplanehq/superplane/pull/6837",
    });
    const verifyStream = fixture.phases.find((phase) => phase.id === "ci-loop-3")?.stream ?? [];
    expect(verifyStream.map((line) => [line.at, line.componentType, line.componentName, line.action])).toEqual([
      ["19:52:40", "On Run", "CI verification", "triggered"],
      ["20:02:50", "Report Work Order Check", "Report CI Check", "passed"],
      ["20:02:50", "github.markPullRequestReadyForReview", "Mark Pull Request Ready", "passed"],
      ["19:52:40", "loop", "loop", "passed"],
      ["19:52:40", "semaphore.runWorkflow", "Run Semaphore CI", "passed"],
    ]);
    const previewStream = fixture.phases.find((phase) => phase.id === "ui-preview-storybook-coverage-5")?.stream ?? [];
    expect(previewStream.map((line) => [line.at, line.componentType, line.componentName, line.action])).toEqual([
      ["20:03:22", "On Run", "Start", "triggered"],
      ["20:03:22", "Run Bash", "Detect UI Changes", "passed"],
      ["20:03:23", "If", "Has UI changes?", "passed"],
      ["20:03:23", "Run Claude Code", "Assess Storybook Coverage", "passed"],
      ["20:03:57", "Run JavaScript", "Format Coverage Review", "passed"],
      ["20:03:23", "Run Bash", "Deploy Storybook", "passed"],
      ["20:03:58", "Report Work Order Check", "Report Coverage Check", "passed"],
      ["20:04:46", "github.updatePullRequest", "Update PR with preview links", "passed"],
    ]);
  });
});
