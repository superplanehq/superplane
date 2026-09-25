import { describe, expect, it } from "bun:test";

import type { FactoriesFactoryPrFeedbackHandler, FactoriesFactoryPullRequest } from "@/api-client";

import {
  findPullRequestDiscussionHandler,
  pullRequestFeedbackMention,
  pullRequestMentionNote,
} from "./pullRequestMentionNote";

function handler(overrides: Partial<FactoriesFactoryPrFeedbackHandler> = {}): FactoriesFactoryPrFeedbackHandler {
  return {
    id: "handler-1",
    factoryId: "factory-1",
    source: "SOURCE_PULL_REQUEST_DISCUSSION",
    healthy: true,
    ...overrides,
  };
}

const OPEN_PULL_REQUEST: FactoriesFactoryPullRequest = {
  id: "pr-1",
  workOrderId: "wo-1",
  repository: "acme/payments",
  url: "https://github.com/acme/payments/pull/42",
  number: "42",
  state: "STATE_OPEN",
};

describe("findPullRequestDiscussionHandler", () => {
  it("finds a healthy handler whose repository matches", () => {
    const target = handler({ settings: { subject: { repository: "acme/payments" } } });
    expect(findPullRequestDiscussionHandler("acme/payments", [target])).toBe(target);
  });

  it("finds a healthy handler that covers every repository", () => {
    const target = handler();
    expect(findPullRequestDiscussionHandler("acme/payments", [target])).toBe(target);
  });

  it("skips a handler for a different repository", () => {
    const other = handler({ settings: { subject: { repository: "acme/other" } } });
    expect(findPullRequestDiscussionHandler("acme/payments", [other])).toBeUndefined();
  });

  it("skips an unhealthy handler", () => {
    const unhealthy = handler({ healthy: false });
    expect(findPullRequestDiscussionHandler("acme/payments", [unhealthy])).toBeUndefined();
  });

  it("skips a checks handler", () => {
    const checks = handler({ source: "SOURCE_PULL_REQUEST_CHECKS" });
    expect(findPullRequestDiscussionHandler("acme/payments", [checks])).toBeUndefined();
  });

  it("returns undefined without handlers", () => {
    expect(findPullRequestDiscussionHandler("acme/payments", undefined)).toBeUndefined();
  });
});

describe("pullRequestFeedbackMention", () => {
  it("uses the handler's configured mention", () => {
    expect(pullRequestFeedbackMention(handler({ settings: { discussion: { mention: "@superplaneagent" } } }))).toBe(
      "@superplaneagent",
    );
  });

  it("falls back to the default mention when none is set", () => {
    expect(pullRequestFeedbackMention(handler())).toBe("@superplaneagent");
  });
});

describe("pullRequestMentionNote", () => {
  it("builds a note with the mention and a link to the open pull request", () => {
    const note = pullRequestMentionNote([OPEN_PULL_REQUEST], "wo-1", [
      handler({ settings: { subject: { repository: "acme/payments" }, discussion: { mention: "@superplaneagent" } } }),
    ]);

    expect(note).toEqual({
      key: "pull-request-mention",
      headline: "Ask for changes in the pull request",
      text: "Mention @superplaneagent in a pull request comment or review to request changes.",
      cta: { label: "Open pull request", href: "https://github.com/acme/payments/pull/42" },
    });
  });

  it("is absent without a matching handler", () => {
    expect(pullRequestMentionNote([OPEN_PULL_REQUEST], "wo-1", [])).toBeUndefined();
    expect(pullRequestMentionNote([OPEN_PULL_REQUEST], "wo-1", undefined)).toBeUndefined();
  });

  it("is absent when the only handler is unhealthy", () => {
    const unhealthy = handler({ healthy: false, settings: { subject: { repository: "acme/payments" } } });
    expect(pullRequestMentionNote([OPEN_PULL_REQUEST], "wo-1", [unhealthy])).toBeUndefined();
  });

  it("is absent once the pull request closes", () => {
    const closed = { ...OPEN_PULL_REQUEST, state: "STATE_CLOSED" as const };
    expect(pullRequestMentionNote([closed], "wo-1", [handler()])).toBeUndefined();
  });

  it("is absent without a tracked pull request", () => {
    expect(pullRequestMentionNote([], "wo-1", [handler()])).toBeUndefined();
  });
});
