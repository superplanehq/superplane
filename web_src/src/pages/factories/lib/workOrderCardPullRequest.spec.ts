import { describe, expect, it } from "vitest";

import type { FactoriesFactoryPullRequest } from "@/api-client";

import {
  selectWorkOrderCardPullRequest,
  workOrderCardPullRequestAriaLabel,
  workOrderCardPullRequestVisibleLabel,
} from "./workOrderCardPullRequest";

function pr(overrides: Partial<FactoriesFactoryPullRequest> = {}): FactoriesFactoryPullRequest {
  return {
    id: "pr-1",
    workOrderId: "wo-1",
    number: "2323",
    url: "https://github.com/acme/payments/pull/2323",
    title: "Ship idempotent refund retries",
    state: "STATE_OPEN",
    ...overrides,
  };
}

describe("selectWorkOrderCardPullRequest", () => {
  it("returns null when the task has no pull request", () => {
    expect(selectWorkOrderCardPullRequest([], "wo-1")).toBeNull();
    expect(selectWorkOrderCardPullRequest(undefined, "wo-1")).toBeNull();
  });

  it("keeps pull requests that belong to this task", () => {
    const selected = selectWorkOrderCardPullRequest(
      [pr({ workOrderId: "wo-other", number: "9" }), pr({ id: "pr-mine" })],
      "wo-1",
    );
    expect(selected?.pullRequest.id).toBe("pr-mine");
    expect(selected?.extraCount).toBe(0);
  });

  it("prefers an open pull request over a merged one", () => {
    const selected = selectWorkOrderCardPullRequest(
      [pr({ id: "merged", number: "99", state: "STATE_MERGED" }), pr({ id: "open", number: "12" })],
      "wo-1",
    );
    expect(selected?.pullRequest.id).toBe("open");
    expect(selected?.extraCount).toBe(1);
  });

  it("picks the higher number when both pull requests are open", () => {
    const selected = selectWorkOrderCardPullRequest(
      [pr({ id: "older", number: "10" }), pr({ id: "newer", number: "2323" })],
      "wo-1",
    );
    expect(selected?.pullRequest.id).toBe("newer");
    expect(selected?.extraCount).toBe(1);
  });
});

describe("workOrderCardPullRequestVisibleLabel", () => {
  it("names an open pull request as Review plus the number", () => {
    expect(workOrderCardPullRequestVisibleLabel(pr())).toBe("Review #2323");
  });

  it("names draft, merged, and closed states", () => {
    expect(workOrderCardPullRequestVisibleLabel(pr({ state: "STATE_DRAFT" }))).toBe("Draft #2323");
    expect(workOrderCardPullRequestVisibleLabel(pr({ state: "STATE_MERGED" }))).toBe("Merged #2323");
    expect(workOrderCardPullRequestVisibleLabel(pr({ state: "STATE_CLOSED" }))).toBe("Closed #2323");
  });

  it("appends a count when more pull requests are attached", () => {
    expect(workOrderCardPullRequestVisibleLabel(pr(), 1)).toBe("Review #2323 +1");
  });
});

describe("workOrderCardPullRequestAriaLabel", () => {
  it("says Review pull request and the number", () => {
    expect(workOrderCardPullRequestAriaLabel(pr())).toBe("Review pull request #2323.");
  });

  it("names extra pull requests in the accessible label", () => {
    expect(workOrderCardPullRequestAriaLabel(pr(), 1)).toBe("Review pull request #2323. 1 more pull request.");
    expect(workOrderCardPullRequestAriaLabel(pr(), 2)).toBe("Review pull request #2323. 2 more pull requests.");
  });
});
