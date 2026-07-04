import { describe, expect, it } from "vitest";

import type { ExecutionDetailsContext, ExecutionInfo, NodeInfo } from "../types";
import { deleteHeartbeatMapper } from "./delete_heartbeat";
import { eventStateRegistry } from "./index";

function buildNode(): NodeInfo {
  return {
    id: "node-1",
    name: "Delete",
    componentName: "jira.deleteHeartbeat",
    isCollapsed: false,
    configuration: { team: "team-1", heartbeat: "DNS Checker" },
    metadata: {},
  };
}

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

describe("deleteHeartbeatMapper", () => {
  it("getExecutionDetails includes deleted and name", () => {
    const ctx: ExecutionDetailsContext = {
      nodes: [buildNode()],
      node: buildNode(),
      execution: buildExecution({
        outputs: {
          default: [
            {
              type: "jira.heartbeat.deleted",
              timestamp: new Date().toISOString(),
              data: { deleted: true, name: "DNS Checker" },
            },
          ],
        },
      }),
    };
    const details = deleteHeartbeatMapper.getExecutionDetails!(ctx);
    expect(details["Deleted"]).toBe("true");
    expect(details["Name"]).toBe("DNS Checker");
  });

  it("event state registry maps success to deleted", () => {
    expect(eventStateRegistry.deleteHeartbeat.getState(buildExecution({ result: "RESULT_PASSED" }))).toBe("deleted");
  });
});
