import { describe, expect, it } from "bun:test";
import type { ExecutionInfo } from "../types";
import { eventStateRegistry } from "./index";

describe("review check execution state", () => {
  it.each([
    ["STATE_FINISHED", "RESULT_PASSED", "applied", "applied"],
    ["STATE_FINISHED", "RESULT_PASSED", "superseded", "superseded"],
    ["STATE_FINISHED", "RESULT_PASSED", "skipped", "skipped"],
    ["STATE_STARTED", "RESULT_PASSED", undefined, "running"],
    ["STATE_FINISHED", "RESULT_FAILED", undefined, "failed"],
    ["STATE_FINISHED", "RESULT_CANCELLED", undefined, "cancelled"],
  ] as const)("maps %s / %s / %s to %s", (state, result, status, expected) => {
    const execution: ExecutionInfo = {
      id: "review-check",
      createdAt: new Date().toISOString(),
      updatedAt: new Date().toISOString(),
      state,
      result,
      resultReason: "RESULT_REASON_OK",
      resultMessage: "",
      metadata: {},
      configuration: {},
      rootEvent: undefined,
      outputs: status ? { default: [{ data: { status } }] } : {},
    };

    expect(eventStateRegistry.setPullRequestReviewCheck.getState(execution)).toBe(expected);
  });
});
