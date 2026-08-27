import { describe, expect, it } from "vitest";

import type { FactoriesFactoryPrFeedbackHandlerRun } from "@/api-client";

import {
  activePRFeedbackWorkOrderIds,
  isActivePRFeedbackRunStatus,
  matchPRFeedbackRunsForWorkOrder,
  oldestActivePRFeedbackRun,
  prFeedbackDraftFromHandler,
  prFeedbackDraftIsValid,
  statusForPRFeedbackRun,
} from "./prFeedbackSettingsModel";
import { matchOldestActivePRFeedbackRun } from "./useWorkOrderPRFeedbackRunHref";

function run(overrides: FactoriesFactoryPrFeedbackHandlerRun): FactoriesFactoryPrFeedbackHandlerRun {
  return overrides;
}

describe("matchPRFeedbackRunsForWorkOrder", () => {
  it("returns matching runs oldest first and skips other work orders", () => {
    const matches = matchPRFeedbackRunsForWorkOrder(
      [
        { id: "h-1", canvasId: "c-1" },
        { id: "h-2", canvasId: "c-2" },
      ],
      [
        [
          run({
            id: "later",
            workOrderId: "wo-1",
            status: "STATUS_RUNNING",
            createdAt: "2026-08-26T12:00:00Z",
          }),
          run({
            id: "other",
            workOrderId: "wo-2",
            status: "STATUS_PASSED",
            createdAt: "2026-08-26T08:00:00Z",
          }),
        ],
        [
          run({
            id: "older",
            workOrderId: "wo-1",
            status: "STATUS_PASSED",
            createdAt: "2026-08-26T11:00:00Z",
          }),
        ],
      ],
      "wo-1",
    );

    expect(matches.map((match) => match.run.id)).toEqual(["older", "later"]);
    expect(matches[0]?.handler.id).toBe("h-2");
  });

  it("skips runs without an id", () => {
    expect(
      matchPRFeedbackRunsForWorkOrder(
        [{ id: "h-1", canvasId: "c-1" }],
        [[run({ workOrderId: "wo-1", status: "STATUS_PASSED", createdAt: "2026-08-26T11:00:00Z" })]],
        "wo-1",
      ),
    ).toEqual([]);
  });
});

describe("statusForPRFeedbackRun", () => {
  it("maps handler run status onto log phase status", () => {
    expect(statusForPRFeedbackRun("STATUS_QUEUED")).toBe("pending");
    expect(statusForPRFeedbackRun("STATUS_RUNNING")).toBe("running");
    expect(statusForPRFeedbackRun("STATUS_PASSED")).toBe("passed");
    expect(statusForPRFeedbackRun("STATUS_FAILED")).toBe("failed");
    expect(statusForPRFeedbackRun("STATUS_CANCELLED")).toBe("failed");
    expect(statusForPRFeedbackRun("STATUS_UNSPECIFIED")).toBe("pending");
  });
});

describe("oldestActivePRFeedbackRun", () => {
  it("returns the oldest queued or running run for the work order", () => {
    const selected = oldestActivePRFeedbackRun(
      [
        run({
          id: "later",
          workOrderId: "wo-1",
          status: "STATUS_RUNNING",
          createdAt: "2026-08-26T12:00:00Z",
        }),
        run({
          id: "older",
          workOrderId: "wo-1",
          status: "STATUS_QUEUED",
          createdAt: "2026-08-26T11:00:00Z",
        }),
        run({
          id: "other",
          workOrderId: "wo-2",
          status: "STATUS_QUEUED",
          createdAt: "2026-08-26T10:00:00Z",
        }),
        run({
          id: "done",
          workOrderId: "wo-1",
          status: "STATUS_PASSED",
          createdAt: "2026-08-26T09:00:00Z",
        }),
      ],
      "wo-1",
    );

    expect(selected?.id).toBe("older");
  });

  it("returns nothing when no matching run is active", () => {
    expect(
      oldestActivePRFeedbackRun(
        [run({ id: "done", workOrderId: "wo-1", status: "STATUS_PASSED", createdAt: "2026-08-26T09:00:00Z" })],
        "wo-1",
      ),
    ).toBeUndefined();
  });
});

describe("isActivePRFeedbackRunStatus", () => {
  it("treats queued and running as active", () => {
    expect(isActivePRFeedbackRunStatus("STATUS_QUEUED")).toBe(true);
    expect(isActivePRFeedbackRunStatus("STATUS_RUNNING")).toBe(true);
    expect(isActivePRFeedbackRunStatus("STATUS_PASSED")).toBe(false);
    expect(isActivePRFeedbackRunStatus("STATUS_FAILED")).toBe(false);
  });
});

describe("activePRFeedbackWorkOrderIds", () => {
  it("collects work orders with a queued or running PR-feedback run", () => {
    expect(
      activePRFeedbackWorkOrderIds([
        [
          run({ id: "a", workOrderId: "wo-1", status: "STATUS_RUNNING" }),
          run({ id: "b", workOrderId: "wo-2", status: "STATUS_PASSED" }),
        ],
        [run({ id: "c", workOrderId: "wo-3", status: "STATUS_QUEUED" })],
      ]),
    ).toEqual(new Set(["wo-1", "wo-3"]));
  });
});

describe("prFeedbackDraftIsValid", () => {
  it("requires a name, repository, and mention that starts with @", () => {
    const draft = prFeedbackDraftFromHandler({
      name: "Address PR feedback",
      settings: { repository: "acme/app", mention: "@superplaneagent", ignoreBots: true },
    });
    expect(prFeedbackDraftIsValid(draft)).toBe(true);
    expect(prFeedbackDraftIsValid({ ...draft, mention: "superplaneagent" })).toBe(false);
    expect(prFeedbackDraftIsValid({ ...draft, repository: "  " })).toBe(false);
  });
});

describe("matchOldestActivePRFeedbackRun", () => {
  it("picks the oldest active run across handlers", () => {
    const match = matchOldestActivePRFeedbackRun(
      [
        { id: "h-1", canvasId: "c-1" },
        { id: "h-2", canvasId: "c-2" },
      ],
      [
        [run({ id: "new", workOrderId: "wo-1", status: "STATUS_RUNNING", createdAt: "2026-08-26T12:00:00Z" })],
        [run({ id: "old", workOrderId: "wo-1", status: "STATUS_QUEUED", createdAt: "2026-08-26T11:00:00Z" })],
      ],
      "wo-1",
    );

    expect(match?.run.id).toBe("old");
    expect(match?.handler.id).toBe("h-2");
  });
});
