import { describe, expect, it } from "vitest";
import type { ExecutionInfo } from "../../../pages/app/mappers/types";
import { isExecutionInFlight } from "./types";

function execution(state: ExecutionInfo["state"]): ExecutionInfo {
  return {
    id: "execution-1",
    createdAt: "2026-07-22T15:46:48.000Z",
    updatedAt: "2026-07-22T15:54:58.000Z",
    state,
    result: "RESULT_UNKNOWN",
    resultReason: "RESULT_REASON_ERROR",
    resultMessage: "",
    metadata: {},
    configuration: {},
    rootEvent: undefined,
  } as ExecutionInfo;
}

describe("runner live log execution state", () => {
  it("treats pending, started, and cancelling executions as log-active", () => {
    expect(isExecutionInFlight(execution("STATE_PENDING"))).toBe(true);
    expect(isExecutionInFlight(execution("STATE_STARTED"))).toBe(true);
    expect(isExecutionInFlight(execution("STATE_CANCELLING"))).toBe(true);
  });

  it("treats finished executions as no longer in flight", () => {
    expect(isExecutionInFlight(execution("STATE_FINISHED"))).toBe(false);
  });
});
