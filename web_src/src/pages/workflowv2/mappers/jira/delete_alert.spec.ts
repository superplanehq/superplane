import { describe, expect, it } from "vitest";

import type { ExecutionDetailsContext, ExecutionInfo, NodeInfo } from "../types";
import { deleteAlertMapper } from "./delete_alert";
import { eventStateRegistry } from "./index";

function buildNode(overrides?: Partial<NodeInfo>): NodeInfo {
  return {
    id: "n1",
    name: "Delete alert",
    componentName: "jira.deleteAlert",
    isCollapsed: false,
    configuration: {},
    metadata: {},
    ...overrides,
  };
}

function buildExecution(overrides?: Partial<ExecutionInfo>): ExecutionInfo {
  return {
    id: "e1",
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

function buildDetailsCtx(execution: Partial<ExecutionInfo>): ExecutionDetailsContext {
  const node = buildNode();
  return { nodes: [], node, execution: buildExecution(execution) };
}

describe("deleteAlertMapper", () => {
  it("getExecutionDetails surfaces executed at and deleted flag", () => {
    const createdAt = "2026-01-15T10:00:00.000Z";
    const details = deleteAlertMapper.getExecutionDetails(
      buildDetailsCtx({
        createdAt,
        outputs: {
          default: [{ data: { deleted: true, alertId: "a9", requestId: "req-del" } }],
        },
      }),
    );
    expect(details["Executed At"]).toBe(new Date(createdAt).toLocaleString());
    expect(details.Deleted).toBe("Yes");
  });

  it("eventStateRegistry maps success to deleted", () => {
    expect(eventStateRegistry.deleteAlert.getState(buildExecution({ result: "RESULT_PASSED" }))).toBe("deleted");
  });
});
