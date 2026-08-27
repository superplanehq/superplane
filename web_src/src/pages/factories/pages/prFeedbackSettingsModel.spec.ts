import { describe, expect, it } from "vitest";

import type { FactoriesFactoryPrFeedbackHandlerRun, FactoriesWorkOrderPrFeedbackRun } from "@/api-client";

import {
  activePRFeedbackWorkOrderIds,
  isActivePRFeedbackRunStatus,
  oldestActivePRFeedbackRun,
  prFeedbackDraftFromHandler,
  prFeedbackDraftIsValid,
  statusForPRFeedbackRun,
} from "./prFeedbackSettingsModel";
import { prFeedbackLogRunsFromItems } from "./useWorkOrderPRFeedbackRunHref";

function run(overrides: FactoriesFactoryPrFeedbackHandlerRun): FactoriesFactoryPrFeedbackHandlerRun {
  return overrides;
}

describe("prFeedbackLogRunsFromItems", () => {
  it("returns matching runs oldest first and skips items without a canvas or run id", () => {
    const items: FactoriesWorkOrderPrFeedbackRun[] = [
      {
        handlerId: "h-1",
        canvasId: "c-1",
        handlerName: "Address PR feedback",
        run: run({
          id: "later",
          status: "STATUS_RUNNING",
          createdAt: "2026-08-26T12:00:00Z",
        }),
      },
      {
        handlerId: "h-2",
        canvasId: "c-2",
        run: run({
          status: "STATUS_PASSED",
          createdAt: "2026-08-26T08:00:00Z",
        }),
      },
      {
        handlerId: "h-2",
        canvasId: "c-2",
        handlerName: "Second handler",
        run: run({
          id: "older",
          status: "STATUS_PASSED",
          createdAt: "2026-08-26T11:00:00Z",
        }),
      },
    ];

    const matches = prFeedbackLogRunsFromItems(items);
    expect(matches.map((match) => match.run.id)).toEqual(["older", "later"]);
    expect(matches[0]?.handlerName).toBe("Second handler");
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
  it("returns the oldest queued or running run", () => {
    const selected = oldestActivePRFeedbackRun([
      run({
        id: "later",
        status: "STATUS_RUNNING",
        createdAt: "2026-08-26T12:00:00Z",
      }),
      run({
        id: "older",
        status: "STATUS_QUEUED",
        createdAt: "2026-08-26T11:00:00Z",
      }),
      run({
        id: "done",
        status: "STATUS_PASSED",
        createdAt: "2026-08-26T09:00:00Z",
      }),
    ]);

    expect(selected?.id).toBe("older");
  });

  it("returns nothing when no run is active", () => {
    expect(
      oldestActivePRFeedbackRun([run({ id: "done", status: "STATUS_PASSED", createdAt: "2026-08-26T09:00:00Z" })]),
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
  it("collects work orders that have a queued or running PR feedback run", () => {
    expect(
      activePRFeedbackWorkOrderIds([
        { id: "wo-1", prFeedbackRuns: [{ run: { status: "STATUS_QUEUED" } }] },
        { id: "wo-2", prFeedbackRuns: [{ run: { status: "STATUS_PASSED" } }] },
        { id: "wo-3", prFeedbackRuns: [{ run: { status: "STATUS_RUNNING" } }] },
        { prFeedbackRuns: [{ run: { status: "STATUS_QUEUED" } }] },
        { id: "wo-4" },
      ]),
    ).toEqual(new Set(["wo-1", "wo-3"]));
  });
});

describe("prFeedbackDraftIsValid", () => {
  it("requires a name, repository, and mention that starts with @", () => {
    const draft = prFeedbackDraftFromHandler({
      name: "Address PR feedback",
      settings: {
        subject: { repository: "acme/app" },
        discussion: { mention: "@superplaneagent", ignoreBots: true },
      },
    });
    expect(prFeedbackDraftIsValid(draft)).toBe(true);
    expect(prFeedbackDraftIsValid({ ...draft, mention: "superplaneagent" })).toBe(false);
    expect(prFeedbackDraftIsValid({ ...draft, repository: "  " })).toBe(false);
  });
});
