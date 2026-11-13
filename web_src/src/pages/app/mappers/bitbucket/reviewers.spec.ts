import { describe, expect, it } from "bun:test";
import type { ExecutionInfo } from "../types";
import { eventStateRegistry } from "./index";

describe("Bitbucket reviewer execution state", () => {
  it.each([
    ["STATE_FINISHED", "RESULT_PASSED", "requested", "requested"],
    ["STATE_FINISHED", "RESULT_PASSED", "already_covered", "already_covered"],
    ["STATE_FINISHED", "RESULT_PASSED", "stale_pr", "stale_pr"],
    ["STATE_STARTED", "RESULT_PASSED", "requested", "running"],
    ["STATE_FINISHED", "RESULT_FAILED", "requested", "failed"],
    ["STATE_FINISHED", "RESULT_FAILED", "requested", "error"],
    ["STATE_FINISHED", "RESULT_CANCELLED", "requested", "cancelled"],
  ] as const)("maps %s / %s / %s to %s", (state, result, status, expected) => {
    const execution: ExecutionInfo = {
      id: "review-request",
      createdAt: new Date().toISOString(),
      updatedAt: new Date().toISOString(),
      state,
      result,
      resultReason: expected === "error" ? "RESULT_REASON_ERROR" : "RESULT_REASON_OK",
      resultMessage: expected === "error" ? "Reviewer update is uncertain" : "",
      metadata: {},
      configuration: {},
      rootEvent: undefined,
      outputs: { default: [{ data: { status } }] },
    };
    expect(eventStateRegistry.ensurePullRequestReviewers.getState(execution)).toBe(expected);
  });
});
