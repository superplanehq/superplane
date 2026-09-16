import { describe, expect, it } from "bun:test";

import type { CanvasesCanvasRunRef, FactoriesFactoryPullRequest } from "@/api-client";

import {
  activePRFeedbackWorkOrderIds,
  addressingFeedbackLabelsByWorkOrder,
  addressingFeedbackWorkOrderIds,
  checksPassedWorkOrderIds,
  fixesPausedWorkOrderIds,
  waitingOnChecksWorkOrderIds,
} from "./prFeedbackSettingsModel";

function run(overrides: CanvasesCanvasRunRef): CanvasesCanvasRunRef {
  return overrides;
}

describe("activePRFeedbackWorkOrderIds", () => {
  it("returns tasks that have an active pull request run", () => {
    expect(
      activePRFeedbackWorkOrderIds([
        { workOrderId: "wo-1", runs: [{ run: { id: "r1", state: "STATE_PENDING" } }] },
        { workOrderId: "wo-2", runs: [{ run: { id: "r2", state: "STATE_FINISHED", result: "RESULT_PASSED" } }] },
        { workOrderId: "wo-3", runs: [{ run: { id: "r3", state: "STATE_STARTED" } }] },
        { runs: [{ run: { id: "r4", state: "STATE_PENDING" } }] },
      ]),
    ).toEqual(new Set(["wo-1", "wo-3"]));
  });

  it("does not treat a cancelling or cancelled check wait as waiting", () => {
    expect(
      waitingOnChecksWorkOrderIds([
        {
          workOrderId: "wo-cancelling",
          activities: [
            {
              access: "concurrent",
              state: "active",
              description: "Waiting for checks on a82fd91",
              run: run({ id: "r-cancelling", state: "STATE_CANCELLING" }),
            },
          ],
        },
        {
          workOrderId: "wo-cancelled",
          activities: [
            {
              access: "concurrent",
              state: "active",
              description: "Waiting for checks on a82fd91",
              run: run({ id: "r-cancelled", state: "STATE_FINISHED", result: "RESULT_CANCELLED" }),
            },
          ],
        },
      ]),
    ).toEqual(new Set());
  });

  it("labels concurrent custom automation activity with its description", () => {
    const pullRequests: FactoriesFactoryPullRequest[] = [
      {
        workOrderId: "wo-custom",
        activities: [
          {
            access: "concurrent",
            state: "active",
            description: "Creating an ephemeral environment",
            run: run({ id: "r-custom", state: "STATE_STARTED" }),
          },
        ],
      },
    ];

    expect(waitingOnChecksWorkOrderIds(pullRequests)).toEqual(new Set());
    expect(addressingFeedbackWorkOrderIds(pullRequests)).toEqual(new Set(["wo-custom"]));
    expect(addressingFeedbackLabelsByWorkOrder(pullRequests).get("wo-custom")).toBe(
      "Creating an ephemeral environment",
    );
    expect(checksPassedWorkOrderIds(pullRequests)).toEqual(new Set());
    expect(fixesPausedWorkOrderIds(pullRequests)).toEqual(new Set());
  });

  it("labels exclusive custom automation activity with its description", () => {
    expect(
      addressingFeedbackLabelsByWorkOrder(
        [
          {
            workOrderId: "wo-custom",
            activities: [
              {
                access: "exclusive",
                state: "active",
                description: "Deploying preview changes",
                run: run({ id: "r-custom", state: "STATE_STARTED", canvasId: "app-custom" }),
              },
            ],
          },
        ],
        new Set(),
      ).get("wo-custom"),
    ).toBe("Deploying preview changes");
  });

  it("does not treat a concurrent check wait as addressing feedback", () => {
    const pullRequests: FactoriesFactoryPullRequest[] = [
      {
        workOrderId: "wo-checks",
        activities: [
          {
            access: "concurrent",
            state: "active",
            description: "Waiting for checks on a82fd91",
            run: run({ id: "r1", state: "STATE_STARTED" }),
          },
        ],
      },
      {
        workOrderId: "wo-repair",
        activities: [
          {
            access: "exclusive",
            state: "active",
            description: "Fixing failed checks on a82fd91",
            run: run({ id: "r2", state: "STATE_STARTED" }),
          },
        ],
      },
    ];

    expect(waitingOnChecksWorkOrderIds(pullRequests)).toEqual(new Set(["wo-checks"]));
    expect(addressingFeedbackWorkOrderIds(pullRequests)).toEqual(new Set(["wo-repair"]));
    expect(checksPassedWorkOrderIds(pullRequests)).toEqual(new Set());
    expect(fixesPausedWorkOrderIds(pullRequests)).toEqual(new Set());
    expect(activePRFeedbackWorkOrderIds(pullRequests)).toEqual(new Set(["wo-repair"]));
    expect(addressingFeedbackLabelsByWorkOrder(pullRequests).get("wo-repair")).toBe("Fixing failed checks on a82fd91");
  });

  it("treats a finished passed check wait as checks passed", () => {
    expect(
      checksPassedWorkOrderIds([
        {
          workOrderId: "wo-passed",
          activities: [
            {
              access: "concurrent",
              state: "finished",
              description: "Checks passed on a82fd91",
              run: run({ id: "r1", state: "STATE_FINISHED", result: "RESULT_PASSED" }),
            },
          ],
        },
        {
          workOrderId: "wo-waiting",
          activities: [
            {
              access: "concurrent",
              state: "finished",
              description: "Checks passed on a82fd91",
              run: run({
                id: "r-old",
                state: "STATE_FINISHED",
                result: "RESULT_PASSED",
                createdAt: "2026-08-31T11:00:00Z",
              }),
            },
            {
              access: "concurrent",
              state: "active",
              description: "Waiting for checks on b91ce02",
              run: run({ id: "r-new", state: "STATE_STARTED", createdAt: "2026-08-31T12:00:00Z" }),
            },
          ],
        },
      ]),
    ).toEqual(new Set(["wo-passed"]));
  });

  it("treats a limit-reached check activity as fixes paused", () => {
    const pullRequests: FactoriesFactoryPullRequest[] = [
      {
        workOrderId: "wo-paused",
        activities: [
          {
            access: "released",
            state: "limit_reached",
            description: "Automatic fixes paused after 3 attempts",
            revision: { sha: "a82fd91" },
            run: run({ id: "r-paused", state: "STATE_FINISHED", result: "RESULT_FAILED" }),
          },
        ],
      },
      {
        workOrderId: "wo-waiting",
        activities: [
          {
            access: "released",
            state: "limit_reached",
            description: "Automatic fixes paused after 3 attempts",
            revision: { sha: "a82fd91" },
            run: run({
              id: "r-old",
              state: "STATE_FINISHED",
              result: "RESULT_FAILED",
              createdAt: "2026-08-31T11:00:00Z",
            }),
          },
          {
            access: "concurrent",
            state: "active",
            description: "Waiting for checks on b91ce02",
            run: run({ id: "r-new", state: "STATE_STARTED", createdAt: "2026-08-31T12:00:00Z" }),
          },
        ],
      },
    ];

    expect(fixesPausedWorkOrderIds(pullRequests)).toEqual(new Set(["wo-paused"]));
    expect(checksPassedWorkOrderIds(pullRequests)).toEqual(new Set());
    expect(waitingOnChecksWorkOrderIds(pullRequests)).toEqual(new Set(["wo-waiting"]));
  });

  it("keeps the generic addressing label for discussion runs", () => {
    expect(
      addressingFeedbackLabelsByWorkOrder([
        {
          workOrderId: "wo-discussion",
          activities: [
            {
              access: "exclusive",
              state: "active",
              description: "Please fix the flaky test in checkout.",
              run: run({ id: "r1", state: "STATE_STARTED" }),
            },
          ],
        },
      ]).get("wo-discussion"),
    ).toBe("Addressing user feedback");
  });

  it("uses active activities when they are present", () => {
    expect(
      activePRFeedbackWorkOrderIds([
        {
          workOrderId: "wo-1",
          activities: [{ state: "active", run: { id: "r1", state: "STATE_STARTED" } }],
          runs: [{ run: { id: "r1", state: "STATE_FINISHED", result: "RESULT_PASSED" } }],
        },
        {
          workOrderId: "wo-2",
          activities: [{ state: "finished", run: { id: "r2", state: "STATE_FINISHED", result: "RESULT_PASSED" } }],
        },
        {
          workOrderId: "wo-3",
          activities: [{ state: "finished", run: { id: "r3", state: "STATE_FINISHED", result: "RESULT_CANCELLED" } }],
        },
      ]),
    ).toEqual(new Set(["wo-1"]));
  });
});
