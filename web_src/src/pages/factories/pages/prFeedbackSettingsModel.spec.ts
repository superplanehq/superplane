import { describe, expect, it } from "bun:test";

import type { CanvasesCanvasRunRef } from "@/api-client";

import type { FactoriesFactoryPrFeedbackHandler } from "@/api-client";

import {
  appendUniqueTrimmedString,
  toggleUniqueString,
  availablePRFeedbackSources,
  hasAvailablePRFeedbackSource,
  isPRFeedbackSettingsTab,
  prFeedbackHandlerForSource,
  prFeedbackSettingsTabs,
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

describe("PR feedback settings tabs", () => {
  it("accepts General, Agent, and Automation", () => {
    expect(isPRFeedbackSettingsTab("general")).toBe(true);
    expect(isPRFeedbackSettingsTab("agent")).toBe(true);
    expect(isPRFeedbackSettingsTab("automation")).toBe(true);
    expect(isPRFeedbackSettingsTab("runs")).toBe(false);
    expect(prFeedbackSettingsTabs(false)).toEqual(["general", "automation"]);
    expect(prFeedbackSettingsTabs(true)).toEqual(["general", "agent", "automation"]);
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

describe("prFeedbackActivityLabel", () => {
  it("removes the legacy addressing suffix from persisted titles", () => {
    expect(
      prFeedbackActivityLabel({
        state: "finished",
        title:
          "[@lucaspin](https://github.com/lucaspin) left a [review](https://github.com/acme/app/pull/42#review) - addressing",
      }),
    ).toBe("[@lucaspin](https://github.com/lucaspin) left a [review](https://github.com/acme/app/pull/42#review)");
  });

  it("shows waiting and limit labels", () => {
    expect(
      prFeedbackActivityLabel({
        access: "waiting",
        state: "active",
        description: "Fixing failed checks on a82fd91",
      }),
    ).toBe("Fixing failed checks on a82fd91");
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
    expect(prFeedbackActivityAttemptLabel({ attempt: 2, attemptLimit: 3 })).toBe("· 2/3");
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
  });
});

describe("availablePRFeedbackSources", () => {
  it("offers discussion and status-check setup", () => {
    expect(availablePRFeedbackSources().map((source) => source.id)).toEqual(["discussion", "checks"]);
    expect(hasAvailablePRFeedbackSource([])).toBe(true);
    expect(hasAvailablePRFeedbackSource(["discussion"])).toBe(true);
    expect(hasAvailablePRFeedbackSource(["discussion", "checks"])).toBe(false);
  });
});

describe("prFeedbackHandlerForSource", () => {
  it("returns the handler for a source when it has an id", () => {
    expect(
      prFeedbackHandlerForSource(
        [
          { id: "handler-discussion", source: "SOURCE_PULL_REQUEST_DISCUSSION" },
          { id: "handler-checks", source: "SOURCE_PULL_REQUEST_CHECKS" },
        ],
        "discussion",
      ),
    ).toEqual({ id: "handler-discussion", source: "SOURCE_PULL_REQUEST_DISCUSSION" });
  });
});

describe("prFeedbackListenTitle", () => {
  it("names the listener from the source", () => {
    expect(prFeedbackListenTitle("SOURCE_PULL_REQUEST_DISCUSSION")).toBe("Listening to pull request comments");
    expect(prFeedbackListenTitle("SOURCE_PULL_REQUEST_CHECKS")).toBe("Monitoring pull request checks");
  });
});

describe("prFeedbackDraftIsValid", () => {
  it("requires a name and repository", () => {
    expect(prFeedbackDraftIsValid(discussionDraft())).toBe(true);
    expect(prFeedbackDraftIsValid(discussionDraft({ name: "" }))).toBe(false);
    expect(prFeedbackDraftIsValid(discussionDraft({ mention: "" }))).toBe(true);
    expect(prFeedbackDraftIsValid(discussionDraft({ mention: "superplaneagent" }))).toBe(false);
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
          checkNames: ["lint"],
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
          checkNames: [],
          maximumAttempts: 3,
        }),
      ),
    ).toBe(false);
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
    expect(
      prFeedbackDraftIsValid(
        discussionDraft({
          source: "checks",
          name: "Fix pull request checks",
          mention: "",
          checkNames: ["lint"],
          maximumAttempts: 5.5,
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

  it("keeps an empty mention", () => {
    const handler: FactoriesFactoryPrFeedbackHandler = {
      name: "Address PR feedback",
      settings: { subject: { repository: "acme/app" }, discussion: { mention: "" } },
    };

    expect(prFeedbackDraftFromHandler(handler).mention).toBe("");
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

describe("toggleUniqueString", () => {
  it("adds a missing name and removes a matching name without case", () => {
    expect(toggleUniqueString(["lint"], "e2e")).toEqual(["lint", "e2e"]);
    expect(toggleUniqueString(["lint", "e2e"], "LINT")).toEqual(["e2e"]);
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
