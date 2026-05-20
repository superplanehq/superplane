import { describe, expect, it } from "vitest";

import type { ExecutionDetailsContext, ExecutionInfo, NodeInfo } from "../types";
import { getIncidentMapper } from "./get_incident";
import { eventStateRegistry } from "./index";

function buildNode(overrides?: Partial<NodeInfo>): NodeInfo {
  return {
    id: "n1",
    name: "Get",
    componentName: "jira.getIncident",
    isCollapsed: false,
    configuration: { issue: "IT-1", project: "IT" },
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

describe("getIncidentMapper", () => {
  it("getExecutionDetails reads summary and status", () => {
    const ctx: ExecutionDetailsContext = {
      nodes: [],
      node: buildNode(),
      execution: buildExecution({
        outputs: {
          default: [
            {
              type: "jira.incident.fetched",
              timestamp: new Date().toISOString(),
              data: { summary: "Sev1", status: { name: "Open" } },
            },
          ],
        },
      }),
    };
    const d = getIncidentMapper.getExecutionDetails(ctx);
    expect(d.Summary).toBe("Sev1");
    expect(d.Status).toBe("Open");
  });

  it("event state registry maps success to fetched", () => {
    const exec = buildExecution({ result: "RESULT_PASSED" });
    expect(eventStateRegistry.getIncident.getState(exec)).toBe("fetched");
  });
});
