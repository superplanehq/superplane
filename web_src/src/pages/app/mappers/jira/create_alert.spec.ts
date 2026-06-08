import { describe, expect, it } from "vitest";

import type { ExecutionDetailsContext, ExecutionInfo, NodeInfo } from "../types";
import { createAlertMapper } from "./create_alert";
import { eventStateRegistry } from "./index";

function buildNode(overrides?: Partial<NodeInfo>): NodeInfo {
  return {
    id: "n1",
    name: "Create alert",
    componentName: "jira.createAlert",
    isCollapsed: false,
    configuration: { message: "CPU", priority: "P1" },
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

describe("createAlertMapper", () => {
  it("getExecutionDetails surfaces core alert payload fields", () => {
    const details = createAlertMapper.getExecutionDetails(
      buildDetailsCtx({
        outputs: {
          default: [
            {
              type: "jira.alert.created",
              timestamp: new Date().toISOString(),
              data: {
                message: "Disk full",
                description: "Boot volume",
                status: "open",
                priority: "P2",
              },
            },
          ],
        },
      }),
    );
    expect(details.Message).toBe("Disk full");
    expect(details.Description).toBe("Boot volume");
    expect(details.Status).toBe("open");
    expect(details.Priority).toBe("P2");
  });

  it("eventStateRegistry maps success to created", () => {
    expect(eventStateRegistry.createAlert.getState(buildExecution({ result: "RESULT_PASSED" }))).toBe("created");
  });
});
