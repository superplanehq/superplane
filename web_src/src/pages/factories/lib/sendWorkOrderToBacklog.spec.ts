import { describe, expect, it } from "bun:test";

import {
  closePullRequestsLabel,
  workOrderHasClearableArtifacts,
  workOrderHasCloseablePullRequests,
} from "./sendWorkOrderToBacklog";

describe("workOrderHasCloseablePullRequests", () => {
  it("is true for open or draft pull requests", () => {
    expect(workOrderHasCloseablePullRequests([{ id: "pr-1", state: "STATE_OPEN" }])).toBe(true);
    expect(workOrderHasCloseablePullRequests([{ id: "pr-2", state: "STATE_DRAFT" }])).toBe(true);
  });

  it("is false for merged, closed, or missing pull requests", () => {
    expect(workOrderHasCloseablePullRequests([{ id: "pr-3", state: "STATE_MERGED" }])).toBe(false);
    expect(workOrderHasCloseablePullRequests([{ id: "pr-4", state: "STATE_CLOSED" }])).toBe(false);
    expect(workOrderHasCloseablePullRequests([])).toBe(false);
    expect(workOrderHasCloseablePullRequests(undefined)).toBe(false);
  });
});

describe("closePullRequestsLabel", () => {
  it("labels Bitbucket pull requests as declines", () => {
    expect(closePullRequestsLabel([{ id: "pr-1", state: "STATE_OPEN", provider: "PROVIDER_BITBUCKET" }])).toBe(
      "Decline pull request",
    );
  });

  it("keeps the close label for GitHub, mixed, or missing pull requests", () => {
    expect(closePullRequestsLabel([{ id: "pr-1", state: "STATE_OPEN", provider: "PROVIDER_GITHUB" }])).toBe(
      "Close previous PRs",
    );
    expect(
      closePullRequestsLabel([
        { id: "pr-1", state: "STATE_OPEN", provider: "PROVIDER_BITBUCKET" },
        { id: "pr-2", state: "STATE_OPEN", provider: "PROVIDER_GITHUB" },
      ]),
    ).toBe("Close previous PRs");
    expect(closePullRequestsLabel([{ id: "pr-1", state: "STATE_OPEN" }])).toBe("Close previous PRs");
    expect(closePullRequestsLabel([])).toBe("Close previous PRs");
    expect(closePullRequestsLabel(undefined)).toBe("Close previous PRs");
  });
});

describe("workOrderHasClearableArtifacts", () => {
  it("is true for markdown, branch, link, and file rows", () => {
    expect(workOrderHasClearableArtifacts([{ type: "TYPE_MARKDOWN" }])).toBe(true);
    expect(workOrderHasClearableArtifacts([{ type: "TYPE_BRANCH" }])).toBe(true);
    expect(workOrderHasClearableArtifacts([{ type: "TYPE_LINK" }])).toBe(true);
    expect(workOrderHasClearableArtifacts([{ type: "TYPE_FILE" }])).toBe(true);
  });

  it("is false for unspecified types and empty lists", () => {
    expect(workOrderHasClearableArtifacts([{ type: "TYPE_UNSPECIFIED" }])).toBe(false);
    expect(workOrderHasClearableArtifacts([])).toBe(false);
    expect(workOrderHasClearableArtifacts(undefined)).toBe(false);
  });
});
