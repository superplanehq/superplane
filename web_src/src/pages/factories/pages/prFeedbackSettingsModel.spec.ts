import { describe, expect, it } from "vitest";

import type { CanvasesCanvasRunRef, FactoriesFactoryPullRequest } from "@/api-client";

import type { FactoriesFactoryPrFeedbackHandler } from "@/api-client";

import {
  activePRFeedbackWorkOrderIds,
  addressingFeedbackLabelsByWorkOrder,
  addressingFeedbackWorkOrderIds,
  apiPRFeedbackSource,
  checksPassedWorkOrderIds,
  fixesPausedWorkOrderIds,
  waitingOnChecksWorkOrderIds,
  appendUniqueTrimmedString,
  hasAvailablePRFeedbackSource,
  PR_FEEDBACK_SOURCES,
  takenPRFeedbackSourceIds,
  normalizePRFeedbackDraft,
  oldestActivePRFeedbackRun,
  prFeedbackActivityAttemptLabel,
  prFeedbackActivityLabel,
  prFeedbackDraftFromHandler,
  prFeedbackDraftIsValid,
  prFeedbackListenTitle,
  prFeedbackSettingsToApi,
  prFeedbackSourceId,
  type PRFeedbackDraftSettings,
} from "./prFeedbackSettingsModel";
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
    baseBranch: "main",
    runnerIntegrationIds: [],
    ...overrides,
  };
}

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

describe("PR_FEEDBACK_SOURCES", () => {
  it("offers discussion, checks, and conflicts", () => {
    expect(PR_FEEDBACK_SOURCES.map((source) => source.id)).toEqual(["discussion", "checks", "conflicts"]);
  });

  it("round-trips the conflicts source through the API enum", () => {
    expect(prFeedbackSourceId("SOURCE_PULL_REQUEST_CONFLICTS")).toBe("conflicts");
    expect(apiPRFeedbackSource("conflicts")).toBe("SOURCE_PULL_REQUEST_CONFLICTS");
  });
});

describe("takenPRFeedbackSourceIds", () => {
  it("collects sources that already have a handler", () => {
    expect(
      takenPRFeedbackSourceIds([
        { source: "SOURCE_PULL_REQUEST_DISCUSSION" },
        { source: "SOURCE_PULL_REQUEST_CHECKS" },
        { source: "SOURCE_PULL_REQUEST_CONFLICTS" },
      ]),
    ).toEqual(["discussion", "checks", "conflicts"]);
    expect(hasAvailablePRFeedbackSource(["discussion"])).toBe(true);
    expect(hasAvailablePRFeedbackSource(["discussion", "checks"])).toBe(true);
    expect(hasAvailablePRFeedbackSource(["discussion", "checks", "conflicts"])).toBe(false);
  });
});

describe("prFeedbackListenTitle", () => {
  it("names the listener from the source", () => {
    expect(prFeedbackListenTitle("SOURCE_PULL_REQUEST_DISCUSSION")).toBe("Listening to pull request comments");
    expect(prFeedbackListenTitle("SOURCE_PULL_REQUEST_CHECKS")).toBe("Monitoring pull request checks");
    expect(prFeedbackListenTitle("SOURCE_PULL_REQUEST_CONFLICTS")).toBe("Monitoring pull request conflicts");
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

  it("requires a base branch and a valid attempt limit for conflicts", () => {
    expect(
      prFeedbackDraftIsValid(
        discussionDraft({
          source: "conflicts",
          name: "Resolve pull request conflicts",
          mention: "",
          maximumAttempts: 3,
          baseBranch: "main",
        }),
      ),
    ).toBe(true);
    expect(
      prFeedbackDraftIsValid(
        discussionDraft({
          source: "conflicts",
          name: "Resolve pull request conflicts",
          mention: "",
          maximumAttempts: 3,
          baseBranch: "",
        }),
      ),
    ).toBe(false);
    expect(
      prFeedbackDraftIsValid(
        discussionDraft({
          source: "conflicts",
          name: "Resolve pull request conflicts",
          mention: "",
          maximumAttempts: 0,
          baseBranch: "main",
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

  it("reads conflict settings", () => {
    const handler: FactoriesFactoryPrFeedbackHandler = {
      name: "Resolve pull request conflicts",
      source: "SOURCE_PULL_REQUEST_CONFLICTS",
      settings: {
        subject: { repository: "acme/app" },
        conflicts: { baseBranch: "develop", maximumAttempts: 5 },
      },
    };

    expect(prFeedbackDraftFromHandler(handler)).toMatchObject({
      source: "conflicts",
      baseBranch: "develop",
      maximumAttempts: 5,
    });
  });

  it("defaults the base branch to main", () => {
    const handler: FactoriesFactoryPrFeedbackHandler = {
      name: "Resolve pull request conflicts",
      source: "SOURCE_PULL_REQUEST_CONFLICTS",
      settings: { subject: { repository: "acme/app" } },
    };

    expect(prFeedbackDraftFromHandler(handler).baseBranch).toBe("main");
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
    expect(
      prFeedbackSettingsToApi(
        discussionDraft({
          source: "conflicts",
          baseBranch: "develop",
          maximumAttempts: 5,
        }),
      ),
    ).toEqual({
      subject: { repository: "acme/app" },
      conflicts: { maximumAttempts: 5, baseBranch: "develop" },
    });
  });
});
