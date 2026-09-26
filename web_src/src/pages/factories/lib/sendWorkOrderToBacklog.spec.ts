import { describe, expect, it } from "bun:test";

import {
  CLOSED_STATUS_DIALOG_RESULTS,
  closedStatusEmptyLabel,
  closedStatusTaskMatchesSearch,
  shouldLoadClosedStatusSearchPage,
  workOrderHasClearableArtifacts,
  workOrderHasCloseablePullRequests,
} from "./sendWorkOrderToBacklog";

describe("CLOSED_STATUS_DIALOG_RESULTS", () => {
  it("loads Failed and Rejected together", () => {
    expect(CLOSED_STATUS_DIALOG_RESULTS).toEqual(["RESULT_FAILED", "RESULT_REJECTED"]);
  });
});

describe("closedStatusEmptyLabel", () => {
  it("explains an empty search when more pages remain", () => {
    expect(closedStatusEmptyLabel(false, false)).toBe("No closed tasks.");
    expect(closedStatusEmptyLabel(true, false)).toBe("No tasks match this search.");
    expect(closedStatusEmptyLabel(true, true)).toBe("No matching tasks on this page.");
  });
});

describe("shouldLoadClosedStatusSearchPage", () => {
  const ready = {
    open: true,
    searchActive: true,
    matchCount: 0,
    isLoading: false,
    isFetchingNextPage: false,
    hasNextPage: true,
    isFetchNextPageError: false,
  };

  it("loads the next page only when search has no match and more pages remain", () => {
    expect(shouldLoadClosedStatusSearchPage(ready)).toBe(true);
    expect(shouldLoadClosedStatusSearchPage({ ...ready, open: false })).toBe(false);
    expect(shouldLoadClosedStatusSearchPage({ ...ready, searchActive: false })).toBe(false);
    expect(shouldLoadClosedStatusSearchPage({ ...ready, matchCount: 1 })).toBe(false);
    expect(shouldLoadClosedStatusSearchPage({ ...ready, hasNextPage: false })).toBe(false);
  });

  it("does not request the same page again after a failed load", () => {
    expect(shouldLoadClosedStatusSearchPage({ ...ready, isLoading: true })).toBe(false);
    expect(shouldLoadClosedStatusSearchPage({ ...ready, isFetchingNextPage: true })).toBe(false);
    expect(shouldLoadClosedStatusSearchPage({ ...ready, isFetchNextPageError: true })).toBe(false);
  });
});

describe("closedStatusTaskMatchesSearch", () => {
  const failed = {
    id: "wo-failed",
    number: "106",
    title: "Fix refund dispatcher timeout loop",
    key: "RF-106",
    result: "RESULT_FAILED" as const,
    state: "STATE_CLOSED" as const,
  };

  it("keeps every task when the query is empty", () => {
    expect(closedStatusTaskMatchesSearch(failed, "RF", "")).toBe(true);
    expect(closedStatusTaskMatchesSearch(failed, "RF", "   ")).toBe(true);
  });

  it("matches title, key, and status label", () => {
    expect(closedStatusTaskMatchesSearch(failed, "RF", "refund")).toBe(true);
    expect(closedStatusTaskMatchesSearch(failed, "RF", "rf-106")).toBe(true);
    expect(closedStatusTaskMatchesSearch(failed, "RF", "failed")).toBe(true);
    expect(closedStatusTaskMatchesSearch(failed, "RF", "rejected")).toBe(false);
  });
});

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
