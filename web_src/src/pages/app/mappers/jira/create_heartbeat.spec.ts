import { describe, expect, it } from "vitest";

import type { ComponentDefinition, ExecutionInfo, NodeInfo } from "../types";
import { createHeartbeatMapper } from "./create_heartbeat";
import { eventStateRegistry } from "./index";

function buildNode(overrides?: Partial<NodeInfo>): NodeInfo {
  return {
    id: "node-1",
    name: "Heartbeat",
    componentName: "jira.createHeartbeat",
    isCollapsed: false,
    configuration: {},
    metadata: {},
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

const defaultDefinition: ComponentDefinition = {
  name: "jira.createHeartbeat",
  label: "Create Heartbeat",
  description: "",
  icon: "jira",
  color: "orange",
};

describe("createHeartbeatMapper", () => {
  it("props metadata shows team, name, and interval", () => {
    const props = createHeartbeatMapper.props!({
      nodes: [],
      node: buildNode({
        configuration: { team: "team-1", name: "DNS Checker", interval: 5, intervalUnit: "minutes" },
        metadata: { teamName: "On-call" },
      }),
      componentDefinition: defaultDefinition,
      lastExecutions: [],
      currentUser: undefined,
      actions: { invokeNodeExecutionHook: async () => {} },
    });
    expect(props.metadata?.map((m) => m.label)).toEqual(["On-call", "DNS Checker", "5 minutes"]);
  });

  it("getExecutionDetails includes name and status", () => {
    const details = createHeartbeatMapper.getExecutionDetails!({
      nodes: [buildNode()],
      node: buildNode(),
      execution: buildExecution({
        outputs: {
          default: [
            {
              type: "jira.heartbeat.created",
              timestamp: new Date().toISOString(),
              data: { name: "DNS Checker", status: "Pending" },
            },
          ],
        },
      }),
    });
    expect(details["Name"]).toBe("DNS Checker");
    expect(details["Status"]).toBe("Pending");
  });

  it("event state registry maps success to created", () => {
    expect(eventStateRegistry.createHeartbeat.getState(buildExecution({ result: "RESULT_PASSED" }))).toBe("created");
  });
});
