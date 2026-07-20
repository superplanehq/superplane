import { describe, expect, it } from "vitest";

import type { CanvasesCanvasRunRef } from "@/api-client";

import { ACTIVE_RUN_API_STATES, getRunRefStatus, statusFiltersToApiFilters } from "./runPresentation";

function runRef(overrides: Partial<CanvasesCanvasRunRef> = {}): CanvasesCanvasRunRef {
  return {
    state: "STATE_FINISHED",
    result: "RESULT_PASSED",
    ...overrides,
  };
}

describe("getRunRefStatus", () => {
  it("returns cancelling for runs in STATE_CANCELLING", () => {
    expect(getRunRefStatus(runRef({ state: "STATE_CANCELLING", result: "RESULT_UNKNOWN" }))).toBe("cancelling");
  });

  it("returns running for pending and started runs", () => {
    expect(getRunRefStatus(runRef({ state: "STATE_PENDING", result: "RESULT_UNKNOWN" }))).toBe("running");
    expect(getRunRefStatus(runRef({ state: "STATE_STARTED", result: "RESULT_UNKNOWN" }))).toBe("running");
  });

  it("returns terminal statuses from result when finished", () => {
    expect(getRunRefStatus(runRef({ state: "STATE_FINISHED", result: "RESULT_FAILED" }))).toBe("failed");
    expect(getRunRefStatus(runRef({ state: "STATE_FINISHED", result: "RESULT_CANCELLED" }))).toBe("cancelled");
    expect(getRunRefStatus(runRef({ state: "STATE_FINISHED", result: "RESULT_PASSED" }))).toBe("passed");
  });
});

describe("statusFiltersToApiFilters", () => {
  it("includes pending runs in the running state filter", () => {
    expect(statusFiltersToApiFilters(["running"])).toEqual({
      states: [...ACTIVE_RUN_API_STATES],
      results: [],
    });
  });
});
