import { describe, expect, it } from "vitest";

import { DRAFT_WORK_ORDER, OPEN_WORK_ORDER, RUNNING_WORK_ORDER } from "../../__fixtures__/factoryPageResponses";
import {
  BOARD_DONE_REJECTED_ORDER,
  BOARD_IMPLEMENT_FAILED_ORDER,
  BOARD_IMPLEMENT_NOTIFY_ORDER,
} from "../../__fixtures__/lineMetricsBoardOrders";
import { LINE_BOARD_DONE_RECEIPTS_ORDER } from "../../__fixtures__/lineMetricsFactoriesFixture";
import { REVIEW_CANDIDATE_WORK_ORDERS } from "../onboarding/first-run/reviewCandidates";
import { autoExpandedPhaseId, splitRunFixtureForWorkOrder } from "./splitRunMocks";
import { SPLIT_RUN_RUNNING } from "./splitRunRunningFixture";

describe("autoExpandedPhaseId", () => {
  it("expands a running step", () => {
    expect(autoExpandedPhaseId(SPLIT_RUN_RUNNING)).toBe("implement");
    expect(autoExpandedPhaseId(splitRunFixtureForWorkOrder(RUNNING_WORK_ORDER))).toMatch(/^implement-/);
  });

  it("expands a failed step", () => {
    const fixture = splitRunFixtureForWorkOrder(BOARD_IMPLEMENT_FAILED_ORDER);
    expect(autoExpandedPhaseId(fixture)).toBe(fixture.currentPhaseId);
    expect(fixture.phases.find((phase) => phase.id === fixture.currentPhaseId)?.status).toBe("failed");
  });

  it("expands a waiting step that needs attention", () => {
    const fixture = {
      ...SPLIT_RUN_RUNNING,
      currentPhaseId: "implement",
      phases: SPLIT_RUN_RUNNING.phases.map((phase) =>
        phase.id === "implement" ? { ...phase, status: "waiting" as const } : phase,
      ),
    };
    expect(autoExpandedPhaseId(fixture)).toBe("implement");
  });

  it("keeps backlog and done steps collapsed", () => {
    expect(autoExpandedPhaseId(splitRunFixtureForWorkOrder(DRAFT_WORK_ORDER))).toBeNull();
    expect(autoExpandedPhaseId(splitRunFixtureForWorkOrder(REVIEW_CANDIDATE_WORK_ORDERS[0]))).toBeNull();
    expect(autoExpandedPhaseId(splitRunFixtureForWorkOrder(LINE_BOARD_DONE_RECEIPTS_ORDER))).toBeNull();
    expect(autoExpandedPhaseId(splitRunFixtureForWorkOrder(BOARD_DONE_REJECTED_ORDER))).toBeNull();
  });

  it("expands the completed PR Creation step on the notify implement card", () => {
    const fixture = splitRunFixtureForWorkOrder(BOARD_IMPLEMENT_NOTIFY_ORDER);
    expect(fixture.openPhaseId).toBe("pr-creation-2");
    expect(autoExpandedPhaseId(fixture)).toBe("pr-creation-2");
  });

  it("expands the oldest active PR feedback run", () => {
    const fixture = splitRunFixtureForWorkOrder(OPEN_WORK_ORDER, {
      prFeedbackRuns: [
        {
          canvasId: "canvas-fb",
          run: {
            id: "run-fb",
            title: "Address feedback on PR #12",
            status: "STATUS_QUEUED",
            createdAt: "2026-08-26T11:00:00Z",
          },
        },
      ],
    });

    expect(autoExpandedPhaseId(fixture)).toBe("pr-feedback-run-fb");
  });

  it("does not expand a passed current step while the order waits", () => {
    const fixture = splitRunFixtureForWorkOrder({
      ...OPEN_WORK_ORDER,
      lineDispatches: [
        {
          id: "dispatch-1",
          line: { id: "line-1", name: "plan-and-implement" },
          state: "STATE_FINISHED",
          stepExecutions: [
            { id: "e-impl", step: "Implement", stepIndex: 0, state: "STATE_FINISHED", result: "RESULT_PASSED" },
          ],
        },
      ],
    });
    expect(fixture.footerTone).toBe("waiting");
    expect(autoExpandedPhaseId(fixture)).toBeNull();
  });
});
