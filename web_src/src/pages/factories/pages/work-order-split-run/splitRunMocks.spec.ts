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

  it("keeps the implement stream on the running reconciliation card", () => {
    const fixture = splitRunFixtureForWorkOrder(RUNNING_WORK_ORDER);
    expect(fixture.title).toBe("Add refund reconciliation test");
    expect(fixture.costUsd).toBe("$0.73");
    expect(fixture.tokensLabel).toBe("2.7k tokens");
    expect(fixture.lineStatus).toBe("running");
    expect(fixture.currentPhaseId).toBe("implement");
    const implement = fixture.phases.find((phase) => phase.id === "implement");
    expect(implement?.status).toBe("running");
    expect(implement?.stream.map((line) => line.componentName)).toContain("Refund Implementer");
    expect(implement?.stream.map((line) => line.componentName)).toContain("Write File");
    expect(fixture.waitingNotes).toEqual([]);
    expect(fixture.checks).toEqual([]);
  });

  it("pins a pull request review on a waiting implement card", () => {
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
    expect(fixture.waitingNotes.map((note) => note.headline)).toEqual(["Review the pull request"]);
    expect(fixture.waitingNotes[0]?.cta?.label).toBe("Review PR #6812");
    expect(fixture.checks).toEqual([]);
  });

  it("shows checks on verify and done cards", () => {
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
    expect(verify.checks.map((check) => check.name)).toEqual([
      "Risk review",
      "Code coverage",
      "Test coverage",
      "Confidence score",
      "CI",
    ]);
    expect(verify.waitingNotes).toEqual([]);

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
    expect(done.checks.map((check) => check.name)).toContain("Risk review");
    expect(done.waitingNotes).toEqual([]);
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
    expect(fixture.phases.map((phase) => [phase.name, phase.status])).toEqual([
      ["Backlog", "passed"],
      ["Plan", "running"],
    ]);
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
    expect(splitRunStatusLabel(fixture.phases.at(-1)!.status)).toBe("Waiting");
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

  it("keeps a backlog card on the create-work-order step", () => {
    const fixture = splitRunFixtureForWorkOrder(
      order({
        title: OPEN_WORK_ORDER.title,
        description: OPEN_WORK_ORDER.description,
        state: "STATE_DRAFT",
      }),
    );

    expect(fixture.lineStatus).toBe("pending");
    expect(fixture.phases).toHaveLength(1);
    expect(fixture.phases[0]?.name).toBe("Backlog");
    expect(fixture.phases[0]?.artifacts[0]?.data).toMatchObject({ name: "description.md" });
    expect(fixture.currentPhaseId).toBe("backlog");
    expect(fixture.footerTone).toBe("draft");
    expect(fixture.waitingNotes.map((note) => note.headline)).toEqual(["Start the next stage"]);
    expect(fixture.waitingNotes[0]?.cta?.label).toBe("Start Plan");
    expect(fixture.checks).toEqual([]);
  });
});
