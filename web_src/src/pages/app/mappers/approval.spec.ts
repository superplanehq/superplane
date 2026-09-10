import { describe, expect, it } from "vitest";
import { approvalMapper } from "./approval";
import type { ExecutionDetailsContext, ExecutionInfo, SubtitleContext } from "./types";

const DEFAULT_NODE = { id: "n1", name: "Approval Node", componentName: "approval", isCollapsed: false };

function makeExecution(overrides?: Partial<ExecutionInfo>): ExecutionInfo {
  return {
    id: "e1",
    createdAt: "",
    updatedAt: "",
    state: "STATE_FINISHED",
    result: "RESULT_PASSED",
    resultReason: "RESULT_REASON_OK",
    resultMessage: "",
    metadata: {},
    configuration: {},
    outputs: {},
    rootEvent: undefined,
    ...overrides,
  };
}

function makeSubtitleContext(execution: ExecutionInfo): SubtitleContext {
  return { node: DEFAULT_NODE, execution };
}

function makeExecutionDetailsContext(execution: ExecutionInfo): ExecutionDetailsContext {
  return { execution };
}

describe("approvalMapper.subtitle", () => {
  it("does not throw when metadata exists but records is missing (STATE_STARTED)", () => {
    const ctx = makeSubtitleContext(
      makeExecution({
        state: "STATE_STARTED",
        metadata: { result: "approved" },
      }),
    );

    expect(() => approvalMapper.subtitle(ctx)).not.toThrow();
    expect(approvalMapper.subtitle(ctx)).toBe("");
  });

  it("renders progress string for in-progress approvals (STATE_STARTED)", () => {
    const ctx = makeSubtitleContext(
      makeExecution({
        state: "STATE_STARTED",
        metadata: {
          result: "approved",
          records: [{ index: 0, state: "approved", type: "user" }],
        },
      }),
    );

    expect(approvalMapper.subtitle(ctx)).toBe("1/1 approved");
  });
});

describe("approvalMapper.getExecutionDetails", () => {
  it("does not throw when metadata exists but records is missing", () => {
    const ctx = makeExecutionDetailsContext(
      makeExecution({
        createdAt: "2026-08-24T10:00:00.000Z",
        updatedAt: "2026-08-24T10:05:00.000Z",
        metadata: { result: "approved" },
      }),
    );

    expect(() => approvalMapper.getExecutionDetails(ctx)).not.toThrow();
    expect(approvalMapper.getExecutionDetails(ctx)).toMatchObject({
      "Started at": expect.any(String),
      "Finished at": expect.any(String),
    });
    expect(approvalMapper.getExecutionDetails(ctx)["State"]).toBeUndefined();
  });
});
