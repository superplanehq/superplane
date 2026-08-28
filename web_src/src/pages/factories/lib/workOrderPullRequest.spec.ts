import { describe, expect, it } from "vitest";

import type { FactoriesFactoryPullRequest } from "@/api-client";

import { prFeedbackRunTitle } from "./workOrderPullRequest";

function pullRequest(overrides: FactoriesFactoryPullRequest): FactoriesFactoryPullRequest {
  return overrides;
}

describe("prFeedbackRunTitle", () => {
  it("names the run after the pull request number", () => {
    expect(prFeedbackRunTitle(pullRequest({ number: "6812" }))).toBe("Activity on PR #6812");
  });

  it("strips a leading # from the pull request number", () => {
    expect(prFeedbackRunTitle(pullRequest({ number: "#6812" }))).toBe("Activity on PR #6812");
  });

  it("falls back to a neutral title when there is no pull request number", () => {
    expect(prFeedbackRunTitle(pullRequest({}))).toBe("Activity on pull request");
  });
});
