import { describe, expect, it } from "vitest";
import type { FactoriesWorkOrder, FactoriesWorkOrderExecution, FactoriesWorkOrderLineDispatch } from "@/api-client";

import { OPEN_WORK_ORDER } from "../../__fixtures__/factoryPageResponses";
import { popupFixtureForWorkOrder } from "./workOrderPopupMocks";

const WAITING_NOTE = OPEN_WORK_ORDER.statusNotes;

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

describe("popupFixtureForWorkOrder", () => {
  it("hides notes and scores on a running implement job", () => {
    const fixture = popupFixtureForWorkOrder(
      order({
        title: "Implement job",
        state: "STATE_OPEN",
        statusNotes: WAITING_NOTE,
        lineDispatches: [
          dispatch("STATE_ACTIVE", [
            { id: "e-impl", step: "Implement", stepIndex: 0, state: "STATE_STARTED", result: "RESULT_UNKNOWN" },
          ]),
        ],
      }),
    );

    expect(fixture.waitingNotes).toEqual([]);
    expect(fixture.checks).toEqual([]);
    expect(fixture.log.map((entry) => [entry.actor, entry.state])).toEqual([
      ["Backlog", "passed"],
      ["Implement", "running"],
    ]);
    expect(fixture.log[0]?.title).toBe("Create work order");
    expect(fixture.log[0]?.artifactId).toBe("art-description");
  });

  it("shows scores only on the verify phase", () => {
    const fixture = popupFixtureForWorkOrder(
      order({
        title: "Verify job",
        state: "STATE_OPEN",
        statusNotes: WAITING_NOTE,
        lineDispatches: [
          dispatch("STATE_ACTIVE", [
            { id: "e-impl", step: "Implement", stepIndex: 0, state: "STATE_FINISHED", result: "RESULT_PASSED" },
            { id: "e-verify", step: "Verify", stepIndex: 1, state: "STATE_STARTED", result: "RESULT_UNKNOWN" },
          ]),
        ],
      }),
    );

    expect(fixture.waitingNotes).toEqual([]);
    expect(fixture.checks.length).toBeGreaterThan(0);
    expect(fixture.log.map((entry) => entry.actor)).toEqual(["Backlog", "Implement", "Verify"]);
    expect(fixture.log.at(-1)?.state).toBe("running");
  });

  it("shows notes only when the work order is waiting", () => {
    const fixture = popupFixtureForWorkOrder(
      order({
        title: "Waiting job",
        state: "STATE_OPEN",
        statusNotes: WAITING_NOTE,
        lineDispatches: [
          dispatch("STATE_FINISHED", [
            { id: "e-impl", step: "Implement", stepIndex: 0, state: "STATE_FINISHED", result: "RESULT_FAILED" },
          ]),
        ],
      }),
    );

    expect(fixture.waitingNotes.map((note) => note.headline)).toEqual(["Review the pull request"]);
    expect(fixture.checks).toEqual([]);
    expect(fixture.log.at(-1)?.state).toBe("failed");
  });

  it("hides notes and scores on a completed done job", () => {
    const fixture = popupFixtureForWorkOrder(
      order({
        title: "Done job",
        state: "STATE_CLOSED",
        result: "RESULT_COMPLETED",
        statusNotes: WAITING_NOTE,
        lineDispatches: [
          dispatch("STATE_FINISHED", [
            { id: "e-impl", step: "Implement", stepIndex: 0, state: "STATE_FINISHED", result: "RESULT_PASSED" },
            { id: "e-verify", step: "Verify", stepIndex: 1, state: "STATE_FINISHED", result: "RESULT_PASSED" },
            { id: "e-done", step: "Done", stepIndex: 2, state: "STATE_FINISHED", result: "RESULT_PASSED" },
          ]),
        ],
      }),
    );

    expect(fixture.waitingNotes).toEqual([]);
    expect(fixture.checks).toEqual([]);
    expect(fixture.log.map((entry) => [entry.actor, entry.title])).toEqual([
      ["Backlog", "Create work order"],
      ["Implement", "Implement"],
      ["Verify", "Verify"],
      ["Done", "Complete work order"],
    ]);
  });

  it("shows a PR Closure complete step with the pull request", () => {
    const fixture = popupFixtureForWorkOrder(
      order({
        title: "Merged job",
        state: "STATE_CLOSED",
        result: "RESULT_COMPLETED",
        lineDispatches: [
          dispatch("STATE_FINISHED", [
            { id: "e-impl", step: "Implement", stepIndex: 0, state: "STATE_FINISHED", result: "RESULT_PASSED" },
            { id: "e-verify", step: "Verify", stepIndex: 1, state: "STATE_FINISHED", result: "RESULT_PASSED" },
            {
              id: "e-done",
              step: "Done",
              stepIndex: 2,
              state: "STATE_FINISHED",
              result: "RESULT_PASSED",
              run: { appId: "app-refund-done", appName: "PR Closure" },
            },
          ]),
        ],
      }),
    );

    expect(fixture.log.at(-1)).toMatchObject({
      actor: "Done",
      title: "Complete work order from merged pull request",
      artifactId: "art-pr-closure",
    });
  });

  it("shows reject titles for a user and for PR Closure", () => {
    const userReject = popupFixtureForWorkOrder(
      order({
        title: "User reject",
        state: "STATE_CLOSED",
        result: "RESULT_REJECTED",
        lineDispatches: [
          dispatch("STATE_FINISHED", [
            {
              id: "e-done",
              step: "Done",
              stepIndex: 2,
              state: "STATE_FINISHED",
              result: "RESULT_PASSED",
            },
          ]),
        ],
      }),
    );
    const automationReject = popupFixtureForWorkOrder(
      order({
        title: "Automation reject",
        state: "STATE_CLOSED",
        result: "RESULT_REJECTED",
        lineDispatches: [
          dispatch("STATE_FINISHED", [
            {
              id: "e-done",
              step: "Done",
              stepIndex: 2,
              state: "STATE_FINISHED",
              result: "RESULT_PASSED",
              run: { appId: "app-refund-done", appName: "PR Closure" },
            },
          ]),
        ],
      }),
    );

    expect(userReject.log.at(-1)?.title).toBe("Reject work order");
    expect(automationReject.log.at(-1)?.title).toBe("Reject work order from closed pull request");
  });

  it("shows a backlog create step with description.md for a user draft", () => {
    const fixture = popupFixtureForWorkOrder(
      order({
        title: "Draft job",
        description: "User-written scope.",
        state: "STATE_DRAFT",
        createdBy: { user: { id: "user-1", name: "Ada" } },
        statusNotes: WAITING_NOTE,
        lineDispatches: [],
      }),
    );

    expect(fixture.waitingNotes).toEqual([]);
    expect(fixture.checks).toEqual([]);
    expect(fixture.log.map((entry) => [entry.actor, entry.title, entry.state])).toEqual([
      ["Backlog", "Create work order", "passed"],
    ]);
    expect(fixture.log[0]?.artifactId).toBe("art-description");
    expect(fixture.description.data).toMatchObject({ name: "description.md", body: "User-written scope." });
  });

  it("shows a backlog ingest step for an automation draft", () => {
    const fixture = popupFixtureForWorkOrder(
      order({
        title: "Ingested job",
        description: "Issue body from GitHub.",
        state: "STATE_DRAFT",
        createdBy: { automation: { appId: "app-ingest", appName: "Ingest", nodeName: "Create Work Order" } },
        lineDispatches: [],
      }),
    );

    expect(fixture.log.map((entry) => [entry.actor, entry.title, entry.state])).toEqual([
      ["Backlog", "Create work order from GitHub issue", "passed"],
    ]);
    expect(fixture.log[0]?.artifactId).toBe("art-description");
  });
});
