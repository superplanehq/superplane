import { describe, expect, it } from "vitest";

import type { ExecutionDetailsContext, ExecutionInfo, NodeInfo } from "../types";
import { pingHeartbeatMapper } from "./ping_heartbeat";
import { eventStateRegistry } from "./index";

function buildNode(overrides?: Partial<NodeInfo>): NodeInfo {
  return {
    id: "node-1",
    name: "Ping",
    componentName: "jira.pingHeartbeat",
    isCollapsed: false,
    configuration: { team: "team-1", heartbeat: "DNS Checker" },
    metadata: { teamName: "On-call" },
    ...overrides,
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

describe("pingHeartbeatMapper", () => {
  it("getExecutionDetails includes message", () => {
    const ctx: ExecutionDetailsContext = {
      nodes: [buildNode()],
      node: buildNode(),
      execution: buildExecution({
        outputs: {
          default: [
            {
              type: "jira.heartbeat.pinged",
              timestamp: new Date().toISOString(),
              data: { message: "PONG - Heartbeat received" },
            },
          ],
        },
      }),
    };
    expect(pingHeartbeatMapper.getExecutionDetails!(ctx)["Message"]).toBe("PONG - Heartbeat received");
  });

  it("event state registry maps success to pinged", () => {
    expect(eventStateRegistry.pingHeartbeat.getState(buildExecution({ result: "RESULT_PASSED" }))).toBe("pinged");
  });
});
