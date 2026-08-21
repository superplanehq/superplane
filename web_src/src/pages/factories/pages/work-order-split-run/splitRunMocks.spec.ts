import type { FactoriesWorkOrder, FactoriesWorkOrderExecution, FactoriesWorkOrderLineDispatch } from "@/api-client";
import { describe, expect, it } from "vitest";

import { OPEN_WORK_ORDER, RUNNING_WORK_ORDER } from "../../__fixtures__/factoryPageResponses";
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

describe("splitRunFixtureForWorkOrder", () => {
  it("uses the designed running fixture when the order is missing", () => {
    const fixture = splitRunFixtureForWorkOrder();
    expect(fixture.title).toBe("Add refund reconciliation test");
    expect(fixture.currentPhaseId).toBe("implement");
    expect(fixture.phases.map((phase) => [phase.name, phase.status])).toEqual([
      ["Backlog", "passed"],
      ["Plan", "passed"],
      ["Implement", "running"],
    ]);
  });

  it("maps the running reconciliation card from the work-order execution", () => {
    const fixture = splitRunFixtureForWorkOrder(RUNNING_WORK_ORDER);
    expect(fixture.title).toBe("Add refund reconciliation test");
    expect(fixture.costUsd).toBe("$0.73");
    expect(fixture.tokensLabel).toBe("2.7k tokens");
    expect(fixture.lineStatus).toBe("running");
    expect(fixture.currentPhaseId).toMatch(/^refund-implementer-/);
    const implement = fixture.phases.find((phase) => phase.name === "Refund Implementer");
    expect(implement?.status).toBe("running");
    expect(implement?.appId).toBe("app-refund-implementer");
    expect(implement?.runId).toBe(RUNNING_WORK_ORDER.lineDispatches?.[0]?.stepExecutions?.[1]?.run?.id);
    expect(implement?.stream.map((line) => line.componentName)).toEqual(["Refund Implementer"]);
    expect(fixture.waitingNotes).toEqual([]);
    expect(fixture.checks).toEqual([]);
  });

  it("pins a pull request review on a waiting implement card", () => {
    const fixture = splitRunFixtureForWorkOrder(
      order({
        title: "Ship idempotent refund retries",
        state: "STATE_OPEN",
        statusNotes: OPEN_WORK_ORDER.statusNotes,
        lineDispatches: [
          dispatch("STATE_FINISHED", [
            { id: "e-plan", step: "Plan", stepIndex: 0, state: "STATE_FINISHED", result: "RESULT_PASSED" },
            { id: "e-impl", step: "Implement", stepIndex: 1, state: "STATE_FINISHED", result: "RESULT_PASSED" },
          ]),
        ],
      }),
    );
    expect(fixture.footerTone).toBe("waiting");
    expect(fixture.waitingNotes.map((note) => note.headline)).toEqual(["Review the pull request"]);
    expect(fixture.waitingNotes[0]?.cta?.label).toBe("Review PR #6812");
    expect(fixture.checks).toEqual([]);
  });

  it("does not invent a pull request review when the order has no notes", () => {
    const fixture = splitRunFixtureForWorkOrder(
      order({
        title: "Ship idempotent refund retries",
        state: "STATE_OPEN",
        lineDispatches: [
          dispatch("STATE_FINISHED", [
            { id: "e-plan", step: "Plan", stepIndex: 0, state: "STATE_FINISHED", result: "RESULT_PASSED" },
            { id: "e-impl", step: "Implement", stepIndex: 1, state: "STATE_FINISHED", result: "RESULT_PASSED" },
          ]),
        ],
      }),
    );
    expect(fixture.footerTone).toBe("waiting");
    expect(fixture.waitingNotes).toEqual([]);
  });

  it("shows checks on verify and done cards only when the API supplies them", () => {
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
    );
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
              { id: "e-plan", step: "Plan", stepIndex: 0, state: "STATE_STARTED", result: "RESULT_UNKNOWN" },
            ],
          },
        ],
      }),
      { lineId: "line-1" },
    );
    expect(fixture.lineName).toBe("plan-and-implement");
    expect(fixture.currentPhaseId).toMatch(/^plan-/);
    expect(fixture.phases.some((phase) => phase.name === "Plan")).toBe(true);
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

  it("opens a running plan on the current phase and hides later steps", () => {
    const fixture = splitRunFixtureForWorkOrder(
      order({
        title: "Plan job",
        state: "STATE_OPEN",
        lineDispatches: [
          dispatch("STATE_ACTIVE", [
            { id: "e-plan", step: "Plan", stepIndex: 0, state: "STATE_STARTED", result: "RESULT_UNKNOWN" },
          ]),
        ],
      }),
    );

    expect(fixture.title).toBe("Plan job");
    expect(fixture.lineStatus).toBe("running");
    expect(fixture.phases.map((phase) => [phase.name, phase.status])).toEqual([["Plan", "running"]]);
    expect(fixture.currentPhaseId).toBe("plan-0");
    expect(fixture.phases.at(-1)?.canvasSteps.at(-1)?.status).toBe("running");
  });

  it("marks a pending pull-request step as waiting", () => {
    const fixture = splitRunFixtureForWorkOrder(
      order({
        title: "Waiting job",
        state: "STATE_OPEN",
        lineDispatches: [
          dispatch("STATE_FINISHED", [
            { id: "e-plan", step: "Plan", stepIndex: 0, state: "STATE_FINISHED", result: "RESULT_PASSED" },
            { id: "e-pr", step: "Open pull request", stepIndex: 1, state: "STATE_PENDING", result: "RESULT_UNKNOWN" },
          ]),
        ],
      }),
    );

    expect(fixture.lineStatus).toBe("waiting");
    expect(fixture.phases.at(-1)?.status).toBe("waiting");
    expect(splitRunStatusLabel(fixture.phases.at(-1)!.status)).toBe("Needs attention");
  });

  it("marks a failed implement step as failed", () => {
    const fixture = splitRunFixtureForWorkOrder(
      order({
        title: "Failed job",
        state: "STATE_OPEN",
        lineDispatches: [
          dispatch("STATE_FINISHED", [
            { id: "e-plan", step: "Plan", stepIndex: 0, state: "STATE_FINISHED", result: "RESULT_PASSED" },
            { id: "e-impl", step: "Implement", stepIndex: 1, state: "STATE_FINISHED", result: "RESULT_FAILED" },
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
    expect(fixture.waitingNotes[0]?.cta?.label).toBe("Open failed run");
    expect(fixture.checks).toEqual([]);
  });

  it("lists no log rows when the work order has no step runs", () => {
    const fixture = splitRunFixtureForWorkOrder(
      order({
        title: OPEN_WORK_ORDER.title,
        description: OPEN_WORK_ORDER.description,
        state: "STATE_DRAFT",
      }),
    );

    expect(fixture.lineStatus).toBe("pending");
    expect(fixture.phases).toEqual([]);
    expect(fixture.currentPhaseId).toBe("");
    expect(fixture.footerTone).toBe("draft");
    expect(fixture.waitingNotes.map((note) => note.headline)).toEqual(["Start the next stage"]);
    expect(fixture.waitingNotes[0]?.cta?.label).toBe("Dispatch");
    expect(fixture.checks).toEqual([]);
  });
});
