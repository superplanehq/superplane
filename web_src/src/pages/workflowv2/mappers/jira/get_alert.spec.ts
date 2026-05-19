import { describe, expect, it } from "vitest";

import type { ExecutionDetailsContext, ExecutionInfo, NodeInfo } from "../types";
import { getAlertMapper } from "./get_alert";
import { eventStateRegistry } from "./index";

function buildNode(overrides?: Partial<NodeInfo>): NodeInfo {
  return {
    id: "n1",
    name: "Get alert",
    componentName: "jira.getAlert",
    isCollapsed: false,
    configuration: { alert: "a1" },
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

describe("getAlertMapper", () => {
  it("getExecutionDetails surfaces core alert fields incl. description without tiny id", () => {
    const details = getAlertMapper.getExecutionDetails(
      buildDetailsCtx({
        outputs: {
          default: [
            {
              data: {
                message: "m1",
                description: "d1",
                status: "open",
                priority: "P2",
                tinyId: "42",
              },
            },
          ],
        },
      }),
    );
    expect(details.Message).toBe("m1");
    expect(details.Description).toBe("d1");
    expect(details.Status).toBe("open");
    expect(details.Priority).toBe("P2");
    expect(details["Tiny ID"]).toBeUndefined();
  });

  it("eventStateRegistry maps success to fetched", () => {
    expect(eventStateRegistry.getAlert.getState(buildExecution({ result: "RESULT_PASSED" }))).toBe("fetched");
  });
});
