import { DRAFT_WORK_ORDER, RUNNING_WORK_ORDER } from "../__fixtures__/factoryPageResponses";
import { splitRunFixtureForWorkOrder } from "./work-order-split-run/splitRunMocks";
import { expect, it } from "bun:test";
import { applyPlanningSessionLiveRun, createWithAgentViewFromSession } from "./planningSessionView";

it("keeps a quota-failed planning attempt failed after it produced a score", () => {
  const view = createWithAgentViewFromSession(
    {
      id: "session-quota",
      state: "ended",
      executionId: "execution-quota",
      runnerState: "failed",
      subscriptionFailure: { reason: "chatgpt_quota_exhausted", subscriptionOwnerId: "owner-1" },
      messages: [{ id: "score", role: "plan", text: JSON.stringify({ score: 3 }) }],
    },
    { composer: "", right: { kind: "empty" }, endConfirmOpen: false },
  );
  expect(applyPlanningSessionLiveRun(view, { result: "RESULT_FAILED" }, true).machineStatus).toBe("failed");
  expect(view.messages).toEqual([expect.objectContaining({ kind: "plan", score: 3 })]);
});

it.each(["STATE_PENDING", "STATE_STARTED"] as const)(
  "clears the current quota notice for a new %s execution and keeps history",
  (state) => {
    const dispatch = RUNNING_WORK_ORDER.lineDispatches?.[0];
    const execution = dispatch?.stepExecutions?.[0];
    if (!dispatch || !execution) throw new Error("A running execution fixture is required");
    const failure = { reason: "chatgpt_quota_exhausted", subscriptionOwnerId: "owner-1" };
    const previous = {
      ...dispatch,
      id: "old-dispatch",
      state: "STATE_FINISHED" as const,
      stepExecutions: [
        {
          ...execution,
          id: "old-execution",
          state: "STATE_FINISHED" as const,
          result: "RESULT_FAILED" as const,
          subscriptionFailure: failure,
          usageState: "partial",
        },
      ],
    };
    const current = {
      ...dispatch,
      id: "new-dispatch",
      stepExecutions: [
        {
          ...execution,
          id: "new-execution",
          state,
          result: "RESULT_UNKNOWN" as const,
          run: { ...execution.run, id: "new-run" },
        },
      ],
    };
    const fixture = splitRunFixtureForWorkOrder({
      ...RUNNING_WORK_ORDER,
      planningSession: { id: "old-planning", subscriptionFailure: failure, usageState: "unavailable" },
      lineDispatches: [previous, current],
    });
    expect(fixture.subscriptionFailure).toBeUndefined();
    expect(fixture.usageState).toBeUndefined();
    expect(
      fixture.phases.some(
        (phase) => phase.historyRun && phase.subscriptionFailure?.reason === "chatgpt_quota_exhausted",
      ),
    ).toBe(true);
  },
);

it("keeps a saved planning phase failed when a quota-limited run already produced its plan and score", () => {
  const failure = { reason: "chatgpt_quota_exhausted", subscriptionOwnerId: "owner-1" };
  const fixture = splitRunFixtureForWorkOrder(DRAFT_WORK_ORDER, {
    demoArtifacts: false,
    checks: [{ key: "clarity", name: "Clarity score", score: 3, runId: "quota-history" }],
    artifacts: [{ type: "TYPE_MARKDOWN", data: { name: "plan.md", body: "# Plan" } }],
    analysisRuns: [
      {
        canvasId: "canvas",
        workOrderId: DRAFT_WORK_ORDER.id ?? "draft",
        run: {
          id: "quota-history",
          state: "STATE_FINISHED",
          result: "RESULT_FAILED",
          subscriptionFailure: failure,
          usageState: "partial",
        },
      },
    ],
  });
  const phase = fixture.phases.find((phase) => phase.runId === "quota-history");
  expect(phase?.status).toBe("failed");
  expect(phase?.subscriptionFailure).toEqual(failure);
  expect(phase?.usageState).toBe("partial");
});

it("shows a quota failure from the live run before the planning session refreshes", () => {
  const view = createWithAgentViewFromSession(
    { id: "session", state: "active", executionId: "execution" },
    { composer: "", right: { kind: "empty" }, endConfirmOpen: false },
  );
  const stopped = applyPlanningSessionLiveRun(
    view,
    {
      result: "RESULT_FAILED",
      subscriptionFailure: { reason: "chatgpt_quota_exhausted", subscriptionOwnerId: "owner-1" },
    },
    true,
  );
  expect(stopped.machineStatus).toBe("failed");
  expect(stopped.subscriptionFailure?.subscriptionOwnerId).toBe("owner-1");
});

it("retains an earlier quota failure when planning merges a new attempt into the same conversation", () => {
  const failure = { reason: "chatgpt_quota_exhausted", subscriptionOwnerId: "owner-1" };
  const now = Date.now();
  const fixture = splitRunFixtureForWorkOrder(DRAFT_WORK_ORDER, {
    demoArtifacts: false,
    analysisRuns: [
      {
        canvasId: "canvas",
        workOrderId: "draft",
        run: {
          id: "old-attempt",
          createdAt: new Date(now - 60000).toISOString(),
          state: "STATE_FINISHED",
          result: "RESULT_FAILED",
          subscriptionFailure: failure,
        },
      },
      {
        canvasId: "canvas",
        workOrderId: "draft",
        run: { id: "new-attempt", createdAt: new Date(now).toISOString(), state: "STATE_STARTED" },
      },
    ],
  });
  const phase = fixture.phases.find((phase) => phase.runId === "new-attempt");
  expect(phase?.subscriptionFailure).toBeUndefined();
  expect(phase?.subscriptionFailureHistory).toEqual([{ runId: "old-attempt", failure }]);
});
