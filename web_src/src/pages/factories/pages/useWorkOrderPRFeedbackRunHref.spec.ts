import { describe, expect, it } from "vitest";

import type { CanvasesCanvasRunRef, FactoriesFactoryPullRequest } from "@/api-client";

import { isActiveCanvasRun, statusForCanvasRun } from "../lib/workOrderPullRequest";
import { prFeedbackLogRunsFromPullRequests } from "./useWorkOrderPRFeedbackRunHref";

function run(overrides: CanvasesCanvasRunRef): CanvasesCanvasRunRef {
  return overrides;
}

describe("prFeedbackLogRunsFromPullRequests", () => {
  it("returns matching runs oldest first and skips items without a canvas or run id", () => {
    const pullRequests: FactoriesFactoryPullRequest[] = [
      {
        workOrderId: "wo-1",
        number: "42",
        runs: [
          { run: run({ id: "later", canvasId: "c-1", state: "STATE_STARTED", createdAt: "2026-08-26T12:00:00Z" }) },
          {
            run: run({
              canvasId: "c-2",
              state: "STATE_FINISHED",
              result: "RESULT_PASSED",
              createdAt: "2026-08-26T08:00:00Z",
            }),
          },
          {
            run: run({
              id: "older",
              canvasId: "c-2",
              state: "STATE_FINISHED",
              result: "RESULT_PASSED",
              createdAt: "2026-08-26T11:00:00Z",
            }),
          },
        ],
      },
    ];

    const matches = prFeedbackLogRunsFromPullRequests(pullRequests, [{ canvasId: "c-2", name: "Second handler" }]);
    expect(matches.map((match) => match.run.id)).toEqual(["older", "later"]);
    expect(matches[0]?.handlerName).toBe("Second handler");
    expect(matches[0]?.pullRequestNumber).toBe("42");
  });

  it("prefers activity labels over the raw run description", () => {
    const pullRequests: FactoriesFactoryPullRequest[] = [
      {
        workOrderId: "wo-1",
        number: "7",
        runs: [
          {
            description: "Fixing failed checks on a82fd91",
            costCents: "12",
            run: run({ id: "r1", canvasId: "c-1", state: "STATE_STARTED", createdAt: "2026-08-26T12:00:00Z" }),
          },
        ],
        activities: [
          {
            description: "Fixing failed checks on a82fd91",
            access: "waiting",
            state: "active",
            attempt: 2,
            attemptLimit: 3,
            run: run({ id: "r1", canvasId: "c-1", state: "STATE_STARTED", createdAt: "2026-08-26T12:00:00Z" }),
          },
        ],
      },
    ];

    const matches = prFeedbackLogRunsFromPullRequests(pullRequests, [
      { canvasId: "c-1", name: "Fix pull request checks" },
    ]);
    expect(matches).toHaveLength(1);
    expect(matches[0]?.description).toBe("Waiting for another pull request activity");
    expect(matches[0]?.attemptLabel).toBe("Attempt 2 of 3");
    expect(matches[0]?.costCents).toBe("12");
  });

  it("uses linked-run spend when activity fields are zero from emit-unpopulated", () => {
    const matches = prFeedbackLogRunsFromPullRequests(
      [
        {
          workOrderId: "wo-1",
          number: "7",
          runs: [
            {
              description: "Fixing failed checks on d8b80c2",
              costCents: "45",
              totalTokens: "1200",
              run: run({ id: "r1", canvasId: "c-1", state: "STATE_FINISHED", createdAt: "2026-08-26T12:00:00Z" }),
            },
          ],
          activities: [
            {
              description: "Fixing failed checks on d8b80c2",
              access: "exclusive",
              state: "finished",
              costCents: "0",
              totalTokens: "0",
              run: run({ id: "r1", canvasId: "c-1", state: "STATE_FINISHED", createdAt: "2026-08-26T12:00:00Z" }),
            },
          ],
        },
      ],
      [{ canvasId: "c-1", name: "Fix pull request checks" }],
    );
    expect(matches[0]?.costCents).toBe("45");
    expect(matches[0]?.totalTokens).toBe("1200");
  });

  it("reads cost and tokens from the activity when the legacy run list is empty", () => {
    const matches = prFeedbackLogRunsFromPullRequests(
      [
        {
          workOrderId: "wo-1",
          number: "7",
          activities: [
            {
              description: "Fixing failed checks on d8b80c2",
              access: "exclusive",
              state: "active",
              costCents: "45",
              totalTokens: "1200",
              run: run({ id: "r1", canvasId: "c-1", state: "STATE_STARTED", createdAt: "2026-08-26T12:00:00Z" }),
            },
          ],
        },
      ],
      [{ canvasId: "c-1", name: "Fix pull request checks" }],
    );
    expect(matches).toHaveLength(1);
    expect(matches[0]?.costCents).toBe("45");
    expect(matches[0]?.totalTokens).toBe("1200");
  });
});

describe("statusForCanvasRun", () => {
  it("maps generic run state onto log phase status", () => {
    expect(statusForCanvasRun(run({ state: "STATE_PENDING" }))).toBe("pending");
    expect(statusForCanvasRun(run({ state: "STATE_STARTED" }))).toBe("running");
    expect(statusForCanvasRun(run({ state: "STATE_FINISHED", result: "RESULT_PASSED" }))).toBe("passed");
    expect(statusForCanvasRun(run({ state: "STATE_FINISHED", result: "RESULT_FAILED" }))).toBe("failed");
    expect(statusForCanvasRun(run({ state: "STATE_FINISHED", result: "RESULT_CANCELLED" }))).toBe("failed");
  });
});

describe("isActiveCanvasRun", () => {
  it("treats pending, started, and cancelling as active", () => {
    expect(isActiveCanvasRun(run({ id: "1", state: "STATE_PENDING" }))).toBe(true);
    expect(isActiveCanvasRun(run({ id: "2", state: "STATE_STARTED" }))).toBe(true);
    expect(isActiveCanvasRun(run({ id: "3", state: "STATE_CANCELLING" }))).toBe(true);
    expect(isActiveCanvasRun(run({ id: "4", state: "STATE_FINISHED", result: "RESULT_PASSED" }))).toBe(false);
    expect(isActiveCanvasRun(run({ id: "5", state: "STATE_FINISHED", result: "RESULT_FAILED" }))).toBe(false);
  });
});
