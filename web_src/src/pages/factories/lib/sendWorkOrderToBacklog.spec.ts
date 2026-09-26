import { describe, expect, it } from "bun:test";

import {
  closedWorkOrderResultForDialogStatus,
  workOrderHasClearableArtifacts,
  workOrderHasCloseablePullRequests,
} from "./sendWorkOrderToBacklog";

describe("closedWorkOrderResultForDialogStatus", () => {
  it("maps Failed and Rejected onto the stored close results", () => {
    expect(closedWorkOrderResultForDialogStatus("failed")).toBe("RESULT_FAILED");
    expect(closedWorkOrderResultForDialogStatus("rejected")).toBe("RESULT_REJECTED");
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
