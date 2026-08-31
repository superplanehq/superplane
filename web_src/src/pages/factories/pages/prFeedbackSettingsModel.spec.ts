import { describe, expect, it } from "vitest";

import type { CanvasesCanvasRunRef, FactoriesFactoryPullRequest } from "@/api-client";

import type { FactoriesFactoryPrFeedbackHandler } from "@/api-client";

import {
  activePRFeedbackWorkOrderIds,
  normalizePRFeedbackDraft,
  oldestActivePRFeedbackRun,
  prFeedbackDraftFromHandler,
  prFeedbackDraftIsValid,
} from "./prFeedbackSettingsModel";
import { prFeedbackLogRunsFromPullRequests } from "./useWorkOrderPRFeedbackRunHref";
import { isActiveCanvasRun, statusForCanvasRun } from "../lib/workOrderPullRequest";

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

describe("oldestActivePRFeedbackRun", () => {
  it("returns the oldest pending or started run", () => {
    const selected = oldestActivePRFeedbackRun([
      run({
        id: "later",
        state: "STATE_STARTED",
        createdAt: "2026-08-26T12:00:00Z",
      }),
      run({
        id: "older",
        state: "STATE_PENDING",
        createdAt: "2026-08-26T11:00:00Z",
      }),
    ]);
    expect(selected?.id).toBe("older");
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
});

describe("prFeedbackDraftIsValid", () => {
  it("requires a name, repository, and mention", () => {
    expect(
      prFeedbackDraftIsValid({
        name: "Address PR feedback",
        repository: "acme/app",
        mention: "@bot",
        ignoreBots: true,
        allowedBots: [],
      }),
    ).toBe(true);
    expect(
      prFeedbackDraftIsValid({
        name: "",
        repository: "acme/app",
        mention: "@bot",
        ignoreBots: true,
        allowedBots: [],
      }),
    ).toBe(false);
  });

  it("does not require an allowed bots list", () => {
    expect(
      prFeedbackDraftIsValid({
        name: "Address PR feedback",
        repository: "acme/app",
        mention: "@bot",
        ignoreBots: true,
        allowedBots: ["coderabbitai"],
      }),
    ).toBe(true);
  });
});

describe("prFeedbackDraftFromHandler", () => {
  it("reads the allowed bots list from the handler settings", () => {
    const handler: FactoriesFactoryPrFeedbackHandler = {
      name: "Address PR feedback",
      settings: {
        subject: { repository: "acme/app" },
        discussion: { mention: "@superplaneagent", ignoreBots: true, allowedBots: ["coderabbitai", "bugbot"] },
      },
    };

    expect(prFeedbackDraftFromHandler(handler).allowedBots).toEqual(["coderabbitai", "bugbot"]);
  });

  it("defaults to an empty allowed bots list", () => {
    const handler: FactoriesFactoryPrFeedbackHandler = {
      name: "Address PR feedback",
      settings: { subject: { repository: "acme/app" }, discussion: { mention: "@superplaneagent" } },
    };

    expect(prFeedbackDraftFromHandler(handler).allowedBots).toEqual([]);
  });
});

describe("normalizePRFeedbackDraft", () => {
  it("trims entries, strips a leading @, drops blanks, and de-duplicates", () => {
    const normalized = normalizePRFeedbackDraft({
      name: "Address PR feedback",
      repository: "acme/app",
      mention: "@bot",
      ignoreBots: true,
      allowedBots: [" @CodeRabbitAI ", "coderabbitai", "bugbot", "", "   "],
    });

    expect(normalized.allowedBots).toEqual(["CodeRabbitAI", "bugbot"]);
  });
});
