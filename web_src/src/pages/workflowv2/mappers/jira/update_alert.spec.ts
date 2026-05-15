import { describe, expect, it } from "vitest";

import type { ExecutionDetailsContext, ExecutionInfo, NodeInfo } from "../types";
import { updateAlertMapper } from "./update_alert";
import { eventStateRegistry } from "./index";

function buildNode(overrides?: Partial<NodeInfo>): NodeInfo {
  return {
    id: "n1",
    name: "Update alert",
    componentName: "jira.updateAlert",
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

describe("updateAlertMapper", () => {
  it("getExecutionDetails lists operations", () => {
    const details = updateAlertMapper.getExecutionDetails(
      buildDetailsCtx({
        outputs: {
          default: [{ data: { operations: ["patchMessage", "acknowledge"] } }],
        },
      }),
    );
    expect(details["Operations"]).toBe("patchMessage, acknowledge");
  });

  it("eventStateRegistry maps success to updated", () => {
    expect(eventStateRegistry.updateAlert.getState(buildExecution({ result: "RESULT_PASSED" }))).toBe("updated");
  });
});
