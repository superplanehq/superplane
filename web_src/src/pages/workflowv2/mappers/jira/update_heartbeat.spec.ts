import { describe, expect, it } from "vitest";

import type { ExecutionInfo } from "../types";
import { eventStateRegistry } from "./index";

function buildExecution(overrides?: Partial<ExecutionInfo>): ExecutionInfo {
  return {
    id: "exec-1",
    createdAt: new Date().toISOString(),
    updatedAt: new Date().toISOString(),
    state: "STATE_FINISHED",
    result: "RESULT_PASSED",
    resultReason: "RESULT_REASON_OK",
    resultMessage: "",
    metadata: {},
    configuration: {},
    rootEvent: undefined,
    ...overrides,
  };
}

describe("updateHeartbeatMapper registry", () => {
  it("maps success to updated", () => {
    expect(eventStateRegistry.updateHeartbeat.getState(buildExecution({ result: "RESULT_PASSED" }))).toBe("updated");
  });
});
