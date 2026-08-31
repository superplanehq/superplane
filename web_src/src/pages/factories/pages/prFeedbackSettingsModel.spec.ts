import { describe, expect, it } from "vitest";

import type { CanvasesCanvasRunRef, FactoriesFactoryPullRequest } from "@/api-client";

import type { FactoriesFactoryPrFeedbackHandler } from "@/api-client";

import {
  activePRFeedbackWorkOrderIds,
  addressingFeedbackLabelsByWorkOrder,
  addressingFeedbackWorkOrderIds,
  checksPassedWorkOrderIds,
  waitingOnChecksWorkOrderIds,
  appendUniqueTrimmedString,
  hasAvailablePRFeedbackSource,
  takenPRFeedbackSourceIds,
  normalizePRFeedbackDraft,
  oldestActivePRFeedbackRun,
  prFeedbackActivityAttemptLabel,
  prFeedbackActivityLabel,
  prFeedbackDraftFromHandler,
  prFeedbackDraftIsValid,
  prFeedbackListenTitle,
  prFeedbackSettingsToApi,
  type PRFeedbackDraftSettings,
} from "./prFeedbackSettingsModel";
import { prFeedbackLogRunsFromPullRequests } from "./useWorkOrderPRFeedbackRunHref";
import { isActiveCanvasRun, statusForCanvasRun } from "../lib/workOrderPullRequest";

function run(overrides: CanvasesCanvasRunRef): CanvasesCanvasRunRef {
  return overrides;
}

function discussionDraft(overrides: Partial<PRFeedbackDraftSettings> = {}): PRFeedbackDraftSettings {
  return {
    source: "discussion",
    name: "Address PR feedback",
    repository: "acme/app",
    mention: "@bot",
    ignoreBots: true,
    allowedBots: [],
    checkNames: [],
    maximumAttempts: 3,
    runnerIntegrationIds: [],
    ...overrides,
  };
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

describe("prFeedbackActivityLabel", () => {
  it("shows waiting and limit labels", () => {
    expect(
      prFeedbackActivityLabel({
        access: "waiting",
        state: "active",
        description: "Fixing failed checks on a82fd91",
      }),
    ).toBe("Waiting for another pull request activity");
    expect(prFeedbackActivityLabel({ state: "finished", description: "Waiting for checks on d1209da" })).toBe(
      "Waiting for checks on d1209da",
    );
    expect(prFeedbackActivityLabel({ state: "limit_reached", attemptLimit: 3 })).toBe(
      "Automatic fixes paused after 3 attempts",
    );
    expect(
      prFeedbackActivityLabel({
        access: "exclusive",
        state: "active",
        description: "Fixing failed checks on a82fd91",
      }),
    ).toBe("Fixing failed checks on a82fd91");
  });
});

describe("prFeedbackActivityAttemptLabel", () => {
  it("shows the attempt count when an attempt exists", () => {
    expect(prFeedbackActivityAttemptLabel({ attempt: 2, attemptLimit: 3 })).toBe("Attempt 2 of 3");
    expect(prFeedbackActivityAttemptLabel({ attempt: 0, attemptLimit: 3 })).toBeUndefined();
    expect(prFeedbackActivityAttemptLabel({})).toBeUndefined();
  });
});

describe("takenPRFeedbackSourceIds", () => {
  it("collects sources that already have a handler", () => {
    expect(
      takenPRFeedbackSourceIds([
        { source: "SOURCE_PULL_REQUEST_DISCUSSION" },
        { source: "SOURCE_PULL_REQUEST_CHECKS" },
      ]),
    ).toEqual(["discussion", "checks"]);
    expect(hasAvailablePRFeedbackSource(["discussion"])).toBe(true);
    expect(hasAvailablePRFeedbackSource(["discussion", "checks"])).toBe(false);
  });
});

describe("prFeedbackListenTitle", () => {
  it("names the listener from the source", () => {
    expect(prFeedbackListenTitle("SOURCE_PULL_REQUEST_DISCUSSION")).toBe("Listening to pull request comments");
    expect(prFeedbackListenTitle("SOURCE_PULL_REQUEST_CHECKS")).toBe("Monitoring pull request checks");
  });
});

describe("prFeedbackDraftIsValid", () => {
  it("requires a name, repository, and mention", () => {
    expect(prFeedbackDraftIsValid(discussionDraft())).toBe(true);
    expect(prFeedbackDraftIsValid(discussionDraft({ name: "" }))).toBe(false);
  });

  it("does not require an allowed bots list", () => {
    expect(prFeedbackDraftIsValid(discussionDraft({ allowedBots: ["coderabbitai"] }))).toBe(true);
  });

  it("requires a valid attempt limit for checks", () => {
    expect(
      prFeedbackDraftIsValid(
        discussionDraft({
          source: "checks",
          name: "Fix pull request checks",
          mention: "",
          maximumAttempts: 3,
        }),
      ),
    ).toBe(true);
    expect(
      prFeedbackDraftIsValid(
        discussionDraft({
          source: "checks",
          name: "Fix pull request checks",
          mention: "",
          maximumAttempts: 0,
        }),
      ),
    ).toBe(false);
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
    expect(prFeedbackDraftFromHandler(handler).source).toBe("discussion");
  });

  it("defaults to an empty allowed bots list", () => {
    const handler: FactoriesFactoryPrFeedbackHandler = {
      name: "Address PR feedback",
      settings: { subject: { repository: "acme/app" }, discussion: { mention: "@superplaneagent" } },
    };

    expect(prFeedbackDraftFromHandler(handler).allowedBots).toEqual([]);
  });

  it("reads check settings", () => {
    const handler: FactoriesFactoryPrFeedbackHandler = {
      name: "Fix pull request checks",
      source: "SOURCE_PULL_REQUEST_CHECKS",
      settings: {
        subject: { repository: "acme/app" },
        checks: { names: ["lint", "unit"], maximumAttempts: 4, runnerIntegrationIds: ["int-1"] },
      },
    };

    expect(prFeedbackDraftFromHandler(handler)).toMatchObject({
      source: "checks",
      checkNames: ["lint", "unit"],
      maximumAttempts: 4,
      runnerIntegrationIds: ["int-1"],
    });
  });
});

describe("appendUniqueTrimmedString", () => {
  it("keeps commas inside one name and skips blanks and duplicates", () => {
    expect(appendUniqueTrimmedString(["lint"], " lint, typecheck ")).toEqual(["lint", "lint, typecheck"]);
    expect(appendUniqueTrimmedString(["lint"], "lint")).toEqual(["lint"]);
    expect(appendUniqueTrimmedString(["lint"], "   ")).toEqual(["lint"]);
  });
});

describe("normalizePRFeedbackDraft", () => {
  it("trims entries, strips a leading @, drops blanks, and de-duplicates", () => {
    const normalized = normalizePRFeedbackDraft(
      discussionDraft({
        allowedBots: [" @CodeRabbitAI ", "coderabbitai", "bugbot", "", "   "],
        checkNames: [" lint ", "lint", "unit"],
      }),
    );

    expect(normalized.allowedBots).toEqual(["CodeRabbitAI", "bugbot"]);
    expect(normalized.checkNames).toEqual(["lint", "unit"]);
  });
});

describe("prFeedbackSettingsToApi", () => {
  it("sends discussion or check settings for the selected source", () => {
    expect(prFeedbackSettingsToApi(discussionDraft())).toEqual({
      subject: { repository: "acme/app" },
      discussion: { mention: "@bot", ignoreBots: true, allowedBots: [] },
    });
    expect(
      prFeedbackSettingsToApi(
        discussionDraft({
          source: "checks",
          checkNames: ["lint"],
          maximumAttempts: 4,
          runnerIntegrationIds: ["int-1"],
        }),
      ),
    ).toEqual({
      subject: { repository: "acme/app" },
      checks: { names: ["lint"], maximumAttempts: 4, runnerIntegrationIds: ["int-1"] },
    });
  });
});
