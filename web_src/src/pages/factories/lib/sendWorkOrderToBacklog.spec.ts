import { describe, expect, it } from "bun:test";

import {
  CLOSED_STATUS_DIALOG_RESULTS,
  closedStatusTaskMatchesSearch,
  workOrderHasClearableArtifacts,
  workOrderHasCloseablePullRequests,
} from "./sendWorkOrderToBacklog";

describe("CLOSED_STATUS_DIALOG_RESULTS", () => {
  it("loads Failed and Rejected together", () => {
    expect(CLOSED_STATUS_DIALOG_RESULTS).toEqual(["RESULT_FAILED", "RESULT_REJECTED"]);
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
