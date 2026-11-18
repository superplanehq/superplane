import { describe, expect, it } from "bun:test";
import type { EventState } from "@/ui/componentBase";
import type { ExecutionInfo, OutputPayload } from "../types";
import { runPipelineStateFunction } from "./run_pipeline";

function buildExecution(overrides: Partial<ExecutionInfo>): ExecutionInfo {
  const timestamp = new Date().toISOString();
  return {
    id: "exec-1",
    createdAt: timestamp,
    updatedAt: timestamp,
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

const output: OutputPayload = {
  type: "gitlab.pipeline",
  timestamp: new Date().toISOString(),
  data: {},
};

describe("runPipelineStateFunction", () => {
  const statuses: Array<[string, EventState]> = [
    ["success", "passed"],
    ["failed", "failed"],
    ["canceled", "failed"],
    ["cancelled", "failed"],
    ["skipped", "failed"],
    ["manual", "failed"],
    ["blocked", "failed"],
    ["", "neutral"],
    ["unknown", "neutral"],
    ["toString", "neutral"],
  ];

  it.each(statuses)("maps metadata status %s to %s", (status, expected) => {
    expect(runPipelineStateFunction(buildExecution({ metadata: { pipeline: { status } } }))).toBe(expected);
  });

  it.each([
    ["metadata", undefined],
    ["pipeline", {}],
    ["status", { pipeline: {} }],
  ])("returns neutral with missing %s", (_name, metadata) => {
    expect(runPipelineStateFunction(buildExecution({ metadata }))).toBe("neutral");
  });

  const precedenceCases: Array<{ name: string; overrides: Partial<ExecutionInfo>; expected: EventState }> = [
    {
      name: "execution error overrides cancellation, active state, outputs, and metadata",
      overrides: {
        result: "RESULT_CANCELLED",
        resultReason: "RESULT_REASON_ERROR",
        resultMessage: "Pipeline request failed",
        state: "STATE_STARTED",
        outputs: { failed: [output], passed: [output] },
      },
      expected: "error",
    },
    {
      name: "a failed execution with a message overrides active state and outputs",
      overrides: {
        result: "RESULT_FAILED",
        resultMessage: "Pipeline request failed",
        state: "STATE_PENDING",
        outputs: { passed: [output] },
      },
      expected: "error",
    },
    {
      name: "cancellation overrides active state and outputs",
      overrides: {
        result: "RESULT_CANCELLED",
        state: "STATE_STARTED",
        outputs: { failed: [output], passed: [output] },
      },
      expected: "cancelled",
    },
    ...(["STATE_PENDING", "STATE_STARTED"] as const).map((state) => ({
      name: `${state} overrides outputs`,
      overrides: { state, outputs: { failed: [output], passed: [output] } },
      expected: "running" as const,
    })),
    {
      name: "failed output overrides passed output and success metadata",
      overrides: { outputs: { failed: [output], passed: [output] } },
      expected: "failed",
    },
    {
      name: "passed output overrides failed metadata",
      overrides: { outputs: { passed: [output] }, metadata: { pipeline: { status: "failed" } } },
      expected: "passed",
    },
    {
      name: "empty failed output falls through to passed output",
      overrides: { outputs: { failed: [], passed: [output] }, metadata: { pipeline: { status: "failed" } } },
      expected: "passed",
    },
    {
      name: "empty output arrays fall through to metadata",
      overrides: { outputs: { failed: [], passed: [] } },
      expected: "passed",
    },
    {
      name: "resolved execution errors fall through to metadata",
      overrides: {
        result: "RESULT_FAILED",
        resultReason: "RESULT_REASON_ERROR_RESOLVED",
        resultMessage: "Pipeline request recovered",
      },
      expected: "passed",
    },
    {
      name: "a failed execution without a message falls through to metadata",
      overrides: { result: "RESULT_FAILED" },
      expected: "passed",
    },
    {
      name: "an explicit error without a message falls through to metadata",
      overrides: { result: "RESULT_FAILED", resultReason: "RESULT_REASON_ERROR" },
      expected: "passed",
    },
  ];

  it.each(precedenceCases)("$name", ({ overrides, expected }) => {
    const execution = buildExecution({ metadata: { pipeline: { status: "success" } }, ...overrides });
    expect(runPipelineStateFunction(execution)).toBe(expected);
  });
});
