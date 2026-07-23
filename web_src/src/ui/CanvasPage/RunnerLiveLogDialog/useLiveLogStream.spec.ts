import { describe, expect, it } from "vitest";
import type { ExecutionInfo } from "../../../pages/app/mappers/types";
import type { LogState } from "./types";
import {
  finalizeRunningCommandSections,
  terminalCommandStatusForExecution,
  terminalTimeMsForExecution,
} from "./useLiveLogStream";

function baseLogState(): LogState {
  return {
    sections: [
      {
        index: 0,
        text: "completed",
        lines: [],
        status: "passed",
        duration_ms: 100,
        started_at: 1_000,
        collapsed: true,
      },
      {
        index: 1,
        text: "Set up DevEnv",
        lines: ["docker compose up"],
        status: "running",
        duration_ms: null,
        started_at: 2_000,
        collapsed: false,
      },
    ],
    orphanLines: [],
    error: null,
    isStreaming: false,
  };
}

function execution(overrides: Partial<ExecutionInfo>): ExecutionInfo {
  return {
    id: "execution-1",
    createdAt: "2026-07-22T15:46:48.000Z",
    updatedAt: "2026-07-22T15:54:58.000Z",
    state: "STATE_FINISHED",
    result: "RESULT_FAILED",
    resultReason: "RESULT_REASON_ERROR",
    resultMessage: "",
    metadata: {},
    configuration: {},
    rootEvent: undefined,
    ...overrides,
  } as ExecutionInfo;
}

describe("runner live log state", () => {
  it("finalizes unfinished command sections when a terminal execution failed", () => {
    const finalized = finalizeRunningCommandSections(baseLogState(), "failed", 5_000);

    expect(finalized.sections[0]).toMatchObject({ status: "passed", duration_ms: 100 });
    expect(finalized.sections[1]).toMatchObject({
      status: "failed",
      duration_ms: 3_000,
      collapsed: false,
    });
  });

  it("collapses unfinished command sections when a terminal execution passed", () => {
    const finalized = finalizeRunningCommandSections(baseLogState(), "passed", 5_000);

    expect(finalized.sections[1]).toMatchObject({
      status: "passed",
      duration_ms: 3_000,
      collapsed: true,
    });
  });

  it("maps terminal execution result to command status", () => {
    expect(terminalCommandStatusForExecution(execution({ result: "RESULT_PASSED" }))).toBe("passed");
    expect(terminalCommandStatusForExecution(execution({ result: "RESULT_FAILED" }))).toBe("failed");
    expect(terminalCommandStatusForExecution(execution({ result: "RESULT_CANCELLED" }))).toBe("failed");
    expect(
      terminalCommandStatusForExecution(execution({ state: "STATE_STARTED", result: "RESULT_UNKNOWN" })),
    ).toBeNull();
  });

  it("uses the execution update time as the terminal command timestamp", () => {
    expect(terminalTimeMsForExecution(execution({ updatedAt: "2026-07-22T15:54:58.000Z" }))).toBe(
      Date.parse("2026-07-22T15:54:58.000Z"),
    );
  });

  it("does not expose a terminal command timestamp while execution is in flight", () => {
    expect(
      terminalTimeMsForExecution(
        execution({
          state: "STATE_CANCELLING",
          result: "RESULT_UNKNOWN",
          updatedAt: "2026-07-22T15:54:58.000Z",
        }),
      ),
    ).toBeNull();
  });
});
